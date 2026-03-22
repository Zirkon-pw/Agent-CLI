package clarificationflow

import (
	. "github.com/docup/agentctl/internal/service/clarificationflow"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/core/clarification"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/fsstore"
	"gopkg.in/yaml.v3"
)

func setupManager(t *testing.T) (*Manager, *fsstore.TaskStore, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "tasks"), 0755)
	os.MkdirAll(filepath.Join(dir, "clarifications"), 0755)

	taskStore := fsstore.NewTaskStore(dir)
	clarStore := fsstore.NewClarificationStore(dir)
	mgr := NewManager(taskStore, clarStore)
	return mgr, taskStore, dir
}

func createTestTask(store *fsstore.TaskStore, status task.TaskStatus) {
	store.Save(&task.Task{
		ID:     "TASK-001",
		Title:  "Test",
		Goal:   "Test",
		Status: status,
		Agent:  "claude",
		Clarifications: task.Clarifications{
			Attached: []string{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

func TestGenerateRequest_Draft(t *testing.T) {
	mgr, store, _ := setupManager(t)
	createTestTask(store, task.StatusDraft)

	questions := []clarification.Question{
		{ID: "q1", Text: "What about X?"},
	}

	req, path, err := mgr.GenerateRequest("TASK-001", questions, "ambiguous scope")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if req.RequestID != "CLAR-REQ-001" {
		t.Errorf("expected CLAR-REQ-001, got %s", req.RequestID)
	}
	if path == "" {
		t.Error("path should not be empty")
	}

	// Check pending request set on task
	tk, _ := store.Load("TASK-001")
	if tk.Clarifications.PendingRequest == nil {
		t.Fatal("pending request should be set")
	}
	if *tk.Clarifications.PendingRequest != "CLAR-REQ-001" {
		t.Errorf("wrong pending: %s", *tk.Clarifications.PendingRequest)
	}
}

func TestGenerateRequest_InvalidStatus(t *testing.T) {
	mgr, store, _ := setupManager(t)
	createTestTask(store, task.StatusRunning)

	_, _, err := mgr.GenerateRequest("TASK-001", nil, "")
	if err == nil {
		t.Fatal("expected error for running task")
	}
}

func TestAttachClarification(t *testing.T) {
	mgr, store, dir := setupManager(t)
	createTestTask(store, task.StatusNeedsClarification)

	// Set pending request
	tk, _ := store.Load("TASK-001")
	reqID := "CLAR-REQ-001"
	tk.SetPendingClarification(reqID)
	store.Save(tk)

	// Create clarification file
	clar := &clarification.Clarification{
		TaskID:          "TASK-001",
		RequestID:       "CLAR-REQ-001",
		ClarificationID: "CLAR-001",
		Answers: []clarification.Answer{
			{QuestionID: "q1", Text: "Answer 1"},
		},
		CreatedAt: time.Now(),
	}
	data, _ := yaml.Marshal(clar)
	clarPath := filepath.Join(dir, "clarifications", "TASK-001", "clarification_CLAR-001.yml")
	os.MkdirAll(filepath.Dir(clarPath), 0755)
	os.WriteFile(clarPath, data, 0644)

	// Attach
	if err := mgr.AttachClarification("TASK-001", clarPath); err != nil {
		t.Fatalf("attach: %v", err)
	}

	tk, _ = store.Load("TASK-001")
	if tk.Status != task.StatusReadyToResume {
		t.Errorf("expected ready_to_resume, got %s", tk.Status)
	}
	if len(tk.Clarifications.Attached) != 1 {
		t.Error("expected 1 attached")
	}
	if tk.Clarifications.PendingRequest != nil {
		t.Error("pending should be cleared")
	}
}

func TestAttachClarification_NoPending(t *testing.T) {
	mgr, store, _ := setupManager(t)
	createTestTask(store, task.StatusDraft)

	err := mgr.AttachClarification("TASK-001", "/fake/path")
	if err == nil {
		t.Fatal("expected error when no pending request")
	}
}

func TestAttachClarification_TaskMismatch(t *testing.T) {
	mgr, store, dir := setupManager(t)
	createTestTask(store, task.StatusNeedsClarification)

	tk, _ := store.Load("TASK-001")
	tk.SetPendingClarification("CLAR-REQ-001")
	store.Save(tk)

	// Create clar with wrong task ID
	clar := &clarification.Clarification{
		TaskID:          "TASK-999", // Mismatch!
		ClarificationID: "CLAR-001",
		CreatedAt:       time.Now(),
	}
	data, _ := yaml.Marshal(clar)
	clarPath := filepath.Join(dir, "wrong_clar.yml")
	os.WriteFile(clarPath, data, 0644)

	err := mgr.AttachClarification("TASK-001", clarPath)
	if err == nil {
		t.Fatal("expected error for task ID mismatch")
	}
}

func TestShowPending(t *testing.T) {
	mgr, store, _ := setupManager(t)
	createTestTask(store, task.StatusDraft)

	questions := []clarification.Question{{ID: "q1", Text: "Question"}}
	mgr.GenerateRequest("TASK-001", questions, "reason")

	req, err := mgr.ShowPending("TASK-001")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if req.RequestID != "CLAR-REQ-001" {
		t.Errorf("wrong request ID: %s", req.RequestID)
	}
}

func TestShowPending_NoPending(t *testing.T) {
	mgr, store, _ := setupManager(t)
	createTestTask(store, task.StatusDraft)

	_, err := mgr.ShowPending("TASK-001")
	if err == nil {
		t.Fatal("expected error when no pending request")
	}
}
