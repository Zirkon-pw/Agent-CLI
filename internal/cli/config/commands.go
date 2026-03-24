package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/docup/agentctl/internal/config/global"
)

// NewConfigCmd creates the config command group.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage global configuration",
		Long:  "View and modify global agentctl settings stored in ~/.agentcli-conf/.",
	}

	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigResetCmd())

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a global config value",
		Long:  "Get a value by dot-notation path (e.g. execution.default_agent).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := global.EnsureDir(); err != nil {
				return err
			}

			cfg, err := global.LoadConfig()
			if err != nil {
				return err
			}

			value, err := global.GetValue(cfg, args[0])
			if err != nil {
				return err
			}

			fmt.Println(value)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key>=<value> | <key> <value>",
		Short: "Set a global config value",
		Long:  "Set a value by dot-notation path (e.g. execution.default_agent=codex or execution.default_agent codex).\nFor list values, use comma-separated format (e.g. prompting.builtin_templates=a,b,c).",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 || len(args) == 2 {
				return nil
			}
			return cobra.ExactArgs(2)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value, err := parseSetArgs(args)
			if err != nil {
				return err
			}

			if _, err := global.EnsureDir(); err != nil {
				return err
			}

			cfg, err := global.LoadConfig()
			if err != nil {
				return err
			}

			updated, err := global.SetValue(cfg, key, value)
			if err != nil {
				return err
			}

			if err := global.SaveConfig(updated); err != nil {
				return err
			}

			fmt.Printf("%s = %s\n", key, value)
			return nil
		},
	}
}

func parseSetArgs(args []string) (string, string, error) {
	if len(args) == 2 {
		return args[0], args[1], nil
	}

	parts := strings.SplitN(args[0], "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected key=value or key value format, got %q", args[0])
	}
	return parts[0], parts[1], nil
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "Show all global configuration",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := global.EnsureDir(); err != nil {
				return err
			}

			cfg, err := global.LoadConfig()
			if err != nil {
				return err
			}

			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}

			fmt.Print(string(data))
			return nil
		},
	}
}

func newConfigResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset global config to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := global.ResetDefaults(); err != nil {
				return err
			}

			dir, err := global.Dir()
			if err != nil {
				return err
			}

			fmt.Printf("Global config reset to defaults at %s\n", dir)
			return nil
		},
	}
}
