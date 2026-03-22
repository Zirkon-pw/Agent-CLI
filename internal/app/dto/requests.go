package dto

// CreateTaskRequest holds parameters for task creation.
type CreateTaskRequest struct {
	Title             string
	Goal              string
	Agent             string
	Templates         []string
	Scope             ScopeDTO
	Guidelines        []string
	TitleSet          bool
	GoalSet           bool
	AgentSet          bool
	TemplatesSet      bool
	GuidelinesSet     bool
	AllowedPathsSet   bool
	ForbiddenPathsSet bool
	MustReadSet       bool
}

type ScopeDTO struct {
	AllowedPaths   []string
	ForbiddenPaths []string
	MustRead       []string
}

// UpdateTaskRequest holds parameters for task update.
type UpdateTaskRequest struct {
	TaskID               string
	Title                *string
	Goal                 *string
	Agent                *string
	AddTemplates         []string
	RemoveTemplates      []string
	AddGuidelines        []string
	RemoveGuidelines     []string
	AddAllowedPaths      []string
	RemoveAllowedPaths   []string
	AddForbiddenPaths    []string
	RemoveForbiddenPaths []string
	AddMustRead          []string
	RemoveMustRead       []string
	Mutations            []TaskMutation
}

// MutationKind is the operation to apply to a task path.
type MutationKind string

const (
	MutationSet    MutationKind = "set"
	MutationAdd    MutationKind = "add"
	MutationRemove MutationKind = "remove"
)

// TaskMutation describes a generic change against a dot-notated task path.
type TaskMutation struct {
	Kind  MutationKind
	Path  string
	Value interface{}
}

// RunTaskRequest holds parameters for running a task.
type RunTaskRequest struct {
	TaskID string
}

// ClarificationGenerateRequest holds parameters for generating clarifications.
type ClarificationGenerateRequest struct {
	TaskID string
	Reason string
}

// ClarificationAttachRequest holds parameters for attaching a clarification.
type ClarificationAttachRequest struct {
	TaskID string
	Path   string
}

// RouteTaskRequest holds parameters for rerouting a task.
type RouteTaskRequest struct {
	TaskID string
	Agent  string
}
