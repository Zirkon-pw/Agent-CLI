package runtime

import (
	. "github.com/docup/agentctl/internal/infra/runtime"
	"testing"
	"time"
)

func TestHeartbeatManager_WriteAndRead(t *testing.T) {
	dir := tmpRuntimeDir(t)
	mgr := NewHeartbeatManager(dir)

	if err := mgr.Write("TASK-001", "RUN-001"); err != nil {
		t.Fatalf("write: %v", err)
	}

	hb, err := mgr.Read("TASK-001")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if hb.TaskID != "TASK-001" {
		t.Errorf("wrong task: %s", hb.TaskID)
	}
	if hb.RunID != "RUN-001" {
		t.Errorf("wrong run: %s", hb.RunID)
	}
	if !hb.Alive {
		t.Error("should be alive")
	}
}

func TestHeartbeatManager_Read_NotFound(t *testing.T) {
	dir := tmpRuntimeDir(t)
	mgr := NewHeartbeatManager(dir)

	_, err := mgr.Read("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHeartbeatManager_IsStale(t *testing.T) {
	dir := tmpRuntimeDir(t)
	mgr := NewHeartbeatManager(dir)

	mgr.Write("TASK-001", "RUN-001")

	stale, err := mgr.IsStale("TASK-001", 5*time.Second)
	if err != nil {
		t.Fatalf("is stale: %v", err)
	}
	if stale {
		t.Error("fresh heartbeat should not be stale")
	}
}

func TestHeartbeatManager_IsStale_NoHeartbeat(t *testing.T) {
	dir := tmpRuntimeDir(t)
	mgr := NewHeartbeatManager(dir)

	stale, err := mgr.IsStale("NONEXISTENT", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for missing heartbeat")
	}
	if !stale {
		t.Error("missing heartbeat should be considered stale")
	}
}
