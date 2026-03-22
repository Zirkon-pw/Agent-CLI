package run

import "fmt"

// RunStatus represents the current state of a task run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusSuccess   RunStatus = "success"
	RunStatusFailed    RunStatus = "failed"
	RunStatusStopped   RunStatus = "stopped"
	RunStatusKilled    RunStatus = "killed"
	RunStatusRetrying  RunStatus = "retrying"
)

var validRunTransitions = map[RunStatus][]RunStatus{
	RunStatusPending:  {RunStatusRunning},
	RunStatusRunning:  {RunStatusSuccess, RunStatusFailed, RunStatusStopped, RunStatusKilled, RunStatusRetrying},
	RunStatusRetrying: {RunStatusRunning},
	RunStatusSuccess:  {},
	RunStatusFailed:   {},
	RunStatusStopped:  {},
	RunStatusKilled:   {},
}

// CanTransitionTo checks if the run can transition to the target status.
func (s RunStatus) CanTransitionTo(target RunStatus) bool {
	allowed, ok := validRunTransitions[s]
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
func (s RunStatus) ValidateTransition(target RunStatus) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("invalid run transition: %s → %s", s, target)
	}
	return nil
}

// IsTerminal returns true if the run is in a terminal state.
func (s RunStatus) IsTerminal() bool {
	return s == RunStatusSuccess || s == RunStatusFailed || s == RunStatusStopped || s == RunStatusKilled
}

func (s RunStatus) String() string {
	return string(s)
}
