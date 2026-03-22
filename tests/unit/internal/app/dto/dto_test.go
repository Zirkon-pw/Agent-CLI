package dto

import (
	. "github.com/docup/agentctl/internal/app/dto"
	"testing"
	"time"
)

func TestCreateTaskRequest_Fields(t *testing.T) {
	req := CreateTaskRequest{
		Title:      "Test",
		Goal:       "Goal",
		Agent:      "claude",
		Templates:  []string{"strict_executor"},
		Guidelines: []string{"backend"},
		Scope: ScopeDTO{
			AllowedPaths:   []string{"src/"},
			ForbiddenPaths: []string{"vendor/"},
			MustRead:       []string{"README.md"},
		},
	}
	if req.Title != "Test" {
		t.Error("title not set")
	}
	if len(req.Scope.AllowedPaths) != 1 {
		t.Error("allowed paths not set")
	}
}

func TestUpdateTaskRequest_Fields(t *testing.T) {
	req := UpdateTaskRequest{
		TaskID:          "TASK-001",
		AddTemplates:    []string{"a"},
		RemoveTemplates: []string{"b"},
		AddGuidelines:   []string{"c"},
	}
	if req.TaskID != "TASK-001" {
		t.Error("task ID not set")
	}
}

func TestTaskSummary_Fields(t *testing.T) {
	s := TaskSummary{
		ID:        "TASK-001",
		Title:     "Title",
		Status:    "draft",
		Agent:     "claude",
		CreatedAt: time.Now(),
	}
	if s.ID != "TASK-001" {
		t.Error("ID not set")
	}
}

func TestTaskDetail_Fields(t *testing.T) {
	d := TaskDetail{
		ID:             "TASK-001",
		Title:          "Title",
		Goal:           "Goal",
		Status:         "running",
		Agent:          "claude",
		Templates:      []string{"strict_executor"},
		Guidelines:     []string{"backend"},
		Validation:     ValidationDTO{Mode: "full", MaxRetries: 3, Commands: []string{"go test"}},
		Runtime:        RuntimeDTO{MaxExecutionMinutes: 45, AllowPause: true},
		Clarifications: ClarificationsDTO{Attached: []string{"/path"}},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if d.Validation.Mode != "full" {
		t.Error("validation mode not set")
	}
	if d.Runtime.MaxExecutionMinutes != 45 {
		t.Error("runtime not set")
	}
	if len(d.Clarifications.Attached) != 1 {
		t.Error("clarifications not set")
	}
}

func TestRunSummary_Fields(t *testing.T) {
	s := RunSummary{
		ID:        "RUN-001",
		TaskID:    "TASK-001",
		Status:    "success",
		Agent:     "claude",
		Duration:  "2m30s",
		CreatedAt: time.Now(),
	}
	if s.Duration != "2m30s" {
		t.Error("duration not set")
	}
}

func TestActiveRunDTO_Fields(t *testing.T) {
	a := ActiveRunDTO{
		TaskID:    "TASK-001",
		RunID:     "RUN-001",
		Agent:     "codex",
		StartedAt: time.Now(),
		Duration:  "1m",
	}
	if a.Agent != "codex" {
		t.Error("agent not set")
	}
}

func TestRouteTaskRequest_Fields(t *testing.T) {
	r := RouteTaskRequest{TaskID: "TASK-001", Agent: "qwen"}
	if r.Agent != "qwen" {
		t.Error("agent not set")
	}
}

func TestClarificationRequests_Fields(t *testing.T) {
	g := ClarificationGenerateRequest{TaskID: "TASK-001", Reason: "ambiguous"}
	if g.Reason != "ambiguous" {
		t.Error("reason not set")
	}
	a := ClarificationAttachRequest{TaskID: "TASK-001", Path: "/path"}
	if a.Path != "/path" {
		t.Error("path not set")
	}
}
