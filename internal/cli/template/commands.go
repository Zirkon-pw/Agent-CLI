package template

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/docup/agentctl/internal/config/builtin_templates"
	coretemplate "github.com/docup/agentctl/internal/core/template"
	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewTemplateCmd creates the template command group.
func NewTemplateCmd(templateStore *fsstore.TemplateStore) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage prompt templates",
	}

	cmd.AddCommand(newListCmd(templateStore))
	cmd.AddCommand(newShowCmd(templateStore))
	cmd.AddCommand(newAddCmd(templateStore))

	return cmd
}

func newListCmd(templateStore *fsstore.TemplateStore) *cobra.Command {
	var builtin bool

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List available templates",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tTYPE\tDESCRIPTION")

			if builtin {
				for _, t := range builtin_templates.All() {
					fmt.Fprintf(w, "%s\t%s\tbuiltin\t%s\n", t.ID, t.Name, truncateDesc(t.Description, 60))
				}
			}

			custom, err := templateStore.List()
			if err == nil {
				for _, t := range custom {
					fmt.Fprintf(w, "%s\t%s\tcustom\t%s\n", t.ID, t.Name, truncateDesc(t.Description, 60))
				}
			}

			w.Flush()
			return nil
		},
	}

	cmd.Flags().BoolVar(&builtin, "builtin", false, "Show built-in templates")
	return cmd
}

func newShowCmd(templateStore *fsstore.TemplateStore) *cobra.Command {
	return &cobra.Command{
		Use:   "show <template-id>",
		Short: "Show template details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			tmpl := builtin_templates.ByID(id)
			if tmpl == nil {
				var err error
				tmpl, err = templateStore.Load(id)
				if err != nil {
					return fmt.Errorf("template %q not found", id)
				}
			}

			data, err := yaml.Marshal(tmpl)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		},
	}
}

func newAddCmd(templateStore *fsstore.TemplateStore) *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a custom template from a YAML file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("opening template file: %w", err)
			}

			var tmpl coretemplate.PromptTemplate
			if err := yaml.Unmarshal(data, &tmpl); err != nil {
				return fmt.Errorf("parsing template: %w", err)
			}

			if tmpl.ID == "" {
				return fmt.Errorf("template must have an 'id' field")
			}

			tmpl.IsBuiltin = false
			tmpl.FilePath = path

			if err := templateStore.Save(&tmpl); err != nil {
				return err
			}

			fmt.Printf("Added template %s: %s\n", tmpl.ID, tmpl.Name)
			return nil
		},
	}
}

func truncateDesc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
