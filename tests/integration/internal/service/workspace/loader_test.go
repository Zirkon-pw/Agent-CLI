package workspace

import (
	. "github.com/docup/agentctl/internal/service/workspace"
	"os"
	"path/filepath"
	"testing"

	"github.com/docup/agentctl/internal/infra/fsstore"
)

func TestLoad_Success(t *testing.T) {
	dir := t.TempDir()
	fsstore.InitWorkspace(dir)

	ws, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if ws.Config == nil {
		t.Fatal("config should not be nil")
	}
	if ws.Agents == nil {
		t.Fatal("agents should not be nil")
	}
	if ws.Routing == nil {
		t.Fatal("routing should not be nil")
	}
	if ws.Config.Project.Name != "my-project" {
		t.Errorf("wrong project name: %s", ws.Config.Project.Name)
	}
	if len(ws.Agents.Agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(ws.Agents.Agents))
	}
}

func TestLoad_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error when no workspace")
	}
}

func TestLoad_FromNestedDir(t *testing.T) {
	dir := t.TempDir()
	fsstore.InitWorkspace(dir)

	nested := filepath.Join(dir, "src", "pkg", "deep")
	os.MkdirAll(nested, 0755)

	ws, err := Load(nested)
	if err != nil {
		t.Fatalf("load from nested: %v", err)
	}
	if ws.Workspace.Root != dir {
		t.Errorf("wrong root: %s", ws.Workspace.Root)
	}
}

func TestLoadGuideline_Found(t *testing.T) {
	dir := t.TempDir()
	fsstore.InitWorkspace(dir)

	agentctlDir := filepath.Join(dir, ".agentctl")
	os.WriteFile(filepath.Join(agentctlDir, "guidelines", "backend.md"), []byte("# Backend Rules"), 0644)

	content, err := LoadGuideline(agentctlDir, "backend")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if content != "# Backend Rules" {
		t.Errorf("wrong content: %s", content)
	}
}

func TestLoadGuideline_WithExtension(t *testing.T) {
	dir := t.TempDir()
	fsstore.InitWorkspace(dir)

	agentctlDir := filepath.Join(dir, ".agentctl")
	os.WriteFile(filepath.Join(agentctlDir, "guidelines", "testing-policy.md"), []byte("# Testing"), 0644)

	content, err := LoadGuideline(agentctlDir, "testing-policy")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if content != "# Testing" {
		t.Errorf("wrong content: %s", content)
	}
}

func TestLoadGuideline_NotFound(t *testing.T) {
	dir := t.TempDir()
	fsstore.InitWorkspace(dir)

	_, err := LoadGuideline(filepath.Join(dir, ".agentctl"), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing guideline")
	}
}
