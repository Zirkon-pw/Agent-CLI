package template

import (
	"bytes"
	. "github.com/docup/agentctl/internal/cli/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coretemplate "github.com/docup/agentctl/internal/core/template"
	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/docup/agentctl/tests/support/testio"
)

func setupTemplateStore(t *testing.T) *fsstore.TemplateStore {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "templates", "prompts"), 0755)
	return fsstore.NewTemplateStore(dir)
}

func TestTemplateCmd_Structure(t *testing.T) {
	store := setupTemplateStore(t)
	cmd := NewTemplateCmd(store)

	if cmd.Use != "template" {
		t.Errorf("expected use 'template', got %q", cmd.Use)
	}

	subs := cmd.Commands()
	names := make(map[string]bool)
	for _, sub := range subs {
		names[sub.Name()] = true
	}

	for _, expected := range []string{"list", "show", "add"} {
		if !names[expected] {
			t.Errorf("missing subcommand: %s", expected)
		}
	}
}

func TestTemplateListCmd_Builtin(t *testing.T) {
	store := setupTemplateStore(t)
	cmd := NewTemplateCmd(store)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"list", "--builtin"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestTemplateListCmd_TruncatesDescription(t *testing.T) {
	store := setupTemplateStore(t)
	longDesc := "this is a very long description that definitely exceeds the maximum character limit we set"
	if err := store.Save(&coretemplate.PromptTemplate{
		ID:          "custom",
		Name:        "Custom",
		Description: longDesc,
	}); err != nil {
		t.Fatalf("save template: %v", err)
	}

	cmd := NewTemplateCmd(store)
	cmd.SetArgs([]string{"list"})
	output := testio.CaptureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	expected := longDesc[:57] + "..."
	if !strings.Contains(output, expected) {
		t.Fatalf("expected truncated description %q in output %q", expected, output)
	}
	if strings.Contains(output, longDesc) {
		t.Fatalf("expected full description to be truncated in output %q", output)
	}
}
