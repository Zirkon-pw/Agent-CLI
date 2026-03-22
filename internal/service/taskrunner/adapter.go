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
	"strings"
	"sync"
	"syscall"

	"github.com/docup/agentctl/internal/config/loader"
	rt "github.com/docup/agentctl/internal/core/runtime"
)

// AgentRuntimeDriver launches an agent according to its configured runtime kind.
type AgentRuntimeDriver interface {
	ID() string
	Kind() loader.AgentRuntimeKind
	Capabilities() rt.AdapterCapabilities
	SupportsStage(rt.StageType) bool
	Start(ctx context.Context, spec *rt.StageSpec, specPath string) (DriverHandle, error)
}

// DriverHandle represents a live driver process.
type DriverHandle interface {
	PID() int
	ProcessGroupID() int
	Events() <-chan rt.ProtocolEvent
	Stdout() <-chan string
	Stderr() <-chan string
	Errors() <-chan error
	Done() <-chan error
	Send(rt.ProtocolCommand) error
	Kill() error
}

// AgentRuntimeRegistry resolves configured agent drivers.
type AgentRuntimeRegistry struct {
	agents map[string]loader.AgentDef
}

// NewAgentRuntimeRegistry creates a registry from agents.yaml definitions.
func NewAgentRuntimeRegistry(cfg *loader.AgentsConfig) *AgentRuntimeRegistry {
	agents := make(map[string]loader.AgentDef, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		agents[agent.ID] = agent
	}
	return &AgentRuntimeRegistry{agents: agents}
}

// Get returns the driver configured for a specific agent.
func (r *AgentRuntimeRegistry) Get(agentID string) (AgentRuntimeDriver, error) {
	def, ok := r.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", agentID)
	}

	switch def.Runtime.Kind {
	case loader.AgentRuntimeKindProtocolAdapter:
		return &protocolAdapterDriver{def: def}, nil
	case loader.AgentRuntimeKindRawCLI:
		return &rawCLIDriver{def: def}, nil
	default:
		return nil, fmt.Errorf("agent %s uses unsupported runtime.kind %q", agentID, def.Runtime.Kind)
	}
}

type protocolAdapterDriver struct {
	def loader.AgentDef
}

func (d *protocolAdapterDriver) ID() string {
	return d.def.ID
}

func (d *protocolAdapterDriver) Kind() loader.AgentRuntimeKind {
	return d.def.Runtime.Kind
}

func (d *protocolAdapterDriver) Capabilities() rt.AdapterCapabilities {
	return d.def.Capabilities()
}

func (d *protocolAdapterDriver) SupportsStage(stage rt.StageType) bool {
	return d.def.SupportsStage(stage)
}

func (d *protocolAdapterDriver) Start(ctx context.Context, spec *rt.StageSpec, specPath string) (DriverHandle, error) {
	args := append([]string{}, d.def.Runtime.Exec.Args...)
	args = append(args, specPath)

	cmd := exec.CommandContext(ctx, d.def.Runtime.Exec.Command, args...)
	cmd.Dir = spec.WorkDir
	cmd.Env = append(os.Environ(), buildStageEnv(spec, specPath, d.def)...)
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
		return nil, fmt.Errorf("starting adapter %s: %w", d.def.ID, err)
	}

	pgid, _ := syscall.Getpgid(cmd.Process.Pid)
	stdoutCh := make(chan string)
	close(stdoutCh)
	handle := &driverHandle{
		cmd:    cmd,
		pgid:   pgid,
		stdin:  stdin,
		kind:   loader.AgentRuntimeKindProtocolAdapter,
		events: make(chan rt.ProtocolEvent, 32),
		stdout: stdoutCh,
		stderr: make(chan string, 32),
		errors: make(chan error, 8),
		done:   make(chan error, 1),
	}
	go handle.readProtocolEvents(stdout)
	go handle.readLines(stderr, handle.stderr)
	go handle.wait()
	return handle, nil
}

type rawCLIDriver struct {
	def loader.AgentDef
}

func (d *rawCLIDriver) ID() string {
	return d.def.ID
}

func (d *rawCLIDriver) Kind() loader.AgentRuntimeKind {
	return d.def.Runtime.Kind
}

func (d *rawCLIDriver) Capabilities() rt.AdapterCapabilities {
	return d.def.Capabilities()
}

func (d *rawCLIDriver) SupportsStage(stage rt.StageType) bool {
	return d.def.SupportsStage(stage)
}

