package runtime

import (
	"encoding/json"
	"time"

	"github.com/docup/agentctl/internal/core/clarification"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/core/validation"
)

// StageType identifies the type of work executed by an adapter stage.
type StageType string

const (
	StageTypeExecute     StageType = "execute"
	StageTypeValidateFix StageType = "validate_fix"
	StageTypeReview      StageType = "review"
	StageTypeHandoff     StageType = "handoff"
)

// SessionStatus captures the current state of a run session.
type SessionStatus string

const (
	SessionStatusQueued               SessionStatus = "queued"
	SessionStatusStageRunning         SessionStatus = "stage_running"
	SessionStatusWaitingClarification SessionStatus = "waiting_clarification"
	SessionStatusPaused               SessionStatus = "paused"
	SessionStatusReviewing            SessionStatus = "reviewing"
	SessionStatusHandoffPending       SessionStatus = "handoff_pending"
	SessionStatusCompleted            SessionStatus = "completed"
	SessionStatusFailed               SessionStatus = "failed"
	SessionStatusCanceled             SessionStatus = "canceled"
)

// StageState captures the current state of a stage within a session.
type StageState string

const (
	StageStatePending   StageState = "pending"
	StageStateRunning   StageState = "running"
	StageStateCompleted StageState = "completed"
	StageStateFailed    StageState = "failed"
	StageStateCanceled  StageState = "canceled"
	StageStatePaused    StageState = "paused"
)

// ProtocolCommandType is the machine-readable control command sent to adapters.
type ProtocolCommandType string

const (
	CommandTypeCancel ProtocolCommandType = "cancel"
	CommandTypePause  ProtocolCommandType = "pause"
	CommandTypeResume ProtocolCommandType = "resume"
	CommandTypeKill   ProtocolCommandType = "kill"
	CommandTypePing   ProtocolCommandType = "ping"
)

// ProtocolEventType is the machine-readable event emitted by adapters.
type ProtocolEventType string

const (
	EventTypeHello                  ProtocolEventType = "hello"
	EventTypeStageStarted           ProtocolEventType = "stage_started"
	EventTypeProgress               ProtocolEventType = "progress"
	EventTypeHeartbeat              ProtocolEventType = "heartbeat"
	EventTypeArtifact               ProtocolEventType = "artifact"
	EventTypeClarificationRequested ProtocolEventType = "clarification_requested"
	EventTypeReviewReport           ProtocolEventType = "review_report"
	EventTypeHandoffRequested       ProtocolEventType = "handoff_requested"
	EventTypeWarning                ProtocolEventType = "warning"
	EventTypeError                  ProtocolEventType = "error"
	EventTypeStageCompleted         ProtocolEventType = "stage_completed"
)

// AdapterCapabilities declares which control and workflow features an adapter supports.
type AdapterCapabilities struct {
	ProtocolVersion       string `json:"protocol_version" yaml:"protocol_version"`
	SupportsCancel        bool   `json:"supports_cancel" yaml:"supports_cancel"`
	SupportsPause         bool   `json:"supports_pause" yaml:"supports_pause"`
	SupportsResume        bool   `json:"supports_resume" yaml:"supports_resume"`
	SupportsKill          bool   `json:"supports_kill" yaml:"supports_kill"`
	SupportsHeartbeat     bool   `json:"supports_heartbeat" yaml:"supports_heartbeat"`
	SupportsClarification bool   `json:"supports_clarification" yaml:"supports_clarification"`
	SupportsReview        bool   `json:"supports_review" yaml:"supports_review"`
	SupportsHandoff       bool   `json:"supports_handoff" yaml:"supports_handoff"`
}

