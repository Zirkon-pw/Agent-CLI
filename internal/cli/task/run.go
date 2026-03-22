package task

import (
	"fmt"

	"github.com/docup/agentctl/internal/app/command"
	"github.com/spf13/cobra"
)

// NewRunCmd creates the task run command.
func NewRunCmd(handler *command.RunTask) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <task-id>",
		Short: "Execute a task with the assigned agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			fmt.Printf("Running task %s...\n", taskID)
			if err := handler.Execute(cmd.Context(), taskID); err != nil {
				return err
			}
			fmt.Printf("Task %s execution completed.\n", taskID)
			return nil
		},
	}
	return cmd
}
