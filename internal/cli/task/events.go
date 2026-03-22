package task

import (
	"fmt"

	"github.com/docup/agentctl/internal/service/runtimecontrol"
	"github.com/spf13/cobra"
)

// NewEventsCmd creates the task events command.
func NewEventsCmd(rtMgr *runtimecontrol.Manager) *cobra.Command {
	var tail int

	cmd := &cobra.Command{
		Use:   "events <task-id>",
		Short: "Show task lifecycle events",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			events, err := rtMgr.TaskEvents(taskID, tail)
			if err != nil {
				return err
			}

			if len(events) == 0 {
				fmt.Println("No events found.")
				return nil
			}

			for _, ev := range events {
				ts := ev.Timestamp.Format("15:04:05")
				if ev.Details != "" {
					fmt.Printf("[%s] %s %s — %s\n", ts, ev.RunID, ev.EventType, ev.Details)
				} else {
					fmt.Printf("[%s] %s %s\n", ts, ev.RunID, ev.EventType)
				}
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&tail, "tail", 0, "Show last N events")
	return cmd
}
