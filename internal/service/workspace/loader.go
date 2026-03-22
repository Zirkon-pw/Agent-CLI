package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docup/agentctl/internal/config/loader"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// LoadedWorkspace holds all loaded workspace resources.
type LoadedWorkspace struct {
	Workspace *fsstore.Workspace
	Config    *loader.ProjectConfig
	Agents    *loader.AgentsConfig
	Routing   *loader.RoutingConfig
}

// Load finds and loads the workspace from the given directory.
func Load(startDir string) (*LoadedWorkspace, error) {
	ws, err := fsstore.FindWorkspace(startDir)
	if err != nil {
		return nil, err
	}

	cfg, err := loader.LoadProjectConfig(ws.AgentctlDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	agents, err := loader.LoadAgentsConfig(ws.AgentctlDir)
	if err != nil {
		return nil, fmt.Errorf("loading agents: %w", err)
	}

	routing, err := loader.LoadRoutingConfig(ws.AgentctlDir)
	if err != nil {
		return nil, fmt.Errorf("loading routing: %w", err)
	}

	return &LoadedWorkspace{
		Workspace: ws,
		Config:    cfg,
		Agents:    agents,
		Routing:   routing,
	}, nil
}

// LoadGuideline reads a guideline file by name.
func LoadGuideline(agentctlDir, name string) (string, error) {
	path := filepath.Join(agentctlDir, "guidelines", name+".md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Try without .md extension
		path = filepath.Join(agentctlDir, "guidelines", name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return "", fmt.Errorf("guideline %q not found", name)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
