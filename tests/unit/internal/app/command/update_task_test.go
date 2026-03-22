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

func TestUpdateTask_SetScalars(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{Title: "T", Goal: "G", Agent: "claude"})

	updateHandler := NewUpdateTask(store)
	title := "Updated title"
	goal := "Updated goal"
	agent := ""
	tk, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID: "TASK-001",
		Title:  &title,
		Goal:   &goal,
		Agent:  &agent,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if tk.Title != title {
		t.Errorf("expected title %q, got %q", title, tk.Title)
	}
	if tk.Goal != goal {
		t.Errorf("expected goal %q, got %q", goal, tk.Goal)
	}
	if tk.Agent != "" {
		t.Errorf("expected cleared agent, got %q", tk.Agent)
	}
}

func TestUpdateTask_ScopePatch(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{
		Title: "T",
		Goal:  "G",
		Scope: dto.ScopeDTO{
			AllowedPaths:   []string{"src/"},
			ForbiddenPaths: []string{"vendor/"},
			MustRead:       []string{"README.md"},
		},
	})

	updateHandler := NewUpdateTask(store)
	tk, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID:               "TASK-001",
		AddAllowedPaths:      []string{"tests/"},
		RemoveAllowedPaths:   []string{"src/"},
		AddForbiddenPaths:    []string{"tmp/"},
		RemoveForbiddenPaths: []string{"vendor/"},
		AddMustRead:          []string{"go.mod"},
		RemoveMustRead:       []string{"README.md"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(tk.Scope.AllowedPaths) != 1 || tk.Scope.AllowedPaths[0] != "tests/" {
		t.Errorf("wrong allowed paths: %v", tk.Scope.AllowedPaths)
	}
	if len(tk.Scope.ForbiddenPaths) != 1 || tk.Scope.ForbiddenPaths[0] != "tmp/" {
		t.Errorf("wrong forbidden paths: %v", tk.Scope.ForbiddenPaths)
	}
	if len(tk.Scope.MustRead) != 1 || tk.Scope.MustRead[0] != "go.mod" {
		t.Errorf("wrong must-read: %v", tk.Scope.MustRead)
	}
}

func TestUpdateTask_GenericSet(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{Title: "T", Goal: "G"})

	updateHandler := NewUpdateTask(store)
	tk, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID: "TASK-001",
		Mutations: []dto.TaskMutation{
			{Kind: dto.MutationSet, Path: "validation.mode", Value: "full"},
			{Kind: dto.MutationSet, Path: "runtime.max_execution_minutes", Value: 15},
			{Kind: dto.MutationSet, Path: "context.include_patterns", Value: []string{"internal/**/*.go"}},
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if tk.Validation.Mode != task.ValidationModeFull {
		t.Errorf("expected full validation mode, got %s", tk.Validation.Mode)
	}
	if tk.Runtime.MaxExecutionMinutes != 15 {
		t.Errorf("expected max execution minutes 15, got %d", tk.Runtime.MaxExecutionMinutes)
	}
	if len(tk.Context.IncludePatterns) != 1 || tk.Context.IncludePatterns[0] != "internal/**/*.go" {
		t.Errorf("wrong include patterns: %v", tk.Context.IncludePatterns)
	}
}

func TestUpdateTask_GenericAddRemove(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{
		Title:     "T",
		Goal:      "G",
		Templates: []string{"strict_executor"},
	})

	updateHandler := NewUpdateTask(store)
	tk, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID: "TASK-001",
		Mutations: []dto.TaskMutation{
			{Kind: dto.MutationAdd, Path: "validation.commands", Value: "go test ./..."},
			{Kind: dto.MutationAdd, Path: "validation.commands", Value: "go test ./..."},
			{Kind: dto.MutationAdd, Path: "guidelines", Value: "backend"},
			{Kind: dto.MutationRemove, Path: "prompt_templates.builtin", Value: "strict_executor"},
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(tk.Validation.Commands) != 1 || tk.Validation.Commands[0] != "go test ./..." {
		t.Errorf("wrong validation commands: %v", tk.Validation.Commands)
	}
	if len(tk.Guidelines) != 1 || tk.Guidelines[0] != "backend" {
		t.Errorf("wrong guidelines: %v", tk.Guidelines)
	}
	if len(tk.PromptTemplates.Builtin) != 0 {
		t.Errorf("expected templates to be removed, got %v", tk.PromptTemplates.Builtin)
	}
}

func TestUpdateTask_GenericOverlap(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{Title: "T", Goal: "G"})

	updateHandler := NewUpdateTask(store)
	_, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID:       "TASK-001",
		AddTemplates: []string{"strict_executor"},
		Mutations: []dto.TaskMutation{
			{Kind: dto.MutationAdd, Path: "prompt_templates.builtin", Value: "clarify_if_needed"},
		},
	})
	if err == nil {
		t.Fatal("expected overlap error")
	}
}

func TestUpdateTask_GenericInvalidPath(t *testing.T) {
	createHandler, store := setupCreateTask(t)
	createHandler.Execute(dto.CreateTaskRequest{Title: "T", Goal: "G"})

	updateHandler := NewUpdateTask(store)
	_, err := updateHandler.Execute(dto.UpdateTaskRequest{
		TaskID: "TASK-001",
		Mutations: []dto.TaskMutation{
			{Kind: dto.MutationSet, Path: "status", Value: "running"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid path error")
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
