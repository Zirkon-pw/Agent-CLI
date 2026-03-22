package template

import (
	. "github.com/docup/agentctl/internal/core/template"
	"testing"
)

func TestIsCompatibleWith_Compatible(t *testing.T) {
	a := &PromptTemplate{ID: "a", IncompatibleWith: []string{"c"}}
	b := &PromptTemplate{ID: "b", IncompatibleWith: []string{"c"}}

	if !a.IsCompatibleWith(b) {
		t.Error("a and b should be compatible")
	}
}

func TestIsCompatibleWith_Incompatible_Forward(t *testing.T) {
	a := &PromptTemplate{ID: "a", IncompatibleWith: []string{"b"}}
	b := &PromptTemplate{ID: "b", IncompatibleWith: []string{}}

	if a.IsCompatibleWith(b) {
		t.Error("a should be incompatible with b")
	}
}

func TestIsCompatibleWith_Incompatible_Reverse(t *testing.T) {
	a := &PromptTemplate{ID: "a", IncompatibleWith: []string{}}
	b := &PromptTemplate{ID: "b", IncompatibleWith: []string{"a"}}

	if a.IsCompatibleWith(b) {
		t.Error("a should be incompatible with b (reverse)")
	}
}

func TestIsCompatibleWith_EmptyLists(t *testing.T) {
	a := &PromptTemplate{ID: "a"}
	b := &PromptTemplate{ID: "b"}

	if !a.IsCompatibleWith(b) {
		t.Error("empty incompatible lists should be compatible")
	}
}

func TestIsCompatibleWith_SelfCheck(t *testing.T) {
	a := &PromptTemplate{ID: "a", IncompatibleWith: []string{"a"}}
	if a.IsCompatibleWith(a) {
		t.Error("self-incompatibility should be detected")
	}
}
