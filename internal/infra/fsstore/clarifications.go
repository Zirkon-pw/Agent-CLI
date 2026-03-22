package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docup/agentctl/internal/core/clarification"
	"gopkg.in/yaml.v3"
)

// ClarificationStore handles clarification YAML files.
type ClarificationStore struct {
	baseDir string // .agentctl/clarifications
}

// NewClarificationStore creates a new ClarificationStore.
func NewClarificationStore(agentctlDir string) *ClarificationStore {
	return &ClarificationStore{baseDir: filepath.Join(agentctlDir, "clarifications")}
}

// SaveRequest writes a clarification request to disk.
func (s *ClarificationStore) SaveRequest(req *clarification.Request) (string, error) {
	dir := filepath.Join(s.baseDir, req.TaskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating clarification dir: %w", err)
	}
	data, err := yaml.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshaling clarification request: %w", err)
	}
	filename := fmt.Sprintf("clarification_request_%s.yml", req.RequestID)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// SaveClarification writes a user-filled clarification to disk.
func (s *ClarificationStore) SaveClarification(clar *clarification.Clarification) (string, error) {
	dir := filepath.Join(s.baseDir, clar.TaskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(clar)
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("clarification_%s.yml", clar.ClarificationID)
	path := filepath.Join(dir, filename)
	return path, os.WriteFile(path, data, 0644)
}

// LoadRequest reads a clarification request.
func (s *ClarificationStore) LoadRequest(taskID, requestID string) (*clarification.Request, error) {
	dir := filepath.Join(s.baseDir, taskID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading clarification dir for %s: %w", taskID, err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), requestID) && strings.HasPrefix(entry.Name(), "clarification_request_") {
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}
			var req clarification.Request
			if err := yaml.Unmarshal(data, &req); err != nil {
				return nil, err
			}
			return &req, nil
		}
	}
	return nil, fmt.Errorf("clarification request %s not found for task %s", requestID, taskID)
}

// LoadClarification reads a user-filled clarification file.
func (s *ClarificationStore) LoadClarification(path string) (*clarification.Clarification, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var clar clarification.Clarification
	if err := yaml.Unmarshal(data, &clar); err != nil {
		return nil, err
	}
	return &clar, nil
}

// ListClarifications returns all attached clarifications for a task.
func (s *ClarificationStore) ListClarifications(taskID string) ([]*clarification.Clarification, error) {
	dir := filepath.Join(s.baseDir, taskID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []*clarification.Clarification
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "clarification_") && !strings.HasPrefix(entry.Name(), "clarification_request_") {
			path := filepath.Join(dir, entry.Name())
			clar, err := s.LoadClarification(path)
			if err != nil {
				continue
			}
			result = append(result, clar)
		}
	}
	return result, nil
}
