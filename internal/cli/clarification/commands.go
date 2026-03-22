package clarification

import (
	"fmt"

	"github.com/docup/agentctl/internal/core/clarification"
	"github.com/docup/agentctl/internal/service/clarificationflow"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewClarificationCmd creates the clarification command group.
func NewClarificationCmd(mgr *clarificationflow.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clarification",
		Short:   "Manage task clarifications",
		Aliases: []string{"clar"},
	}

	cmd.AddCommand(newGenerateCmd(mgr))
	cmd.AddCommand(newShowCmd(mgr))
	cmd.AddCommand(newAttachCmd(mgr))

	return cmd
}

func newGenerateCmd(mgr *clarificationflow.Manager) *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "generate <task-id>",
		Short: "Generate a clarification request template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Create default questions template
			questions := []clarification.Question{
				{ID: "q1", Text: "Enter your first question here"},
				{ID: "q2", Text: "Enter your second question here"},
			}

			req, path, err := mgr.GenerateRequest(taskID, questions, reason)
			if err != nil {
				return err
			}

			fmt.Printf("Generated clarification request %s\n", req.RequestID)
			fmt.Printf("File: %s\n", path)
			fmt.Println("\nEdit the questions, then create a clarification file with answers.")
			fmt.Printf("Attach with: agentctl clarification attach %s <clarification-file>\n", taskID)
			return nil
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Reason for clarification")
	return cmd
}

func newShowCmd(mgr *clarificationflow.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show pending clarification request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := mgr.ShowPending(args[0])
			if err != nil {
				return err
			}

			data, err := yaml.Marshal(req)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		},
	}
}

func newAttachCmd(mgr *clarificationflow.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "attach <task-id> <clarification-file>",
		Short: "Attach a filled clarification to a task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			path := args[1]

			if err := mgr.AttachClarification(taskID, path); err != nil {
				return err
			}

			fmt.Printf("Clarification attached to task %s\n", taskID)
			fmt.Printf("Task is now ready to resume. Run: agentctl task resume %s\n", taskID)
			return nil
		},
	}
}
