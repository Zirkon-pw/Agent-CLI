package review

import (
	"encoding/json"
	. "github.com/docup/agentctl/internal/core/review"
	"testing"
	"time"
)

func TestDecisionConstants(t *testing.T) {
	decisions := []Decision{DecisionAccepted, DecisionRejected, DecisionNeedsChanges, DecisionReroute}
	expected := []string{"accepted", "rejected", "needs_changes", "reroute"}
	for i, d := range decisions {
		if string(d) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], d)
		}
	}
}

func TestReview_JSONRoundtrip(t *testing.T) {
	r := Review{
		TaskID:    "TASK-001",
		RunID:     "RUN-001",
		Decision:  DecisionAccepted,
		Reason:    "looks good",
		Reviewer:  "human",
		CreatedAt: time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Review
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.Decision != DecisionAccepted {
		t.Errorf("wrong decision: %s", loaded.Decision)
	}
	if loaded.Reason != "looks good" {
		t.Errorf("wrong reason: %s", loaded.Reason)
	}
	if loaded.Reviewer != "human" {
		t.Errorf("wrong reviewer: %s", loaded.Reviewer)
	}
}
