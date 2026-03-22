package taskrunner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"

	"github.com/docup/agentctl/internal/config/loader"
	rt "github.com/docup/agentctl/internal/core/runtime"
)

// AgentAdapter launches a machine-readable adapter process for a stage.
type AgentAdapter interface {
	ID() string
	Capabilities() rt.AdapterCapabilities
	Start(ctx context.Context, spec *rt.StageSpec, specPath string) (AdapterHandle, error)
}

// AdapterHandle represents a live adapter process.
type AdapterHandle interface {
	PID() int
	ProcessGroupID() int
	Events() <-chan rt.ProtocolEvent
	Stderr() <-chan string
	Errors() <-chan error
	Done() <-chan error
	Send(rt.ProtocolCommand) error
	Kill() error
}

// AgentAdapterRegistry resolves configured agent adapters.
type AgentAdapterRegistry struct {
	agents map[string]loader.AgentDef
}

// NewAgentAdapterRegistry creates a registry from agents.yaml definitions.
func NewAgentAdapterRegistry(cfg *loader.AgentsConfig) *AgentAdapterRegistry {
	agents := make(map[string]loader.AgentDef, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		agents[agent.ID] = agent
	}
	return &AgentAdapterRegistry{agents: agents}
}

// Get returns the adapter configured for a specific agent.
func (r *AgentAdapterRegistry) Get(agentID string) (AgentAdapter, error) {
	def, ok := r.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", agentID)
	}
	if def.Transport == "" {
		def.Transport = "ndjson_stdio"
	}
	if def.Transport != "ndjson_stdio" {
		return nil, fmt.Errorf("agent %s uses unsupported transport %q", agentID, def.Transport)
	}
	if def.AdapterCommand == "" {
		return nil, fmt.Errorf("agent %s is missing adapter_command", agentID)
	}
	return &stdioAdapter{def: def}, nil
}

type stdioAdapter struct {
	def loader.AgentDef
}

func (a *stdioAdapter) ID() string {
	return a.def.ID
}

func (a *stdioAdapter) Capabilities() rt.AdapterCapabilities {
	return a.def.Capabilities
}

func (a *stdioAdapter) Start(ctx context.Context, spec *rt.StageSpec, specPath string) (AdapterHandle, error) {
	args := append([]string{}, a.def.AdapterArgs...)
	args = append(args, specPath)

	cmd := exec.CommandContext(ctx, a.def.AdapterCommand, args...)
	cmd.Dir = spec.WorkDir
	cmd.Env = append(os.Environ(),
		"AGENTCTL_STAGE_SPEC="+specPath,
		"AGENTCTL_SESSION_DIR="+spec.SessionDir,
		"AGENTCTL_STAGE_DIR="+spec.StageDir,
		"AGENTCTL_TASK_ID="+spec.TaskID,
		"AGENTCTL_SESSION_ID="+spec.SessionID,
		"AGENTCTL_STAGE_ID="+spec.StageID,
		"AGENTCTL_AGENT_ID="+spec.AgentID,
		"AGENTCTL_CHILD_CLI_COMMAND="+a.def.ChildCLICommand,
		"AGENTCTL_CHILD_CLI_ARGS_JSON="+mustMarshalStringSlice(a.def.ChildCLIArgs),
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("opening adapter stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("opening adapter stderr: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("opening adapter stdin: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting adapter %s: %w", a.def.ID, err)
	}

	pgid, _ := syscall.Getpgid(cmd.Process.Pid)
	handle := &stdioAdapterHandle{
		cmd:    cmd,
		pgid:   pgid,
		stdin:  stdin,
		events: make(chan rt.ProtocolEvent, 32),
		stderr: make(chan string, 32),
		errors: make(chan error, 8),
		done:   make(chan error, 1),
	}
	go handle.readEvents(stdout)
	go handle.readStderr(stderr)
	go handle.wait()
	return handle, nil
}

type stdioAdapterHandle struct {
	cmd    *exec.Cmd
	pgid   int
	stdin  io.WriteCloser
	events chan rt.ProtocolEvent
	stderr chan string
	errors chan error
	done   chan error
	mu     sync.Mutex
}

func (h *stdioAdapterHandle) PID() int {
	if h.cmd.Process == nil {
		return 0
	}
	return h.cmd.Process.Pid
}

func (h *stdioAdapterHandle) ProcessGroupID() int {
	return h.pgid
}

func (h *stdioAdapterHandle) Events() <-chan rt.ProtocolEvent {
	return h.events
}

func (h *stdioAdapterHandle) Stderr() <-chan string {
	return h.stderr
}

func (h *stdioAdapterHandle) Errors() <-chan error {
	return h.errors
}

func (h *stdioAdapterHandle) Done() <-chan error {
	return h.done
}

func (h *stdioAdapterHandle) Send(cmd rt.ProtocolCommand) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	if _, err := h.stdin.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (h *stdioAdapterHandle) Kill() error {
	if h.cmd.Process == nil {
		return nil
	}
	if h.pgid > 0 {
		return syscall.Kill(-h.pgid, syscall.SIGKILL)
	}
	return h.cmd.Process.Kill()
}

func (h *stdioAdapterHandle) readEvents(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		var ev rt.ProtocolEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			h.errors <- fmt.Errorf("parsing adapter event: %w", err)
			continue
		}
		h.events <- ev
	}
	if err := scanner.Err(); err != nil {
		h.errors <- fmt.Errorf("reading adapter stdout: %w", err)
	}
	close(h.events)
}

func (h *stdioAdapterHandle) readStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		h.stderr <- scanner.Text()
	}
	close(h.stderr)
	if err := scanner.Err(); err != nil {
		h.errors <- fmt.Errorf("reading adapter stderr: %w", err)
	}
}

func (h *stdioAdapterHandle) wait() {
	err := h.cmd.Wait()
	close(h.errors)
	h.done <- err
	close(h.done)
}

func mustMarshalStringSlice(values []string) string {
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func nextSequence(current int64) int64 {
	return current + 1
}

func intPtr(n int) *int {
	return &n
}

func atoiOrZero(value string) int {
	n, _ := strconv.Atoi(value)
	return n
}
