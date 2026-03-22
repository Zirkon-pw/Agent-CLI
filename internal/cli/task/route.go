package task

import (
	"fmt"

	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/spf13/cobra"
)

// NewRouteCmd creates the task route command.
func NewRouteCmd(taskStore *fsstore.TaskStore) *cobra.Command {
	var agent string

	cmd := &cobra.Command{
		Use:   "route <task-id>",
		Short: "Reroute a task to a different agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if agent == "" {
				return fmt.Errorf("--agent is required")
			}
			taskID := args[0]
			t, err := taskStore.Load(taskID)
			if err != nil {
				return err
			}
			t.Agent = agent
			if err := taskStore.Save(t); err != nil {
				return err
			}
			fmt.Printf("Task %s routed to agent %s\n", taskID, agent)
			return nil
		},
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Target agent (required)")
	return cmd
}
