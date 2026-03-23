package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig is the root configuration loaded from .agentctl/config.yaml.
type ProjectConfig struct {
	Project        ProjectInfo      `yaml:"project"`
	Execution      ExecutionConfig  `yaml:"execution"`
	Prompting      PromptingConfig  `yaml:"prompting"`
	Clarifications ClarificationCfg `yaml:"clarifications"`
	Runtime        RuntimeCfg       `yaml:"runtime"`
	Validation     ValidationCfg    `yaml:"validation"`
	Artifacts      ArtifactsCfg     `yaml:"artifacts"`
}

type ProjectInfo struct {
	Name     string `yaml:"name"`
	Language string `yaml:"language"`
}

type ExecutionConfig struct {
	DefaultAgent string `yaml:"default_agent"`
	Mode         string `yaml:"mode"`
}

type PromptingConfig struct {
	BuiltinTemplates       []string `yaml:"builtin_templates"`
	DefaultTemplate        string   `yaml:"default_template"`
	AllowMultipleTemplates bool     `yaml:"allow_multiple_templates"`
}

type ClarificationCfg struct {
	Dir                string `yaml:"dir"`
	Strategy           string `yaml:"strategy"`
	AllowMultipleFiles bool   `yaml:"allow_multiple_files"`
}

type RuntimeCfg struct {
	MaxParallelTasks       int  `yaml:"max_parallel_tasks"`
	HeartbeatIntervalSec   int  `yaml:"heartbeat_interval_sec"`
	StaleAfterSec          int  `yaml:"stale_after_sec"`
	GracefulStopTimeoutSec int  `yaml:"graceful_stop_timeout_sec"`
	AllowForceKill         bool `yaml:"allow_force_kill"`
}

type ValidationCfg struct {
	DefaultMode       string   `yaml:"default_mode"`
	DefaultMaxRetries int      `yaml:"default_max_retries"`
	DefaultCommands   []string `yaml:"default_commands"`
}

type ArtifactsCfg struct {
	RunsDir    string `yaml:"runs_dir"`
	ContextDir string `yaml:"context_dir"`
	ReviewsDir string `yaml:"reviews_dir"`
}

type AgentDriver string

const (
	AgentDriverClaude AgentDriver = "claude"
	AgentDriverCodex  AgentDriver = "codex"
	AgentDriverQwen   AgentDriver = "qwen"
)

// AgentDef describes an available agent from agents.yaml.
type AgentDef struct {
	ID      string            `yaml:"id"`
	Driver  AgentDriver       `yaml:"driver"`
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	Enabled *bool             `yaml:"enabled,omitempty"`
}

func (a AgentDef) IsEnabled() bool {
	return a.Enabled == nil || *a.Enabled
}

// AgentsConfig wraps the list of agents.
type AgentsConfig struct {
	Agents []AgentDef `yaml:"agents"`
}

// RoutingRule defines an agent routing rule.
type RoutingRule struct {
	When  string `yaml:"when"`
	Agent string `yaml:"agent"`
}

// RoutingConfig wraps routing rules.
type RoutingConfig struct {
	Routing []RoutingRule `yaml:"routing"`
}

// LoadProjectConfig reads config.yaml from the .agentctl directory.
func LoadProjectConfig(agentctlDir string) (*ProjectConfig, error) {
	path := filepath.Join(agentctlDir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config.yaml: %w", err)
	}
	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config.yaml: %w", err)
	}
	return &cfg, nil
}

// LoadAgentsConfig reads agents.yaml.
func LoadAgentsConfig(agentctlDir string) (*AgentsConfig, error) {
	path := filepath.Join(agentctlDir, "agents.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading agents.yaml: %w", err)
	}
	if err := validateAgentsConfigSchema(data); err != nil {
		return nil, fmt.Errorf("parsing agents.yaml: %w", err)
	}

	var cfg AgentsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing agents.yaml: %w", err)
	}
	if err := validateAgentsConfig(cfg); err != nil {
		return nil, fmt.Errorf("parsing agents.yaml: %w", err)
	}
	return &cfg, nil
}

// LoadRoutingConfig reads routing.yaml.
func LoadRoutingConfig(agentctlDir string) (*RoutingConfig, error) {
	path := filepath.Join(agentctlDir, "routing.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading routing.yaml: %w", err)
	}
	var cfg RoutingConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing routing.yaml: %w", err)
	}
	return &cfg, nil
}