func (d *rawCLIDriver) Start(ctx context.Context, spec *rt.StageSpec, specPath string) (DriverHandle, error) {
	promptArg, err := rawPromptArg(spec, specPath)
	if err != nil {
		return nil, err
	}

	args := append([]string{}, d.def.Runtime.Exec.Args...)
	if promptArg != "" {
		args = append(args, promptArg)
	}

	cmd := exec.CommandContext(ctx, d.def.Runtime.Exec.Command, args...)
	cmd.Dir = spec.WorkDir
	cmd.Env = append(os.Environ(), buildStageEnv(spec, specPath, d.def)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("opening raw stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("opening raw stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting raw cli %s: %w", d.def.ID, err)
	}

	pgid, _ := syscall.Getpgid(cmd.Process.Pid)
	eventsCh := make(chan rt.ProtocolEvent)
	close(eventsCh)
	handle := &driverHandle{
		cmd:    cmd,
		pgid:   pgid,
		kind:   loader.AgentRuntimeKindRawCLI,
		events: eventsCh,
		stdout: make(chan string, 32),
		stderr: make(chan string, 32),
		errors: make(chan error, 8),
		done:   make(chan error, 1),
	}
	go handle.readLines(stdout, handle.stdout)
	go handle.readLines(stderr, handle.stderr)
	go handle.wait()
	return handle, nil
}

type driverHandle struct {
	cmd    *exec.Cmd
	pgid   int
	stdin  io.WriteCloser
	kind   loader.AgentRuntimeKind
	events chan rt.ProtocolEvent
	stdout chan string
	stderr chan string
	errors chan error
	done   chan error
	mu     sync.Mutex
}

func (h *driverHandle) PID() int {
	if h.cmd.Process == nil {
		return 0
	}
	return h.cmd.Process.Pid
}

func (h *driverHandle) ProcessGroupID() int {
	return h.pgid
}

func (h *driverHandle) Events() <-chan rt.ProtocolEvent {
	return h.events
}

func (h *driverHandle) Stdout() <-chan string {
	return h.stdout
}

func (h *driverHandle) Stderr() <-chan string {
	return h.stderr
}

func (h *driverHandle) Errors() <-chan error {
	return h.errors
}

func (h *driverHandle) Done() <-chan error {
	return h.done
}

func (h *driverHandle) Send(cmd rt.ProtocolCommand) error {
	switch h.kind {
	case loader.AgentRuntimeKindProtocolAdapter:
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
	case loader.AgentRuntimeKindRawCLI:
		switch cmd.Type {
		case rt.CommandTypeCancel:
			return h.signal(syscall.SIGTERM)
		case rt.CommandTypeKill:
			return h.signal(syscall.SIGKILL)
		case rt.CommandTypePing:
			return nil
		default:
			return fmt.Errorf("raw cli runtime does not support %s control", cmd.Type)
		}
	default:
		return fmt.Errorf("unsupported runtime kind %q", h.kind)
	}
}

func (h *driverHandle) Kill() error {
	return h.signal(syscall.SIGKILL)
}

func (h *driverHandle) signal(sig syscall.Signal) error {
	if h.cmd.Process == nil {
		return nil
	}
	if h.pgid > 0 {
		return syscall.Kill(-h.pgid, sig)
	}
	return h.cmd.Process.Signal(sig)
}

func (h *driverHandle) readProtocolEvents(r io.Reader) {
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

func (h *driverHandle) readLines(r io.Reader, out chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		out <- scanner.Text()
	}
	close(out)
	if err := scanner.Err(); err != nil {
		h.errors <- fmt.Errorf("reading process stream: %w", err)
	}
}

func (h *driverHandle) wait() {
	err := h.cmd.Wait()
	close(h.errors)
	h.done <- err
	close(h.done)
}

func buildStageEnv(spec *rt.StageSpec, specPath string, def loader.AgentDef) []string {
	childCommand := ""
	childArgsJSON := "[]"
	if def.Runtime.ChildCLI != nil {
		childCommand = def.Runtime.ChildCLI.Command
		childArgsJSON = mustMarshalStringSlice(def.Runtime.ChildCLI.Args)
	}

	return []string{
		"AGENTCTL_STAGE_SPEC=" + specPath,
		"AGENTCTL_SESSION_DIR=" + spec.SessionDir,
		"AGENTCTL_STAGE_DIR=" + spec.StageDir,
		"AGENTCTL_TASK_ID=" + spec.TaskID,
		"AGENTCTL_SESSION_ID=" + spec.SessionID,
		"AGENTCTL_STAGE_ID=" + spec.StageID,
		"AGENTCTL_AGENT_ID=" + spec.AgentID,
		"AGENTCTL_TASK_PATH=" + spec.TaskPath,
		"AGENTCTL_CONTEXT_DIR=" + spec.ContextDir,
		"AGENTCTL_PROMPT_PATH=" + spec.PromptPath,
		"AGENTCTL_CHILD_CLI_COMMAND=" + childCommand,
		"AGENTCTL_CHILD_CLI_ARGS_JSON=" + childArgsJSON,
	}
}

func rawPromptArg(spec *rt.StageSpec, specPath string) (string, error) {
	if spec.PromptPath != "" {
		data, err := os.ReadFile(spec.PromptPath)
		if err != nil {
			return "", fmt.Errorf("reading prompt for raw cli: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return specPath, nil
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
