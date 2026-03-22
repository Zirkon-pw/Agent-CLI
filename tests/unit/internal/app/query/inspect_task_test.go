package query

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/docup/agentctl/internal/app/query"
	rt "github.com/docup/agentctl/internal/core/runtime"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

func setupInspectStores(t *testing.T) (*fsstore.TaskStore, *fsstore.RunStore) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".agentctl")
	_ = os.MkdirAll(filepath.Join(dir, "tasks"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "runs"), 0755)
	return fsstore.NewTaskStore(dir), fsstore.NewRunStore(dir)
}

func TestInspectTask_Found(t *testing.T) {
	store, runStore := setupInspectStores(t)
	now := time.Now()

	_ = store.Save(&task.Task{
		ID:     "TASK-001",
		Title:  "Inspect me",
		Goal:   "Test goal",
		Status: task.StatusRunning,
		Agent:  "claude",
		PromptTemplates: task.PromptTemplates{
			Builtin: []string{"strict_executor", "clarify_if_needed"},
			Custom:  []string{"my_custom"},
		},
		Guidelines: []string{"backend-guidelines"},
		Scope: task.Scope{
			AllowedPaths:   []string{"src/"},
			ForbiddenPaths: []string{"vendor/"},
		},
		Validation: task.ValidationConfig{
			Mode:       task.ValidationModeFull,
			MaxRetries: 5,
			Commands:   []string{"go test"},
		},
		Runtime:   task.DefaultRuntimeConfig(),
		CreatedAt: now,
		UpdatedAt: now,
	})

	session := &rt.RunSession{
		ID:             "RUN-001",
		TaskID:         "TASK-001",
		Status:         rt.SessionStatusFailed,
		CurrentAgentID: "claude",
		ArtifactManifest: rt.ArtifactManifest{
			Items: []rt.ArtifactRecord{{Name: "runtime_errors.log"}},
		},
		StageHistory: []rt.StageRun{
			{
				StageID: "STAGE-002",
				Type:    rt.StageTypeReview,
				State:   rt.StageStateFailed,
				Result:  &rt.StageResult{Outcome: "failed", Message: "review unsupported"},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := runStore.SaveSession(session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	q := NewInspectTask(store, runStore)
	detail, err := q.Execute("TASK-001")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}

	if detail.ID != "TASK-001" {
		t.Errorf("wrong ID: %s", detail.ID)
	}
	if detail.Status != "stage_running" {
		t.Errorf("wrong status: %s", detail.Status)
	}
	if len(detail.Templates) != 3 {
		t.Errorf("expected 3 templates (2 builtin + 1 custom), got %d", len(detail.Templates))
	}
	if detail.Validation.Mode != "full" {
		t.Errorf("expected full mode, got %s", detail.Validation.Mode)
	}
	if detail.Validation.MaxRetries != 5 {
		t.Errorf("expected 5 retries, got %d", detail.Validation.MaxRetries)
	}
	if len(detail.Scope.AllowedPaths) != 1 {
		t.Error("expected 1 allowed path")
	}
	if detail.LatestSession == nil {
		t.Fatal("expected latest session details")
	}
	if detail.LatestSession.LastError != "review unsupported" {
		t.Fatalf("expected latest session error, got %q", detail.LatestSession.LastError)
	}
}

func TestInspectTask_NotFound(t *testing.T) {
	store, runStore := setupInspectStores(t)
	q := NewInspectTask(store, runStore)
	_, err := q.Execute("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error")
	}
}
