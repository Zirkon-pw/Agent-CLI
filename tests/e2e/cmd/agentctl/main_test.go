package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "agentctl")
	root := projectRoot(t)

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/agentctl")
	cmd.Dir = root
	cmd.Env = buildEnv(root)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Binary should exist and be executable
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("binary not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("binary is empty")
	}
}

func TestRun_Help(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "agentctl")
	root := projectRoot(t)

	build := exec.Command("go", "build", "-o", bin, "./cmd/agentctl")
	build.Dir = root
	build.Env = buildEnv(root)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	cmd := exec.Command(bin, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Fatal("help output is empty")
	}
}

func TestRun_InitAndTaskList(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "agentctl")
	root := projectRoot(t)

	build := exec.Command("go", "build", "-o", bin, "./cmd/agentctl")
	build.Dir = root
	build.Env = buildEnv(root)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	workDir := t.TempDir()

	// Init workspace
	initCmd := exec.Command(bin, "init")
	initCmd.Dir = workDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	// Verify .agentctl was created
	if _, err := os.Stat(filepath.Join(workDir, ".agentctl")); err != nil {
		t.Fatalf(".agentctl not created: %v", err)
	}

	// Task list should work on empty workspace
	listCmd := exec.Command(bin, "task", "list")
	listCmd.Dir = workDir
	if out, err := listCmd.CombinedOutput(); err != nil {
		t.Fatalf("task list failed: %v\n%s", err, out)
	}
}

func TestRun_TaskHelpDoesNotExposePauseOrResume(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "agentctl")
	root := projectRoot(t)

	build := exec.Command("go", "build", "-o", bin, "./cmd/agentctl")
	build.Dir = root
	build.Env = buildEnv(root)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	workDir := t.TempDir()
	initCmd := exec.Command(bin, "init")
	initCmd.Dir = workDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	cmd := exec.Command(bin, "task", "--help")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("task help failed: %v\n%s", err, out)
	}

	output := string(out)
	if strings.Contains(output, "pause") {
		t.Fatalf("task help should not expose pause command, got:\n%s", output)
	}
	if strings.Contains(output, "resume") {
		t.Fatalf("task help should not expose resume command, got:\n%s", output)
	}
}

func TestRun_TaskPauseAndResumeAreUnknownCommands(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "agentctl")
	root := projectRoot(t)

	build := exec.Command("go", "build", "-o", bin, "./cmd/agentctl")
	build.Dir = root
	build.Env = buildEnv(root)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	workDir := t.TempDir()
	initCmd := exec.Command(bin, "init")
	initCmd.Dir = workDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	for _, args := range [][]string{{"task", "pause", "TASK-001"}, {"task", "resume", "TASK-001"}} {
		cmd := exec.Command(bin, args...)
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected %v to fail as unknown command", args)
		}
		if !strings.Contains(string(out), "unknown command") {
			t.Fatalf("expected unknown command for %v, got:\n%s", args, out)
		}
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate module root from test working directory")
		}
		dir = parent
	}
}

func buildEnv(root string) []string {
	return append(os.Environ(),
		"CGO_ENABLED=0",
		"GOCACHE="+filepath.Join(root, ".gocache"),
	)
}
