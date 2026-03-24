package result

import (
	. "github.com/docup/agentctl/internal/cli/result"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rt "github.com/docup/agentctl/internal/core/runtime"
	"github.com/docup/agentctl/internal/infra/fsstore"
	"github.com/docup/agentctl/tests/support/testio"
)

func TestResultCmd_Structure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agentctl")
	os.MkdirAll(filepath.Join(dir, "runs"), 0755)
	store := fsstore.NewRunStore(dir)

	cmd := NewResultCmd(store)
	if cmd.Use != "result" {
		t.Errorf("expected use 'result', got %q", cmd.Use)
	}

	subs := cmd.Commands()
	names := make(map[string]bool)
	for _, sub := range subs {
		names[sub.Name()] = true
	}
	for _, expected := range []string{"show", "diff", "list"} {
		if !names[expected] {
			t.Errorf("missing subcommand: %s", expected)
		}
	}
}

func setupResultStore(t *testing.T) *fsstore.RunStore {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".agentctl")
	if err := os.MkdirAll(filepath.Join(dir, "runs"), 0755); err != nil {
		t.Fatalf("mkdir runs: %v", err)
	}
	return fsstore.NewRunStore(dir)
}

func saveSession(t *testing.T, store *fsstore.RunStore, session *rt.RunSession) {
	t.Helper()
	if err := store.SaveSession(session); err != nil {
		t.Fatalf("save session: %v", err)
	}
}

func TestResultDiff_UsesManifestArtifact(t *testing.T) {
	store := setupResultStore(t)
	now := time.Now()
	diffPath := filepath.Join(store.RunDir("TASK-001", "RUN-001"), "custom-diff.patch")
	if err := os.MkdirAll(filepath.Dir(diffPath), 0755); err != nil {
		t.Fatalf("mkdir diff dir: %v", err)
	}
	if err := os.WriteFile(diffPath, []byte("manifest diff\n"), 0644); err != nil {
		t.Fatalf("write diff: %v", err)
	}

	saveSession(t, store, &rt.RunSession{
		ID:             "RUN-001",
		TaskID:         "TASK-001",
		Status:         rt.SessionStatusReviewing,
		CurrentAgentID: "qwen",
		ArtifactManifest: rt.ArtifactManifest{
			Items: []rt.ArtifactRecord{{
				Name:      "diff.patch",
				Kind:      "diff",
				Path:      diffPath,
				CreatedAt: now,
			}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	cmd := NewResultCmd(store)
	cmd.SetArgs([]string{"diff", "TASK-001"})
	output := testio.CaptureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if !strings.Contains(output, "manifest diff") {
		t.Fatalf("expected manifest diff output, got %q", output)
	}
}

func TestResultDiff_FallsBackToRunDiffPatch(t *testing.T) {
	store := setupResultStore(t)
	now := time.Now()
	if err := store.WriteArtifact("TASK-001", "RUN-001", "diff.patch", []byte("fallback diff\n")); err != nil {
		t.Fatalf("write fallback diff: %v", err)
	}

	saveSession(t, store, &rt.RunSession{
		ID:             "RUN-001",
		TaskID:         "TASK-001",
		Status:         rt.SessionStatusReviewing,
		CurrentAgentID: "qwen",
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	cmd := NewResultCmd(store)
	cmd.SetArgs([]string{"diff", "TASK-001"})
	output := testio.CaptureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if !strings.Contains(output, "fallback diff") {
		t.Fatalf("expected fallback diff output, got %q", output)
	}
}
