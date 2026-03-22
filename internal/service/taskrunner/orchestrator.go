package taskrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/docup/agentctl/internal/core/run"
	rt "github.com/docup/agentctl/internal/core/runtime"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/events"
	"github.com/docup/agentctl/internal/infra/executor"
	"github.com/docup/agentctl/internal/infra/fsstore"
	infrart "github.com/docup/agentctl/internal/infra/runtime"
	"github.com/docup/agentctl/internal/service/contextpack"
	"github.com/docup/agentctl/internal/service/prompting"
	"github.com/docup/agentctl/internal/service/validationrunner"
)

// Orchestrator coordinates the full task execution pipeline.
type Orchestrator struct {
	taskStore      *fsstore.TaskStore
	runStore       *fsstore.RunStore
	registry       *infrart.Registry
	heartbeatMgr   *infrart.HeartbeatManager
	eventSink      *events.Sink
	contextBuilder *contextpack.Builder
	promptBuilder  *prompting.Builder
	executor       *executor.AgentExecutor
	validator      *validationrunner.Runner
	agentctlDir    string
	projectRoot    string
}

// NewOrchestrator creates a task runner orchestrator.
func NewOrchestrator(
	taskStore *fsstore.TaskStore,
	runStore *fsstore.RunStore,
	registry *infrart.Registry,
	heartbeatMgr *infrart.HeartbeatManager,
	eventSink *events.Sink,
	contextBuilder *contextpack.Builder,
	promptBuilder *prompting.Builder,
	exec *executor.AgentExecutor,
	validator *validationrunner.Runner,
	agentctlDir string,
	projectRoot string,
) *Orchestrator {
	return &Orchestrator{
		taskStore:      taskStore,
		runStore:       runStore,
		registry:       registry,
		heartbeatMgr:   heartbeatMgr,
		eventSink:      eventSink,
		contextBuilder: contextBuilder,
		promptBuilder:  promptBuilder,
		executor:       exec,
		validator:      validator,
		agentctlDir:    agentctlDir,
		projectRoot:    projectRoot,
	}
}

// Run executes the full task pipeline.
func (o *Orchestrator) Run(ctx context.Context, taskID string) error {
	t, err := o.taskStore.Load(taskID)
	if err != nil {
		return err
	}

	// Transition to queued
	if t.Status == task.StatusDraft || t.Status == task.StatusReadyToResume ||
		t.Status == task.StatusPaused || t.Status == task.StatusStopped || t.Status == task.StatusKilled {
		if err := t.TransitionTo(task.StatusQueued); err != nil {
			return fmt.Errorf("cannot run task: %w", err)
		}
		o.taskStore.Save(t)
	}

	o.eventSink.Emit(taskID, "", "queued", "")

	// Prepare context
	if err := t.TransitionTo(task.StatusPreparingContext); err != nil {
		return err
	}
	o.taskStore.Save(t)
	o.eventSink.Emit(taskID, "", "preparing_context", "")

	contextDir, err := o.contextBuilder.Build(t)
	if err != nil {
		t.TransitionTo(task.StatusFailed)
		o.taskStore.Save(t)
		return fmt.Errorf("building context: %w", err)
	}
	o.eventSink.Emit(taskID, "", "context_prepared", contextDir)

	// Create run
	runID, err := o.runStore.NextRunID(taskID)
	if err != nil {
		return err
	}
	runDir := o.runStore.RunDir(taskID, runID)

	// Build prompt
	promptContent, err := o.promptBuilder.BuildPrompt(t, contextDir, runDir)
	if err != nil {
		t.TransitionTo(task.StatusFailed)
		o.taskStore.Save(t)
		return fmt.Errorf("building prompt: %w", err)
	}

	// Create run entity
	r := &run.Run{
		ID:              runID,
		TaskID:          taskID,
		Status:          run.RunStatusPending,
		Agent:           t.Agent,
		PromptFile:      filepath.Join(runDir, "prompt.md"),
		TemplateLockFile: filepath.Join(runDir, "prompt_template_lock.yml"),
		Clarifications:  t.Clarifications.Attached,
		CreatedAt:       time.Now(),
	}

	// Transition to running
	if err := t.TransitionTo(task.StatusRunning); err != nil {
		return err
	}
	o.taskStore.Save(t)

	// Register in runtime
	activeRun := rt.ActiveRun{
		TaskID:    taskID,
		RunID:     runID,
		Agent:     t.Agent,
		StartedAt: time.Now(),
	}
	if err := o.registry.RegisterRun(activeRun); err != nil {
		return fmt.Errorf("registering run: %w", err)
	}
	o.eventSink.Emit(taskID, runID, "running", fmt.Sprintf("agent=%s", t.Agent))

	// Start heartbeat goroutine
	hbCtx, hbCancel := context.WithCancel(ctx)
	defer hbCancel()
	go o.heartbeatLoop(hbCtx, taskID, runID, time.Duration(t.Runtime.HeartbeatIntervalSec)*time.Second)

	// Execute agent
	result, err := o.executor.ExecuteWithPromptFile(ctx, t.Agent, promptContent, o.projectRoot, taskID, runID, o.agentctlDir)
	hbCancel()

	if err != nil {
		slog.Error("agent execution failed", "task", taskID, "error", err)
		r.TransitionTo(run.RunStatusFailed)
		o.runStore.Save(r)
		o.registry.UnregisterRun(taskID, runID)
		t.TransitionTo(task.StatusFailed)
		o.taskStore.Save(t)
		o.eventSink.Emit(taskID, runID, "failed", err.Error())
		return fmt.Errorf("executing agent: %w", err)
	}

	r.MarkStarted(result.PID)
	r.MarkFinished(result.ExitCode, "completed")
	o.runStore.Save(r)

	// Save stdout/stderr as logs
	o.runStore.WriteArtifact(taskID, runID, "logs.txt", []byte(result.Stdout+"\n---STDERR---\n"+result.Stderr))

	o.eventSink.Emit(taskID, runID, "execution_completed", fmt.Sprintf("exit_code=%d", result.ExitCode))

	// Unregister from runtime
	o.registry.UnregisterRun(taskID, runID)

	// Check if agent requested clarification
	clarReqPath := filepath.Join(runDir, "clarification_request.yml")
	if _, err := os.Stat(clarReqPath); err == nil {
		t.TransitionTo(task.StatusNeedsClarification)
		o.taskStore.Save(t)
		o.eventSink.Emit(taskID, runID, "needs_clarification", "")
		return nil
	}

	// Run validation
	if len(t.Validation.Commands) > 0 {
		if err := t.TransitionTo(task.StatusValidating); err != nil {
			return err
		}
		o.taskStore.Save(t)
		o.eventSink.Emit(taskID, runID, "validating", "")

		report, err := o.validator.Validate(ctx, t, r)
		if err != nil {
			slog.Error("validation failed", "task", taskID, "error", err)
		}

		// Save validation report
		reportData, _ := json.MarshalIndent(report, "", "  ")
		o.runStore.WriteArtifact(taskID, runID, "validation.json", reportData)

		if report != nil && !report.AllPassed {
			o.eventSink.Emit(taskID, runID, "validation_failed", fmt.Sprintf("retries=%d/%d", report.TotalRetries, report.MaxRetries))
			if report.CanRetry() && t.Validation.Mode == task.ValidationModeFull {
				// Will be handled by validation runner retry loop
				slog.Info("validation failed, retries exhausted or mode is simple", "task", taskID)
			}
			t.TransitionTo(task.StatusFailed)
			o.taskStore.Save(t)
			return nil
		}

		o.eventSink.Emit(taskID, runID, "validation_passed", "")
	}

	// Move to review
	if err := t.TransitionTo(task.StatusReview); err != nil {
		return err
	}
	o.taskStore.Save(t)
	o.eventSink.Emit(taskID, runID, "review", "awaiting human review")

	return nil
}

