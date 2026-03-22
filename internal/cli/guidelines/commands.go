package guidelines

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewGuidelinesCmd creates the guidelines command group.
func NewGuidelinesCmd(agentctlDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guidelines",
		Short: "Manage project guidelines",
	}

	cmd.AddCommand(newAddGuidelineCmd(agentctlDir))
	cmd.AddCommand(newListGuidelinesCmd(agentctlDir))
	cmd.AddCommand(newShowGuidelineCmd(agentctlDir))

	return cmd
}

func newAddGuidelineCmd(agentctlDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a guideline file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPath := args[0]
			f, err := os.Open(srcPath)
			if err != nil {
				return fmt.Errorf("opening file: %w", err)
			}
			defer f.Close()

			data, err := io.ReadAll(f)
			if err != nil {
				return err
			}

			name := filepath.Base(srcPath)
			dstDir := filepath.Join(agentctlDir, "guidelines")
			os.MkdirAll(dstDir, 0755)
			dstPath := filepath.Join(dstDir, name)

			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}

			fmt.Printf("Added guideline: %s\n", name)
			return nil
		},
	}
}

func newListGuidelinesCmd(agentctlDir string) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List available guidelines",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := filepath.Join(agentctlDir, "guidelines")
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No guidelines found.")
					return nil
				}
				return err
			}

			if len(entries) == 0 {
				fmt.Println("No guidelines found.")
				return nil
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
					fmt.Printf("  %s\n", name)
				}
			}
			return nil
		},
	}
}

func newShowGuidelineCmd(agentctlDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show a guideline content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dir := filepath.Join(agentctlDir, "guidelines")

			// Try with .md extension first
			path := filepath.Join(dir, name+".md")
			data, err := os.ReadFile(path)
			if err != nil {
				path = filepath.Join(dir, name)
				data, err = os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("guideline %q not found", name)
				}
			}

			fmt.Print(string(data))
			return nil
		},
	}
}
