package fsstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docup/agentctl/internal/core/run"
)

// RunStore handles reading and writing run artifacts.
type RunStore struct {
	baseDir string // .agentctl/runs
}

// NewRunStore creates a new RunStore.
func NewRunStore(agentctlDir string) *RunStore {
	return &RunStore{baseDir: filepath.Join(agentctlDir, "runs")}
}

// RunDir returns the path for a specific run.
func (s *RunStore) RunDir(taskID, runID string) string {
	return filepath.Join(s.baseDir, taskID, runID)
}

// Save writes run metadata to disk.
func (s *RunStore) Save(r *run.Run) error {
	dir := s.RunDir(r.TaskID, r.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating run dir: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling run: %w", err)
	}
	path := filepath.Join(dir, "metadata.json")
	return os.WriteFile(path, data, 0644)
}

// Load reads run metadata from disk.
func (s *RunStore) Load(taskID, runID string) (*run.Run, error) {
	path := filepath.Join(s.RunDir(taskID, runID), "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading run %s/%s: %w", taskID, runID, err)
	}
	var r run.Run
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing run: %w", err)
	}
	return &r, nil
}

// ListRuns returns all runs for a task, newest first.
func (s *RunStore) ListRuns(taskID string) ([]*run.Run, error) {
	taskDir := filepath.Join(s.baseDir, taskID)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var runs []*run.Run
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		r, err := s.Load(taskID, entry.Name())
		if err != nil {
			continue
		}
		runs = append(runs, r)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
	return runs, nil
}

// LatestRun returns the most recent run for a task.
func (s *RunStore) LatestRun(taskID string) (*run.Run, error) {
	runs, err := s.ListRuns(taskID)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("no runs found for task %s", taskID)
	}
	return runs[0], nil
}

// NextRunID generates the next run ID for a task.
func (s *RunStore) NextRunID(taskID string) (string, error) {
	taskDir := filepath.Join(s.baseDir, taskID)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "RUN-001", nil
		}
		return "", err
	}
	maxNum := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "RUN-") {
			continue
		}
		numStr := strings.TrimPrefix(entry.Name(), "RUN-")
		var num int
		if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil && num > maxNum {
			maxNum = num
		}
	}
	return fmt.Sprintf("RUN-%03d", maxNum+1), nil
}

// WriteArtifact writes a named artifact file into a run directory.
func (s *RunStore) WriteArtifact(taskID, runID, filename string, data []byte) error {
	dir := s.RunDir(taskID, runID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), data, 0644)
}

// ReadArtifact reads a named artifact from a run directory.
func (s *RunStore) ReadArtifact(taskID, runID, filename string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.RunDir(taskID, runID), filename))
}
