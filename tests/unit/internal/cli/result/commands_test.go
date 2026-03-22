package result

import (
	. "github.com/docup/agentctl/internal/cli/result"
	"os"
	"path/filepath"
	"testing"

	"github.com/docup/agentctl/internal/infra/fsstore"
)

func TestResultCmd_Structure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "runs"), 0755)
	store := fsstore.NewRunStore(dir)

	cmd := NewResultCmd(store)
	if cmd.Use != "result" {
		t.Errorf("expected use 'result', got %q", cmd.Use)
	}

	subs := cmd.Commands()
	names := make(map[string]bool)
	for _, sub := range subs {
		names[sub.Name()] = true
	}
	for _, expected := range []string{"show", "diff", "list"} {
		if !names[expected] {
			t.Errorf("missing subcommand: %s", expected)
		}
	}
}
