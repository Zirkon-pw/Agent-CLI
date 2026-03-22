package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docup/agentctl/internal/core/task"
	"gopkg.in/yaml.v3"
)

// TaskStore handles reading and writing task YAML files.
type TaskStore struct {
	baseDir string // .agentctl/tasks
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore(agentctlDir string) *TaskStore {
	return &TaskStore{baseDir: filepath.Join(agentctlDir, "tasks")}
}

// Save writes a task to disk as YAML.
func (s *TaskStore) Save(t *task.Task) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("creating tasks dir: %w", err)
	}
	data, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshaling task %s: %w", t.ID, err)
	}
	path := filepath.Join(s.baseDir, t.ID+".yml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing task %s: %w", t.ID, err)
	}
	return nil
}

// Load reads a task from disk by ID.
func (s *TaskStore) Load(id string) (*task.Task, error) {
	path := filepath.Join(s.baseDir, id+".yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task %s not found", id)
		}
		return nil, fmt.Errorf("reading task %s: %w", id, err)
	}
	var t task.Task
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing task %s: %w", id, err)
	}
	return &t, nil
}

// List returns all tasks sorted by creation time (newest first).
func (s *TaskStore) List() ([]*task.Task, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	var tasks []*task.Task
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".yml")
		t, err := s.Load(id)
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
	return tasks, nil
}

// Exists checks if a task file exists.
func (s *TaskStore) Exists(id string) bool {
	path := filepath.Join(s.baseDir, id+".yml")
	_, err := os.Stat(path)
	return err == nil
}

// NextID generates the next task ID based on existing tasks.
func (s *TaskStore) NextID() (string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "TASK-001", nil
		}
		return "", err
	}
	maxNum := 0
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".yml")
		if strings.HasPrefix(name, "TASK-") {
			numStr := strings.TrimPrefix(name, "TASK-")
			var num int
			if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil {
				if num > maxNum {
					maxNum = num
				}
			}
		}
	}
	return fmt.Sprintf("TASK-%03d", maxNum+1), nil
}
