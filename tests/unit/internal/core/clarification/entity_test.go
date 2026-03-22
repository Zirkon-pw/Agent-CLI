package clarification

import (
	. "github.com/docup/agentctl/internal/core/clarification"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestRequest_YAMLRoundtrip(t *testing.T) {
	req := &Request{
		TaskID:    "TASK-001",
		RequestID: "CLAR-REQ-001",
		CreatedBy: "claude",
		Reason:    "ambiguous scope",
		Questions: []Question{
			{ID: "q1", Text: "What about X?"},
			{ID: "q2", Text: "Should we include Y?"},
		},
		ContextRefs: []string{"src/auth.go", "docs/spec.md"},
		CreatedAt:   time.Now().Truncate(time.Second),
	}

	data, err := yaml.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Request
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.TaskID != req.TaskID {
		t.Errorf("TaskID: expected %s, got %s", req.TaskID, loaded.TaskID)
	}
	if loaded.RequestID != req.RequestID {
		t.Errorf("RequestID: expected %s, got %s", req.RequestID, loaded.RequestID)
	}
	if len(loaded.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(loaded.Questions))
	}
	if loaded.Questions[0].ID != "q1" || loaded.Questions[1].Text != "Should we include Y?" {
		t.Error("questions not preserved correctly")
	}
	if len(loaded.ContextRefs) != 2 {
		t.Error("context refs not preserved")
	}
}

func TestClarification_YAMLRoundtrip(t *testing.T) {
	clar := &Clarification{
		TaskID:          "TASK-001",
		RequestID:       "CLAR-REQ-001",
		ClarificationID: "CLAR-001",
		Answers: []Answer{
			{QuestionID: "q1", Text: "Single use tokens"},
			{QuestionID: "q2", Text: "Yes include Y"},
		},
		Notes:     []string{"Keep API unchanged", "No migration needed"},
		CreatedAt: time.Now().Truncate(time.Second),
	}

	data, err := yaml.Marshal(clar)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Clarification
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.ClarificationID != "CLAR-001" {
		t.Errorf("wrong ID: %s", loaded.ClarificationID)
	}
	if len(loaded.Answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(loaded.Answers))
	}
	if loaded.Answers[0].Text != "Single use tokens" {
		t.Errorf("wrong answer text: %s", loaded.Answers[0].Text)
	}
	if len(loaded.Notes) != 2 {
		t.Error("notes not preserved")
	}
}

func TestQuestion_Fields(t *testing.T) {
	q := Question{ID: "q1", Text: "test"}
	if q.ID != "q1" || q.Text != "test" {
		t.Error("fields not set")
	}
}
