package clarification

import (
	. "github.com/docup/agentctl/internal/cli/clarification"
	"os"
	"path/filepath"
	"testing"

	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/docup/agentctl/internal/service/clarificationflow"
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
