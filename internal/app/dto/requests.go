package dto

// CreateTaskRequest holds parameters for task creation.
type CreateTaskRequest struct {
	Title     string
	Goal      string
	Agent     string
	Templates []string
	Scope     ScopeDTO
	Guidelines []string
}

type ScopeDTO struct {
	AllowedPaths   []string
	ForbiddenPaths []string
	MustRead       []string
}

// UpdateTaskRequest holds parameters for task update.
type UpdateTaskRequest struct {
	TaskID         string
	AddTemplates   []string
	RemoveTemplates []string
	AddGuidelines  []string
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
