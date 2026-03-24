package task

import (
	"fmt"

	"github.com/docup/agentctl/internal/service/runtimecontrol"
	"github.com/spf13/cobra"
)

// NewEventsCmd creates the task events command.
func NewEventsCmd(rtMgr *runtimecontrol.Manager) *cobra.Command {
	var tail int
	var stageFilter string

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
				if stageFilter != "" && ev.StageID != stageFilter {
					continue
				}

				ts := ev.Timestamp.Format("15:04:05")
				parts := fmt.Sprintf("[%s] %s %s", ts, ev.RunID, ev.EventType)
				if ev.StageID != "" {
					parts += fmt.Sprintf(" [%s]", ev.StageID)
				}
				if ev.AgentID != "" {
					parts += fmt.Sprintf(" (%s)", ev.AgentID)
				}
				if ev.Details != "" {
					parts += " — " + ev.Details
				}
				fmt.Println(parts)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&tail, "tail", 0, "Show last N events")
	cmd.Flags().StringVar(&stageFilter, "stage", "", "Filter events by stage ID")
	return cmd
}
