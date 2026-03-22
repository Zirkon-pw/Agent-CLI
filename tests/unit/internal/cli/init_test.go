package cli

import (
	"bytes"
	. "github.com/docup/agentctl/internal/cli"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd(t *testing.T) {
	cmd := NewInitCmd()
	dir := t.TempDir()
	cmd.SetArgs([]string{"--dir", dir})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	agentctlDir := filepath.Join(dir, ".agentctl")
	if _, err := os.Stat(agentctlDir); err != nil {
		t.Error(".agentctl directory should be created")
	}
	if _, err := os.Stat(filepath.Join(agentctlDir, "config.yaml")); err != nil {
		t.Error("config.yaml should be created")
	}
}

func TestInitCmd_DefaultDir(t *testing.T) {
	cmd := NewInitCmd()
	if cmd.Use != "init" {
		t.Errorf("expected use 'init', got %q", cmd.Use)
	}
}
