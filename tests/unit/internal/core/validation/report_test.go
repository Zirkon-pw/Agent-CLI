package validation

import (
	. "github.com/docup/agentctl/internal/core/validation"
	"testing"
)

func TestHasFailures(t *testing.T) {
	r := &Report{AllPassed: true}
	if r.HasFailures() {
		t.Error("should not have failures")
	}

	r.AllPassed = false
	if !r.HasFailures() {
		t.Error("should have failures")
	}
}

func TestFailedCommands(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Command: "go build", Passed: true},
			{Command: "go test", Passed: false, ExitCode: 1},
			{Command: "go vet", Passed: false, ExitCode: 2},
		},
	}

	failed := r.FailedCommands()
	if len(failed) != 2 {
		t.Fatalf("expected 2 failed, got %d", len(failed))
	}
	if failed[0].Command != "go test" {
		t.Errorf("expected 'go test', got %q", failed[0].Command)
	}
	if failed[1].Command != "go vet" {
		t.Errorf("expected 'go vet', got %q", failed[1].Command)
	}
}

func TestFailedCommands_AllPassed(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Command: "go build", Passed: true},
		},
	}
	if len(r.FailedCommands()) != 0 {
		t.Error("should be empty when all passed")
	}
}

func TestCanRetry(t *testing.T) {
	r := &Report{TotalRetries: 1, MaxRetries: 3}
	if !r.CanRetry() {
		t.Error("should be able to retry (1/3)")
	}

	r.TotalRetries = 3
	if r.CanRetry() {
		t.Error("should NOT be able to retry (3/3)")
	}

	r.TotalRetries = 0
	r.MaxRetries = 0
	if r.CanRetry() {
		t.Error("should NOT be able to retry (0/0)")
	}
}
