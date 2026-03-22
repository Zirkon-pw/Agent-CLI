package root

import (
	. "github.com/docup/agentctl/internal/cli/root"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	if cmd == nil {
		t.Fatal("root cmd should not be nil")
	}
	if cmd.Use != "agentctl" {
		t.Errorf("expected use 'agentctl', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}
