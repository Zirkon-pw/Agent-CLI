package task

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/docup/agentctl/internal/app/query"
	"github.com/spf13/cobra"
)

// NewListCmd creates the task list command.
func NewListCmd(handler *query.ListTasks) *cobra.Command {
	var status string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := handler.Execute()
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tAGENT\tCREATED")
			for _, t := range tasks {
				if status != "" && t.Status != status {
					continue
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					t.ID, truncate(t.Title, 40), t.Status, t.Agent,
					t.CreatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	return cmd
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
