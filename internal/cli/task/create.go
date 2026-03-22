package task

import (
	"fmt"

	"github.com/docup/agentctl/internal/app/command"
	"github.com/docup/agentctl/internal/app/dto"
	"github.com/spf13/cobra"
)

// NewCreateCmd creates the task create command.
func NewCreateCmd(handler *command.CreateTask) *cobra.Command {
	var (
		title      string
		goal       string
		agent      string
		templates  []string
		guidelines []string
		allowed    []string
		forbidden  []string
		mustRead   []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if goal == "" {
				return fmt.Errorf("--goal is required")
			}

			req := dto.CreateTaskRequest{
				Title:     title,
				Goal:      goal,
				Agent:     agent,
				Templates: templates,
				Scope: dto.ScopeDTO{
					AllowedPaths:   allowed,
					ForbiddenPaths: forbidden,
					MustRead:       mustRead,
				},
				Guidelines: guidelines,
			}

			t, err := handler.Execute(req)
			if err != nil {
				return err
			}

			fmt.Printf("Created task %s: %s\n", t.ID, t.Title)
			fmt.Printf("  Agent:     %s\n", t.Agent)
			fmt.Printf("  Status:    %s\n", t.Status)
			fmt.Printf("  Templates: %v\n", t.PromptTemplates.Builtin)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Task title (required)")
	cmd.Flags().StringVar(&goal, "goal", "", "Engineering goal (required)")
	cmd.Flags().StringVar(&agent, "agent", "", "Agent to use (default from config)")
	cmd.Flags().StringSliceVar(&templates, "template", nil, "Prompt templates to apply")
	cmd.Flags().StringSliceVar(&guidelines, "guideline", nil, "Guidelines to include")
	cmd.Flags().StringSliceVar(&allowed, "allowed-path", nil, "Allowed file paths")
	cmd.Flags().StringSliceVar(&forbidden, "forbidden-path", nil, "Forbidden file paths")
	cmd.Flags().StringSliceVar(&mustRead, "must-read", nil, "Files agent must read")

	return cmd
}
