package fsstore

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/docup/agentctl/internal/config/loader"
)

const AgentctlDir = ".agentctl"

// Workspace encapsulates the .agentctl directory and its contents.
type Workspace struct {
	Root       string // project root (parent of .agentctl)
	AgentctlDir string // full path to .agentctl
}

// FindWorkspace walks up from the given directory looking for .agentctl.
func FindWorkspace(startDir string) (*Workspace, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, AgentctlDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return &Workspace{
				Root:        dir,
				AgentctlDir: candidate,
			}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, fmt.Errorf(".agentctl directory not found (searched from %s)", startDir)
}

// InitWorkspace creates the .agentctl directory structure with default configs.
func InitWorkspace(projectRoot string) (*Workspace, error) {
	agentctlDir := filepath.Join(projectRoot, AgentctlDir)

	dirs := []string{
		agentctlDir,
		filepath.Join(agentctlDir, "tasks"),
		filepath.Join(agentctlDir, "templates", "tasks"),
		filepath.Join(agentctlDir, "templates", "prompts"),
		filepath.Join(agentctlDir, "guidelines"),
		filepath.Join(agentctlDir, "clarifications"),
		filepath.Join(agentctlDir, "context"),
		filepath.Join(agentctlDir, "runs"),
		filepath.Join(agentctlDir, "runtime"),
		filepath.Join(agentctlDir, "reviews"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Write default config.yaml
	cfg := loader.DefaultProjectConfig()
	if err := writeYAML(filepath.Join(agentctlDir, "config.yaml"), cfg); err != nil {
		return nil, fmt.Errorf("writing config.yaml: %w", err)
	}

	// Write default agents.yaml
	agents := loader.DefaultAgentsConfig()
	if err := writeYAML(filepath.Join(agentctlDir, "agents.yaml"), agents); err != nil {
		return nil, fmt.Errorf("writing agents.yaml: %w", err)
	}

	// Write default routing.yaml
	routing := loader.DefaultRoutingConfig()
	if err := writeYAML(filepath.Join(agentctlDir, "routing.yaml"), routing); err != nil {
		return nil, fmt.Errorf("writing routing.yaml: %w", err)
	}

	return &Workspace{
		Root:        projectRoot,
		AgentctlDir: agentctlDir,
	}, nil
}

func writeYAML(path string, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
