package template

// PromptTemplate defines a behavioral template applied to task execution.
type PromptTemplate struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	IsBuiltin   bool     `yaml:"is_builtin"`
	FilePath    string   `yaml:"file_path,omitempty"`

	Behavior Behavior `yaml:"behavior"`

	// IncompatibleWith lists template IDs that cannot be combined with this one.
	IncompatibleWith []string `yaml:"incompatible_with"`
}

// Behavior defines the execution rules a template imposes.
type Behavior struct {
	RequireExplicitScope       bool `yaml:"require_explicit_scope"`
	ClarificationIfAmbiguous   bool `yaml:"clarification_if_ambiguous"`
	AllowNonBlockingAssumptions bool `yaml:"allow_non_blocking_assumptions"`
	PlanBeforeExecution        bool `yaml:"plan_before_execution"`
	CodeChangesAllowed         bool `yaml:"code_changes_allowed"`
	ReviewMode                 bool `yaml:"review_mode"`
	ResearchOnly               bool `yaml:"research_only"`
}

// IsCompatibleWith checks if two templates can be used together.
func (t *PromptTemplate) IsCompatibleWith(other *PromptTemplate) bool {
	for _, id := range t.IncompatibleWith {
		if id == other.ID {
			return false
		}
	}
	for _, id := range other.IncompatibleWith {
		if id == t.ID {
			return false
		}
	}
	return true
}
