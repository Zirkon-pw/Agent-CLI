package query

import (
	"github.com/docup/agentctl/internal/app/dto"
	rt "github.com/docup/agentctl/internal/core/runtime"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// InspectTask returns detailed task info.
type InspectTask struct {
	taskStore *fsstore.TaskStore
	runStore  *fsstore.RunStore
}

func NewInspectTask(taskStore *fsstore.TaskStore, runStore *fsstore.RunStore) *InspectTask {
	return &InspectTask{taskStore: taskStore, runStore: runStore}
}

func (q *InspectTask) Execute(taskID string) (*dto.TaskDetail, error) {
	t, err := q.taskStore.Load(taskID)
	if err != nil {
		return nil, err
	}

	var allTemplates []string
	allTemplates = append(allTemplates, t.PromptTemplates.Builtin...)
	allTemplates = append(allTemplates, t.PromptTemplates.Custom...)

	detail := &dto.TaskDetail{
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
	}

	if q.runStore != nil {
		if session, err := q.runStore.LatestSession(taskID); err == nil {
			detail.LatestSession = buildSessionDetail(session)
		}
	}

	return detail, nil
}

func buildSessionDetail(session *rt.RunSession) *dto.SessionDetailDTO {
	if session == nil {
		return nil
	}

	detail := &dto.SessionDetailDTO{
		ID:            session.ID,
		Status:        string(session.Status),
		Agent:         session.CurrentAgentID,
		ArtifactCount: len(session.ArtifactManifest.Items),
	}
	if stage := session.LastStage(); stage != nil {
		detail.LastStageID = stage.StageID
		detail.LastStageType = string(stage.Type)
		if stage.Result != nil {
			detail.LastOutcome = stage.Result.Outcome
			detail.LastError = stage.Result.Message
		}
	}
	return detail
}
