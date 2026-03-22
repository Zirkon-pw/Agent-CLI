package validation

import "time"

// CheckResult represents the outcome of a single validation command.
type CheckResult struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration_ms"`
	Passed   bool          `json:"passed"`
}

// RetryRecord stores info about a single validation retry attempt.
type RetryRecord struct {
	Attempt   int           `json:"attempt"`
	Results   []CheckResult `json:"results"`
	AllPassed bool          `json:"all_passed"`
	FixPrompt string        `json:"fix_prompt,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// Report is the full validation report for a run.
type Report struct {
	TaskID     string        `json:"task_id"`
	RunID      string        `json:"run_id"`
	Mode       string        `json:"mode"`
	Results    []CheckResult `json:"results"`
	AllPassed  bool          `json:"all_passed"`
	Retries    []RetryRecord `json:"retries,omitempty"`
	TotalRetries int         `json:"total_retries"`
	MaxRetries int           `json:"max_retries"`
	CreatedAt  time.Time     `json:"created_at"`
}

// HasFailures returns true if any check failed.
func (r *Report) HasFailures() bool {
	return !r.AllPassed
}

// FailedCommands returns the list of commands that failed.
func (r *Report) FailedCommands() []CheckResult {
	var failed []CheckResult
	for _, cr := range r.Results {
		if !cr.Passed {
			failed = append(failed, cr)
		}
	}
	return failed
}

// CanRetry returns true if more retry attempts are available.
func (r *Report) CanRetry() bool {
	return r.TotalRetries < r.MaxRetries
}
