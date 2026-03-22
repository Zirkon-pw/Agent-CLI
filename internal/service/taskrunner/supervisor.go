package taskrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docup/agentctl/internal/config/loader"
	"github.com/docup/agentctl/internal/core/clarification"
	rt "github.com/docup/agentctl/internal/core/runtime"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/core/validation"
	"github.com/docup/agentctl/internal/infra/events"
	"github.com/docup/agentctl/internal/infra/fsstore"
	infrart "github.com/docup/agentctl/internal/infra/runtime"
	"github.com/docup/agentctl/internal/service/contextpack"
	"github.com/docup/agentctl/internal/service/prompting"
	"gopkg.in/yaml.v3"
)

// TaskSupervisor runs stage-based task sessions using structured adapter protocols.
type TaskSupervisor struct {
	taskStore      *fsstore.TaskStore
	runStore       *fsstore.RunStore
	clarStore      *fsstore.ClarificationStore
	registry       *infrart.Registry
	heartbeatMgr   *infrart.HeartbeatManager
	eventSink      *events.Sink
	contextBuilder *contextpack.Builder
	promptBuilder  *prompting.Builder
	config         *loader.ProjectConfig
	adapters       *AgentAdapterRegistry
	projectRoot    string
}

// NewTaskSupervisor constructs a new stage-based task supervisor.
func NewTaskSupervisor(
	taskStore *fsstore.TaskStore,
	runStore *fsstore.RunStore,
	clarStore *fsstore.ClarificationStore,
	registry *infrart.Registry,
	heartbeatMgr *infrart.HeartbeatManager,
	eventSink *events.Sink,
	contextBuilder *contextpack.Builder,
	promptBuilder *prompting.Builder,
	config *loader.ProjectConfig,
	adapters *AgentAdapterRegistry,
	projectRoot string,
) *TaskSupervisor {
	return &TaskSupervisor{
		taskStore:      taskStore,
		runStore:       runStore,
		clarStore:      clarStore,
		registry:       registry,
		heartbeatMgr:   heartbeatMgr,
		eventSink:      eventSink,
		contextBuilder: contextBuilder,
		promptBuilder:  promptBuilder,
		config:         config,
		adapters:       adapters,
		projectRoot:    projectRoot,
	}
}

// Run executes or resumes the session pipeline for a task until it blocks or reaches review/completion.
func (s *TaskSupervisor) Run(ctx context.Context, t *task.Task) (*rt.RunSession, error) {
	session, err := s.loadOrCreateSession(t)
	if err != nil {
		return nil, err
	}

	for {
		lastStage := session.LastStage()
		if (session.ReviewReport != nil || (lastStage != nil && lastStage.Type == rt.StageTypeReview && lastStage.State == rt.StageStateCompleted)) &&
			session.Status != rt.SessionStatusReviewing {
			session.Status = rt.SessionStatusReviewing
			t.Status = task.StatusReviewing
			t.UpdatedAt = time.Now()
			if err := s.persistSession(t, session); err != nil {
				return nil, err
			}
		}

		if session.PendingHandoff != nil {
			if err := s.runSyntheticHandoff(t, session); err != nil {
				return nil, err
			}
			continue
		}

		stageType, err := s.nextStageType(t, session)
		if err != nil {
			return nil, err
		}
		if stageType == "" {
			if err := s.persistSession(t, session); err != nil {
				return nil, err
			}
			return session, nil
		}

		spec, err := s.prepareStageSpec(t, session, stageType)
		if err != nil {
			return nil, err
		}
		if err := s.runAdapterStage(ctx, t, session, spec); err != nil {
			return nil, err
		}

		if session.Status == rt.SessionStatusWaitingClarification ||
			session.Status == rt.SessionStatusPaused ||
			session.Status == rt.SessionStatusCanceled ||
			session.Status == rt.SessionStatusHandoffPending ||
			session.Status == rt.SessionStatusReviewing ||
			session.Status == rt.SessionStatusCompleted ||
			session.Status == rt.SessionStatusFailed {
			if err := s.persistSession(t, session); err != nil {
				return nil, err
			}
			return session, nil
		}

		if stageType == rt.StageTypeExecute || stageType == rt.StageTypeValidateFix {
			report, err := s.runValidation(ctx, t, session)
			if err != nil {
				return nil, err
			}
			if report != nil && !report.AllPassed {
				if t.Validation.Mode == task.ValidationModeFull && session.Validation.Attempt < t.Validation.MaxRetries {
					session.Status = rt.SessionStatusQueued
					session.UpdatedAt = time.Now()
					if err := s.persistSession(t, session); err != nil {
						return nil, err
					}
					continue
				}
				session.Status = rt.SessionStatusFailed
				t.Status = task.StatusFailed
				t.UpdatedAt = time.Now()
				if err := s.persistSession(t, session); err != nil {
					return nil, err
				}
				return session, nil
			}
		}
	}
}

