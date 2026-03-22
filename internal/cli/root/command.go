package root

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root agentctl command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agentctl",
		Short: "Task-based control plane for AI agent engineering work",
		Long: `agentctl replaces chat-based AI interaction with a structured
task-based control plane. Create formalized tasks, submit them with
controlled context, execute through structured pipelines, and review
results as reproducible artifacts.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return cmd
}
