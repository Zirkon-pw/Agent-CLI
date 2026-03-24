package result

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	rt "github.com/docup/agentctl/internal/core/runtime"
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
	cmd.AddCommand(newListCmd(runStore))

	return cmd
}

func newShowResultCmd(runStore *fsstore.RunStore) *cobra.Command {
	return &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show task result summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			session, err := runStore.LatestSession(taskID)
			if err != nil {
				return fmt.Errorf("no sessions found for task %s", taskID)
			}

			fmt.Printf("Task: %s | Run: %s | Status: %s\n", taskID, session.ID, session.Status)
			fmt.Printf("Agent: %s | Duration: %s\n", session.CurrentAgentID, sessionDuration(session))

			if stage := session.LastStage(); stage != nil && stage.Result != nil {
				if stage.Result.ExitCode != nil {
					fmt.Printf("Exit code: %d\n", *stage.Result.ExitCode)
				}
				if stage.Result.Message != "" {
					fmt.Printf("Last error: %s\n", stage.Result.Message)
				}
			}

			if artifact := findArtifact(session.ArtifactManifest, []string{"summary"}, []string{"summary.md"}); artifact != nil {
				if data, err := os.ReadFile(artifact.Path); err == nil {
					fmt.Println("\n--- Summary ---")
					fmt.Print(string(data))
				}
			}

			if artifact := findArtifact(session.ArtifactManifest, []string{"validation_report"}, []string{"validation.json"}); artifact != nil {
				if data, err := os.ReadFile(artifact.Path); err == nil {
					fmt.Println("\n--- Validation ---")
					fmt.Print(string(data))
				}
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
			session, err := runStore.LatestSession(taskID)
			if err != nil {
				return fmt.Errorf("no sessions found for task %s", taskID)
			}

			diffPath := resolveDiffPath(runStore, taskID, session)
			if diffPath == "" {
				return fmt.Errorf("no diff artifact found for session %s", session.ID)
			}

			data, err := os.ReadFile(diffPath)
			if err != nil {
				return fmt.Errorf("reading diff artifact: %w", err)
			}

			fmt.Print(string(data))
			return nil
		},
	}
}

func resolveDiffPath(runStore *fsstore.RunStore, taskID string, session *rt.RunSession) string {
	if session == nil {
		return ""
	}

	if artifact := findArtifact(session.ArtifactManifest, []string{"diff"}, []string{"diff.patch"}); artifact != nil && artifact.Path != "" {
		if _, err := os.Stat(artifact.Path); err == nil {
			return artifact.Path
		}
	}

	fallback := filepath.Join(runStore.RunDir(taskID, session.ID), "diff.patch")
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	return ""
}

func newListCmd(runStore *fsstore.RunStore) *cobra.Command {
	return &cobra.Command{
		Use:   "list <task-id>",
		Short: "List stored artifacts for the latest task session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			session, err := runStore.LatestSession(taskID)
			if err != nil {
				return fmt.Errorf("no sessions found for task %s", taskID)
			}

			if len(session.ArtifactManifest.Items) == 0 {
				fmt.Printf("No artifacts found for %s (%s)\n", taskID, session.ID)
				return nil
			}

			items := append([]rt.ArtifactRecord(nil), session.ArtifactManifest.Items...)
			sort.SliceStable(items, func(i, j int) bool {
				return items[i].CreatedAt.Before(items[j].CreatedAt)
			})

			fmt.Printf("Task: %s | Run: %s\n", taskID, session.ID)
			for _, item := range items {
				stage := item.StageID
				if stage == "" {
					stage = "-"
				}
				fmt.Printf("%s | %s | %s | %s\n", item.Name, item.Kind, stage, item.Path)
			}
			return nil
		},
	}
}

func findArtifact(manifest rt.ArtifactManifest, kinds []string, names []string) *rt.ArtifactRecord {
	for i := len(manifest.Items) - 1; i >= 0; i-- {
		item := manifest.Items[i]
		if contains(kinds, item.Kind) || contains(names, item.Name) {
			return &manifest.Items[i]
		}
	}
	return nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func sessionDuration(session *rt.RunSession) time.Duration {
	if len(session.StageHistory) == 0 || session.StageHistory[0].StartedAt == nil {
		return 0
	}
	end := session.UpdatedAt
	if stage := session.LastStage(); stage != nil && stage.FinishedAt != nil {
		end = *stage.FinishedAt
	}
	return end.Sub(*session.StageHistory[0].StartedAt)
}