func (s *TaskSupervisor) loadOrCreateSession(t *task.Task) (*rt.RunSession, error) {
	if t.Status == task.StatusWaitingClarification || t.Status == task.StatusHandoffPending || t.Status == task.StatusPaused {
		session, err := s.runStore.LatestSession(t.ID)
		if err == nil {
			return session, nil
		}
	}

	runID, err := s.runStore.NextRunID(t.ID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	session := &rt.RunSession{
		ID:             runID,
		TaskID:         t.ID,
		Status:         rt.SessionStatusQueued,
		CurrentAgentID: t.Agent,
		Validation: rt.ValidationState{
			MaxRetries: t.Validation.MaxRetries,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.runStore.SaveSession(session); err != nil {
		return nil, err
	}
	if err := s.runStore.SaveArtifactManifest(t.ID, session.ID, &session.ArtifactManifest); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *TaskSupervisor) nextStageType(t *task.Task, session *rt.RunSession) (rt.StageType, error) {
	if session.ReviewReport != nil || session.Status == rt.SessionStatusReviewing {
		return "", nil
	}
	if session.Status == rt.SessionStatusWaitingClarification {
		if t.Clarifications.PendingRequest != nil {
			return "", nil
		}
		return rt.StageTypeExecute, nil
	}
	if session.Status == rt.SessionStatusPaused || session.Status == rt.SessionStatusCanceled ||
		session.Status == rt.SessionStatusCompleted || session.Status == rt.SessionStatusFailed {
		return "", nil
	}
	if session.Validation.Attempt > 0 && !allPassed(session.Validation.LastResults) {
		return rt.StageTypeValidateFix, nil
	}
	last := session.LastStage()
	if last == nil {
		return rt.StageTypeExecute, nil
	}
	if last.Type == rt.StageTypeReview && last.State == rt.StageStateCompleted {
		return "", nil
	}
	if last.State == rt.StageStateCompleted && (last.Type == rt.StageTypeExecute || last.Type == rt.StageTypeValidateFix) {
		return rt.StageTypeReview, nil
	}
	return rt.StageTypeExecute, nil
}

func (s *TaskSupervisor) prepareStageSpec(t *task.Task, session *rt.RunSession, stageType rt.StageType) (*rt.StageSpec, error) {
	stageID := fmt.Sprintf("STAGE-%03d", len(session.StageHistory)+1)
	stageDir := s.runStore.StageDir(t.ID, session.ID, stageID)
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return nil, fmt.Errorf("creating stage dir: %w", err)
	}

	contextDir := ""
	promptPath := ""

	switch stageType {
	case rt.StageTypeExecute:
		var err error
		contextDir, err = s.contextBuilder.Build(t)
		if err != nil {
			return nil, err
		}
		promptContent, err := s.promptBuilder.BuildPrompt(t, contextDir, stageDir)
		if err != nil {
			return nil, err
		}
		promptPath = filepath.Join(stageDir, "prompt.md")
		if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
			return nil, err
		}
	case rt.StageTypeValidateFix:
		var err error
		contextDir, err = s.contextBuilder.Build(t)
		if err != nil {
			return nil, err
		}
		promptPath = filepath.Join(stageDir, "prompt.md")
		if err := os.WriteFile(promptPath, []byte(s.buildValidationFixPrompt(t, session.Validation.LastResults, session.Validation.Attempt, t.Validation.MaxRetries)), 0644); err != nil {
			return nil, err
		}
	case rt.StageTypeReview:
		var err error
		contextDir, err = s.contextBuilder.Build(t)
		if err != nil {
			return nil, err
		}
		promptPath = filepath.Join(stageDir, "review_prompt.md")
		if err := os.WriteFile(promptPath, []byte(s.buildReviewPrompt(t)), 0644); err != nil {
			return nil, err
		}
	}

	taskPath := filepath.Join(stageDir, "task_snapshot.yml")
	taskData, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("marshaling task snapshot: %w", err)
	}
	if err := os.WriteFile(taskPath, taskData, 0644); err != nil {
		return nil, fmt.Errorf("writing task snapshot: %w", err)
	}

	spec := &rt.StageSpec{
		ProtocolVersion: "v1",
		SessionID:       session.ID,
		TaskID:          t.ID,
		RunID:           session.ID,
		StageID:         stageID,
		Type:            stageType,
		AgentID:         session.CurrentAgentID,
		WorkDir:         s.projectRoot,
		SessionDir:      s.runStore.RunDir(t.ID, session.ID),
		StageDir:        stageDir,
		TaskPath:        taskPath,
		ContextDir:      contextDir,
		PromptPath:      promptPath,
		Input: rt.StageInput{
			Task:             t,
			ArtifactManifest: session.ArtifactManifest,
			Clarifications:   append([]string{}, t.Clarifications.Attached...),
		},
	}

	switch stageType {
	case rt.StageTypeValidateFix:
		spec.Input.Validation = &rt.ValidationStageInput{
			Attempt:        session.Validation.Attempt,
			MaxRetries:     t.Validation.MaxRetries,
			FailedChecks:   failedChecks(session.Validation.LastResults),
			ValidationPath: session.Validation.LastReportPath,
		}
	case rt.StageTypeReview:
		spec.Input.Review = &rt.ReviewStageInput{
			SummaryPath:    s.findArtifactPath(session.ArtifactManifest, "summary.md"),
			DiffPath:       s.findArtifactPath(session.ArtifactManifest, "diff.patch"),
			ValidationPath: session.Validation.LastReportPath,
			ContextPath:    filepath.Join(contextDir, "context.md"),
		}
	}

	specPath := filepath.Join(stageDir, "stage_spec.json")
	if err := writeJSON(specPath, spec); err != nil {
		return nil, err
	}
	return spec, nil
}

