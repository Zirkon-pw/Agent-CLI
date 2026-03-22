package task

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/docup/agentctl/internal/cli/task"

	"github.com/docup/agentctl/internal/app/command"
	"github.com/docup/agentctl/internal/app/dto"
	"github.com/docup/agentctl/internal/config/loader"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

func setupUpdateCmd(t *testing.T) (*command.UpdateTask, *fsstore.TaskStore) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".agentctl")
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store := fsstore.NewTaskStore(dir)
	createHandler := command.NewCreateTask(store, loader.DefaultProjectConfig())
	if _, err := createHandler.Execute(dto.CreateTaskRequest{Title: "Initial", Goal: "Goal"}); err != nil {
		t.Fatalf("create seed task: %v", err)
	}
	return command.NewUpdateTask(store), store
}

func TestUpdateCmd_Flags(t *testing.T) {
	handler, _ := setupUpdateCmd(t)
	cmd := NewUpdateCmd(handler)

	flags := []string{
		"title", "goal", "agent",
		"add-template", "remove-template",
		"add-guideline", "remove-guideline",
		"add-allowed-path", "remove-allowed-path",
		"add-forbidden-path", "remove-forbidden-path",
		"add-must-read", "remove-must-read",
		"set", "add", "remove",
	}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s to exist", flag)
		}
	}
}

func TestUpdateCmd_Success(t *testing.T) {
	handler, store := setupUpdateCmd(t)
	cmd := NewUpdateCmd(handler)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"TASK-001",
		"--title", "Configured",
		"--agent", "codex",
		"--add-template", "clarify_if_needed",
		"--set", "validation.mode=full",
		"--add", "validation.commands=go test ./...",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	task, err := store.Load("TASK-001")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if task.Title != "Configured" {
		t.Errorf("expected title to be updated, got %q", task.Title)
	}
	if task.Agent != "codex" {
		t.Errorf("expected agent codex, got %q", task.Agent)
	}
	if len(task.PromptTemplates.Builtin) != 1 || task.PromptTemplates.Builtin[0] != "clarify_if_needed" {
		t.Errorf("unexpected templates: %v", task.PromptTemplates.Builtin)
	}
	if string(task.Validation.Mode) != "full" {
		t.Errorf("expected full validation mode, got %q", task.Validation.Mode)
	}
	if len(task.Validation.Commands) != 1 || task.Validation.Commands[0] != "go test ./..." {
		t.Errorf("unexpected validation commands: %v", task.Validation.Commands)
	}
}

func TestUpdateCmd_InvalidMutation(t *testing.T) {
	handler, _ := setupUpdateCmd(t)
	cmd := NewUpdateCmd(handler)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"TASK-001", "--set", "validation.mode"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid mutation error")
	}
}
