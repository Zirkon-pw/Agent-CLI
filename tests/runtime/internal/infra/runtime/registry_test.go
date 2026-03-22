package runtime

import (
	. "github.com/docup/agentctl/internal/infra/runtime"
	"os"
	"path/filepath"
	"testing"
	"time"

	rt "github.com/docup/agentctl/internal/core/runtime"
)

func tmpRuntimeDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "runtime"), 0755)
	return dir
}

func TestRegistry_RegisterAndUnregister(t *testing.T) {
	dir := tmpRuntimeDir(t)
	reg := NewRegistry(dir)

	active := rt.ActiveRun{
		TaskID:    "TASK-001",
		RunID:     "RUN-001",
		Agent:     "claude",
		PID:       1234,
		StartedAt: time.Now(),
	}

	if err := reg.RegisterRun(active); err != nil {
		t.Fatalf("register: %v", err)
	}

	if !reg.IsLocked("TASK-001") {
		t.Error("task should be locked after register")
	}

	runs, err := reg.GetActiveRuns()
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 active run, got %d", len(runs))
	}
	if runs[0].TaskID != "TASK-001" {
		t.Errorf("wrong task ID: %s", runs[0].TaskID)
	}

	if err := reg.UnregisterRun("TASK-001", "RUN-001"); err != nil {
		t.Fatalf("unregister: %v", err)
	}

	if reg.IsLocked("TASK-001") {
		t.Error("task should NOT be locked after unregister")
	}

	runs, _ = reg.GetActiveRuns()
	if len(runs) != 0 {
		t.Errorf("expected 0 active runs, got %d", len(runs))
	}
}

func TestRegistry_RegisterAlreadyLocked(t *testing.T) {
	dir := tmpRuntimeDir(t)
	reg := NewRegistry(dir)

	active := rt.ActiveRun{TaskID: "TASK-001", RunID: "RUN-001", StartedAt: time.Now()}
	reg.RegisterRun(active)

	active2 := rt.ActiveRun{TaskID: "TASK-001", RunID: "RUN-002", StartedAt: time.Now()}
	if err := reg.RegisterRun(active2); err == nil {
		t.Fatal("expected error when registering already locked task")
	}
}

func TestRegistry_Signals(t *testing.T) {
	dir := tmpRuntimeDir(t)
	reg := NewRegistry(dir)

	// No signal initially
	sig, err := reg.ReadSignal("TASK-001")
	if err != nil {
		t.Fatalf("read signal: %v", err)
	}
	if sig != "" {
		t.Errorf("expected empty signal, got %q", sig)
	}

	// Write signal
	if err := reg.WriteSignal("TASK-001", rt.SignalStop); err != nil {
		t.Fatalf("write signal: %v", err)
	}

	sig, err = reg.ReadSignal("TASK-001")
	if err != nil {
		t.Fatalf("read signal: %v", err)
	}
	if sig != rt.SignalStop {
		t.Errorf("expected stop, got %q", sig)
	}

	// Clear signal
	if err := reg.ClearSignal("TASK-001"); err != nil {
		t.Fatalf("clear signal: %v", err)
	}
	sig, _ = reg.ReadSignal("TASK-001")
	if sig != "" {
		t.Error("signal should be cleared")
	}
}

func TestRegistry_GetActiveRuns_NoFile(t *testing.T) {
	dir := tmpRuntimeDir(t)
	reg := NewRegistry(dir)

	runs, err := reg.GetActiveRuns()
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if runs != nil {
		t.Error("expected nil when no active_runs.json")
	}
}

func TestRegistry_IsLocked_NoLock(t *testing.T) {
	dir := tmpRuntimeDir(t)
	reg := NewRegistry(dir)

	if reg.IsLocked("NONEXISTENT") {
		t.Error("should not be locked")
	}
}
