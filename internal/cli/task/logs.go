package task

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/spf13/cobra"
)

// NewLogsCmd creates the task logs command.
func NewLogsCmd(runStore *fsstore.RunStore, agentctlDir string) *cobra.Command {
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs <task-id>",
		Short: "Show execution logs for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			r, err := runStore.LatestRun(taskID)
			if err != nil {
				return fmt.Errorf("no runs found for task %s", taskID)
			}

			logsPath := filepath.Join(runStore.RunDir(taskID, r.ID), "logs.txt")

			if follow {
				return followLogs(logsPath)
			}

			data, err := os.ReadFile(logsPath)
			if err != nil {
				return fmt.Errorf("reading logs: %w", err)
			}
			fmt.Print(string(data))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	return cmd
}

func followLogs(path string) error {
	var lastSize int64
	for {
		info, err := os.Stat(path)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if info.Size() > lastSize {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			f.Seek(lastSize, 0)
			buf := make([]byte, info.Size()-lastSize)
			n, _ := f.Read(buf)
			f.Close()
			fmt.Print(string(buf[:n]))
			lastSize = info.Size()
		}
		time.Sleep(500 * time.Millisecond)
	}
}
