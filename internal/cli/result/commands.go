package result

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/spf13/cobra"
)

// NewResultCmd creates the result command group.
func NewResultCmd(runStore *fsstore.RunStore) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "result",
		Short: "View task execution results",
	}

	cmd.AddCommand(newShowResultCmd(runStore))
	cmd.AddCommand(newDiffCmd(runStore))

	return cmd
}

func newShowResultCmd(runStore *fsstore.RunStore) *cobra.Command {
	return &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show task result summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			r, err := runStore.LatestRun(taskID)
			if err != nil {
				return fmt.Errorf("no runs found for task %s", taskID)
			}

			runDir := runStore.RunDir(taskID, r.ID)
			fmt.Printf("Task: %s | Run: %s | Status: %s\n", taskID, r.ID, r.Status)
			fmt.Printf("Agent: %s | Duration: %s\n", r.Agent, r.Duration())

			if r.ExitCode != nil {
				fmt.Printf("Exit code: %d\n", *r.ExitCode)
			}

			// Show summary if exists
			summaryPath := filepath.Join(runDir, "summary.md")
			if data, err := os.ReadFile(summaryPath); err == nil {
				fmt.Println("\n--- Summary ---")
				fmt.Print(string(data))
			}

			// Show validation if exists
			valPath := filepath.Join(runDir, "validation.json")
			if data, err := os.ReadFile(valPath); err == nil {
				fmt.Println("\n--- Validation ---")
				fmt.Print(string(data))
			}

			return nil
		},
	}
}

func newDiffCmd(runStore *fsstore.RunStore) *cobra.Command {
	return &cobra.Command{
		Use:   "diff <task-id>",
		Short: "Show code changes from task execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			r, err := runStore.LatestRun(taskID)
			if err != nil {
				return fmt.Errorf("no runs found for task %s", taskID)
			}

			diffPath := filepath.Join(runStore.RunDir(taskID, r.ID), "diff.patch")
			data, err := os.ReadFile(diffPath)
			if err != nil {
				return fmt.Errorf("no diff found for run %s", r.ID)
			}

			fmt.Print(string(data))
			return nil
		},
	}
}
