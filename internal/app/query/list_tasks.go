package query

import (
	"github.com/docup/agentctl/internal/app/dto"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// ListTasks returns all tasks as summaries.
type ListTasks struct {
	taskStore *fsstore.TaskStore
}

func NewListTasks(taskStore *fsstore.TaskStore) *ListTasks {
	return &ListTasks{taskStore: taskStore}
}

func (q *ListTasks) Execute() ([]dto.TaskSummary, error) {
	tasks, err := q.taskStore.List()
	if err != nil {
		return nil, err
	}
	var summaries []dto.TaskSummary
	for _, t := range tasks {
		summaries = append(summaries, dto.TaskSummary{
			ID:        t.ID,
			Title:     t.Title,
			Status:    string(t.Status),
			Agent:     t.Agent,
			CreatedAt: t.CreatedAt,
		})
	}
	return summaries, nil
}