func (s *TaskSupervisor) runAdapterStage(ctx context.Context, t *task.Task, session *rt.RunSession, spec *rt.StageSpec) error {
	adapter, err := s.adapters.Get(spec.AgentID)
	if err != nil {
		return err
	}
	if err := s.validateCapabilities(adapter.Capabilities(), spec.Type); err != nil {
		return err
	}

	stage := rt.StageRun{
		StageID: spec.StageID,
		Type:    spec.Type,
		AgentID: spec.AgentID,
		State:   rt.StageStatePending,
		Attempt: s.stageAttempt(session, spec.Type),
	}
	session.StageHistory = append(session.StageHistory, stage)
	now := time.Now()
	current := &session.StageHistory[len(session.StageHistory)-1]
	current.State = rt.StageStateRunning
	current.StartedAt = &now

	session.CurrentStageID = spec.StageID
	session.Status = rt.SessionStatusStageRunning
	session.CurrentAgentID = spec.AgentID
	session.UpdatedAt = now
	t.Status = task.StatusStageRunning
	t.UpdatedAt = now

	if err := s.persistSession(t, session); err != nil {
		return err
	}

	specPath := filepath.Join(spec.StageDir, "stage_spec.json")
	handle, err := adapter.Start(ctx, spec, specPath)
	if err != nil {
		return err
	}
	eventsCh := handle.Events()
	stderrCh := handle.Stderr()
	errorsCh := handle.Errors()
	doneCh := handle.Done()

	session.Capabilities = adapter.Capabilities()
	session.Recovery.AdapterPID = handle.PID()
	session.Recovery.ProcessGroupID = handle.ProcessGroupID()
	session.Recovery.LastHeartbeatAt = now
	session.UpdatedAt = time.Now()
	if err := s.persistSession(t, session); err != nil {
		return err
	}

	active := rt.ActiveRun{
		TaskID:         t.ID,
		RunID:          session.ID,
		SessionID:      session.ID,
		StageID:        spec.StageID,
		Agent:          spec.AgentID,
		Status:         session.Status,
		PID:            handle.PID(),
		ProcessGroupID: handle.ProcessGroupID(),
		StartedAt:      now,
		UpdatedAt:      now,
		Capabilities:   adapter.Capabilities(),
	}
	if err := s.registry.RegisterRun(active); err != nil {
		return err
	}
	defer s.registry.UnregisterRun(t.ID, session.ID)

	s.eventSink.Emit(t.ID, session.ID, "stage_started", string(spec.Type))
	_ = s.heartbeatMgr.Write(t.ID, session.ID)

	ticker := time.NewTicker(time.Duration(t.Runtime.HeartbeatIntervalSec) * time.Second)
	defer ticker.Stop()

	stderrLog := filepath.Join(spec.StageDir, "adapter.stderr.log")
	var gracefulDeadline *time.Time
	stageCompleted := false
	processExited := false
	var processErr error

	for !stageCompleted {
		select {
		case <-ctx.Done():
			_ = handle.Kill()
			current.State = rt.StageStateFailed
			current.Result = &rt.StageResult{Outcome: "failed", Message: ctx.Err().Error()}
			session.Status = rt.SessionStatusFailed
			t.Status = task.StatusFailed
			t.UpdatedAt = time.Now()
			return s.persistSession(t, session)

		case ev, ok := <-eventsCh:
			if !ok {
				eventsCh = nil
				if processExited && current.Result == nil {
					if processErr != nil {
						current.State = rt.StageStateFailed
						current.Result = &rt.StageResult{Outcome: "failed", Message: processErr.Error()}
						session.Status = rt.SessionStatusFailed
						t.Status = task.StatusFailed
					} else {
						current.State = rt.StageStateCompleted
						current.Result = &rt.StageResult{Outcome: "completed"}
						session.Status = rt.SessionStatusQueued
					}
					stageCompleted = true
				}
				continue
			}
			if err := s.handleProtocolEvent(t, session, current, spec, &ev); err != nil {
				return err
			}
			if ev.Type == rt.EventTypeStageCompleted {
				stageCompleted = true
			}

		case line, ok := <-stderrCh:
			if ok && line != "" {
				_ = appendFile(stderrLog, line+"\n")
			}
			if !ok {
				stderrCh = nil
			}

		case err, ok := <-errorsCh:
			if ok && err != nil {
				current.State = rt.StageStateFailed
				current.Result = &rt.StageResult{Outcome: "failed", Message: err.Error()}
				session.Status = rt.SessionStatusFailed
				t.Status = task.StatusFailed
				t.UpdatedAt = time.Now()
				return s.persistSession(t, session)
			}
			if !ok {
				errorsCh = nil
			}

		case err, ok := <-doneCh:
			if !ok {
				doneCh = nil
				continue
			}
			processExited = true
			processErr = err
			doneCh = nil
			if eventsCh == nil && current.Result == nil {
				if err != nil {
					current.State = rt.StageStateFailed
					current.Result = &rt.StageResult{Outcome: "failed", Message: err.Error()}
					session.Status = rt.SessionStatusFailed
					t.Status = task.StatusFailed
				} else {
					current.Result = &rt.StageResult{Outcome: "completed"}
					current.State = rt.StageStateCompleted
					session.Status = rt.SessionStatusQueued
				}
				stageCompleted = true
			}

		case <-ticker.C:
			now := time.Now()
			session.Recovery.LastHeartbeatAt = now
			_ = s.heartbeatMgr.Write(t.ID, session.ID)

			commands, err := s.registry.CommandsAfter(t.ID, session.LastCommandSeq)
			if err != nil {
				return err
			}
			for _, cmd := range commands {
				session.LastCommandSeq = cmd.Seq
				session.PendingControlCommand = &cmd
				session.UpdatedAt = time.Now()
				if err := s.persistSession(t, session); err != nil {
					return err
				}
				if err := handle.Send(cmd); err != nil {
					return err
				}
				switch cmd.Type {
				case rt.CommandTypePause:
					session.Status = rt.SessionStatusPaused
					t.Status = task.StatusPaused
					t.UpdatedAt = time.Now()
				case rt.CommandTypeCancel:
					deadline := time.Now().Add(time.Duration(t.Runtime.GracefulStopTimeoutSec) * time.Second)
					gracefulDeadline = &deadline
				case rt.CommandTypeResume:
					session.Status = rt.SessionStatusStageRunning
					t.Status = task.StatusStageRunning
					t.UpdatedAt = time.Now()
				case rt.CommandTypeKill:
					_ = handle.Kill()
					session.Status = rt.SessionStatusCanceled
					t.Status = task.StatusCanceled
					t.UpdatedAt = time.Now()
					stageCompleted = true
				}
			}
			if gracefulDeadline != nil && time.Now().After(*gracefulDeadline) {
				_ = handle.Kill()
				session.Status = rt.SessionStatusCanceled
				t.Status = task.StatusCanceled
				t.UpdatedAt = time.Now()
				stageCompleted = true
			}
		}
	}

	finished := time.Now()
	current.FinishedAt = &finished
	session.PendingControlCommand = nil
	session.CurrentStageID = ""
	session.UpdatedAt = finished
	if session.Status == rt.SessionStatusStageRunning {
		session.Status = rt.SessionStatusQueued
	}
	return s.persistSession(t, session)
}

