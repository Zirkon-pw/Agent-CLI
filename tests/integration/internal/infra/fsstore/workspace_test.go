package fsstore

import (
	. "github.com/docup/agentctl/internal/infra/fsstore"
	"os"
	"path/filepath"
	"testing"
)

func TestInitWorkspace(t *testing.T) {
	dir := t.TempDir()
	ws, err := InitWorkspace(dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	if ws.Root != dir {
		t.Errorf("wrong root: %s", ws.Root)
	}
	if ws.AgentctlDir != filepath.Join(dir, ".agentctl") {
		t.Errorf("wrong agentctl dir: %s", ws.AgentctlDir)
	}

	// Check directories exist
	expectedDirs := []string{
		"tasks", "templates/tasks", "templates/prompts",
		"guidelines", "clarifications", "context",
		"runs", "runtime", "reviews",
	}
	for _, d := range expectedDirs {
		path := filepath.Join(ws.AgentctlDir, d)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("directory %s should exist", d)
		}
	}

	// Check config files exist
	for _, f := range []string{"config.yaml", "agents.yaml", "routing.yaml"} {
		path := filepath.Join(ws.AgentctlDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("file %s should exist", f)
		}
	}
}

func TestFindWorkspace(t *testing.T) {
	dir := t.TempDir()
	InitWorkspace(dir)

	// Find from root
	ws, err := FindWorkspace(dir)
	if err != nil {
		t.Fatalf("find from root: %v", err)
	}
	if ws.Root != dir {
		t.Errorf("wrong root: %s", ws.Root)
	}

	// Find from nested directory
	nested := filepath.Join(dir, "src", "pkg")
	os.MkdirAll(nested, 0755)
	ws, err = FindWorkspace(nested)
	if err != nil {
		t.Fatalf("find from nested: %v", err)
	}
	if ws.Root != dir {
		t.Errorf("wrong root from nested: %s", ws.Root)
	}
}

func TestFindWorkspace_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindWorkspace(dir)
	if err == nil {
		t.Fatal("expected error when .agentctl not found")
	}
}