// DefaultProjectConfig returns a config with sensible defaults.
func DefaultProjectConfig() *ProjectConfig {
	return &ProjectConfig{
		Project: ProjectInfo{
			Name:     "my-project",
			Language: "go",
		},
		Execution: ExecutionConfig{
			DefaultAgent: "claude",
			Mode:         "strict",
		},
		Prompting: PromptingConfig{
			BuiltinTemplates: []string{
				"clarify_if_needed",
				"plan_before_execution",
				"strict_executor",
				"research_only",
				"review_only",
			},
			DefaultTemplate:        "strict_executor",
			AllowMultipleTemplates: true,
		},
		Clarifications: ClarificationCfg{
			Dir:                ".agentctl/clarifications",
			Strategy:           "by_yml_files",
			AllowMultipleFiles: true,
		},
		Runtime: RuntimeCfg{
			MaxParallelTasks:       4,
			HeartbeatIntervalSec:   5,
			StaleAfterSec:          30,
			GracefulStopTimeoutSec: 20,
			AllowForceKill:         true,
		},
		Validation: ValidationCfg{
			DefaultMode:       "simple",
			DefaultMaxRetries: 3,
			DefaultCommands:   []string{},
		},
		Artifacts: ArtifactsCfg{
			RunsDir:    ".agentctl/runs",
			ContextDir: ".agentctl/context",
			ReviewsDir: ".agentctl/reviews",
		},
	}
}

// DefaultAgentsConfig returns default agent definitions.
func DefaultAgentsConfig() *AgentsConfig {
	return &AgentsConfig{
		Agents: []AgentDef{
			{
				ID:      "claude",
				Driver:  AgentDriverClaude,
				Command: "claude",
				Args:    []string{},
				Enabled: boolPtr(true),
			},
			{
				ID:      "codex",
				Driver:  AgentDriverCodex,
				Command: "codex",
				Args:    []string{},
				Enabled: boolPtr(true),
			},
			{
				ID:      "qwen",
				Driver:  AgentDriverQwen,
				Command: "qwen",
				Args:    []string{},
				Enabled: boolPtr(true),
			},
		},
	}
}

