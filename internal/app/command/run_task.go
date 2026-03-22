package command

import (
	"context"

	"github.com/docup/agentctl/internal/service/taskrunner"
)

// RunTask handles the task execution use case.
type RunTask struct {
	orchestrator *taskrunner.Orchestrator
}

func NewRunTask(orchestrator *taskrunner.Orchestrator) *RunTask {
	return &RunTask{orchestrator: orchestrator}
}

func (r *RunTask) Execute(ctx context.Context, taskID string) error {
	return r.orchestrator.Run(ctx, taskID)
}
