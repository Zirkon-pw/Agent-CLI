package task

import (
	"fmt"

	"github.com/docup/agentctl/internal/app/command"
	"github.com/docup/agentctl/internal/app/dto"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates the task update command.
func NewUpdateCmd(handler *command.UpdateTask) *cobra.Command {
	var (
		addTemplates    []string
		removeTemplates []string
		addGuidelines   []string
	)

	cmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Update a draft task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := dto.UpdateTaskRequest{
				TaskID:          args[0],
				AddTemplates:    addTemplates,
				RemoveTemplates: removeTemplates,
				AddGuidelines:   addGuidelines,
			}
			t, err := handler.Execute(req)
			if err != nil {
				return err
			}
			fmt.Printf("Updated task %s\n", t.ID)
			fmt.Printf("  Templates: %v\n", t.PromptTemplates.Builtin)
			fmt.Printf("  Guidelines: %v\n", t.Guidelines)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&addTemplates, "add-template", nil, "Add prompt template")
	cmd.Flags().StringSliceVar(&removeTemplates, "remove-template", nil, "Remove prompt template")
	cmd.Flags().StringSliceVar(&addGuidelines, "add-guideline", nil, "Add guideline")
	return cmd
}
