package fsstore

import (
	. "github.com/docup/agentctl/internal/infra/fsstore"
	"path/filepath"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/core/run"
)

func makeRun(id, taskID string) *run.Run {
	return &run.Run{
		ID:        id,
		TaskID:    taskID,
		Status:    run.RunStatusPending,
		Agent:     "claude",
		CreatedAt: time.Now(),
	}
}

func TestRunStore_SaveAndLoad(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	r := makeRun("RUN-001", "TASK-001")
	if err := store.Save(r); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load("TASK-001", "RUN-001")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ID != "RUN-001" {
		t.Errorf("expected RUN-001, got %s", loaded.ID)
	}
	if loaded.Agent != "claude" {
		t.Errorf("wrong agent: %s", loaded.Agent)
	}
}

func TestRunStore_Load_NotFound(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	_, err := store.Load("TASK-001", "RUN-999")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunStore_ListRuns(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	store.Save(makeRun("RUN-001", "TASK-001"))
	time.Sleep(time.Millisecond)
	store.Save(makeRun("RUN-002", "TASK-001"))

	runs, err := store.ListRuns("TASK-001")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
}

func TestRunStore_ListRuns_NoRuns(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	runs, err := store.ListRuns("TASK-001")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if runs != nil {
		t.Error("expected nil for task with no runs")
	}
}

func TestRunStore_LatestRun(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	r1 := makeRun("RUN-001", "TASK-001")
	store.Save(r1)
	time.Sleep(time.Millisecond)

	r2 := makeRun("RUN-002", "TASK-001")
	store.Save(r2)

	latest, err := store.LatestRun("TASK-001")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.ID != "RUN-002" {
		t.Errorf("expected RUN-002, got %s", latest.ID)
	}
}

func TestRunStore_LatestRun_NoRuns(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	_, err := store.LatestRun("TASK-001")
	if err == nil {
		t.Fatal("expected error for task with no runs")
	}
}

func TestRunStore_NextRunID(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	id, _ := store.NextRunID("TASK-001")
	if id != "RUN-001" {
		t.Errorf("expected RUN-001, got %s", id)
	}

	store.Save(makeRun("RUN-001", "TASK-001"))
	id, _ = store.NextRunID("TASK-001")
	if id != "RUN-002" {
		t.Errorf("expected RUN-002, got %s", id)
	}
}

func TestRunStore_Artifacts(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	store.Save(makeRun("RUN-001", "TASK-001"))

	content := []byte("test artifact content")
	if err := store.WriteArtifact("TASK-001", "RUN-001", "test.txt", content); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	data, err := store.ReadArtifact("TASK-001", "RUN-001", "test.txt")
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(data) != "test artifact content" {
		t.Errorf("wrong content: %s", string(data))
	}
}

func TestRunStore_ReadArtifact_NotFound(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewRunStore(dir)

	_, err := store.ReadArtifact("TASK-001", "RUN-001", "missing.txt")
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
}

func TestRunStore_RunDir(t *testing.T) {
	store := NewRunStore("/tmp/.agentctl")
	dir := store.RunDir("TASK-001", "RUN-001")
	expected := filepath.Join("/tmp/.agentctl", "runs", "TASK-001", "RUN-001")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}
