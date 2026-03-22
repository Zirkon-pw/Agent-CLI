package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docup/agentctl/internal/core/runtime"
)

// Sink writes events to an NDJSON file.
type Sink struct {
	baseDir string // .agentctl/runtime or .agentctl/runs
}

// NewSink creates an event sink.
func NewSink(baseDir string) *Sink {
	return &Sink{baseDir: baseDir}
}

// Emit writes an event to the events.ndjson file for a task.
func (s *Sink) Emit(taskID, runID, eventType, details string) error {
	event := runtime.Event{
		Timestamp: time.Now(),
		TaskID:    taskID,
		RunID:     runID,
		EventType: eventType,
		Details:   details,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	dir := filepath.Join(s.baseDir, taskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, "events.ndjson")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// Read reads all events for a task.
func (s *Sink) Read(taskID string) ([]runtime.Event, error) {
	path := filepath.Join(s.baseDir, taskID, "events.ndjson")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var events []runtime.Event
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var ev runtime.Event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

// Tail returns the last N events for a task.
func (s *Sink) Tail(taskID string, n int) ([]runtime.Event, error) {
	events, err := s.Read(taskID)
	if err != nil {
		return nil, err
	}
	if len(events) <= n {
		return events, nil
	}
	return events[len(events)-n:], nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
