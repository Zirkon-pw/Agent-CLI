package taskrunner

import (
	"context"
	. "github.com/docup/agentctl/internal/service/taskrunner"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/config/loader"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/events"
	"github.com/docup/agentctl/internal/infra/executor"
	"github.com/docup/agentctl/internal/infra/fsstore"
	infrart "github.com/docup/agentctl/internal/infra/runtime"
	"github.com/docup/agentctl/internal/service/contextpack"
	"github.com/docup/agentctl/internal/service/prompting"
	"github.com/docup/agentctl/internal/service/validationrunner"
)

func setupOrchestrator(t *testing.T) (*Orchestrator, *fsstore.TaskStore, string) {
	t.Helper()
	root := t.TempDir()
	agentctlDir := filepath.Join(root, ".agentctl")
	for _, d := range []string{"tasks", "runs", "runtime", "context", "templates/prompts", "guidelines"} {
		os.MkdirAll(filepath.Join(agentctlDir, d), 0755)
	}

	taskStore := fsstore.NewTaskStore(agentctlDir)
	runStore := fsstore.NewRunStore(agentctlDir)
	registry := infrart.NewRegistry(agentctlDir)
	heartbeatMgr := infrart.NewHeartbeatManager(agentctlDir)
	eventSink := events.NewSink(filepath.Join(agentctlDir, "runtime"))
	ctxBuilder := contextpack.NewBuilder(agentctlDir, root)
	templateStore := fsstore.NewTemplateStore(agentctlDir)
	promptBuilder := prompting.NewBuilder(templateStore, agentctlDir)

	// Use 'echo' as agent for tests
	agentExec := executor.NewAgentExecutor(&loader.AgentsConfig{
		Agents: []loader.AgentDef{
			{ID: "echo", Command: "echo", Args: []string{}},
		},
	})
	validator := validationrunner.NewRunner(root, agentExec, runStore, agentctlDir)
	cfg := loader.DefaultProjectConfig()
	cfg.Execution.DefaultAgent = "echo"

	orch := NewOrchestrator(
		taskStore, runStore, registry, heartbeatMgr, eventSink,
		ctxBuilder, promptBuilder, agentExec, validator,
		cfg,
		agentctlDir, root,
	)
	return orch, taskStore, agentctlDir
}

func createDraftTask(store *fsstore.TaskStore) {
	now := time.Now()
	store.Save(&task.Task{
		ID:     "TASK-001",
		Title:  "Test task",
		Goal:   "Test goal",
		Status: task.StatusDraft,
		Agent:  "echo",
		PromptTemplates: task.PromptTemplates{
			Builtin: []string{"strict_executor"},
		},
		Clarifications: task.Clarifications{Attached: []string{}},
		Runtime:        task.DefaultRuntimeConfig(),
		Validation:     task.ValidationConfig{Mode: task.ValidationModeSimple, Commands: []string{}},
		CreatedAt:      now,
		UpdatedAt:      now,
	})
}

func TestOrchestrator_Run_FullPipeline(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)
	createDraftTask(store)

	err := orch.Run(context.Background(), "TASK-001")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusReview {
		t.Errorf("expected review status, got %s", tk.Status)
	}
}

