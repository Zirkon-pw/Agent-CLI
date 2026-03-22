package fsstore

import (
	. "github.com/docup/agentctl/internal/infra/fsstore"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/core/task"
)

func tmpAgentctlDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	agentctlDir := filepath.Join(dir, ".agentctl")
	os.MkdirAll(filepath.Join(agentctlDir, "tasks"), 0755)
	return agentctlDir
}

func makeTask(id string) *task.Task {
	now := time.Now()
	return &task.Task{
		ID:         id,
		Title:      "Test " + id,
		Goal:       "Goal for " + id,
		Status:     task.StatusDraft,
		Agent:      "claude",
		Runtime:    task.DefaultRuntimeConfig(),
		Validation: task.DefaultValidationConfig(),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestTaskStore_SaveAndLoad(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTaskStore(dir)

	tk := makeTask("TASK-001")
	if err := store.Save(tk); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load("TASK-001")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ID != "TASK-001" {
		t.Errorf("expected TASK-001, got %s", loaded.ID)
	}
	if loaded.Title != "Test TASK-001" {
		t.Errorf("wrong title: %s", loaded.Title)
	}
	if loaded.Status != task.StatusDraft {
		t.Errorf("wrong status: %s", loaded.Status)
	}
}

func TestTaskStore_Load_NotFound(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTaskStore(dir)

	_, err := store.Load("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestTaskStore_List(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTaskStore(dir)

	store.Save(makeTask("TASK-001"))
	time.Sleep(time.Millisecond)
	store.Save(makeTask("TASK-002"))

	tasks, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestTaskStore_List_EmptyDir(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTaskStore(dir)

	tasks, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestTaskStore_List_NoDir(t *testing.T) {
	store := NewTaskStore(filepath.Join(t.TempDir(), "nonexistent"))
	tasks, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if tasks != nil {
		t.Error("expected nil for nonexistent dir")
	}
}

func TestTaskStore_Exists(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTaskStore(dir)

	store.Save(makeTask("TASK-001"))

	if !store.Exists("TASK-001") {
		t.Error("TASK-001 should exist")
	}
	if store.Exists("TASK-999") {
		t.Error("TASK-999 should not exist")
	}
}

func TestTaskStore_NextID(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTaskStore(dir)

	id, err := store.NextID()
	if err != nil {
		t.Fatalf("nextID: %v", err)
	}
	if id != "TASK-001" {
		t.Errorf("expected TASK-001, got %s", id)
	}

	store.Save(makeTask("TASK-001"))
	id, err = store.NextID()
	if err != nil {
		t.Fatalf("nextID: %v", err)
	}
	if id != "TASK-002" {
		t.Errorf("expected TASK-002, got %s", id)
	}

	store.Save(makeTask("TASK-010"))
	id, _ = store.NextID()
	if id != "TASK-011" {
		t.Errorf("expected TASK-011, got %s", id)
	}
}

func TestTaskStore_NextID_NoDir(t *testing.T) {
	store := NewTaskStore(filepath.Join(t.TempDir(), "nonexistent"))
	id, err := store.NextID()
	if err != nil {
		t.Fatalf("nextID: %v", err)
	}
	if id != "TASK-001" {
		t.Errorf("expected TASK-001, got %s", id)
	}
}
