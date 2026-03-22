package command

import (
	. "github.com/docup/agentctl/internal/app/command"
	"testing"

	"github.com/docup/agentctl/internal/app/dto"
	"github.com/docup/agentctl/internal/core/task"
)

func TestUpdateTask_AddTemplate(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{Title: "T", Goal: "G"})

	updateHandler := NewUpdateTask(store)
	tk, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID:       "TASK-001",
		AddTemplates: []string{"clarify_if_needed"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !tk.HasTemplate("clarify_if_needed") {
		t.Error("should have clarify_if_needed after update")
	}
}

func TestUpdateTask_RemoveTemplate(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{
		Title:     "T",
		Goal:      "G",
		Templates: []string{"strict_executor", "clarify_if_needed"},
	})

	updateHandler := NewUpdateTask(store)
	tk, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID:          "TASK-001",
		RemoveTemplates: []string{"clarify_if_needed"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if tk.HasTemplate("clarify_if_needed") {
		t.Error("clarify_if_needed should be removed")
	}
	if !tk.HasTemplate("strict_executor") {
		t.Error("strict_executor should remain")
	}
}

func TestUpdateTask_AddDuplicateTemplate(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{
		Title:     "T",
		Goal:      "G",
		Templates: []string{"strict_executor"},
	})

	updateHandler := NewUpdateTask(store)
	tk, _ := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID:       "TASK-001",
		AddTemplates: []string{"strict_executor"},
	})
	count := 0
	for _, tmpl := range tk.PromptTemplates.Builtin {
		if tmpl == "strict_executor" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 strict_executor, got %d (duplicates)", count)
	}
}

func TestUpdateTask_NonDraftStatus(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{Title: "T", Goal: "G"})

	tk, _ := store.Load("TASK-001")
	tk.Status = task.StatusQueued
	store.Save(tk)

	updateHandler := NewUpdateTask(store)
	_, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID:       "TASK-001",
		AddTemplates: []string{"review_only"},
	})
	if err == nil {
		t.Fatal("expected error for non-draft task")
	}
}

func TestUpdateTask_AddGuideline(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{Title: "T", Goal: "G"})

	updateHandler := NewUpdateTask(store)
	tk, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID:        "TASK-001",
		AddGuidelines: []string{"backend-guidelines"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(tk.Guidelines) != 1 || tk.Guidelines[0] != "backend-guidelines" {
		t.Errorf("wrong guidelines: %v", tk.Guidelines)
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	_, store := setupCreateTask(t)
	updateHandler := NewUpdateTask(store)
	_, err := updateHandler.Execute(dto.UpdateTaskRequest{TaskID: "NONEXISTENT"})
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}
