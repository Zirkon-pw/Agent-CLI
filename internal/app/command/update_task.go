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

	if err := applyTaskUpdate(t, req); err != nil {
		return nil, err
	}

	if err := u.taskStore.Save(t); err != nil {
		return nil, err
	}

	return t, nil
}
