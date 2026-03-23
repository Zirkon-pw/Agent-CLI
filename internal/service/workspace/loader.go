package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docup/agentctl/internal/config/global"
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
// It loads global config first, then merges with project-local config.
func Load(startDir string) (*LoadedWorkspace, error) {
	// Ensure global config directory exists with defaults.
	if _, err := global.EnsureDir(); err != nil {
		return nil, fmt.Errorf("ensuring global config: %w", err)
	}

	// Load global configs.
	globalCfg, err := global.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading global config: %w", err)
	}
	globalAgents, err := global.LoadAgents()
	if err != nil {
		return nil, fmt.Errorf("loading global agents: %w", err)
	}
	globalRouting, err := global.LoadRouting()
	if err != nil {
		return nil, fmt.Errorf("loading global routing: %w", err)
	}

	// Find project workspace.
	ws, err := fsstore.FindWorkspace(startDir)
	if err != nil {
		return nil, err
	}

	// Load project-local config (only project section).
	localCfg, err := loader.LoadProjectLocalConfig(ws.AgentctlDir)
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	// Load project-level agents and routing (may not exist).
	localAgents, _ := loader.LoadAgentsConfig(ws.AgentctlDir)
	localRouting, _ := loader.LoadRoutingConfig(ws.AgentctlDir)

	// Merge: global base + project overrides.
	mergedCfg := loader.MergeConfig(globalCfg, localCfg)
	mergedAgents := loader.MergeAgents(globalAgents, localAgents)
	mergedRouting := loader.MergeRouting(globalRouting, localRouting)

	return &LoadedWorkspace{
		Workspace: ws,
		Config:    mergedCfg,
		Agents:    mergedAgents,
		Routing:   mergedRouting,
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
