package builtin_templates

import "github.com/docup/agentctl/internal/core/template"

// All returns all built-in prompt templates.
func All() []*template.PromptTemplate {
	return []*template.PromptTemplate{
		ClarifyIfNeeded(),
		PlanBeforeExecution(),
		StrictExecutor(),
		ResearchOnly(),
		ReviewOnly(),
	}
}

// ByID returns a built-in template by its ID, or nil if not found.
func ByID(id string) *template.PromptTemplate {
	for _, t := range All() {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func ClarifyIfNeeded() *template.PromptTemplate {
	return &template.PromptTemplate{
		ID:          "clarify_if_needed",
		Name:        "Clarify If Needed",
		Description: "Ask blocking questions instead of guessing. If the task has ambiguities, create a clarification request YAML instead of making assumptions.",
		IsBuiltin:   true,
		Behavior: template.Behavior{
			ClarificationIfAmbiguous:    true,
			AllowNonBlockingAssumptions: false,
			CodeChangesAllowed:          true,
		},
		IncompatibleWith: []string{},
	}
}

func PlanBeforeExecution() *template.PromptTemplate {
	return &template.PromptTemplate{
		ID:          "plan_before_execution",
		Name:        "Plan Before Execution",
		Description: "Build a detailed plan before making any code changes. Output the plan as part of the run artifacts.",
		IsBuiltin:   true,
		Behavior: template.Behavior{
			PlanBeforeExecution: true,
			CodeChangesAllowed:  true,
		},
		IncompatibleWith: []string{"research_only", "review_only"},
	}
}

func StrictExecutor() *template.PromptTemplate {
	return &template.PromptTemplate{
		ID:          "strict_executor",
		Name:        "Strict Executor",
		Description: "Work without assumptions, follow scope strictly. Only modify files within allowed paths, never touch forbidden paths.",
		IsBuiltin:   true,
		Behavior: template.Behavior{
			RequireExplicitScope:        true,
			AllowNonBlockingAssumptions: false,
			CodeChangesAllowed:          true,
		},
		IncompatibleWith: []string{"research_only", "review_only"},
	}
}

func ResearchOnly() *template.PromptTemplate {
	return &template.PromptTemplate{
		ID:          "research_only",
		Name:        "Research Only",
		Description: "Analyze and propose, no code changes. Output analysis, recommendations, and findings as artifacts.",
		IsBuiltin:   true,
		Behavior: template.Behavior{
			ResearchOnly:       true,
			CodeChangesAllowed: false,
		},
		IncompatibleWith: []string{"strict_executor", "plan_before_execution", "review_only"},
	}
}

func ReviewOnly() *template.PromptTemplate {
	return &template.PromptTemplate{
		ID:          "review_only",
		Name:        "Review Only",
		Description: "Independent review of completed work. Analyze diff, summary, validation report and provide notes, risks, violations, recommendations.",
		IsBuiltin:   true,
		Behavior: template.Behavior{
			ReviewMode:         true,
			CodeChangesAllowed: false,
		},
		IncompatibleWith: []string{"strict_executor", "plan_before_execution", "research_only"},
	}
}
