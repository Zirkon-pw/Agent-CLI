package query

import (
	"github.com/docup/agentctl/internal/app/dto"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// InspectTask returns detailed task info.
type InspectTask struct {
	taskStore *fsstore.TaskStore
}

func NewInspectTask(taskStore *fsstore.TaskStore) *InspectTask {
	return &InspectTask{taskStore: taskStore}
}

func (q *InspectTask) Execute(taskID string) (*dto.TaskDetail, error) {
	t, err := q.taskStore.Load(taskID)
	if err != nil {
		return nil, err
	}

	var allTemplates []string
	allTemplates = append(allTemplates, t.PromptTemplates.Builtin...)
	allTemplates = append(allTemplates, t.PromptTemplates.Custom...)

	return &dto.TaskDetail{
		ID:         t.ID,
		Title:      t.Title,
		Goal:       t.Goal,
		Status:     string(t.Status),
		Agent:      t.Agent,
		Templates:  allTemplates,
		Guidelines: t.Guidelines,
		Scope: dto.ScopeDTO{
			AllowedPaths:   t.Scope.AllowedPaths,
			ForbiddenPaths: t.Scope.ForbiddenPaths,
			MustRead:       t.Scope.MustRead,
		},
		Validation: dto.ValidationDTO{
			Mode:       string(t.Validation.Mode),
			MaxRetries: t.Validation.MaxRetries,
			Commands:   t.Validation.Commands,
		},
		Runtime: dto.RuntimeDTO{
			MaxExecutionMinutes:    t.Runtime.MaxExecutionMinutes,
			HeartbeatIntervalSec:   t.Runtime.HeartbeatIntervalSec,
			GracefulStopTimeoutSec: t.Runtime.GracefulStopTimeoutSec,
			AllowPause:             t.Runtime.AllowPause,
		},
		Clarifications: dto.ClarificationsDTO{
			PendingRequest: t.Clarifications.PendingRequest,
			Attached:       t.Clarifications.Attached,
		},
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}, nil
}
