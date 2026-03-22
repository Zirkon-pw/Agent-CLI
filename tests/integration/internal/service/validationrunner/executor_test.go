package validationrunner

import (
	"context"
	. "github.com/docup/agentctl/internal/service/validationrunner"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/config/loader"
	"github.com/docup/agentctl/internal/core/run"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/executor"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

func setupRunner(t *testing.T) (*Runner, string) {
	t.Helper()
	root := t.TempDir()
	agentctlDir := filepath.Join(root, ".agentctl")
	os.MkdirAll(filepath.Join(agentctlDir, "runs"), 0755)

	agentExec := executor.NewAgentExecutor(&loader.AgentsConfig{
		Agents: []loader.AgentDef{
			{ID: "echo", Command: "echo", Args: []string{}},
		},
	})
	runStore := fsstore.NewRunStore(agentctlDir)
	runner := NewRunner(root, agentExec, runStore, agentctlDir)
	return runner, agentctlDir
}

func makeTestRun() *run.Run {
	return &run.Run{
		ID:        "RUN-001",
		TaskID:    "TASK-001",
		Status:    run.RunStatusRunning,
		Agent:     "echo",
		CreatedAt: time.Now(),
	}
}

func TestValidate_AllPass(t *testing.T) {
	runner, _ := setupRunner(t)

	tk := &task.Task{
		ID:    "TASK-001",
		Agent: "echo",
		Validation: task.ValidationConfig{
			Mode:     task.ValidationModeSimple,
			Commands: []string{"true", "echo ok"},
		},
	}

	report, err := runner.Validate(context.Background(), tk, makeTestRun())
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !report.AllPassed {
		t.Error("all commands should pass")
	}
	if len(report.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(report.Results))
	}
}

func TestValidate_SomeFail_SimpleMode(t *testing.T) {
	runner, _ := setupRunner(t)

	tk := &task.Task{
		ID:    "TASK-001",
		Agent: "echo",
		Validation: task.ValidationConfig{
			Mode:     task.ValidationModeSimple,
			Commands: []string{"true", "false"},
		},
	}

	report, err := runner.Validate(context.Background(), tk, makeTestRun())
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if report.AllPassed {
		t.Error("should not all pass")
	}
	if report.TotalRetries != 0 {
		t.Error("simple mode should not retry")
	}
}

func TestValidate_FullMode_WithRetries(t *testing.T) {
	runner, _ := setupRunner(t)

	// Use 'false' as validation — it always fails.
	// Agent is 'echo' so "fix" does nothing, but retry loop runs.
	tk := &task.Task{
		ID:    "TASK-001",
		Agent: "echo",
		Validation: task.ValidationConfig{
			Mode:       task.ValidationModeFull,
			MaxRetries: 2,
			Commands:   []string{"false"},
		},
	}

	report, err := runner.Validate(context.Background(), tk, makeTestRun())
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if report.AllPassed {
		t.Error("should not pass (false always fails)")
	}
	if report.TotalRetries != 2 {
		t.Errorf("expected 2 retries, got %d", report.TotalRetries)
	}
	if len(report.Retries) != 2 {
		t.Errorf("expected 2 retry records, got %d", len(report.Retries))
	}
}

func TestValidate_EmptyCommands(t *testing.T) {
	runner, _ := setupRunner(t)

	tk := &task.Task{
		ID:    "TASK-001",
		Agent: "echo",
		Validation: task.ValidationConfig{
			Mode:     task.ValidationModeSimple,
			Commands: []string{},
		},
	}

	report, err := runner.Validate(context.Background(), tk, makeTestRun())
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !report.AllPassed {
		t.Error("empty commands should pass")
	}
}

func TestValidate_CapturesCommandOutput(t *testing.T) {
	runner, _ := setupRunner(t)
	tk := &task.Task{
		ID:    "TASK-001",
		Agent: "echo",
		Validation: task.ValidationConfig{
			Mode:     task.ValidationModeSimple,
			Commands: []string{"echo hello"},
		},
	}

	report, err := runner.Validate(context.Background(), tk, makeTestRun())
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}

	result := report.Results[0]
	if !result.Passed {
		t.Error("echo should pass")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Fatalf("expected stdout to contain hello, got %q", result.Stdout)
	}
	if result.Duration <= 0 {
		t.Error("duration should be positive")
	}
}

func TestValidate_CapturesFailedCommand(t *testing.T) {
	runner, _ := setupRunner(t)
	tk := &task.Task{
		ID:    "TASK-001",
		Agent: "echo",
		Validation: task.ValidationConfig{
			Mode:     task.ValidationModeSimple,
			Commands: []string{"exit 1"},
		},
	}

	report, err := runner.Validate(context.Background(), tk, makeTestRun())
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}

	result := report.Results[0]
	if result.Passed {
		t.Error("should fail")
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit 1, got %d", result.ExitCode)
	}
}

func TestValidate_FullMode_WritesFixPrompt(t *testing.T) {
	runner, agentctlDir := setupRunner(t)

	tk := &task.Task{
		ID:    "TASK-001",
		Goal:  "Fix stuff",
		Agent: "echo",
		Validation: task.ValidationConfig{
			Mode:       task.ValidationModeFull,
			MaxRetries: 1,
			Commands:   []string{"echo FAIL pkg/foo 1>&2; exit 1"},
		},
	}

	report, err := runner.Validate(context.Background(), tk, makeTestRun())
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(report.Retries) != 1 {
		t.Fatalf("expected 1 retry, got %d", len(report.Retries))
	}

	promptPath := filepath.Join(agentctlDir, "runs", "TASK-001", "RUN-001", "prompt.md")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read fix prompt: %v", err)
	}

	prompt := string(data)
	if !strings.Contains(prompt, "echo FAIL pkg/foo 1>&2; exit 1") {
		t.Fatalf("expected fix prompt to include failed command, got %q", prompt)
	}
	if !strings.Contains(prompt, "FAIL pkg/foo") {
		t.Fatalf("expected fix prompt to include stderr, got %q", prompt)
	}
}
