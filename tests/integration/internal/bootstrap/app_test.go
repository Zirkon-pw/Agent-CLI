package bootstrap

import (
	. "github.com/docup/agentctl/internal/bootstrap"
	"os"
	"testing"

	"github.com/docup/agentctl/internal/infra/fsstore"
)

func TestNewApp_NoWorkspace(t *testing.T) {
	// Save and restore cwd
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	dir := t.TempDir()
	os.Chdir(dir)

	_, err := NewApp()
	if err == nil {
		t.Fatal("expected error when no .agentctl")
	}
}

func TestNewApp_WithWorkspace(t *testing.T) {
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	dir := t.TempDir()
	fsstore.InitWorkspace(dir)
	os.Chdir(dir)

	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	if app == nil {
		t.Fatal("app should not be nil")
	}
	if app.TaskStore == nil {
		t.Error("TaskStore should be wired")
	}
	if app.RunStore == nil {
		t.Error("RunStore should be wired")
	}
	if app.Orchestrator == nil {
		t.Error("Orchestrator should be wired")
	}
	if app.CreateTask == nil {
		t.Error("CreateTask should be wired")
	}
	if app.ListTasks == nil {
		t.Error("ListTasks should be wired")
	}
	if app.RuntimeMgr == nil {
		t.Error("RuntimeMgr should be wired")
	}
	if app.AgentctlDir == "" {
		t.Error("AgentctlDir should be set")
	}
}