// ArtifactRecord tracks a file artifact produced or consumed by the runtime.
type ArtifactRecord struct {
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Path      string    `json:"path"`
	StageID   string    `json:"stage_id,omitempty"`
	MediaType string    `json:"media_type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ArtifactManifest contains all registered artifacts for a session.
type ArtifactManifest struct {
	Items []ArtifactRecord `json:"items"`
}

// Add registers or replaces an artifact by name and path.
func (m *ArtifactManifest) Add(item ArtifactRecord) {
	for i := range m.Items {
		if m.Items[i].Name == item.Name && m.Items[i].Path == item.Path {
			m.Items[i] = item
			return
		}
	}
	m.Items = append(m.Items, item)
}

// ValidationStageInput is structured input for validate-fix stages.
type ValidationStageInput struct {
	Attempt        int                      `json:"attempt"`
	MaxRetries     int                      `json:"max_retries"`
	FailedChecks   []validation.CheckResult `json:"failed_checks"`
	ValidationPath string                   `json:"validation_path,omitempty"`
}

// ReviewFinding captures a structured reviewer finding.
type ReviewFinding struct {
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	File       string  `json:"file,omitempty"`
	StartLine  int     `json:"start_line,omitempty"`
	EndLine    int     `json:"end_line,omitempty"`
	Priority   int     `json:"priority,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// ReviewStageInput is structured input for review stages.
type ReviewStageInput struct {
	SummaryPath    string `json:"summary_path,omitempty"`
	DiffPath       string `json:"diff_path,omitempty"`
	ValidationPath string `json:"validation_path,omitempty"`
	ContextPath    string `json:"context_path,omitempty"`
}

// HandoffStageInput is structured input for handoff stages.
type HandoffStageInput struct {
	NextAgentID string `json:"next_agent_id"`
	Reason      string `json:"reason,omitempty"`
}

// StageInput groups stage-specific machine-readable inputs.
type StageInput struct {
	Task             *task.Task            `json:"task,omitempty"`
	ArtifactManifest ArtifactManifest      `json:"artifact_manifest,omitempty"`
	Clarifications   []string              `json:"clarifications,omitempty"`
	Validation       *ValidationStageInput `json:"validation,omitempty"`
	Review           *ReviewStageInput     `json:"review,omitempty"`
	Handoff          *HandoffStageInput    `json:"handoff,omitempty"`
}

// StageSpec is the input contract handed to an adapter wrapper.
type StageSpec struct {
	ProtocolVersion string     `json:"protocol_version"`
	SessionID       string     `json:"session_id"`
	TaskID          string     `json:"task_id"`
	RunID           string     `json:"run_id"`
	StageID         string     `json:"stage_id"`
	Type            StageType  `json:"type"`
	AgentID         string     `json:"agent_id"`
	WorkDir         string     `json:"work_dir"`
	SessionDir      string     `json:"session_dir"`
	StageDir        string     `json:"stage_dir"`
	TaskPath        string     `json:"task_path"`
	ContextDir      string     `json:"context_dir,omitempty"`
	PromptPath      string     `json:"prompt_path,omitempty"`
	Input           StageInput `json:"input"`
}

// ProtocolCommand is a machine-readable control message sent to an adapter.
type ProtocolCommand struct {
	SessionID string              `json:"session_id"`
	TaskID    string              `json:"task_id"`
	RunID     string              `json:"run_id"`
	StageID   string              `json:"stage_id"`
	Seq       int64               `json:"seq"`
	Timestamp time.Time           `json:"ts"`
	Type      ProtocolCommandType `json:"type"`
	Payload   json.RawMessage     `json:"payload,omitempty"`
}

// ProtocolEvent is a machine-readable event emitted by an adapter.
type ProtocolEvent struct {
	SessionID string            `json:"session_id"`
	TaskID    string            `json:"task_id"`
	RunID     string            `json:"run_id"`
	StageID   string            `json:"stage_id"`
	Seq       int64             `json:"seq"`
	Timestamp time.Time         `json:"ts"`
	Type      ProtocolEventType `json:"type"`
	Payload   json.RawMessage   `json:"payload,omitempty"`
}

// HelloPayload describes adapter identity and capabilities.
type HelloPayload struct {
	AdapterID    string              `json:"adapter_id"`
	Capabilities AdapterCapabilities `json:"capabilities"`
}

// StageStartedPayload describes a stage start event.
type StageStartedPayload struct {
	Type    StageType `json:"type"`
	Message string    `json:"message,omitempty"`
}

// ProgressPayload describes incremental stage progress.
type ProgressPayload struct {
	Message string `json:"message"`
	Percent int    `json:"percent,omitempty"`
}

// HeartbeatPayload is emitted by adapters to indicate liveness.
type HeartbeatPayload struct {
	Message string `json:"message,omitempty"`
}

// ArtifactPayload registers an artifact produced by an adapter.
type ArtifactPayload struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	MediaType string `json:"media_type,omitempty"`
}

// ClarificationRequestedPayload captures structured clarification needs.
type ClarificationRequestedPayload struct {
	RequestID   string                   `json:"request_id,omitempty"`
	Reason      string                   `json:"reason,omitempty"`
	Questions   []clarification.Question `json:"questions"`
	ContextRefs []string                 `json:"context_refs,omitempty"`
}

// ReviewReportPayload captures structured review results.
type ReviewReportPayload struct {
	Summary      string          `json:"summary,omitempty"`
	Findings     []ReviewFinding `json:"findings,omitempty"`
	ArtifactPath string          `json:"artifact_path,omitempty"`
}

