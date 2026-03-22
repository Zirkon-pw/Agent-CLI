package runtime

import (
	. "github.com/docup/agentctl/internal/core/runtime"
	"testing"
	"time"
)

func TestHeartbeat_IsStale(t *testing.T) {
	hb := &Heartbeat{
		TaskID:    "TASK-001",
		RunID:     "RUN-001",
		Timestamp: time.Now().Add(-60 * time.Second),
		Alive:     true,
	}

	if !hb.IsStale(30 * time.Second) {
		t.Error("heartbeat 60s old should be stale with 30s threshold")
	}
	if hb.IsStale(120 * time.Second) {
		t.Error("heartbeat 60s old should NOT be stale with 120s threshold")
	}
}

func TestHeartbeat_IsStale_Fresh(t *testing.T) {
	hb := &Heartbeat{
		Timestamp: time.Now(),
		Alive:     true,
	}
	if hb.IsStale(5 * time.Second) {
		t.Error("fresh heartbeat should not be stale")
	}
}