func (s *TaskSupervisor) handleProtocolEvent(
	t *task.Task,
	session *rt.RunSession,
	stage *rt.StageRun,
	spec *rt.StageSpec,
	ev *rt.ProtocolEvent,
) error {
	if ev.SessionID == "" {
		ev.SessionID = session.ID
	}
	if ev.TaskID == "" {
		ev.TaskID = t.ID
	}
	if ev.RunID == "" {
		ev.RunID = session.ID
	}
	if ev.StageID == "" {
		ev.StageID = spec.StageID
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	if err := s.runStore.AppendProtocolEvent(t.ID, session.ID, ev); err != nil {
		return err
	}

	if ev.Seq > session.LastEventSeq {
		session.LastEventSeq = ev.Seq
	}
	session.Recovery.LastHeartbeatAt = time.Now()

	switch ev.Type {
	case rt.EventTypeHello:
		var payload rt.HelloPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			session.Capabilities = payload.Capabilities
		}
	case rt.EventTypeProgress:
		var payload rt.ProgressPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			s.eventSink.Emit(t.ID, session.ID, "progress", payload.Message)
		}
	case rt.EventTypeHeartbeat:
		_ = s.heartbeatMgr.Write(t.ID, session.ID)
	case rt.EventTypeArtifact:
		var payload rt.ArtifactPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}
		absPath := payload.Path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(spec.StageDir, payload.Path)
		}
		record := rt.ArtifactRecord{
			Name:      payload.Name,
			Kind:      payload.Kind,
			Path:      absPath,
			StageID:   spec.StageID,
			MediaType: payload.MediaType,
			CreatedAt: time.Now(),
		}
		session.ArtifactManifest.Add(record)
		if err := s.runStore.SaveArtifactManifest(t.ID, session.ID, &session.ArtifactManifest); err != nil {
			return err
		}
		if err := s.materializeSessionArtifact(t.ID, session.ID, record); err != nil {
			return err
		}
	case rt.EventTypeClarificationRequested:
		var payload rt.ClarificationRequestedPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}
		reqID := payload.RequestID
		if reqID == "" {
			reqID = fmt.Sprintf("CLAR-REQ-%03d", len(t.Clarifications.Attached)+1)
		}
		req := &clarification.Request{
			TaskID:      t.ID,
			RequestID:   reqID,
			CreatedBy:   spec.AgentID,
			Reason:      payload.Reason,
			Questions:   payload.Questions,
			ContextRefs: payload.ContextRefs,
			CreatedAt:   time.Now(),
		}
		path, err := s.clarStore.SaveRequest(req)
		if err != nil {
			return err
		}
		t.SetPendingClarification(reqID)
		t.Status = task.StatusWaitingClarification
		t.UpdatedAt = time.Now()
		session.PendingClarificationID = &reqID
		session.Status = rt.SessionStatusWaitingClarification
		session.ArtifactManifest.Add(rt.ArtifactRecord{
			Name:      filepath.Base(path),
			Kind:      "clarification_request",
			Path:      path,
			StageID:   spec.StageID,
			MediaType: "application/yaml",
			CreatedAt: time.Now(),
		})
		_ = s.runStore.SaveArtifactManifest(t.ID, session.ID, &session.ArtifactManifest)
		s.eventSink.Emit(t.ID, session.ID, "waiting_clarification", reqID)
	case rt.EventTypeReviewReport:
		var payload rt.ReviewReportPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}
		report := &rt.ReviewReport{
			StageID:      spec.StageID,
			Summary:      payload.Summary,
			Findings:     payload.Findings,
			ArtifactPath: payload.ArtifactPath,
			CreatedAt:    time.Now(),
		}
		session.ReviewReport = report
		data, _ := json.MarshalIndent(report, "", "  ")
		if err := s.runStore.WriteArtifact(t.ID, session.ID, "review_report.json", data); err == nil {
			session.ArtifactManifest.Add(rt.ArtifactRecord{
				Name:      "review_report.json",
				Kind:      "review_report",
				Path:      filepath.Join(s.runStore.RunDir(t.ID, session.ID), "review_report.json"),
				StageID:   spec.StageID,
				MediaType: "application/json",
				CreatedAt: time.Now(),
			})
			_ = s.runStore.SaveArtifactManifest(t.ID, session.ID, &session.ArtifactManifest)
		}
	case rt.EventTypeHandoffRequested:
		var payload rt.HandoffRequestedPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}
		session.PendingHandoff = &rt.PendingHandoff{
			NextAgentID: payload.NextAgentID,
			Reason:      payload.Reason,
			RequestedAt: time.Now(),
		}
		session.Status = rt.SessionStatusHandoffPending
		t.Status = task.StatusHandoffPending
		t.UpdatedAt = time.Now()
		s.eventSink.Emit(t.ID, session.ID, "handoff_pending", payload.NextAgentID)
	case rt.EventTypeWarning:
		var payload rt.ErrorPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			s.eventSink.Emit(t.ID, session.ID, "warning", payload.Message)
		}
	case rt.EventTypeError:
		var payload rt.ErrorPayload
		if err := json.Unmarshal(ev.Payload, &payload); err == nil {
			s.eventSink.Emit(t.ID, session.ID, "error", payload.Message)
		}
	case rt.EventTypeStageCompleted:
		var payload rt.StageCompletedPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}
		stage.Result = &payload.Result
		switch payload.Result.Outcome {
		case "completed":
			stage.State = rt.StageStateCompleted
			session.Status = rt.SessionStatusQueued
		case "clarification_requested":
			stage.State = rt.StageStateCompleted
			session.Status = rt.SessionStatusWaitingClarification
			t.Status = task.StatusWaitingClarification
		case "handoff_requested", "handoff_pending":
			stage.State = rt.StageStateCompleted
			session.Status = rt.SessionStatusHandoffPending
			t.Status = task.StatusHandoffPending
			if payload.Result.NextAgentID != "" && session.PendingHandoff == nil {
				session.PendingHandoff = &rt.PendingHandoff{
					NextAgentID: payload.Result.NextAgentID,
					RequestedAt: time.Now(),
				}
			}
		case "paused":
			stage.State = rt.StageStatePaused
			session.Status = rt.SessionStatusPaused
			t.Status = task.StatusPaused
		case "canceled":
			stage.State = rt.StageStateCanceled
			session.Status = rt.SessionStatusCanceled
			t.Status = task.StatusCanceled
		default:
			stage.State = rt.StageStateFailed
			session.Status = rt.SessionStatusFailed
			t.Status = task.StatusFailed
		}
		if payload.Result.ReviewPath != "" {
			session.ArtifactManifest.Add(rt.ArtifactRecord{
				Name:      filepath.Base(payload.Result.ReviewPath),
				Kind:      "review_report",
				Path:      payload.Result.ReviewPath,
				StageID:   spec.StageID,
				MediaType: "application/json",
				CreatedAt: time.Now(),
			})
		}
		if payload.Result.SummaryPath != "" {
			session.ArtifactManifest.Add(rt.ArtifactRecord{
				Name:      filepath.Base(payload.Result.SummaryPath),
				Kind:      "summary",
				Path:      payload.Result.SummaryPath,
				StageID:   spec.StageID,
				MediaType: "text/markdown",
				CreatedAt: time.Now(),
			})
		}
		if payload.Result.DiffPath != "" {
			session.ArtifactManifest.Add(rt.ArtifactRecord{
				Name:      filepath.Base(payload.Result.DiffPath),
				Kind:      "diff",
				Path:      payload.Result.DiffPath,
				StageID:   spec.StageID,
				MediaType: "text/x-diff",
				CreatedAt: time.Now(),
			})
		}
		_ = s.runStore.SaveArtifactManifest(t.ID, session.ID, &session.ArtifactManifest)
		s.eventSink.Emit(t.ID, session.ID, "stage_completed", payload.Result.Outcome)
	}

	return s.persistSession(t, session)
}

