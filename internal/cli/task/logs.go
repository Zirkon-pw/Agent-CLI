package task

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rt "github.com/docup/agentctl/internal/core/runtime"
	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/spf13/cobra"
)

// NewLogsCmd creates the task logs command.
func NewLogsCmd(runStore *fsstore.RunStore) *cobra.Command {
	var follow bool
	var stageID string
	var protocol bool

	cmd := &cobra.Command{
		Use:   "logs <task-id>",
		Short: "Show execution logs for a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			session, err := runStore.LatestSession(taskID)
			if err != nil {
				return fmt.Errorf("no sessions found for task %s", taskID)
			}

			logsPath, err := resolveLogsPath(runStore, taskID, session, stageID, protocol)
			if err != nil {
				return err
			}

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
	cmd.Flags().StringVar(&stageID, "stage", "", "Read logs for a specific stage ID")
	cmd.Flags().BoolVar(&protocol, "protocol", false, "Show raw protocol.ndjson log for the latest session")
	return cmd
}

func resolveLogsPath(runStore *fsstore.RunStore, taskID string, session *rt.RunSession, stageID string, protocol bool) (string, error) {
	if protocol {
		path := filepath.Join(runStore.RunDir(taskID, session.ID), "protocol.ndjson")
		if existsAndNotEmpty(path) {
			return path, nil
		}
		return "", fmt.Errorf("no protocol log found for session %s", session.ID)
	}

	stage := resolveStage(session, stageID)
	if stage == nil {
		return "", fmt.Errorf("no stage found for task %s", taskID)
	}

	stageDir := runStore.StageDir(taskID, session.ID, stage.StageID)
	candidates := []string{
		filepath.Join(stageDir, "runtime_errors.log"),
		filepath.Join(stageDir, "adapter.stderr.log"),
		filepath.Join(stageDir, "raw.stderr.log"),
		filepath.Join(stageDir, "raw.stdout.log"),
	}
	for _, candidate := range candidates {
		if existsAndNotEmpty(candidate) {
			return candidate, nil
		}
	}
	for _, candidate := range candidates {
		if exists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no logs found for stage %s; try --protocol for session log", stage.StageID)
}

func resolveStage(session *rt.RunSession, stageID string) *rt.StageRun {
	if stageID == "" {
		return session.LastStage()
	}
	for i := range session.StageHistory {
		if session.StageHistory[i].StageID == stageID {
			return &session.StageHistory[i]
		}
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func existsAndNotEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
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
			if _, err := f.Seek(lastSize, 0); err != nil {
				_ = f.Close()
				return err
			}
			buf := make([]byte, info.Size()-lastSize)
			n, _ := f.Read(buf)
			_ = f.Close()
			fmt.Print(string(buf[:n]))
			lastSize = info.Size()
		}
		time.Sleep(500 * time.Millisecond)
	}
}
