package loader

import (
	. "github.com/docup/agentctl/internal/config/loader"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func tmpConfigDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func writeFile(t *testing.T, dir, name string, v interface{}) {
	t.Helper()
	data, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, name), data, 0644)
}

func TestLoadProjectConfig(t *testing.T) {
	dir := tmpConfigDir(t)
	cfg := DefaultProjectConfig()
	writeFile(t, dir, "config.yaml", cfg)

	loaded, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Project.Name != "my-project" {
		t.Errorf("wrong name: %s", loaded.Project.Name)
	}
	if loaded.Execution.DefaultAgent != "claude" {
		t.Errorf("wrong agent: %s", loaded.Execution.DefaultAgent)
	}
	if loaded.Validation.DefaultMaxRetries != 3 {
		t.Errorf("wrong retries: %d", loaded.Validation.DefaultMaxRetries)
	}
}

func TestLoadProjectConfig_NotFound(t *testing.T) {
	_, err := LoadProjectConfig("/nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadProjectConfig_InvalidYAML(t *testing.T) {
	dir := tmpConfigDir(t)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("invalid: [yaml: broken"), 0644)
	_, err := LoadProjectConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadAgentsConfig(t *testing.T) {
	dir := tmpConfigDir(t)
	agents := DefaultAgentsConfig()
	writeFile(t, dir, "agents.yaml", agents)

	loaded, err := LoadAgentsConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(loaded.Agents))
	}
	if loaded.Agents[0].ID != "claude" {
		t.Errorf("first agent should be claude, got %s", loaded.Agents[0].ID)
	}
	if loaded.Agents[0].Command != "claude" {
		t.Errorf("claude command should be 'claude', got %s", loaded.Agents[0].Command)
	}
}

func TestLoadAgentsConfig_NotFound(t *testing.T) {
	_, err := LoadAgentsConfig("/nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRoutingConfig(t *testing.T) {
	dir := tmpConfigDir(t)
	routing := DefaultRoutingConfig()
	writeFile(t, dir, "routing.yaml", routing)

	loaded, err := LoadRoutingConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Routing) != 3 {
		t.Errorf("expected 3 rules, got %d", len(loaded.Routing))
	}
}

func TestDefaultProjectConfig(t *testing.T) {
	cfg := DefaultProjectConfig()

	if cfg.Project.Language != "go" {
		t.Errorf("expected go, got %s", cfg.Project.Language)
	}
	if cfg.Execution.Mode != "strict" {
		t.Errorf("expected strict, got %s", cfg.Execution.Mode)
	}
	if len(cfg.Prompting.BuiltinTemplates) != 5 {
		t.Errorf("expected 5 builtin templates, got %d", len(cfg.Prompting.BuiltinTemplates))
	}
	if cfg.Runtime.MaxParallelTasks != 4 {
		t.Errorf("expected 4 parallel tasks, got %d", cfg.Runtime.MaxParallelTasks)
	}
	if cfg.Validation.DefaultMode != "simple" {
		t.Errorf("expected simple, got %s", cfg.Validation.DefaultMode)
	}
}

func TestDefaultAgentsConfig(t *testing.T) {
	cfg := DefaultAgentsConfig()
	if len(cfg.Agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(cfg.Agents))
	}

	ids := map[string]bool{}
	for _, a := range cfg.Agents {
		ids[a.ID] = true
		if a.Command == "" {
			t.Errorf("agent %s has empty command", a.ID)
		}
	}
	for _, expected := range []string{"claude", "codex", "qwen"} {
		if !ids[expected] {
			t.Errorf("missing agent: %s", expected)
		}
	}
}

func TestDefaultRoutingConfig(t *testing.T) {
	cfg := DefaultRoutingConfig()
	if len(cfg.Routing) != 3 {
		t.Errorf("expected 3 rules, got %d", len(cfg.Routing))
	}
	for _, r := range cfg.Routing {
		if r.When == "" || r.Agent == "" {
			t.Error("routing rule has empty fields")
		}
	}
}
