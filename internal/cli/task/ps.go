package task

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/docup/agentctl/internal/service/runtimecontrol"
	"github.com/spf13/cobra"
)

// NewPsCmd creates the task ps command.
func NewPsCmd(rtMgr *runtimecontrol.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List actively running tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			runs, err := rtMgr.ActiveRuns()
			if err != nil {
				return err
			}

			if len(runs) == 0 {
				fmt.Println("No active tasks.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TASK\tRUN\tAGENT\tDURATION\tSTARTED")
			for _, r := range runs {
				duration := time.Since(r.StartedAt).Truncate(time.Second)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					r.TaskID, r.RunID, r.Agent,
					duration, r.StartedAt.Format("15:04:05"))
			}
			w.Flush()
			return nil
		},
	}
	return cmd
}
