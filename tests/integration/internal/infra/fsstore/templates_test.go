package fsstore

import (
	. "github.com/docup/agentctl/internal/infra/fsstore"
	"os"
	"path/filepath"
	"testing"

	"github.com/docup/agentctl/internal/core/template"
)

func TestTemplateStore_SaveAndLoad(t *testing.T) {
	dir := tmpAgentctlDir(t)
	os.MkdirAll(filepath.Join(dir, "templates", "prompts"), 0755)
	store := NewTemplateStore(dir)

	tmpl := &template.PromptTemplate{
		ID:          "custom_test",
		Name:        "Custom Test",
		Description: "A test template",
		IsBuiltin:   false,
		Behavior: template.Behavior{
			RequireExplicitScope: true,
			CodeChangesAllowed:   true,
		},
	}

	if err := store.Save(tmpl); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load("custom_test")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ID != "custom_test" {
		t.Errorf("wrong ID: %s", loaded.ID)
	}
	if loaded.Name != "Custom Test" {
		t.Errorf("wrong name: %s", loaded.Name)
	}
	if !loaded.Behavior.RequireExplicitScope {
		t.Error("RequireExplicitScope should be true")
	}
}

func TestTemplateStore_Load_NotFound(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTemplateStore(dir)

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTemplateStore_List(t *testing.T) {
	dir := tmpAgentctlDir(t)
	os.MkdirAll(filepath.Join(dir, "templates", "prompts"), 0755)
	store := NewTemplateStore(dir)

	store.Save(&template.PromptTemplate{ID: "tmpl_a", Name: "A"})
	store.Save(&template.PromptTemplate{ID: "tmpl_b", Name: "B"})

	templates, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}
}

func TestTemplateStore_List_EmptyDir(t *testing.T) {
	dir := tmpAgentctlDir(t)
	store := NewTemplateStore(dir)

	templates, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if templates != nil {
		t.Error("expected nil for empty dir")
	}
}
