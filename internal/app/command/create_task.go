package command

import (
	"fmt"
	"time"

	"github.com/docup/agentctl/internal/app/dto"
	"github.com/docup/agentctl/internal/config/loader"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// CreateTask handles the task creation use case.
type CreateTask struct {
	taskStore *fsstore.TaskStore
	config    *loader.ProjectConfig
}

// NewCreateTask creates the use case handler.
func NewCreateTask(taskStore *fsstore.TaskStore, config *loader.ProjectConfig) *CreateTask {
	return &CreateTask{taskStore: taskStore, config: config}
}

// Execute creates a new task.
func (c *CreateTask) Execute(req dto.CreateTaskRequest) (*task.Task, error) {
	id, err := c.taskStore.NextID()
	if err != nil {
		return nil, fmt.Errorf("generating task ID: %w", err)
	}

	agent := req.Agent
	if agent == "" {
		agent = c.config.Execution.DefaultAgent
	}

	templates := req.Templates
	if len(templates) == 0 {
		templates = []string{c.config.Prompting.DefaultTemplate}
	}

	now := time.Now()
	t := &task.Task{
		ID:     id,
		Title:  req.Title,
		Goal:   req.Goal,
		Status: task.StatusDraft,
		Agent:  agent,
		PromptTemplates: task.PromptTemplates{
			Builtin: templates,
			Custom:  []string{},
		},
		Scope: task.Scope{
			AllowedPaths:   req.Scope.AllowedPaths,
			ForbiddenPaths: req.Scope.ForbiddenPaths,
			MustRead:       req.Scope.MustRead,
		},
		Guidelines: req.Guidelines,
		Context:    task.ContextConfig{},
		Constraints: task.Constraints{
			NoBreakingChanges: false,
			RequireTests:      false,
		},
		Interaction: task.Interaction{
			ClarificationStrategy: "by_yml_files",
		},
		Clarifications: task.Clarifications{
			Attached: []string{},
		},
		Runtime:    task.DefaultRuntimeConfig(),
		Validation: task.ValidationConfig{
			Mode:       task.ValidationMode(c.config.Validation.DefaultMode),
			MaxRetries: c.config.Validation.DefaultMaxRetries,
			Commands:   c.config.Validation.DefaultCommands,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := c.taskStore.Save(t); err != nil {
		return nil, fmt.Errorf("saving task: %w", err)
	}

	return t, nil
}
