package fsstore

import (
	. "github.com/docup/agentctl/internal/infra/fsstore"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/core/clarification"
)

func TestClarificationStore_SaveAndLoadRequest(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewClarificationStore(dir)

	req := &clarification.Request{
		TaskID:    "TASK-001",
		RequestID: "CLAR-REQ-001",
		CreatedBy: "claude",
		Reason:    "ambiguous scope",
		Questions: []clarification.Question{
			{ID: "q1", Text: "What about X?"},
		},
		CreatedAt: time.Now(),
	}

	path, err := store.SaveRequest(req)
	if err != nil {
		t.Fatalf("save request: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}

	loaded, err := store.LoadRequest("TASK-001", "CLAR-REQ-001")
	if err != nil {
		t.Fatalf("load request: %v", err)
	}
	if loaded.RequestID != "CLAR-REQ-001" {
		t.Errorf("wrong request ID: %s", loaded.RequestID)
	}
	if len(loaded.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(loaded.Questions))
	}
}

func TestClarificationStore_LoadRequest_NotFound(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewClarificationStore(dir)

	_, err := store.LoadRequest("TASK-001", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent request")
	}
}

func TestClarificationStore_SaveAndLoadClarification(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewClarificationStore(dir)

	clar := &clarification.Clarification{
		TaskID:          "TASK-001",
		RequestID:       "CLAR-REQ-001",
		ClarificationID: "CLAR-001",
		Answers: []clarification.Answer{
			{QuestionID: "q1", Text: "Answer 1"},
		},
		Notes:     []string{"Keep API stable"},
		CreatedAt: time.Now(),
	}

	path, err := store.SaveClarification(clar)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.LoadClarification(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ClarificationID != "CLAR-001" {
		t.Errorf("wrong ID: %s", loaded.ClarificationID)
	}
	if len(loaded.Answers) != 1 {
		t.Error("expected 1 answer")
	}
}

func TestClarificationStore_ListClarifications(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewClarificationStore(dir)

	// Save a request (should NOT appear in list)
	store.SaveRequest(&clarification.Request{
		TaskID:    "TASK-001",
		RequestID: "CLAR-REQ-001",
		CreatedAt: time.Now(),
	})

	// Save a clarification (should appear)
	store.SaveClarification(&clarification.Clarification{
		TaskID:          "TASK-001",
		ClarificationID: "CLAR-001",
		CreatedAt:       time.Now(),
	})

	clars, err := store.ListClarifications("TASK-001")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(clars) != 1 {
		t.Fatalf("expected 1 clarification (not request), got %d", len(clars))
	}
}

func TestClarificationStore_ListClarifications_Empty(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewClarificationStore(dir)

	clars, err := store.ListClarifications("TASK-001")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if clars != nil {
		t.Error("expected nil for nonexistent task")
	}
}
