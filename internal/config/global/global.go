package global

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/docup/agentctl/internal/config/loader"
)

const (
	// DefaultDirName is the name of the global config directory.
	DefaultDirName = ".agentcli-conf"

	// EnvVar is the environment variable to override the global config directory.
	EnvVar = "AGENTCTL_GLOBAL_CONFIG"
)

// Dir returns the path to the global config directory.
// Uses AGENTCTL_GLOBAL_CONFIG env var if set, otherwise ~/.agentcli-conf/.
func Dir() (string, error) {
	if v := os.Getenv(EnvVar); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, DefaultDirName), nil
}

// EnsureDir creates the global config directory and default files if they don't exist.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating global config directory: %w", err)
	}

	// Write default config files only if they don't exist.
	if err := writeIfNotExists(filepath.Join(dir, "config.yaml"), loader.DefaultGlobalConfig()); err != nil {
		return "", fmt.Errorf("writing default global config.yaml: %w", err)
	}
	if err := writeIfNotExists(filepath.Join(dir, "agents.yaml"), loader.DefaultAgentsConfig()); err != nil {
		return "", fmt.Errorf("writing default global agents.yaml: %w", err)
	}
	if err := writeIfNotExists(filepath.Join(dir, "routing.yaml"), loader.DefaultRoutingConfig()); err != nil {
		return "", fmt.Errorf("writing default global routing.yaml: %w", err)
	}

	return dir, nil
}

// LoadConfig loads the global config.yaml.
func LoadConfig() (*loader.GlobalConfig, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	return loader.LoadGlobalConfig(dir)
}

// LoadAgents loads the global agents.yaml.
func LoadAgents() (*loader.AgentsConfig, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	return loader.LoadAgentsConfig(dir)
}

// LoadRouting loads the global routing.yaml.
func LoadRouting() (*loader.RoutingConfig, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	return loader.LoadRoutingConfig(dir)
}

// SaveConfig writes the global config.yaml.
func SaveConfig(cfg *loader.GlobalConfig) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return writeYAML(filepath.Join(dir, "config.yaml"), cfg)
}

// ResetDefaults overwrites the global config directory with default files.
func ResetDefaults() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating global config directory: %w", err)
	}

	if err := writeYAML(filepath.Join(dir, "config.yaml"), loader.DefaultGlobalConfig()); err != nil {
		return fmt.Errorf("writing default global config.yaml: %w", err)
	}
	if err := writeYAML(filepath.Join(dir, "agents.yaml"), loader.DefaultAgentsConfig()); err != nil {
		return fmt.Errorf("writing default global agents.yaml: %w", err)
	}
	if err := writeYAML(filepath.Join(dir, "routing.yaml"), loader.DefaultRoutingConfig()); err != nil {
		return fmt.Errorf("writing default global routing.yaml: %w", err)
	}

	return nil
}

func writeIfNotExists(path string, v interface{}) error {
	if _, err := os.Stat(path); err == nil {
		return nil // file already exists
	}
	return writeYAML(path, v)
}

func writeYAML(path string, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