func (s *TaskSupervisor) runValidation(ctx context.Context, t *task.Task, session *rt.RunSession) (*validation.Report, error) {
	if len(t.Validation.Commands) == 0 {
		session.Validation.LastResults = nil
		session.Validation.LastReportPath = ""
		session.Validation.Attempt = 0
		session.UpdatedAt = time.Now()
		return nil, s.persistSession(t, session)
	}

	results := make([]validation.CheckResult, 0, len(t.Validation.Commands))
	for _, cmdStr := range t.Validation.Commands {
		results = append(results, runValidationCommand(ctx, s.projectRoot, cmdStr))
	}

	if !allPassed(results) {
		session.Validation.Attempt++
	} else {
		session.Validation.Attempt = 0
	}
	session.Validation.LastResults = results
	report := &validation.Report{
		TaskID:     t.ID,
		RunID:      session.ID,
		Mode:       string(t.Validation.Mode),
		MaxRetries: t.Validation.MaxRetries,
		CreatedAt:  time.Now(),
		Results:    results,
		AllPassed:  allPassed(results),
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := s.runStore.WriteArtifact(t.ID, session.ID, "validation.json", data); err != nil {
		return nil, err
	}
	session.Validation.LastReportPath = filepath.Join(s.runStore.RunDir(t.ID, session.ID), "validation.json")
	session.ArtifactManifest.Add(rt.ArtifactRecord{
		Name:      "validation.json",
		Kind:      "validation_report",
		Path:      session.Validation.LastReportPath,
		MediaType: "application/json",
		CreatedAt: time.Now(),
	})
	if err := s.runStore.SaveArtifactManifest(t.ID, session.ID, &session.ArtifactManifest); err != nil {
		return nil, err
	}
	if err := s.persistSession(t, session); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *TaskSupervisor) runSyntheticHandoff(t *task.Task, session *rt.RunSession) error {
	if session.PendingHandoff == nil {
		return nil
	}
	stageID := fmt.Sprintf("STAGE-%03d", len(session.StageHistory)+1)
	now := time.Now()
	stage := rt.StageRun{
		StageID:    stageID,
		Type:       rt.StageTypeHandoff,
		AgentID:    session.CurrentAgentID,
		State:      rt.StageStateCompleted,
		Attempt:    s.stageAttempt(session, rt.StageTypeHandoff),
		StartedAt:  &now,
		FinishedAt: &now,
		Result: &rt.StageResult{
			Outcome:     "completed",
			Message:     "handoff completed",
			NextAgentID: session.PendingHandoff.NextAgentID,
		},
	}
	session.StageHistory = append(session.StageHistory, stage)
	handoffPath := filepath.Join(s.runStore.RunDir(t.ID, session.ID), "handoff.json")
	if err := writeJSON(handoffPath, session.PendingHandoff); err != nil {
		return err
	}
	session.ArtifactManifest.Add(rt.ArtifactRecord{
		Name:      "handoff.json",
		Kind:      "handoff",
		Path:      handoffPath,
		StageID:   stageID,
		MediaType: "application/json",
		CreatedAt: now,
	})
	t.Agent = session.PendingHandoff.NextAgentID
	t.Status = task.StatusQueued
	t.UpdatedAt = now
	session.CurrentAgentID = t.Agent
	session.PendingHandoff = nil
	session.Status = rt.SessionStatusQueued
	session.UpdatedAt = now
	s.eventSink.Emit(t.ID, session.ID, "handoff_completed", t.Agent)
	return s.persistSession(t, session)
}

func (s *TaskSupervisor) persistSession(t *task.Task, session *rt.RunSession) error {
	session.UpdatedAt = time.Now()
	if err := s.runStore.SaveSession(session); err != nil {
		return err
	}
	if err := s.runStore.SaveArtifactManifest(t.ID, session.ID, &session.ArtifactManifest); err != nil {
		return err
	}
	return s.taskStore.Save(t)
}

func (s *TaskSupervisor) validateCapabilities(cap rt.AdapterCapabilities, stageType rt.StageType) error {
	switch stageType {
	case rt.StageTypeReview:
		if !cap.SupportsReview {
			return fmt.Errorf("adapter does not support review stages")
		}
	case rt.StageTypeExecute, rt.StageTypeValidateFix:
		if !cap.SupportsCancel || !cap.SupportsKill {
			return fmt.Errorf("adapter must support cancel and kill control commands")
		}
	}
	return nil
}

func (s *TaskSupervisor) stageAttempt(session *rt.RunSession, stageType rt.StageType) int {
	attempt := 0
	for _, stage := range session.StageHistory {
		if stage.Type == stageType {
			attempt++
		}
	}
	return attempt + 1
}

func (s *TaskSupervisor) materializeSessionArtifact(taskID, sessionID string, artifact rt.ArtifactRecord) error {
	base := filepath.Base(artifact.Path)
	if base == artifact.Name {
		return nil
	}
	if artifact.Name == "" {
		return nil
	}
	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		return nil
	}
	return s.runStore.WriteArtifact(taskID, sessionID, artifact.Name, data)
}

