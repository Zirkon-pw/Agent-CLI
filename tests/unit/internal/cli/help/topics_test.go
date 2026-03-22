package help

import (
	"bytes"
	. "github.com/docup/agentctl/internal/cli/help"
	"testing"
)

func TestNewHelpCmd(t *testing.T) {
	cmd := NewHelpCmd()
	if cmd == nil {
		t.Fatal("help cmd should not be nil")
	}
	if cmd.Use != "topics [topic]" {
		t.Errorf("expected use 'topics [topic]', got %q", cmd.Use)
	}
}

func TestHelpCmd_UnknownTopic(t *testing.T) {
	cmd := NewHelpCmd()
	cmd.SetArgs([]string{"nonexistent"})
	cmd.SetOut(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown topic")
	}
}

func TestHelpCmd_KnownTopics(t *testing.T) {
	for _, topic := range []string{"task", "template", "clarification", "validation", "workflow"} {
		cmd := NewHelpCmd()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{topic})
		if err := cmd.Execute(); err != nil {
			t.Errorf("topic %q: unexpected error: %v", topic, err)
		}
	}
}
