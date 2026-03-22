package command

import (
	"fmt"

	"github.com/docup/agentctl/internal/app/dto"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// UpdateTask handles task update use case.
type UpdateTask struct {
	taskStore *fsstore.TaskStore
}

func NewUpdateTask(taskStore *fsstore.TaskStore) *UpdateTask {
	return &UpdateTask{taskStore: taskStore}
}

func (u *UpdateTask) Execute(req dto.UpdateTaskRequest) (*task.Task, error) {
	t, err := u.taskStore.Load(req.TaskID)
	if err != nil {
		return nil, err
	}

	if t.Status != task.StatusDraft {
		return nil, fmt.Errorf("can only update tasks in draft status, current: %s", t.Status)
	}

	for _, tmpl := range req.AddTemplates {
		if !t.HasTemplate(tmpl) {
			t.PromptTemplates.Builtin = append(t.PromptTemplates.Builtin, tmpl)
		}
	}

	for _, tmpl := range req.RemoveTemplates {
		filtered := make([]string, 0)
		for _, existing := range t.PromptTemplates.Builtin {
			if existing != tmpl {
				filtered = append(filtered, existing)
			}
		}
		t.PromptTemplates.Builtin = filtered
	}

	for _, g := range req.AddGuidelines {
		found := false
		for _, existing := range t.Guidelines {
			if existing == g {
				found = true
				break
			}
		}
		if !found {
			t.Guidelines = append(t.Guidelines, g)
		}
	}

	if err := u.taskStore.Save(t); err != nil {
		return nil, err
	}

	return t, nil
}
