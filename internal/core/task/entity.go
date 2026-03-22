package task

import "time"

// Task is the central domain entity representing a formalized engineering task.
type Task struct {
	ID     string     `yaml:"id"`
	Title  string     `yaml:"title"`
	Goal   string     `yaml:"goal"`
	Status TaskStatus `yaml:"status"`
	Agent  string     `yaml:"agent"`

	PromptTemplates PromptTemplates `yaml:"prompt_templates"`
	Scope           Scope           `yaml:"scope"`
	Guidelines      []string        `yaml:"guidelines"`
	Context         ContextConfig   `yaml:"context"`
	Constraints     Constraints     `yaml:"constraints"`
	Interaction     Interaction     `yaml:"interaction"`
	Clarifications  Clarifications  `yaml:"clarifications"`
	Runtime         RuntimeConfig   `yaml:"runtime"`
	Validation      ValidationConfig `yaml:"validation"`

	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// PromptTemplates holds references to built-in and custom templates.
type PromptTemplates struct {
	Builtin []string `yaml:"builtin"`
	Custom  []string `yaml:"custom"`
}

// Scope defines file boundaries for the task.
type Scope struct {
	AllowedPaths   []string `yaml:"allowed_paths"`
	ForbiddenPaths []string `yaml:"forbidden_paths"`
	MustRead       []string `yaml:"must_read"`
}

// ContextConfig defines which files to include in the context pack.
type ContextConfig struct {
	IncludeFiles    []string `yaml:"include_files"`
	IncludePatterns []string `yaml:"include_patterns"`
}

// Constraints define rules the agent must follow.
type Constraints struct {
	NoBreakingChanges bool `yaml:"no_breaking_changes"`
	RequireTests      bool `yaml:"require_tests"`
}

// Interaction defines how the agent communicates back.
type Interaction struct {
	ClarificationStrategy string `yaml:"clarification_strategy"`
}

// Clarifications holds pending requests and attached clarifications.
type Clarifications struct {
	PendingRequest *string  `yaml:"pending_request"`
	Attached       []string `yaml:"attached"`
}

// RuntimeConfig holds execution parameters.
type RuntimeConfig struct {
	MaxExecutionMinutes    int `yaml:"max_execution_minutes"`
	HeartbeatIntervalSec   int `yaml:"heartbeat_interval_sec"`
	GracefulStopTimeoutSec int `yaml:"graceful_stop_timeout_sec"`
	AllowPause             bool `yaml:"allow_pause"`
}

// ValidationMode defines how validation results are handled.
type ValidationMode string

const (
	ValidationModeSimple ValidationMode = "simple"
	ValidationModeFull   ValidationMode = "full"
)

// ValidationConfig holds validation settings for the task.
type ValidationConfig struct {
	Mode       ValidationMode `yaml:"mode"`
	MaxRetries int            `yaml:"max_retries"`
	Commands   []string       `yaml:"commands"`
}

// TransitionTo attempts to change the task status to target.
func (t *Task) TransitionTo(target TaskStatus) error {
	if err := t.Status.ValidateTransition(target); err != nil {
		return err
	}
	t.Status = target
	t.UpdatedAt = time.Now()
	return nil
}

// HasTemplate checks if a built-in template is attached.
func (t *Task) HasTemplate(name string) bool {
	for _, tmpl := range t.PromptTemplates.Builtin {
		if tmpl == name {
			return true
		}
	}
	return false
}

// HasClarifyTemplate returns true if clarify_if_needed is attached.
func (t *Task) HasClarifyTemplate() bool {
	return t.HasTemplate("clarify_if_needed")
}

// AddClarification attaches a clarification file reference.
func (t *Task) AddClarification(path string) {
	t.Clarifications.Attached = append(t.Clarifications.Attached, path)
	t.Clarifications.PendingRequest = nil
	t.UpdatedAt = time.Now()
}

// SetPendingClarification marks a pending clarification request.
func (t *Task) SetPendingClarification(requestID string) {
	t.Clarifications.PendingRequest = &requestID
	t.UpdatedAt = time.Now()
}

// DefaultRuntimeConfig returns sensible defaults.
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		MaxExecutionMinutes:    45,
		HeartbeatIntervalSec:   5,
		GracefulStopTimeoutSec: 20,
		AllowPause:             true,
	}
}

// DefaultValidationConfig returns sensible defaults.
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		Mode:       ValidationModeSimple,
		MaxRetries: 3,
		Commands:   []string{},
	}
}
