package clarificationflow

import (
	"fmt"
	"time"

	"github.com/docup/agentctl/internal/core/clarification"
	"github.com/docup/agentctl/internal/core/task"
	"github.com/docup/agentctl/internal/infra/fsstore"
)

// Manager handles the clarification request/response flow.
type Manager struct {
	taskStore *fsstore.TaskStore
	clarStore *fsstore.ClarificationStore
}

// NewManager creates a clarification flow manager.
func NewManager(taskStore *fsstore.TaskStore, clarStore *fsstore.ClarificationStore) *Manager {
	return &Manager{taskStore: taskStore, clarStore: clarStore}
}

// GenerateRequest creates a clarification request for a task.
func (m *Manager) GenerateRequest(taskID string, questions []clarification.Question, reason string) (*clarification.Request, string, error) {
	t, err := m.taskStore.Load(taskID)
	if err != nil {
		return nil, "", err
	}

	if t.Status != task.StatusNeedsClarification && t.Status != task.StatusDraft {
		return nil, "", fmt.Errorf("cannot generate clarification for task in status %s", t.Status)
	}

	reqID := fmt.Sprintf("CLAR-REQ-%03d", len(t.Clarifications.Attached)+1)

	req := &clarification.Request{
		TaskID:    taskID,
		RequestID: reqID,
		CreatedBy: t.Agent,
		Reason:    reason,
		Questions: questions,
		CreatedAt: time.Now(),
	}

	path, err := m.clarStore.SaveRequest(req)
	if err != nil {
		return nil, "", err
	}

	t.SetPendingClarification(reqID)
	if err := m.taskStore.Save(t); err != nil {
		return nil, "", err
	}

	return req, path, nil
}

// AttachClarification attaches a user-filled clarification to a task.
func (m *Manager) AttachClarification(taskID, clarificationPath string) error {
	t, err := m.taskStore.Load(taskID)
	if err != nil {
		return err
	}

	if t.Clarifications.PendingRequest == nil {
		return fmt.Errorf("no pending clarification request for task %s", taskID)
	}

	// Validate the clarification file exists and parses
	clar, err := m.clarStore.LoadClarification(clarificationPath)
	if err != nil {
		return fmt.Errorf("loading clarification file: %w", err)
	}

	if clar.TaskID != taskID {
		return fmt.Errorf("clarification task_id mismatch: expected %s, got %s", taskID, clar.TaskID)
	}

	t.AddClarification(clarificationPath)

	if t.Status == task.StatusNeedsClarification {
		if err := t.TransitionTo(task.StatusReadyToResume); err != nil {
			return err
		}
	}

	return m.taskStore.Save(t)
}

// ShowPending returns the pending clarification request for a task.
func (m *Manager) ShowPending(taskID string) (*clarification.Request, error) {
	t, err := m.taskStore.Load(taskID)
	if err != nil {
		return nil, err
	}

	if t.Clarifications.PendingRequest == nil {
		return nil, fmt.Errorf("no pending clarification request for task %s", taskID)
	}

	return m.clarStore.LoadRequest(taskID, *t.Clarifications.PendingRequest)
}
