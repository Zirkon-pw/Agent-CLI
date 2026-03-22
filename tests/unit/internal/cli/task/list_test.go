package task

import (
	"bytes"
	. "github.com/docup/agentctl/internal/cli/task"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docup/agentctl/internal/app/query"
	coretask "github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/docup/agentctl/tests/support/testio"
)

func TestListCmd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "tasks"), 0755)
	store := fsstore.NewTaskStore(dir)
	handler := query.NewListTasks(store)

	cmd := NewListCmd(handler)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestListCmd_HasStatusFlag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "tasks"), 0755)
	store := fsstore.NewTaskStore(dir)
	handler := query.NewListTasks(store)

	cmd := NewListCmd(handler)
	if cmd.Flags().Lookup("status") == nil {
		t.Error("--status flag should exist")
	}
}

func TestListCmd_TruncatesLongTitle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	store := fsstore.NewTaskStore(dir)
	longTitle := strings.Repeat("x", 45)
	now := time.Now()
	if err := store.Save(&coretask.Task{
		ID:        "TASK-001",
		Title:     longTitle,
		Status:    coretask.StatusDraft,
		Agent:     "claude",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	cmd := NewListCmd(query.NewListTasks(store))
	output := testio.CaptureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	expected := longTitle[:37] + "..."
	if !strings.Contains(output, expected) {
		t.Fatalf("expected truncated title %q in output %q", expected, output)
	}
	if strings.Contains(output, longTitle) {
		t.Fatalf("expected full title to be truncated in output %q", output)
	}
}
