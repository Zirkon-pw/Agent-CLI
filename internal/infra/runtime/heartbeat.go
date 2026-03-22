package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	rt "github.com/docup/agentctl/internal/core/runtime"
)

// HeartbeatManager handles heartbeat read/write operations.
type HeartbeatManager struct {
	baseDir string
}

// NewHeartbeatManager creates a new HeartbeatManager.
func NewHeartbeatManager(agentctlDir string) *HeartbeatManager {
	return &HeartbeatManager{baseDir: filepath.Join(agentctlDir, "runtime")}
}

// Write updates the heartbeat file for a task.
func (h *HeartbeatManager) Write(taskID, runID string) error {
	hb := rt.Heartbeat{
		TaskID:    taskID,
		RunID:     runID,
		Timestamp: time.Now(),
		Alive:     true,
	}
	data, err := json.MarshalIndent(hb, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(h.baseDir, taskID)
	os.MkdirAll(dir, 0755)
	return os.WriteFile(filepath.Join(dir, "heartbeat.json"), data, 0644)
}

// Read loads the heartbeat for a task.
func (h *HeartbeatManager) Read(taskID string) (*rt.Heartbeat, error) {
	data, err := os.ReadFile(filepath.Join(h.baseDir, taskID, "heartbeat.json"))
	if err != nil {
		return nil, err
	}
	var hb rt.Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return nil, err
	}
	return &hb, nil
}

// IsStale checks if the heartbeat for a task is stale.
func (h *HeartbeatManager) IsStale(taskID string, threshold time.Duration) (bool, error) {
	hb, err := h.Read(taskID)
	if err != nil {
		return true, err
	}
	return hb.IsStale(threshold), nil
}