func validateAgentsConfigSchema(data []byte) error {
	var raw struct {
		Agents []map[string]any `yaml:"agents"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}

	legacyFields := []string{
		"role",
		"metadata",
		"runtime",
		"transport",
		"adapter_command",
		"adapter_args",
		"capabilities",
		"child_cli_command",
		"child_cli_args",
	}
	for _, agent := range raw.Agents {
		id, _ := agent["id"].(string)
		for _, field := range legacyFields {
			if _, ok := agent[field]; ok {
				return fmt.Errorf("agent %q uses legacy field %q; migrate to metadata/runtime schema", id, field)
			}
		}
	}
	return nil
}

func validateAgentsConfig(cfg AgentsConfig) error {
	seen := make(map[string]struct{}, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent id is required")
		}
		if _, ok := seen[agent.ID]; ok {
			return fmt.Errorf("duplicate agent id %q", agent.ID)
		}
		seen[agent.ID] = struct{}{}
		if agent.Driver == "" {
			return fmt.Errorf("agent %q is missing driver", agent.ID)
		}
		if agent.Command == "" {
			return fmt.Errorf("agent %q is missing command", agent.ID)
		}
		switch agent.Driver {
		case AgentDriverClaude, AgentDriverCodex, AgentDriverQwen:
		default:
			return fmt.Errorf("agent %q uses unsupported driver %q", agent.ID, agent.Driver)
		}
	}
	return nil
}

func boolPtr(v bool) *bool {
	return &v
}

// DefaultRoutingConfig returns default routing rules.
func DefaultRoutingConfig() *RoutingConfig {
	return &RoutingConfig{
		Routing: []RoutingRule{
			{When: "task.type == \"architecture_refactor\"", Agent: "claude"},
			{When: "task.type == \"code_generation\"", Agent: "codex"},
			{When: "task.type == \"bulk_tests\"", Agent: "qwen"},
		},
	}
}

// GlobalConfig contains operational settings loaded from ~/.agentcli-conf/config.yaml.
// It holds all non-project-specific configuration that applies across projects.
type GlobalConfig struct {
	Execution      ExecutionConfig  `yaml:"execution"`
	Prompting      PromptingConfig  `yaml:"prompting"`
	Clarifications ClarificationCfg `yaml:"clarifications"`
	Runtime        RuntimeCfg       `yaml:"runtime"`
	Validation     ValidationCfg    `yaml:"validation"`
	Artifacts      ArtifactsCfg     `yaml:"artifacts"`
}

// ProjectLocalConfig is the slim project-level config from .agentctl/config.yaml.
// It contains only project-specific information.
type ProjectLocalConfig struct {
	Project ProjectInfo `yaml:"project"`
}

// DefaultGlobalConfig returns global config with sensible defaults.
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Execution: ExecutionConfig{
			DefaultAgent: "claude",
			Mode:         "strict",
		},
		Prompting: PromptingConfig{
			BuiltinTemplates: []string{
				"clarify_if_needed",
				"plan_before_execution",
				"strict_executor",
				"research_only",
				"review_only",
			},
			DefaultTemplate:        "strict_executor",
			AllowMultipleTemplates: true,
		},
		Clarifications: ClarificationCfg{
			Dir:                ".agentctl/clarifications",
			Strategy:           "by_yml_files",
			AllowMultipleFiles: true,
		},
		Runtime: RuntimeCfg{
			MaxParallelTasks:       4,
			HeartbeatIntervalSec:   5,
			StaleAfterSec:          30,
			GracefulStopTimeoutSec: 20,
			AllowForceKill:         true,
		},
		Validation: ValidationCfg{
			DefaultMode:       "simple",
			DefaultMaxRetries: 3,
			DefaultCommands:   []string{},
		},
		Artifacts: ArtifactsCfg{
			RunsDir:    ".agentctl/runs",
			ContextDir: ".agentctl/context",
			ReviewsDir: ".agentctl/reviews",
		},
	}
}

// DefaultProjectLocalConfig returns slim project config with defaults.
func DefaultProjectLocalConfig() *ProjectLocalConfig {
	return &ProjectLocalConfig{
		Project: ProjectInfo{
			Name:     "my-project",
			Language: "go",
		},
	}
}

// LoadProjectLocalConfig reads config.yaml and parses only the project section.
func LoadProjectLocalConfig(agentctlDir string) (*ProjectLocalConfig, error) {
	path := filepath.Join(agentctlDir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config.yaml: %w", err)
	}
	var cfg ProjectLocalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config.yaml: %w", err)
	}
	return &cfg, nil
}

// LoadGlobalConfig reads config.yaml from the global config directory.
func LoadGlobalConfig(globalDir string) (*GlobalConfig, error) {
	path := filepath.Join(globalDir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading global config.yaml: %w", err)
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing global config.yaml: %w", err)
	}
	return &cfg, nil
}

// MergeConfig combines global config with project-local config into a full ProjectConfig.
func MergeConfig(global *GlobalConfig, local *ProjectLocalConfig) *ProjectConfig {
	return &ProjectConfig{
		Project:        local.Project,
		Execution:      global.Execution,
		Prompting:      global.Prompting,
		Clarifications: global.Clarifications,
		Runtime:        global.Runtime,
		Validation:     global.Validation,
		Artifacts:      global.Artifacts,
	}
}

// MergeAgents combines global and project agents. Project agents override global by ID.
func MergeAgents(global, local *AgentsConfig) *AgentsConfig {
	if local == nil || len(local.Agents) == 0 {
		return global
	}

	localByID := make(map[string]AgentDef, len(local.Agents))
	for _, a := range local.Agents {
		localByID[a.ID] = a
	}

	merged := make([]AgentDef, 0, len(global.Agents)+len(local.Agents))
	seen := make(map[string]struct{})

	// Global agents, overridden by local if same ID exists.
	for _, a := range global.Agents {
		if override, ok := localByID[a.ID]; ok {
			merged = append(merged, override)
		} else {
			merged = append(merged, a)
		}
		seen[a.ID] = struct{}{}
	}

	// Local-only agents not in global.
	for _, a := range local.Agents {
		if _, ok := seen[a.ID]; !ok {
			merged = append(merged, a)
		}
	}

	return &AgentsConfig{Agents: merged}
}

// MergeRouting combines global and project routing.
// If project has routing rules, they take priority over global.
func MergeRouting(global, local *RoutingConfig) *RoutingConfig {
	if local != nil && len(local.Routing) > 0 {
		return local
	}
	return global
}
