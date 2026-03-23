package fsstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/docup/agentctl/internal/core/run"
	rt "github.com/docup/agentctl/internal/core/runtime"
)

// SaveSession writes the full session state to disk.
func (s *RunStore) SaveSession(session *rt.RunSession) error {
	dir := s.RunDir(session.TaskID, session.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating session dir: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.json"), data, 0644); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	return s.Save(sessionToRunSummary(session))
}

// LoadSession reads a persisted session from disk.
func (s *RunStore) LoadSession(taskID, sessionID string) (*rt.RunSession, error) {
	data, err := os.ReadFile(filepath.Join(s.RunDir(taskID, sessionID), "session.json"))
	if err != nil {
		return nil, fmt.Errorf("reading session %s/%s: %w", taskID, sessionID, err)
	}
	var session rt.RunSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing session: %w", err)
	}
	return &session, nil
}

// LatestSession returns the most recent session for a task.
func (s *RunStore) LatestSession(taskID string) (*rt.RunSession, error) {
	taskDir := filepath.Join(s.baseDir, taskID)
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no sessions found for task %s", taskID)
		}
		return nil, err
	}

	var sessions []*rt.RunSession
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		session, err := s.LoadSession(taskID, entry.Name())
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found for task %s", taskID)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})
	return sessions[0], nil
}

// StageDir returns the path for a stage inside a session.
func (s *RunStore) StageDir(taskID, sessionID, stageID string) string {
	return filepath.Join(s.RunDir(taskID, sessionID), "stages", stageID)
}

// SaveArtifactManifest writes the normalized artifact manifest for a session.
func (s *RunStore) SaveArtifactManifest(taskID, sessionID string, manifest *rt.ArtifactManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling artifact manifest: %w", err)
	}
	return s.WriteArtifact(taskID, sessionID, "artifacts.json", data)
}

// LoadArtifactManifest reads the artifact manifest for a session.
func (s *RunStore) LoadArtifactManifest(taskID, sessionID string) (*rt.ArtifactManifest, error) {
	data, err := s.ReadArtifact(taskID, sessionID, "artifacts.json")
	if err != nil {
		return nil, err
	}
	var manifest rt.ArtifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing artifact manifest: %w", err)
	}
	return &manifest, nil
}

func sessionToRunSummary(session *rt.RunSession) *run.Run {
	summary := &run.Run{
		ID:        session.ID,
		TaskID:    session.TaskID,
		Agent:     session.CurrentAgentID,
		CreatedAt: session.CreatedAt,
		PID:       session.Recovery.AdapterPID,
	}
	if stage := session.LastStage(); stage != nil {
		if stage.Result != nil && stage.Result.ExitCode != nil {
			summary.ExitCode = stage.Result.ExitCode
		}
	}
	if session.ReviewReport != nil {
		summary.Status = run.RunStatusSuccess
	} else {
		switch session.Status {
		case rt.SessionStatusQueued, rt.SessionStatusStageRunning, rt.SessionStatusWaitingClarification, rt.SessionStatusHandoffPending:
			summary.Status = run.RunStatusRunning
		case rt.SessionStatusCompleted, rt.SessionStatusReviewing:
			summary.Status = run.RunStatusSuccess
		case rt.SessionStatusCanceled:
			summary.Status = run.RunStatusStopped
		default:
			summary.Status = run.RunStatusFailed
		}
	}
	if len(session.StageHistory) > 0 {
		first := session.StageHistory[0]
		summary.StartedAt = first.StartedAt
		last := session.StageHistory[len(session.StageHistory)-1]
		summary.FinishedAt = last.FinishedAt
	}
	return summary
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
