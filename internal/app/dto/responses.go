package dto

import "time"

// TaskSummary is a short view of a task for list displays.
type TaskSummary struct {
	ID        string
	Title     string
	Status    string
	Agent     string
	CreatedAt time.Time
}

// TaskDetail is a full view of a task.
type TaskDetail struct {
	ID              string
	Title           string
	Goal            string
	Status          string
	Agent           string
	Templates       []string
	Guidelines      []string
	Scope           ScopeDTO
	Validation      ValidationDTO
	Runtime         RuntimeDTO
	Clarifications  ClarificationsDTO
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ValidationDTO struct {
	Mode       string
	MaxRetries int
	Commands   []string
}

type RuntimeDTO struct {
	MaxExecutionMinutes    int
	HeartbeatIntervalSec   int
	GracefulStopTimeoutSec int
	AllowPause             bool
}

type ClarificationsDTO struct {
	PendingRequest *string
	Attached       []string
}

// RunSummary is a short view of a run.
type RunSummary struct {
	ID        string
	TaskID    string
	Status    string
	Agent     string
	Duration  string
	CreatedAt time.Time
}

// ActiveRunDTO represents an active run for ps command.
type ActiveRunDTO struct {
	TaskID    string
	RunID     string
	Agent     string
	StartedAt time.Time
	Duration  string
}
