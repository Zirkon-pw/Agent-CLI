package validationrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/docup/agentctl/internal/core/run"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/core/validation"
	"github.com/docup/agentctl/internal/infra/executor"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// Runner executes validation commands and handles retry logic.
type Runner struct {
	projectRoot string
	agentExec   *executor.AgentExecutor
	runStore    *fsstore.RunStore
	agentctlDir string
}

// NewRunner creates a validation runner.
func NewRunner(projectRoot string, agentExec *executor.AgentExecutor, runStore *fsstore.RunStore, agentctlDir string) *Runner {
	return &Runner{
		projectRoot: projectRoot,
		agentExec:   agentExec,
		runStore:    runStore,
		agentctlDir: agentctlDir,
	}
}

// Validate runs validation commands according to the task's validation config.
func (r *Runner) Validate(ctx context.Context, t *task.Task, taskRun *run.Run) (*validation.Report, error) {
	report := &validation.Report{
		TaskID:     t.ID,
		RunID:      taskRun.ID,
		Mode:       string(t.Validation.Mode),
		MaxRetries: t.Validation.MaxRetries,
		CreatedAt:  time.Now(),
	}

	// Run initial validation
	results := r.runCommands(ctx, t.Validation.Commands)
	report.Results = results
	report.AllPassed = allPassed(results)

	if report.AllPassed || t.Validation.Mode == task.ValidationModeSimple {
		return report, nil
	}

	// Full mode: retry loop with agent fixes
	for attempt := 1; attempt <= t.Validation.MaxRetries; attempt++ {
		report.TotalRetries = attempt

		retryRecord := validation.RetryRecord{
			Attempt:   attempt,
			Results:   results,
			AllPassed: false,
			Timestamp: time.Now(),
		}

		// Build fix prompt from failed commands
		fixPrompt := r.buildFixPrompt(t, results)
		retryRecord.FixPrompt = fixPrompt

		slog.Info("validation failed, requesting agent fix",
			"task", t.ID,
			"attempt", attempt,
			"max_retries", t.Validation.MaxRetries,
		)

		// Execute agent with fix prompt
		fixResult, err := r.agentExec.ExecuteWithPromptFile(
			ctx, t.Agent, fixPrompt, r.projectRoot,
			t.ID, taskRun.ID, r.agentctlDir,
		)
		if err != nil {
			slog.Error("fix execution failed", "error", err)
			retryRecord.Results = results
			report.Retries = append(report.Retries, retryRecord)
			break
		}

		// Save fix logs
		fixLog := fmt.Sprintf("--- Fix attempt %d ---\nExit: %d\nStdout:\n%s\nStderr:\n%s\n",
			attempt, fixResult.ExitCode, fixResult.Stdout, fixResult.Stderr)
		r.runStore.WriteArtifact(t.ID, taskRun.ID, fmt.Sprintf("fix_attempt_%d.log", attempt), []byte(fixLog))

		// Re-run validation
		results = r.runCommands(ctx, t.Validation.Commands)
		retryRecord.Results = results
		retryRecord.AllPassed = allPassed(results)
		report.Retries = append(report.Retries, retryRecord)

		if retryRecord.AllPassed {
			report.Results = results
			report.AllPassed = true
			break
		}
	}

	// Save final report
	reportData, _ := json.MarshalIndent(report, "", "  ")
	r.runStore.WriteArtifact(t.ID, taskRun.ID, "validation.json", reportData)

	return report, nil
}

func (r *Runner) runCommands(ctx context.Context, commands []string) []validation.CheckResult {
	var results []validation.CheckResult
	for _, cmdStr := range commands {
		result := r.runSingleCommand(ctx, cmdStr)
		results = append(results, result)
	}
	return results
}

func (r *Runner) runSingleCommand(ctx context.Context, cmdStr string) validation.CheckResult {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = r.projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return validation.CheckResult{
		Command:  cmdStr,
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		Passed:   exitCode == 0,
	}
}

func (r *Runner) buildFixPrompt(t *task.Task, results []validation.CheckResult) string {
	var buf bytes.Buffer
	buf.WriteString("# Validation Failed — Fix Required\n\n")
	buf.WriteString(fmt.Sprintf("Task: %s\n", t.ID))
	buf.WriteString(fmt.Sprintf("Goal: %s\n\n", t.Goal))
	buf.WriteString("The following validation commands failed. Fix the issues and ensure all commands pass.\n\n")

	for _, res := range results {
		if !res.Passed {
			buf.WriteString(fmt.Sprintf("## Failed: `%s` (exit code %d)\n", res.Command, res.ExitCode))
			if res.Stdout != "" {
				buf.WriteString("### Stdout\n```\n")
				buf.WriteString(res.Stdout)
				buf.WriteString("\n```\n")
			}
			if res.Stderr != "" {
				buf.WriteString("### Stderr\n```\n")
				buf.WriteString(res.Stderr)
				buf.WriteString("\n```\n")
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString("Fix the errors above. Only modify files within the task's allowed scope.\n")
	return buf.String()
}

func allPassed(results []validation.CheckResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}
