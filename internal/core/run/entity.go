package run

import "time"

// Run represents a single execution attempt of a task.
type Run struct {
	ID     string    `json:"id"`
	TaskID string    `json:"task_id"`
	Status RunStatus `json:"status"`

	Agent            string   `json:"agent"`
	PromptFile       string   `json:"prompt_file"`
	TemplateLockFile string   `json:"template_lock_file"`
	Clarifications   []string `json:"clarifications"`

	PID      int    `json:"pid,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	ExitReason string `json:"exit_reason,omitempty"`

	ValidationAttempt int `json:"validation_attempt"`

	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Duration returns the run duration, or zero if not started.
func (r *Run) Duration() time.Duration {
	if r.StartedAt == nil {
		return 0
	}
	end := time.Now()
	if r.FinishedAt != nil {
		end = *r.FinishedAt
	}
	return end.Sub(*r.StartedAt)
}

// TransitionTo changes the run status.
func (r *Run) TransitionTo(target RunStatus) error {
	if err := r.Status.ValidateTransition(target); err != nil {
		return err
	}
	r.Status = target
	now := time.Now()
	if target == RunStatusRunning && r.StartedAt == nil {
		r.StartedAt = &now
	}
	if target.IsTerminal() {
		r.FinishedAt = &now
	}
	return nil
}

// MarkStarted sets PID and transitions to running.
func (r *Run) MarkStarted(pid int) error {
	r.PID = pid
	return r.TransitionTo(RunStatusRunning)
}

// MarkFinished sets exit code and transitions to terminal state.
func (r *Run) MarkFinished(exitCode int, reason string) error {
	r.ExitCode = &exitCode
	r.ExitReason = reason
	if exitCode == 0 {
		return r.TransitionTo(RunStatusSuccess)
	}
	return r.TransitionTo(RunStatusFailed)
}
