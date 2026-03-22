package clarification

import "time"

// Question represents a single clarification question.
type Question struct {
	ID   string `yaml:"id"`
	Text string `yaml:"text"`
}

// Request is a system-generated clarification request.
type Request struct {
	TaskID      string     `yaml:"task_id"`
	RequestID   string     `yaml:"request_id"`
	CreatedBy   string     `yaml:"created_by"`
	Reason      string     `yaml:"reason"`
	Questions   []Question `yaml:"questions"`
	ContextRefs []string   `yaml:"context_refs"`
	CreatedAt   time.Time  `yaml:"created_at"`
}

// Answer represents a user's answer to a clarification question.
type Answer struct {
	QuestionID string `yaml:"question_id"`
	Text       string `yaml:"text"`
}

// Clarification is the user-filled response to a clarification request.
type Clarification struct {
	TaskID           string   `yaml:"task_id"`
	RequestID        string   `yaml:"request_id"`
	ClarificationID  string   `yaml:"clarification_id"`
	Answers          []Answer `yaml:"answers"`
	Notes            []string `yaml:"notes"`
	CreatedAt        time.Time `yaml:"created_at"`
}
