package run

import (
	. "github.com/docup/agentctl/internal/core/run"
	"testing"
	"time"
)

func newTestRun() *Run {
	return &Run{
		ID:        "RUN-001",
		TaskID:    "TASK-001",
		Status:    RunStatusPending,
		Agent:     "claude",
		CreatedAt: time.Now(),
	}
}

func TestRunTransitionTo_Valid(t *testing.T) {
	r := newTestRun()
	if err := r.TransitionTo(RunStatusRunning); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != RunStatusRunning {
		t.Errorf("expected running, got %s", r.Status)
	}
	if r.StartedAt == nil {
		t.Error("StartedAt should be set on transition to running")
	}
}

func TestRunTransitionTo_Invalid(t *testing.T) {
	r := newTestRun()
	if err := r.TransitionTo(RunStatusSuccess); err == nil {
		t.Fatal("expected error for pending → success")
	}
}

func TestRunTransitionTo_Terminal_SetsFinished(t *testing.T) {
	r := newTestRun()
	r.TransitionTo(RunStatusRunning)
	if err := r.TransitionTo(RunStatusSuccess); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.FinishedAt == nil {
		t.Error("FinishedAt should be set on terminal transition")
	}
}

func TestMarkStarted(t *testing.T) {
	r := newTestRun()
	if err := r.MarkStarted(12345); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", r.PID)
	}
	if r.Status != RunStatusRunning {
		t.Errorf("expected running, got %s", r.Status)
	}
}

func TestMarkFinished_Success(t *testing.T) {
	r := newTestRun()
	r.MarkStarted(100)

	if err := r.MarkFinished(0, "completed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != RunStatusSuccess {
		t.Errorf("expected success, got %s", r.Status)
	}
	if r.ExitCode == nil || *r.ExitCode != 0 {
		t.Error("exit code should be 0")
	}
	if r.ExitReason != "completed" {
		t.Errorf("expected reason 'completed', got %q", r.ExitReason)
	}
}

func TestMarkFinished_Failure(t *testing.T) {
	r := newTestRun()
	r.MarkStarted(100)

	if err := r.MarkFinished(1, "error"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != RunStatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
	if r.ExitCode == nil || *r.ExitCode != 1 {
		t.Error("exit code should be 1")
	}
}

func TestDuration_NotStarted(t *testing.T) {
	r := newTestRun()
	if r.Duration() != 0 {
		t.Error("expected zero duration for unstarted run")
	}
}

func TestDuration_Running(t *testing.T) {
	r := newTestRun()
	r.MarkStarted(100)
	time.Sleep(10 * time.Millisecond)
	d := r.Duration()
	if d < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", d)
	}
}

func TestDuration_Finished(t *testing.T) {
	r := newTestRun()
	r.MarkStarted(100)
	time.Sleep(10 * time.Millisecond)
	r.MarkFinished(0, "done")

	d := r.Duration()
	// Should be fixed (not grow over time)
	time.Sleep(10 * time.Millisecond)
	d2 := r.Duration()
	if d != d2 {
		t.Error("finished run duration should be fixed")
	}
}

func TestRunStatusIsTerminal(t *testing.T) {
	terminals := []RunStatus{RunStatusSuccess, RunStatusFailed, RunStatusStopped, RunStatusKilled}
	for _, s := range terminals {
		if !s.IsTerminal() {
			t.Errorf("expected %s to be terminal", s)
		}
	}
	nonTerminals := []RunStatus{RunStatusPending, RunStatusRunning, RunStatusRetrying}
	for _, s := range nonTerminals {
		if s.IsTerminal() {
			t.Errorf("expected %s to NOT be terminal", s)
		}
	}
}

func TestRunRetryTransition(t *testing.T) {
	r := newTestRun()
	r.TransitionTo(RunStatusRunning)
	if err := r.TransitionTo(RunStatusRetrying); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := r.TransitionTo(RunStatusRunning); err != nil {
		t.Fatalf("unexpected error retrying → running: %v", err)
	}
}
