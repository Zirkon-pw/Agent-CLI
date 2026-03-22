package clarification

import (
	. "github.com/docup/agentctl/internal/cli/clarification"
	"github.com/docup/agentctl/internal/core/clarification"
	"github.com/docup/agentctl/internal/core/task"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/docup/agentctl/internal/service/clarificationflow"
	"github.com/docup/agentctl/tests/support/testio"
	"gopkg.in/yaml.v3"
)

func TestClarificationCmd_Structure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "tasks"), 0755)
	os.MkdirAll(filepath.Join(dir, "clarifications"), 0755)

	taskStore := fsstore.NewTaskStore(dir)
	clarStore := fsstore.NewClarificationStore(dir)
	mgr := clarificationflow.NewManager(taskStore, clarStore)

	cmd := NewClarificationCmd(mgr)

	if cmd.Use != "clarification" {
		t.Errorf("expected use 'clarification', got %q", cmd.Use)
	}

	subs := cmd.Commands()
	names := make(map[string]bool)
	for _, sub := range subs {
		names[sub.Name()] = true
	}

	for _, expected := range []string{"generate", "show", "attach"} {
		if !names[expected] {
			t.Errorf("missing subcommand: %s", expected)
		}
	}
}

func TestClarificationAttach_PrintsTaskRunInstruction(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "clarifications", "TASK-001"), 0755); err != nil {
		t.Fatalf("mkdir clarifications: %v", err)
	}

	taskStore := fsstore.NewTaskStore(dir)
	clarStore := fsstore.NewClarificationStore(dir)
	mgr := clarificationflow.NewManager(taskStore, clarStore)

	now := time.Now()
	reqID := "CLAR-REQ-001"
	if err := taskStore.Save(&task.Task{
		ID:     "TASK-001",
		Title:  "Test",
		Goal:   "Test",
		Status: task.StatusNeedsClarification,
		Clarifications: task.Clarifications{
			PendingRequest: &reqID,
			Attached:       []string{},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save task: %v", err)
	}

	clarPath := filepath.Join(dir, "clarifications", "TASK-001", "clarification_CLAR-001.yml")
	data, err := yaml.Marshal(&clarification.Clarification{
		TaskID:          "TASK-001",
		RequestID:       reqID,
		ClarificationID: "CLAR-001",
		CreatedAt:       now,
	})
	if err != nil {
		t.Fatalf("marshal clarification: %v", err)
	}
	if err := os.WriteFile(clarPath, data, 0644); err != nil {
		t.Fatalf("write clarification: %v", err)
	}

	cmd := NewClarificationCmd(mgr)
	cmd.SetArgs([]string{"attach", "TASK-001", clarPath})

	output := testio.CaptureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if !strings.Contains(output, "agentctl task run TASK-001") {
		t.Fatalf("expected attach output to mention task run, got %q", output)
	}
	if strings.Contains(output, "agentctl task resume TASK-001") {
		t.Fatalf("did not expect attach output to mention task resume, got %q", output)
	}
}
