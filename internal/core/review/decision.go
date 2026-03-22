package review

import "time"

// Decision represents the outcome of a task review.
type Decision string

const (
	DecisionAccepted     Decision = "accepted"
	DecisionRejected     Decision = "rejected"
	DecisionNeedsChanges Decision = "needs_changes"
	DecisionReroute      Decision = "reroute"
)

// Review holds a review decision for a task run.
type Review struct {
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Decision  Decision  `json:"decision"`
	Reason    string    `json:"reason,omitempty"`
	Reviewer  string    `json:"reviewer"`
	CreatedAt time.Time `json:"created_at"`
}
