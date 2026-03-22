package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	rt "github.com/docup/agentctl/internal/core/runtime"
)

// Registry manages active run tracking and file-based locks.
type Registry struct {
	baseDir string // .agentctl/runtime
	mu      sync.Mutex
}

// NewRegistry creates a new runtime registry.
func NewRegistry(agentctlDir string) *Registry {
	return &Registry{baseDir: filepath.Join(agentctlDir, "runtime")}
}

// RegisterRun adds a run to the active runs list and creates a lock.
func (r *Registry) RegisterRun(active rt.ActiveRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	taskDir := filepath.Join(r.baseDir, active.TaskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return err
	}

	// Check for existing lock
	lockPath := filepath.Join(taskDir, "lock")
	if _, err := os.Stat(lockPath); err == nil {
		return fmt.Errorf("task %s is already locked (another run in progress)", active.TaskID)
	}

	// Create lock file
	if err := os.WriteFile(lockPath, []byte(active.RunID), 0644); err != nil {
		return err
	}

	// Write runtime.json
	data, err := json.MarshalIndent(active, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(taskDir, "runtime.json"), data, 0644); err != nil {
		return err
	}

	// Update active_runs.json
	return r.addToActiveRuns(active)
}

// UnregisterRun removes a run from active tracking and releases the lock.
func (r *Registry) UnregisterRun(taskID, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	taskDir := filepath.Join(r.baseDir, taskID)
	os.Remove(filepath.Join(taskDir, "lock"))
	os.Remove(filepath.Join(taskDir, "runtime.json"))
	os.Remove(filepath.Join(taskDir, "heartbeat.json"))
	os.Remove(filepath.Join(taskDir, "control.signal"))

	return r.removeFromActiveRuns(taskID)
}

// GetActiveRuns returns all currently registered active runs.
func (r *Registry) GetActiveRuns() ([]rt.ActiveRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	path := filepath.Join(r.baseDir, "active_runs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var runs []rt.ActiveRun
	return runs, json.Unmarshal(data, &runs)
}

// IsLocked checks if a task has an active lock.
func (r *Registry) IsLocked(taskID string) bool {
	lockPath := filepath.Join(r.baseDir, taskID, "lock")
	_, err := os.Stat(lockPath)
	return err == nil
}

// WriteSignal writes a control signal for a running task.
func (r *Registry) WriteSignal(taskID string, signal rt.Signal) error {
	taskDir := filepath.Join(r.baseDir, taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(taskDir, "control.signal"), []byte(signal), 0644)
}

// ReadSignal reads the current control signal for a task.
func (r *Registry) ReadSignal(taskID string) (rt.Signal, error) {
	data, err := os.ReadFile(filepath.Join(r.baseDir, taskID, "control.signal"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return rt.Signal(data), nil
}

// ClearSignal removes the control signal file.
func (r *Registry) ClearSignal(taskID string) error {
	return os.Remove(filepath.Join(r.baseDir, taskID, "control.signal"))
}

func (r *Registry) addToActiveRuns(active rt.ActiveRun) error {
	path := filepath.Join(r.baseDir, "active_runs.json")
	var runs []rt.ActiveRun
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &runs)
	}
	runs = append(runs, active)
	data, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (r *Registry) removeFromActiveRuns(taskID string) error {
	path := filepath.Join(r.baseDir, "active_runs.json")
	var runs []rt.ActiveRun
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &runs)
	}
	var filtered []rt.ActiveRun
	for _, run := range runs {
		if run.TaskID != taskID {
			filtered = append(filtered, run)
		}
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
