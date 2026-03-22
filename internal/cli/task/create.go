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
		Short: "Create a new draft task",
		RunE: func(cmd *cobra.Command, args []string) error {
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
				Guidelines:        guidelines,
				TitleSet:          cmd.Flags().Changed("title"),
				GoalSet:           cmd.Flags().Changed("goal"),
				AgentSet:          cmd.Flags().Changed("agent"),
				TemplatesSet:      cmd.Flags().Changed("template"),
				GuidelinesSet:     cmd.Flags().Changed("guideline"),
				AllowedPathsSet:   cmd.Flags().Changed("allowed-path"),
				ForbiddenPathsSet: cmd.Flags().Changed("forbidden-path"),
				MustReadSet:       cmd.Flags().Changed("must-read"),
			}

			t, err := handler.Execute(req)
			if err != nil {
				return err
			}

			fmt.Printf("Created task %s\n", t.ID)
			if t.Title != "" {
				fmt.Printf("  Title:     %s\n", t.Title)
			}
			fmt.Printf("  Agent:     %s\n", t.Agent)
			fmt.Printf("  Status:    %s\n", t.Status)
			fmt.Printf("  Templates: %v\n", t.PromptTemplates.Builtin)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Task title")
	cmd.Flags().StringVar(&goal, "goal", "", "Engineering goal")
	cmd.Flags().StringVar(&agent, "agent", "", "Agent to use (default is resolved from config at run time)")
	cmd.Flags().StringSliceVar(&templates, "template", nil, "Prompt templates to apply")
	cmd.Flags().StringSliceVar(&guidelines, "guideline", nil, "Guidelines to include")
	cmd.Flags().StringSliceVar(&allowed, "allowed-path", nil, "Allowed file paths")
	cmd.Flags().StringSliceVar(&forbidden, "forbidden-path", nil, "Forbidden file paths")
	cmd.Flags().StringSliceVar(&mustRead, "must-read", nil, "Files agent must read")

	return cmd
}
