package task

import (
	"encoding/json"
	"fmt"

	"github.com/docup/agentctl/internal/app/query"
	"github.com/docup/agentctl/internal/service/runtimecontrol"
	"github.com/spf13/cobra"
)

// NewInspectCmd creates the task inspect command.
func NewInspectCmd(handler *query.InspectTask, rtMgr *runtimecontrol.Manager) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "inspect <task-id>",
		Short: "Show detailed task info with runtime state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			detail, err := handler.Execute(taskID)
			if err != nil {
				return err
			}

			if asJSON {
				data, _ := json.MarshalIndent(detail, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Task: %s\n", detail.ID)
			fmt.Printf("Title: %s\n", detail.Title)
			fmt.Printf("Goal: %s\n", detail.Goal)
			fmt.Printf("Status: %s\n", detail.Status)
			fmt.Printf("Agent: %s\n", detail.Agent)
			fmt.Printf("Templates: %v\n", detail.Templates)
			fmt.Printf("Guidelines: %v\n", detail.Guidelines)
			fmt.Printf("Created: %s\n", detail.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated: %s\n", detail.UpdatedAt.Format("2006-01-02 15:04:05"))

			if len(detail.Scope.AllowedPaths) > 0 {
				fmt.Printf("Allowed paths: %v\n", detail.Scope.AllowedPaths)
			}
			if len(detail.Scope.ForbiddenPaths) > 0 {
				fmt.Printf("Forbidden paths: %v\n", detail.Scope.ForbiddenPaths)
			}

			fmt.Printf("Validation: mode=%s, max_retries=%d\n", detail.Validation.Mode, detail.Validation.MaxRetries)
			if len(detail.Validation.Commands) > 0 {
				fmt.Printf("Validation commands: %v\n", detail.Validation.Commands)
			}

			// Runtime info
			if rtMgr != nil {
				info, err := rtMgr.Inspect(taskID)
				if err == nil {
					fmt.Printf("\nRuntime:\n")
					fmt.Printf("  Running: %v\n", info.IsRunning)
					fmt.Printf("  Stale: %v\n", info.IsStale)
					if info.Heartbeat != nil {
						fmt.Printf("  Last heartbeat: %s\n", info.Heartbeat.Timestamp.Format("15:04:05"))
					}
					if info.ActiveRun != nil {
						fmt.Printf("  Active run: %s (agent=%s)\n", info.ActiveRun.RunID, info.ActiveRun.Agent)
					}
				}
			}

			if detail.LatestSession != nil {
				fmt.Printf("\nLatest session:\n")
				fmt.Printf("  ID: %s\n", detail.LatestSession.ID)
				fmt.Printf("  Status: %s\n", detail.LatestSession.Status)
				fmt.Printf("  Agent: %s\n", detail.LatestSession.Agent)
				if detail.LatestSession.LastStageID != "" {
					fmt.Printf("  Last stage: %s (%s)\n", detail.LatestSession.LastStageID, detail.LatestSession.LastStageType)
				}
				if detail.LatestSession.LastOutcome != "" {
					fmt.Printf("  Last outcome: %s\n", detail.LatestSession.LastOutcome)
				}
				if detail.LatestSession.LastError != "" {
					fmt.Printf("  Last error: %s\n", detail.LatestSession.LastError)
				}
				fmt.Printf("  Artifacts: %d\n", detail.LatestSession.ArtifactCount)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}
