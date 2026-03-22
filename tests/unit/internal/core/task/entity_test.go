package task

import (
	. "github.com/docup/agentctl/internal/core/task"
	"testing"
	"time"
)

func newTestTask() *Task {
	return &Task{
		ID:     "TASK-001",
		Title:  "Test task",
		Goal:   "Test goal",
		Status: StatusDraft,
		Agent:  "claude",
		PromptTemplates: PromptTemplates{
			Builtin: []string{"strict_executor"},
			Custom:  []string{},
		},
		Clarifications: Clarifications{Attached: []string{}},
		Runtime:        DefaultRuntimeConfig(),
		Validation:     DefaultValidationConfig(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func TestTransitionTo_Valid(t *testing.T) {
	task := newTestTask()
	before := task.UpdatedAt

	time.Sleep(time.Millisecond)
	if err := task.TransitionTo(StatusQueued); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != StatusQueued {
		t.Errorf("expected status queued, got %s", task.Status)
	}
	if !task.UpdatedAt.After(before) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestTransitionTo_Invalid(t *testing.T) {
	task := newTestTask()
	if err := task.TransitionTo(StatusRunning); err == nil {
		t.Fatal("expected error for invalid transition draft → running")
	}
	if task.Status != StatusDraft {
		t.Error("status should not change on invalid transition")
	}
}

func TestHasTemplate(t *testing.T) {
	task := newTestTask()
	if !task.HasTemplate("strict_executor") {
		t.Error("expected to find strict_executor")
	}
	if task.HasTemplate("review_only") {
		t.Error("should not find review_only")
	}
}

func TestHasClarifyTemplate(t *testing.T) {
	task := newTestTask()
	if task.HasClarifyTemplate() {
		t.Error("should not have clarify template")
	}

	task.PromptTemplates.Builtin = append(task.PromptTemplates.Builtin, "clarify_if_needed")
	if !task.HasClarifyTemplate() {
		t.Error("should have clarify template")
	}
}

func TestAddClarification(t *testing.T) {
	task := newTestTask()
	reqID := "CLAR-REQ-001"
	task.SetPendingClarification(reqID)

	if task.Clarifications.PendingRequest == nil || *task.Clarifications.PendingRequest != reqID {
		t.Fatal("pending request not set")
	}

	task.AddClarification("/path/to/clar.yml")

	if len(task.Clarifications.Attached) != 1 {
		t.Fatal("expected 1 attached clarification")
	}
	if task.Clarifications.Attached[0] != "/path/to/clar.yml" {
		t.Error("wrong clarification path")
	}
	if task.Clarifications.PendingRequest != nil {
		t.Error("pending request should be cleared after attach")
	}
}

func TestSetPendingClarification(t *testing.T) {
	task := newTestTask()
	before := task.UpdatedAt
	time.Sleep(time.Millisecond)

	task.SetPendingClarification("REQ-001")

	if task.Clarifications.PendingRequest == nil {
		t.Fatal("pending request should be set")
	}
	if *task.Clarifications.PendingRequest != "REQ-001" {
		t.Error("wrong request ID")
	}
	if !task.UpdatedAt.After(before) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestDefaultRuntimeConfig(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	if cfg.MaxExecutionMinutes != 45 {
		t.Errorf("expected 45 minutes, got %d", cfg.MaxExecutionMinutes)
	}
	if cfg.HeartbeatIntervalSec != 5 {
		t.Errorf("expected 5s heartbeat, got %d", cfg.HeartbeatIntervalSec)
	}
	if cfg.GracefulStopTimeoutSec != 20 {
		t.Errorf("expected 20s timeout, got %d", cfg.GracefulStopTimeoutSec)
	}
	if !cfg.AllowPause {
		t.Error("expected AllowPause true")
	}
}

func TestDefaultValidationConfig(t *testing.T) {
	cfg := DefaultValidationConfig()
	if cfg.Mode != ValidationModeSimple {
		t.Errorf("expected simple mode, got %s", cfg.Mode)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected 3 retries, got %d", cfg.MaxRetries)
	}
	if len(cfg.Commands) != 0 {
		t.Error("expected empty commands")
	}
}