// HandoffRequestedPayload captures an adapter-requested handoff.
type HandoffRequestedPayload struct {
	NextAgentID string `json:"next_agent_id"`
	Reason      string `json:"reason,omitempty"`
}

// ErrorPayload is emitted for warnings and errors.
type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// StageResult describes the terminal result of a stage.
type StageResult struct {
	Outcome       string `json:"outcome"`
	Message       string `json:"message,omitempty"`
	NextAgentID   string `json:"next_agent_id,omitempty"`
	ExitCode      *int   `json:"exit_code,omitempty"`
	SummaryPath   string `json:"summary_path,omitempty"`
	DiffPath      string `json:"diff_path,omitempty"`
	ReviewPath    string `json:"review_path,omitempty"`
	ArtifactsPath string `json:"artifacts_path,omitempty"`
}

// StageCompletedPayload wraps the final stage result.
type StageCompletedPayload struct {
	Result StageResult `json:"result"`
}

// ReviewReport captures normalized persisted review results.
type ReviewReport struct {
	StageID      string          `json:"stage_id"`
	Summary      string          `json:"summary,omitempty"`
	Findings     []ReviewFinding `json:"findings,omitempty"`
	ArtifactPath string          `json:"artifact_path,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// PendingHandoff stores a queued handoff request.
type PendingHandoff struct {
	NextAgentID string    `json:"next_agent_id"`
	Reason      string    `json:"reason,omitempty"`
	RequestedAt time.Time `json:"requested_at"`
}

// RecoverySnapshot stores runtime details needed for recovery.
type RecoverySnapshot struct {
	AdapterPID      int       `json:"adapter_pid,omitempty"`
	ProcessGroupID  int       `json:"process_group_id,omitempty"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at,omitempty"`
}

// ValidationState tracks retry state across validate-fix loops.
type ValidationState struct {
	Attempt        int                      `json:"attempt"`
	MaxRetries     int                      `json:"max_retries"`
	LastResults    []validation.CheckResult `json:"last_results,omitempty"`
	LastReportPath string                   `json:"last_report_path,omitempty"`
}

// StageRun stores the persisted history of a single stage execution.
type StageRun struct {
	StageID    string       `json:"stage_id"`
	Type       StageType    `json:"type"`
	AgentID    string       `json:"agent_id"`
	State      StageState   `json:"state"`
	Attempt    int          `json:"attempt"`
	StartedAt  *time.Time   `json:"started_at,omitempty"`
	FinishedAt *time.Time   `json:"finished_at,omitempty"`
	Result     *StageResult `json:"result,omitempty"`
}

// RunSession is the persisted state for a task execution lifecycle.
type RunSession struct {
	ID                     string              `json:"id"`
	TaskID                 string              `json:"task_id"`
	Status                 SessionStatus       `json:"status"`
	CurrentAgentID         string              `json:"current_agent_id"`
	CurrentStageID         string              `json:"current_stage_id,omitempty"`
	BlockedStageType       StageType           `json:"blocked_stage_type,omitempty"`
	LastEventSeq           int64               `json:"last_event_seq"`
	LastCommandSeq         int64               `json:"last_command_seq"`
	PendingControlCommand  *ProtocolCommand    `json:"pending_control_command,omitempty"`
	PendingClarificationID *string             `json:"pending_clarification_id,omitempty"`
	PendingHandoff         *PendingHandoff     `json:"pending_handoff,omitempty"`
	Capabilities           AdapterCapabilities `json:"capabilities"`
	Recovery               RecoverySnapshot    `json:"recovery"`
	Validation             ValidationState     `json:"validation"`
	ReviewReport           *ReviewReport       `json:"review_report,omitempty"`
	ArtifactManifest       ArtifactManifest    `json:"artifact_manifest"`
	StageHistory           []StageRun          `json:"stage_history"`
	CreatedAt              time.Time           `json:"created_at"`
	UpdatedAt              time.Time           `json:"updated_at"`
	CompletedAt            *time.Time          `json:"completed_at,omitempty"`
}

// ActiveStage returns the current persisted stage, if any.
func (s *RunSession) ActiveStage() *StageRun {
	for i := len(s.StageHistory) - 1; i >= 0; i-- {
		if s.StageHistory[i].StageID == s.CurrentStageID {
			return &s.StageHistory[i]
		}
	}
	return nil
}

// LastStage returns the most recent stage in the history.
func (s *RunSession) LastStage() *StageRun {
	if len(s.StageHistory) == 0 {
		return nil
	}
	return &s.StageHistory[len(s.StageHistory)-1]
}
