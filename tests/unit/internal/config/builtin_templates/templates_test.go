package builtin_templates

import (
	. "github.com/docup/agentctl/internal/config/builtin_templates"
	"testing"
)

func TestAll_Returns5(t *testing.T) {
	all := All()
	if len(all) != 5 {
		t.Fatalf("expected 5 builtin templates, got %d", len(all))
	}
}

func TestAll_UniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, tmpl := range All() {
		if seen[tmpl.ID] {
			t.Errorf("duplicate template ID: %s", tmpl.ID)
		}
		seen[tmpl.ID] = true
	}
}

func TestAll_AllBuiltin(t *testing.T) {
	for _, tmpl := range All() {
		if !tmpl.IsBuiltin {
			t.Errorf("template %s should be builtin", tmpl.ID)
		}
	}
}

func TestByID_Found(t *testing.T) {
	ids := []string{"clarify_if_needed", "plan_before_execution", "strict_executor", "research_only", "review_only"}
	for _, id := range ids {
		tmpl := ByID(id)
		if tmpl == nil {
			t.Errorf("template %s not found", id)
		}
		if tmpl != nil && tmpl.ID != id {
			t.Errorf("expected ID %s, got %s", id, tmpl.ID)
		}
	}
}

func TestByID_NotFound(t *testing.T) {
	if ByID("nonexistent") != nil {
		t.Error("should return nil for unknown ID")
	}
}

func TestIncompatibilities(t *testing.T) {
	plan := ByID("plan_before_execution")
	strict := ByID("strict_executor")
	research := ByID("research_only")
	review := ByID("review_only")
	clarify := ByID("clarify_if_needed")

	// Compatible pairs
	if !plan.IsCompatibleWith(strict) {
		t.Error("plan + strict should be compatible")
	}
	if !plan.IsCompatibleWith(clarify) {
		t.Error("plan + clarify should be compatible")
	}
	if !strict.IsCompatibleWith(clarify) {
		t.Error("strict + clarify should be compatible")
	}

	// Incompatible pairs
	if plan.IsCompatibleWith(research) {
		t.Error("plan + research should be incompatible")
	}
	if plan.IsCompatibleWith(review) {
		t.Error("plan + review should be incompatible")
	}
	if strict.IsCompatibleWith(research) {
		t.Error("strict + research should be incompatible")
	}
	if strict.IsCompatibleWith(review) {
		t.Error("strict + review should be incompatible")
	}
	if research.IsCompatibleWith(review) {
		t.Error("research + review should be incompatible")
	}
}

func TestClarifyIfNeeded_Behavior(t *testing.T) {
	tmpl := ClarifyIfNeeded()
	if !tmpl.Behavior.ClarificationIfAmbiguous {
		t.Error("ClarificationIfAmbiguous should be true")
	}
	if tmpl.Behavior.AllowNonBlockingAssumptions {
		t.Error("AllowNonBlockingAssumptions should be false")
	}
	if !tmpl.Behavior.CodeChangesAllowed {
		t.Error("CodeChangesAllowed should be true")
	}
}

func TestResearchOnly_Behavior(t *testing.T) {
	tmpl := ResearchOnly()
	if !tmpl.Behavior.ResearchOnly {
		t.Error("ResearchOnly should be true")
	}
	if tmpl.Behavior.CodeChangesAllowed {
		t.Error("CodeChangesAllowed should be false")
	}
}

func TestReviewOnly_Behavior(t *testing.T) {
	tmpl := ReviewOnly()
	if !tmpl.Behavior.ReviewMode {
		t.Error("ReviewMode should be true")
	}
	if tmpl.Behavior.CodeChangesAllowed {
		t.Error("CodeChangesAllowed should be false")
	}
}