func (o *Orchestrator) heartbeatLoop(ctx context.Context, taskID, runID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.heartbeatMgr.Write(taskID, runID)
		}
	}
}

// Stop sends a graceful stop signal to a running task.
func (o *Orchestrator) Stop(taskID string) error {
	t, err := o.taskStore.Load(taskID)
	if err != nil {
		return err
	}
	if err := t.TransitionTo(task.StatusStopping); err != nil {
		return fmt.Errorf("cannot stop task: %w", err)
	}
	o.taskStore.Save(t)
	o.registry.WriteSignal(taskID, rt.SignalStop)
	o.eventSink.Emit(taskID, "", "stopping", "graceful stop requested")
	return nil
}

// Kill sends a forced kill signal.
func (o *Orchestrator) Kill(taskID string) error {
	t, err := o.taskStore.Load(taskID)
	if err != nil {
		return err
	}
	if err := t.TransitionTo(task.StatusKilled); err != nil {
		return fmt.Errorf("cannot kill task: %w", err)
	}
	o.taskStore.Save(t)
	o.registry.WriteSignal(taskID, rt.SignalKill)
	o.registry.UnregisterRun(taskID, "")
	o.eventSink.Emit(taskID, "", "killed", "forced kill")
	return nil
}

// Pause sends a pause signal.
func (o *Orchestrator) Pause(taskID string) error {
	t, err := o.taskStore.Load(taskID)
	if err != nil {
		return err
	}
	if !t.Runtime.AllowPause {
		return fmt.Errorf("pause is not allowed for task %s", taskID)
	}
	if err := t.TransitionTo(task.StatusPausing); err != nil {
		return fmt.Errorf("cannot pause task: %w", err)
	}
	o.taskStore.Save(t)
	o.registry.WriteSignal(taskID, rt.SignalPause)
	o.eventSink.Emit(taskID, "", "pausing", "pause requested")
	return nil
}

// Cancel cancels a task that is not actively running.
func (o *Orchestrator) Cancel(taskID string) error {
	t, err := o.taskStore.Load(taskID)
	if err != nil {
		return err
	}
	if err := t.TransitionTo(task.StatusCanceled); err != nil {
		return fmt.Errorf("cannot cancel task: %w", err)
	}
	o.taskStore.Save(t)
	o.eventSink.Emit(taskID, "", "canceled", "")
	return nil
}

// Accept marks a task as completed after review.
func (o *Orchestrator) Accept(taskID string) error {
	t, err := o.taskStore.Load(taskID)
	if err != nil {
		return err
	}
	if err := t.TransitionTo(task.StatusCompleted); err != nil {
		return fmt.Errorf("cannot accept task: %w", err)
	}
	o.taskStore.Save(t)
	o.eventSink.Emit(taskID, "", "completed", "accepted")
	return nil
}

// Reject marks a task as rejected after review.
func (o *Orchestrator) Reject(taskID, reason string) error {
	t, err := o.taskStore.Load(taskID)
	if err != nil {
		return err
	}
	if err := t.TransitionTo(task.StatusRejected); err != nil {
		return fmt.Errorf("cannot reject task: %w", err)
	}
	o.taskStore.Save(t)
	o.eventSink.Emit(taskID, "", "rejected", reason)
	return nil
}
