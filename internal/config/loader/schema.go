package loader

import (
	"fmt"
	"os"
	"path/filepath"

	rt "github.com/docup/agentctl/internal/core/runtime"
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

type AgentRuntimeKind string

const (
	AgentRuntimeKindProtocolAdapter AgentRuntimeKind = "protocol_adapter"
	AgentRuntimeKindRawCLI          AgentRuntimeKind = "raw_cli"
)

// AgentMetadata contains descriptive, non-runtime agent metadata.
type AgentMetadata struct {
	Specialization []string `yaml:"specialization"`
	Strengths      []string `yaml:"strengths"`
	Speed          string   `yaml:"speed"`
	Cost           string   `yaml:"cost"`
	ContextLimit   string   `yaml:"context_limit"`
	Modes          []string `yaml:"modes"`
	Tools          []string `yaml:"tools"`
}

// AgentExec describes an executable command with arguments.
type AgentExec struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

// AgentControl describes supported runtime controls.
type AgentControl struct {
	Cancel    bool `yaml:"cancel"`
	Pause     bool `yaml:"pause"`
	Resume    bool `yaml:"resume"`
	Kill      bool `yaml:"kill"`
	Heartbeat bool `yaml:"heartbeat"`
}

// AgentProtocol describes machine-readable protocol settings for wrappers.
type AgentProtocol struct {
	Version string `yaml:"version"`
}

// AgentRuntime defines how the agent is executed.
type AgentRuntime struct {
	Kind            AgentRuntimeKind `yaml:"kind"`
	Exec            AgentExec        `yaml:"exec"`
	SupportedStages []rt.StageType   `yaml:"supported_stages"`
	Control         AgentControl     `yaml:"control"`
	Protocol        *AgentProtocol   `yaml:"protocol,omitempty"`
	ChildCLI        *AgentExec       `yaml:"child_cli,omitempty"`
}

// AgentDef describes an available agent from agents.yaml.
type AgentDef struct {
	ID       string        `yaml:"id"`
	Role     string        `yaml:"role"`
	Metadata AgentMetadata `yaml:"metadata"`
	Runtime  AgentRuntime  `yaml:"runtime"`
}

// Capabilities derives runtime capabilities from the configured agent runtime.
func (a AgentDef) Capabilities() rt.AdapterCapabilities {
	supportsReview := false
	supportsHandoff := false
	for _, stage := range a.Runtime.SupportedStages {
		if stage == rt.StageTypeReview {
			supportsReview = true
		}
		if stage == rt.StageTypeHandoff {
			supportsHandoff = true
		}
	}

	protocolVersion := ""
	supportsClarification := false
	if a.Runtime.Kind == AgentRuntimeKindProtocolAdapter {
		supportsClarification = true
		if a.Runtime.Protocol != nil {
			protocolVersion = a.Runtime.Protocol.Version
		}
	}

	return rt.AdapterCapabilities{
		ProtocolVersion:       protocolVersion,
		SupportsCancel:        a.Runtime.Control.Cancel,
		SupportsPause:         a.Runtime.Control.Pause,
		SupportsResume:        a.Runtime.Control.Resume,
		SupportsKill:          a.Runtime.Control.Kill,
		SupportsHeartbeat:     a.Runtime.Control.Heartbeat,
		SupportsClarification: supportsClarification,
		SupportsReview:        supportsReview,
		SupportsHandoff:       supportsHandoff,
	}
}

// SupportsStage returns true if the agent is configured to handle the stage type.
func (a AgentDef) SupportsStage(stage rt.StageType) bool {
	for _, candidate := range a.Runtime.SupportedStages {
		if candidate == stage {
			return true
		}
	}
	return false
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
				ID:   "claude",
				Role: "executor",
				Metadata: AgentMetadata{
					Specialization: []string{"architecture_refactor", "deep_analysis"},
					Strengths:      []string{"large_context_reasoning", "architecture_review"},
					Speed:          "medium",
					Cost:           "high",
					ContextLimit:   "large",
					Modes:          []string{"strict", "research"},
					Tools:          []string{"filesystem", "git"},
				},
				Runtime: AgentRuntime{
					Kind:            AgentRuntimeKindProtocolAdapter,
					Exec:            AgentExec{Command: "claude", Args: []string{"-p"}},
					SupportedStages: []rt.StageType{rt.StageTypeExecute, rt.StageTypeValidateFix, rt.StageTypeReview, rt.StageTypeHandoff},
					Control: AgentControl{
						Cancel: true,
						Kill:   true,
					},
					Protocol: &AgentProtocol{Version: "v1"},
					ChildCLI: &AgentExec{Command: "claude", Args: []string{"-p"}},
				},
			},
			{
				ID:   "codex",
				Role: "executor",
				Metadata: AgentMetadata{
					Specialization: []string{"code_generation", "task_execution"},
					Strengths:      []string{"code_edits", "terminal_workflow"},
					Speed:          "high",
					Cost:           "medium",
					ContextLimit:   "medium",
					Modes:          []string{"strict", "fast"},
					Tools:          []string{"filesystem", "terminal"},
				},
				Runtime: AgentRuntime{
					Kind:            AgentRuntimeKindProtocolAdapter,
					Exec:            AgentExec{Command: "codex", Args: []string{"-q"}},
					SupportedStages: []rt.StageType{rt.StageTypeExecute, rt.StageTypeValidateFix, rt.StageTypeReview, rt.StageTypeHandoff},
					Control: AgentControl{
						Cancel: true,
						Kill:   true,
					},
					Protocol: &AgentProtocol{Version: "v1"},
					ChildCLI: &AgentExec{Command: "codex", Args: []string{"-q"}},
				},
			},
			{
				ID:   "qwen",
				Role: "executor",
				Metadata: AgentMetadata{
					Specialization: []string{"bulk_tests", "code_generation"},
					Strengths:      []string{"fast_generation", "code_edits"},
					Speed:          "high",
					Cost:           "low",
					ContextLimit:   "medium",
					Modes:          []string{"strict", "fast"},
					Tools:          []string{"filesystem", "terminal"},
				},
				Runtime: AgentRuntime{
					Kind:            AgentRuntimeKindRawCLI,
					Exec:            AgentExec{Command: "qwen", Args: []string{}},
					SupportedStages: []rt.StageType{rt.StageTypeExecute, rt.StageTypeValidateFix},
					Control: AgentControl{
						Cancel: true,
						Kill:   true,
					},
				},
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
		"command",
		"args",
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
	for _, agent := range cfg.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent id is required")
		}
		if agent.Runtime.Kind == "" {
			return fmt.Errorf("agent %q is missing runtime.kind", agent.ID)
		}
		if agent.Runtime.Exec.Command == "" {
			return fmt.Errorf("agent %q is missing runtime.exec.command", agent.ID)
		}
		if len(agent.Runtime.SupportedStages) == 0 {
			return fmt.Errorf("agent %q must declare at least one runtime.supported_stages entry", agent.ID)
		}
		switch agent.Runtime.Kind {
		case AgentRuntimeKindProtocolAdapter:
			if agent.Runtime.Protocol == nil || agent.Runtime.Protocol.Version == "" {
				return fmt.Errorf("agent %q must declare runtime.protocol.version for protocol_adapter runtime", agent.ID)
			}
		case AgentRuntimeKindRawCLI:
			if agent.Runtime.Protocol != nil {
				return fmt.Errorf("agent %q cannot declare runtime.protocol for raw_cli runtime", agent.ID)
			}
			if agent.Runtime.ChildCLI != nil {
				return fmt.Errorf("agent %q cannot declare runtime.child_cli for raw_cli runtime", agent.ID)
			}
		default:
			return fmt.Errorf("agent %q uses unsupported runtime.kind %q", agent.ID, agent.Runtime.Kind)
		}
	}
	return nil
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
