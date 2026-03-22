package cli

import (
	"fmt"

	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the init command.
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize .agentctl directory structure",
		Long:  "Creates .agentctl/ with default config files and directory structure.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			if dir == "" {
				dir = "."
			}
			ws, err := fsstore.InitWorkspace(dir)
			if err != nil {
				return fmt.Errorf("init failed: %w", err)
			}
			fmt.Printf("Initialized agentctl workspace at %s\n", ws.AgentctlDir)
			fmt.Println("Created:")
			fmt.Println("  config.yaml    — project configuration")
			fmt.Println("  agents.yaml    — available agents")
			fmt.Println("  routing.yaml   — agent routing rules")
			fmt.Println("  tasks/         — task specifications")
			fmt.Println("  templates/     — prompt templates")
			fmt.Println("  guidelines/    — project guidelines")
			fmt.Println("  clarifications/ — clarification files")
			fmt.Println("  context/       — context packs")
			fmt.Println("  runs/          — execution artifacts")
			fmt.Println("  runtime/       — live task state")
			fmt.Println("  reviews/       — review decisions")
			return nil
		},
	}
	cmd.Flags().StringP("dir", "d", ".", "Project root directory")
	return cmd
}
