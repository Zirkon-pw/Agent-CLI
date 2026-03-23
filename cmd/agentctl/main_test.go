package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNewRootCmd_KeepsTaskCommandWhenWorkspaceUnavailable(t *testing.T) {
	rootCmd := newRootCmd(nil, errors.New("loading agents: parsing agents.yaml: agent \"qwen\" uses legacy field \"command\""))
	rootCmd.SetArgs([]string{"task", "run", "TASK-001"})
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected workspace error")
	}
	if strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected real workspace error, got %v", err)
	}
	if !strings.Contains(err.Error(), "legacy field") {
		t.Fatalf("expected legacy schema error, got %v", err)
	}
}