func TestOrchestrator_Run_WithValidation(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:     "TASK-001",
		Title:  "With validation",
		Goal:   "Test",
		Status: task.StatusDraft,
		Agent:  "echo",
		PromptTemplates: task.PromptTemplates{
			Builtin: []string{"strict_executor"},
		},
		Clarifications: task.Clarifications{Attached: []string{}},
		Runtime:        task.DefaultRuntimeConfig(),
		Validation: task.ValidationConfig{
			Mode:     task.ValidationModeSimple,
			Commands: []string{"true"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	err := orch.Run(context.Background(), "TASK-001")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusReview {
		t.Errorf("expected review, got %s", tk.Status)
	}
}

func TestOrchestrator_Run_ValidationFails(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:     "TASK-001",
		Title:  "Fail validation",
		Goal:   "Test",
		Status: task.StatusDraft,
		Agent:  "echo",
		PromptTemplates: task.PromptTemplates{
			Builtin: []string{"strict_executor"},
		},
		Clarifications: task.Clarifications{Attached: []string{}},
		Runtime:        task.DefaultRuntimeConfig(),
		Validation: task.ValidationConfig{
			Mode:     task.ValidationModeSimple,
			Commands: []string{"false"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	err := orch.Run(context.Background(), "TASK-001")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusFailed {
		t.Errorf("expected failed, got %s", tk.Status)
	}
}

func TestOrchestrator_Run_NormalizesAgentAndTemplate(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:              "TASK-001",
		Title:           "Needs defaults",
		Goal:            "Run with normalized defaults",
		Status:          task.StatusDraft,
		PromptTemplates: task.PromptTemplates{},
		Clarifications:  task.Clarifications{Attached: []string{}},
		Runtime:         task.DefaultRuntimeConfig(),
		Validation:      task.ValidationConfig{Mode: task.ValidationModeSimple},
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	if err := orch.Run(context.Background(), "TASK-001"); err != nil {
		t.Fatalf("run: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Agent != "echo" {
		t.Errorf("expected normalized agent echo, got %s", tk.Agent)
	}
	if len(tk.PromptTemplates.Builtin) != 1 || tk.PromptTemplates.Builtin[0] != "strict_executor" {
		t.Errorf("expected normalized template strict_executor, got %v", tk.PromptTemplates.Builtin)
	}
}

func TestOrchestrator_Run_RequiresTitleAndGoal(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:              "TASK-001",
		Goal:            "Goal only",
		Status:          task.StatusDraft,
		PromptTemplates: task.PromptTemplates{},
		Clarifications:  task.Clarifications{Attached: []string{}},
		Runtime:         task.DefaultRuntimeConfig(),
		Validation:      task.ValidationConfig{Mode: task.ValidationModeSimple},
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	err := orch.Run(context.Background(), "TASK-001")
	if err == nil {
		t.Fatal("expected missing title error")
	}
}

func TestOrchestrator_Run_TaskNotFound(t *testing.T) {
	orch, _, _ := setupOrchestrator(t)
	err := orch.Run(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
}

func TestOrchestrator_Stop(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusRunning,
		Agent:     "echo",
		Runtime:   task.DefaultRuntimeConfig(),
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err := orch.Stop("TASK-001"); err != nil {
		t.Fatalf("stop: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusStopping {
		t.Errorf("expected stopping, got %s", tk.Status)
	}
}

func TestOrchestrator_Kill(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusRunning,
		Agent:     "echo",
		Runtime:   task.DefaultRuntimeConfig(),
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err := orch.Kill("TASK-001"); err != nil {
		t.Fatalf("kill: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusKilled {
		t.Errorf("expected killed, got %s", tk.Status)
	}
}

func TestOrchestrator_Cancel(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err := orch.Cancel("TASK-001"); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusCanceled {
		t.Errorf("expected canceled, got %s", tk.Status)
	}
}

func TestOrchestrator_Cancel_Running_Fails(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err := orch.Cancel("TASK-001"); err == nil {
		t.Fatal("expected error canceling running task")
	}
}

func TestOrchestrator_Accept(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusReview,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err := orch.Accept("TASK-001"); err != nil {
		t.Fatalf("accept: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusCompleted {
		t.Errorf("expected completed, got %s", tk.Status)
	}
}

func TestOrchestrator_Reject(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusReview,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err := orch.Reject("TASK-001", "not good enough"); err != nil {
		t.Fatalf("reject: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusRejected {
		t.Errorf("expected rejected, got %s", tk.Status)
	}
}

func TestOrchestrator_Pause_NotAllowed(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusRunning,
		Runtime:   task.RuntimeConfig{AllowPause: false},
		CreatedAt: now,
		UpdatedAt: now,
	})

	err := orch.Pause("TASK-001")
	if err == nil {
		t.Fatal("expected error when pause not allowed")
	}
}

func TestOrchestrator_Pause_Allowed(t *testing.T) {
	orch, store, _ := setupOrchestrator(t)

	now := time.Now()
	store.Save(&task.Task{
		ID:        "TASK-001",
		Status:    task.StatusRunning,
		Runtime:   task.RuntimeConfig{AllowPause: true},
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err := orch.Pause("TASK-001"); err != nil {
		t.Fatalf("pause: %v", err)
	}

	tk, _ := store.Load("TASK-001")
	if tk.Status != task.StatusPausing {
		t.Errorf("expected pausing, got %s", tk.Status)
	}
}
