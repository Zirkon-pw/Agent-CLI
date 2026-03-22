package runtimecontrol

import (
	. "github.com/docup/agentctl/internal/service/runtimecontrol"
	"os"
	"path/filepath"
	"testing"
	"time"

	rt "github.com/docup/agentctl/internal/core/runtime"
	"github.com/docup/agentctl/internal/infra/events"
	infrart "github.com/docup/agentctl/internal/infra/runtime"
)

func setupRuntimeMgr(t *testing.T) (*Manager, *infrart.Registry) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "runtime"), 0755)

	registry := infrart.NewRegistry(dir)
	heartbeatMgr := infrart.NewHeartbeatManager(dir)
	eventSink := events.NewSink(filepath.Join(dir, "runtime"))
	mgr := NewManager(registry, heartbeatMgr, eventSink, 30)
	return mgr, registry
}

func TestActiveRuns_Empty(t *testing.T) {
	mgr, _ := setupRuntimeMgr(t)
	runs, err := mgr.ActiveRuns()
	if err != nil {
		t.Fatalf("active runs: %v", err)
	}
	if runs != nil {
		t.Error("expected nil for no active runs")
	}
}

func TestActiveRuns_WithRuns(t *testing.T) {
	mgr, registry := setupRuntimeMgr(t)

	registry.RegisterRun(rt.ActiveRun{
		TaskID:    "TASK-001",
		RunID:     "RUN-001",
		Agent:     "claude",
		StartedAt: time.Now(),
	})

	runs, err := mgr.ActiveRuns()
	if err != nil {
		t.Fatalf("active runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1, got %d", len(runs))
	}
}

func TestIsRunning(t *testing.T) {
	mgr, registry := setupRuntimeMgr(t)

	if mgr.IsRunning("TASK-001") {
		t.Error("should not be running")
	}

	registry.RegisterRun(rt.ActiveRun{
		TaskID:    "TASK-001",
		RunID:     "RUN-001",
		StartedAt: time.Now(),
	})

	if !mgr.IsRunning("TASK-001") {
		t.Error("should be running")
	}
}

func TestInspect(t *testing.T) {
	mgr, registry := setupRuntimeMgr(t)

	registry.RegisterRun(rt.ActiveRun{
		TaskID:    "TASK-001",
		RunID:     "RUN-001",
		Agent:     "claude",
		StartedAt: time.Now(),
	})

	info, err := mgr.Inspect("TASK-001")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !info.IsRunning {
		t.Error("should show as running")
	}
	if info.ActiveRun == nil {
		t.Error("should have active run")
	}
	if info.ActiveRun.Agent != "claude" {
		t.Errorf("wrong agent: %s", info.ActiveRun.Agent)
	}
}

func TestInspect_NotRunning(t *testing.T) {
	mgr, _ := setupRuntimeMgr(t)

	info, err := mgr.Inspect("TASK-001")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if info.IsRunning {
		t.Error("should not be running")
	}
	if info.ActiveRun != nil {
		t.Error("should not have active run")
	}
}

func TestTaskEvents(t *testing.T) {
	mgr, _ := setupRuntimeMgr(t)

	// Events need to be emitted through the sink (which is internal to mgr)
	// We test via the Inspect path which reads events
	info, _ := mgr.Inspect("TASK-001")
	if len(info.Events) != 0 {
		t.Error("expected no events for new task")
	}
}
