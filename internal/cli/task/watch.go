package task

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docup/agentctl/internal/app/query"
	"github.com/docup/agentctl/internal/service/runtimecontrol"
	"github.com/spf13/cobra"
)

// NewWatchCmd creates the task watch command.
func NewWatchCmd(inspectQuery *query.InspectTask, rtMgr *runtimecontrol.Manager) *cobra.Command {
	var interval int

	cmd := &cobra.Command{
		Use:   "watch <task-id>",
		Short: "Live view of task status, heartbeat, and events",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			dur := time.Duration(interval) * time.Second

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(sigCh)

			fmt.Printf("Watching task %s (Ctrl+C to stop)\n\n", taskID)

			for {
				select {
				case <-sigCh:
					fmt.Println("\nStopped watching.")
					return nil
				default:
				}

				detail, err := inspectQuery.Execute(taskID)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					time.Sleep(dur)
					continue
				}

				// Clear screen
				fmt.Print("\033[2J\033[H")
				fmt.Printf("Task: %s | Status: %s | Agent: %s\n", detail.ID, detail.Status, detail.Agent)
				fmt.Printf("Title: %s\n", detail.Title)
				fmt.Printf("Updated: %s\n\n", detail.UpdatedAt.Format("15:04:05"))

				// Session info
				if detail.LatestSession != nil {
					sess := detail.LatestSession
					fmt.Printf("Session: %s | Status: %s | Agent: %s\n", sess.ID, sess.Status, sess.Agent)
					if sess.LastStageID != "" {
						fmt.Printf("Last stage: %s (%s) — %s\n", sess.LastStageID, sess.LastStageType, sess.LastOutcome)
					}
					fmt.Println()
				}

				// Runtime
				if rtMgr != nil {
					info, err := rtMgr.Inspect(taskID)
					if err == nil {
						fmt.Printf("Running: %v | Stale: %v\n", info.IsRunning, info.IsStale)
						if info.Heartbeat != nil {
							fmt.Printf("Heartbeat: %s\n", info.Heartbeat.Timestamp.Format("15:04:05"))
						}
						fmt.Println("\nRecent events:")
						for _, ev := range info.Events {
							line := fmt.Sprintf("  [%s] %s", ev.Timestamp.Format("15:04:05"), ev.EventType)
							if ev.StageID != "" {
								line += fmt.Sprintf(" [%s]", ev.StageID)
							}
							if ev.Details != "" {
								line += " — " + ev.Details
							}
							fmt.Println(line)
						}
					}
				}

				time.Sleep(dur)
			}
		},
	}

	cmd.Flags().IntVar(&interval, "interval", 2, "Refresh interval in seconds")
	return cmd
}
