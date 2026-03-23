package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/docup/agentctl/internal/bootstrap"
	"github.com/docup/agentctl/internal/cli"
	cliclar "github.com/docup/agentctl/internal/cli/clarification"
	"github.com/docup/agentctl/internal/cli/guidelines"
	"github.com/docup/agentctl/internal/cli/help"
	"github.com/docup/agentctl/internal/cli/result"
	"github.com/docup/agentctl/internal/cli/root"
	clitask "github.com/docup/agentctl/internal/cli/task"
	clitmpl "github.com/docup/agentctl/internal/cli/template"
	"github.com/docup/agentctl/internal/infra/logging"
)

func main() {
	logging.Setup(false)
	rootCmd := buildRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildRootCmd() *cobra.Command {
	app, appErr := bootstrap.NewApp()
	return newRootCmd(app, appErr)
}

func newRootCmd(app *bootstrap.App, appErr error) *cobra.Command {
	rootCmd := root.NewRootCmd()

	// Init doesn't need workspace
	rootCmd.AddCommand(cli.NewInitCmd())
	rootCmd.AddCommand(help.NewHelpCmd())

	if appErr != nil {
		configureWorkspaceUnavailable(rootCmd, appErr)
	}

	if app != nil {
		addWorkspaceCommands(rootCmd, app)
	} else {
		addUnavailableWorkspaceCommands(rootCmd, appErr)
	}

	return rootCmd
}

func configureWorkspaceUnavailable(rootCmd *cobra.Command, appErr error) {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		name := cmd.Name()
		switch name {
		case "agentctl", "init", "help", "topics", "completion", "__complete", "__completeNoDesc":
			return nil
		default:
			return workspaceUnavailableError(appErr)
		}
	}
}

func addWorkspaceCommands(rootCmd *cobra.Command, app *bootstrap.App) {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
			}
			return cmd.Help()
		},
	}
	taskCmd.AddCommand(clitask.NewCreateCmd(app.CreateTask))
	taskCmd.AddCommand(clitask.NewRunCmd(app.RunTask))
	taskCmd.AddCommand(clitask.NewListCmd(app.ListTasks))
	taskCmd.AddCommand(clitask.NewInspectCmd(app.InspectTask, app.RuntimeMgr))
	taskCmd.AddCommand(clitask.NewPsCmd(app.RuntimeMgr))
	taskCmd.AddCommand(clitask.NewStopCmd(app.Orchestrator))
	taskCmd.AddCommand(clitask.NewKillCmd(app.Orchestrator))
	taskCmd.AddCommand(clitask.NewCancelCmd(app.Orchestrator))
	taskCmd.AddCommand(clitask.NewAcceptCmd(app.Orchestrator))
	taskCmd.AddCommand(clitask.NewRejectCmd(app.Orchestrator))
	taskCmd.AddCommand(clitask.NewRerunCmd(app.Orchestrator))
	taskCmd.AddCommand(clitask.NewUpdateCmd(app.UpdateTask))
	taskCmd.AddCommand(clitask.NewLogsCmd(app.RunStore))
	taskCmd.AddCommand(clitask.NewEventsCmd(app.RuntimeMgr))
	taskCmd.AddCommand(clitask.NewWatchCmd(app.InspectTask, app.RuntimeMgr))
	taskCmd.AddCommand(clitask.NewRouteCmd(app.Orchestrator))
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(clitmpl.NewTemplateCmd(app.TemplateStore))
	rootCmd.AddCommand(cliclar.NewClarificationCmd(app.ClarMgr))
	rootCmd.AddCommand(guidelines.NewGuidelinesCmd(app.AgentctlDir))
	rootCmd.AddCommand(result.NewResultCmd(app.RunStore))
}

func addUnavailableWorkspaceCommands(rootCmd *cobra.Command, appErr error) {
	rootCmd.AddCommand(newUnavailableGroupCmd("task", "Manage tasks", appErr))
	rootCmd.AddCommand(newUnavailableGroupCmd("template", "Manage prompt templates", appErr))
	rootCmd.AddCommand(newUnavailableGroupCmd("clarification", "Manage task clarifications", appErr, "clar"))
	rootCmd.AddCommand(newUnavailableGroupCmd("guidelines", "Manage project guidelines", appErr))
	rootCmd.AddCommand(newUnavailableGroupCmd("result", "View task execution results", appErr))
}

func newUnavailableGroupCmd(use, short string, appErr error, aliases ...string) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Short:              short,
		Aliases:            aliases,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return workspaceUnavailableError(appErr)
		},
	}
}

func workspaceUnavailableError(appErr error) error {
	if appErr == nil {
		return fmt.Errorf("workspace unavailable")
	}
	if strings.Contains(appErr.Error(), ".agentctl directory not found") {
		return fmt.Errorf("workspace not initialized: %v\nRun 'agentctl init' first", appErr)
	}
	return fmt.Errorf("workspace unavailable: %v", appErr)
}
