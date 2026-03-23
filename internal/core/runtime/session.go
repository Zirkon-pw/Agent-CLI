package runtime

import (
	"time"

	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/core/validation"
)

// StageType identifies the type of work executed by an adapter stage.
type StageType string

const (
	StageTypeExecute     StageType = "execute"
	StageTypeClarify     StageType = "clarification"
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
)

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

// StageInput groups stage-specific machine-readable inputs.
type StageInput struct {
	Task             *task.Task            `json:"task,omitempty"`
	ArtifactManifest ArtifactManifest      `json:"artifact_manifest,omitempty"`
	Clarifications   []string              `json:"clarifications,omitempty"`
	Validation       *ValidationStageInput `json:"validation,omitempty"`
	Review           *ReviewStageInput     `json:"review,omitempty"`
}

// StageSpec is the normalized input contract handed to a built-in CLI driver.
type StageSpec struct {
	SessionID  string     `json:"session_id"`
	TaskID     string     `json:"task_id"`
	RunID      string     `json:"run_id"`
	StageID    string     `json:"stage_id"`
	Type       StageType  `json:"type"`
	AgentID    string     `json:"agent_id"`
	WorkDir    string     `json:"work_dir"`
	SessionDir string     `json:"session_dir"`
	StageDir   string     `json:"stage_dir"`
	TaskPath   string     `json:"task_path"`
	ContextDir string     `json:"context_dir,omitempty"`
	PromptPath string     `json:"prompt_path,omitempty"`
	Input      StageInput `json:"input"`
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

// DriverState stores CLI-specific continuation state across stages.
type DriverState struct {
	ExternalSessionID string            `json:"external_session_id,omitempty"`
	Values            map[string]string `json:"values,omitempty"`
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
	ID                     string           `json:"id"`
	TaskID                 string           `json:"task_id"`
	Status                 SessionStatus    `json:"status"`
	CurrentAgentID         string           `json:"current_agent_id"`
	CurrentStageID         string           `json:"current_stage_id,omitempty"`
	BlockedStageType       StageType        `json:"blocked_stage_type,omitempty"`
	PendingClarificationID *string          `json:"pending_clarification_id,omitempty"`
	PendingHandoff         *PendingHandoff  `json:"pending_handoff,omitempty"`
	DriverState            DriverState      `json:"driver_state"`
	Recovery               RecoverySnapshot `json:"recovery"`
	Validation             ValidationState  `json:"validation"`
	ReviewReport           *ReviewReport    `json:"review_report,omitempty"`
	ArtifactManifest       ArtifactManifest `json:"artifact_manifest"`
	StageHistory           []StageRun       `json:"stage_history"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
	CompletedAt            *time.Time       `json:"completed_at,omitempty"`
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
