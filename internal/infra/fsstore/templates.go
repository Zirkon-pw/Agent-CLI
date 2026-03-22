package fsstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docup/agentctl/internal/core/template"
	"gopkg.in/yaml.v3"
)

// TemplateStore handles custom prompt template files.
type TemplateStore struct {
	baseDir string // .agentctl/templates/prompts
}

// NewTemplateStore creates a new TemplateStore.
func NewTemplateStore(agentctlDir string) *TemplateStore {
	return &TemplateStore{baseDir: filepath.Join(agentctlDir, "templates", "prompts")}
}

// Save writes a custom template to disk.
func (s *TemplateStore) Save(t *template.PromptTemplate) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshaling template: %w", err)
	}
	path := filepath.Join(s.baseDir, t.ID+".yml")
	return os.WriteFile(path, data, 0644)
}

// Load reads a custom template from disk.
func (s *TemplateStore) Load(id string) (*template.PromptTemplate, error) {
	path := filepath.Join(s.baseDir, id+".yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", id, err)
	}
	var t template.PromptTemplate
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// List returns all custom templates.
func (s *TemplateStore) List() ([]*template.PromptTemplate, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var templates []*template.PromptTemplate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".yml")
		t, err := s.Load(id)
		if err != nil {
			continue
		}
		templates = append(templates, t)
	}
	return templates, nil
}
