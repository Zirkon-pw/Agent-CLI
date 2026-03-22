package runtime

import "time"

// Signal represents a control signal sent to a running task.
type Signal string

const (
	SignalStop  Signal = "stop"
	SignalKill  Signal = "kill"
	SignalPause Signal = "pause"
)

// ActiveRun tracks a currently executing task run.
type ActiveRun struct {
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Agent     string    `json:"agent"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
}

// Heartbeat holds the last heartbeat info for a running task.
type Heartbeat struct {
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Timestamp time.Time `json:"timestamp"`
	Alive     bool      `json:"alive"`
}

// IsStale returns true if heartbeat is older than the given threshold.
func (h *Heartbeat) IsStale(threshold time.Duration) bool {
	return time.Since(h.Timestamp) > threshold
}

// Event represents a task lifecycle event.
type Event struct {
	Timestamp time.Time `json:"ts"`
	TaskID    string    `json:"task"`
	RunID     string    `json:"run"`
	EventType string    `json:"event"`
	Details   string    `json:"details,omitempty"`
}
