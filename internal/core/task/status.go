package task

import "fmt"

// TaskStatus represents the current state of a task in its lifecycle.
type TaskStatus string

const (
	StatusDraft              TaskStatus = "draft"
	StatusQueued             TaskStatus = "queued"
	StatusPreparingContext   TaskStatus = "preparing_context"
	StatusRunning            TaskStatus = "running"
	StatusNeedsClarification TaskStatus = "needs_clarification"
	StatusReadyToResume      TaskStatus = "ready_to_resume"
	StatusPausing            TaskStatus = "pausing"
	StatusPaused             TaskStatus = "paused"
	StatusStopping           TaskStatus = "stopping"
	StatusStopped            TaskStatus = "stopped"
	StatusKilled             TaskStatus = "killed"
	StatusValidating         TaskStatus = "validating"
	StatusReview             TaskStatus = "review"
	StatusCompleted          TaskStatus = "completed"
	StatusFailed             TaskStatus = "failed"
	StatusRejected           TaskStatus = "rejected"
	StatusCanceled           TaskStatus = "canceled"
)

// validTransitions defines all allowed status transitions.
var validTransitions = map[TaskStatus][]TaskStatus{
	StatusDraft:              {StatusQueued, StatusCanceled},
	StatusQueued:             {StatusPreparingContext, StatusCanceled},
	StatusPreparingContext:   {StatusRunning, StatusFailed},
	StatusRunning:            {StatusNeedsClarification, StatusPausing, StatusStopping, StatusValidating, StatusReview, StatusKilled, StatusFailed},
	StatusNeedsClarification: {StatusReadyToResume, StatusCanceled},
	StatusReadyToResume:      {StatusPreparingContext, StatusCanceled},
	StatusPausing:            {StatusPaused},
	StatusPaused:             {StatusPreparingContext, StatusCanceled},
	StatusStopping:           {StatusStopped},
	StatusStopped:            {StatusPreparingContext, StatusCanceled},
	StatusKilled:             {StatusPreparingContext, StatusCanceled},
	StatusValidating:         {StatusRunning, StatusReview, StatusFailed},
	StatusReview:             {StatusCompleted, StatusRejected, StatusQueued},
	StatusCompleted:          {},
	StatusFailed:             {StatusQueued},
	StatusRejected:           {StatusQueued},
	StatusCanceled:           {},
}

// CanTransitionTo checks whether the transition from current status to target is allowed.
func (s TaskStatus) CanTransitionTo(target TaskStatus) bool {
	allowed, ok := validTransitions[s]
	if !ok {
		return false
	}
	for _, t := range allowed {
		if t == target {
			return true
		}
	}
	return false
}

// ValidateTransition returns an error if the transition is not allowed.
func (s TaskStatus) ValidateTransition(target TaskStatus) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("invalid transition: %s → %s", s, target)
	}
	return nil
}

// IsTerminal returns true if no further transitions are possible.
func (s TaskStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusCanceled
}

// IsActive returns true if the task has an active process.
func (s TaskStatus) IsActive() bool {
	return s == StatusRunning || s == StatusPausing || s == StatusStopping || s == StatusPreparingContext || s == StatusValidating
}

// CanCancel returns true if the task can be canceled from its current status.
func (s TaskStatus) CanCancel() bool {
	return s.CanTransitionTo(StatusCanceled)
}

// CanResume returns true if the task can be resumed.
func (s TaskStatus) CanResume() bool {
	return s == StatusReadyToResume || s == StatusPaused || s == StatusStopped || s == StatusKilled
}

// String returns the string representation of the status.
func (s TaskStatus) String() string {
	return string(s)
}