func (s *TaskSupervisor) buildValidationFixPrompt(t *task.Task, results []validation.CheckResult, attempt, maxRetries int) string {
	var buf bytes.Buffer
	buf.WriteString("# Validation Fix Required\n\n")
	buf.WriteString(fmt.Sprintf("Task: %s\n", t.ID))
	buf.WriteString(fmt.Sprintf("Goal: %s\n", t.Goal))
	buf.WriteString(fmt.Sprintf("Attempt: %d/%d\n\n", attempt, maxRetries))
	buf.WriteString("Fix the failing validation checks described in the structured stage input.\n\n")
	for _, res := range results {
		if !res.Passed {
			buf.WriteString(fmt.Sprintf("- %s (exit=%d)\n", res.Command, res.ExitCode))
		}
	}
	return buf.String()
}

func (s *TaskSupervisor) buildReviewPrompt(t *task.Task) string {
	return strings.TrimSpace(fmt.Sprintf(`
# Reviewer Stage

Review the produced summary, diff and validation report for task %s.
Return a structured review report with findings and an overall summary.
`, t.ID))
}

func (s *TaskSupervisor) findArtifactPath(manifest rt.ArtifactManifest, name string) string {
	for i := len(manifest.Items) - 1; i >= 0; i-- {
		if manifest.Items[i].Name == name {
			return manifest.Items[i].Path
		}
	}
	return ""
}

func failedChecks(results []validation.CheckResult) []validation.CheckResult {
	var failed []validation.CheckResult
	for _, result := range results {
		if !result.Passed {
			failed = append(failed, result)
		}
	}
	return failed
}

func allPassed(results []validation.CheckResult) bool {
	for _, result := range results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func runValidationCommand(ctx context.Context, projectRoot, cmdStr string) validation.CheckResult {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return validation.CheckResult{
		Command:  cmdStr,
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		Passed:   exitCode == 0,
	}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func appendFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
