package guidelines

import (
	"bytes"
	. "github.com/docup/agentctl/internal/cli/guidelines"
	"os"
	"path/filepath"
	"testing"
)

func TestGuidelinesCmd_Structure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "guidelines"), 0755)

	cmd := NewGuidelinesCmd(dir)
	if cmd.Use != "guidelines" {
		t.Errorf("expected use 'guidelines', got %q", cmd.Use)
	}

	subs := cmd.Commands()
	names := make(map[string]bool)
	for _, sub := range subs {
		names[sub.Name()] = true
	}
	for _, expected := range []string{"add", "list", "show"} {
		if !names[expected] {
			t.Errorf("missing subcommand: %s", expected)
		}
	}
}

func TestGuidelinesListCmd_Empty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "guidelines"), 0755)

	cmd := NewGuidelinesCmd(dir)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestGuidelinesAddAndShow(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "guidelines"), 0755)

	// Create a file to add
	src := filepath.Join(t.TempDir(), "test-guideline.md")
	os.WriteFile(src, []byte("# Test Guideline\nContent here."), 0644)

	cmd := NewGuidelinesCmd(dir)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"add", src})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("add: %v", err)
	}

	// Show it
	cmd2 := NewGuidelinesCmd(dir)
	var buf2 bytes.Buffer
	cmd2.SetOut(&buf2)
	cmd2.SetErr(&buf2)
	cmd2.SetArgs([]string{"show", "test-guideline"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("show: %v", err)
	}
}
