package directive

import (
	"errors"
	"strings"
	"testing"
)

func TestCompileDirective_Success(t *testing.T) {
	input := CompilerInput{
		ProjectID:          "proj-test",
		GoalID:             "goal-1",
		TaskID:             "task-1",
		TaskIntent:         "Implement feature X",
		Decisions:          []string{"DEC-1: Use Go Core"},
		Requirements:       []string{"REQ-1: Clean Code"},
		Constraints:        []string{"CONST-1: No YAML generation"},
		InScope:            []string{"internal/feature"},
		OutOfScope:         []string{"internal/legacy"},
		AllowedPaths:       []string{"internal/feature/x.go"},
		ForbiddenPaths:     []string{"internal/legacy/*"},
		AllowedActions:     []string{"file_write", "file_read"},
		ForbiddenActions:   []string{"git_force_push"},
		HardLimitCents:     500,
		ReviewReserveCents: 50,
		Currency:           "BRL",
		RequiredEvidence:   []string{"go test passed"},
		QualityGates:       []string{"ci-gate"},
		StopConditions:     []string{"budget_exhausted"},
		ToolCapabilities: []ToolCapability{
			{Name: "file_write", SideEffectClass: "stateful_mutation"},
		},
	}

	ir, err := CompileDirective(input)
	if err != nil {
		t.Fatalf("unexpected compilation error: %v", err)
	}

	if ir.DirectiveIRVersion != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", ir.DirectiveIRVersion)
	}
	if ir.TaskIntent != "Implement feature X" {
		t.Errorf("expected intent 'Implement feature X', got %s", ir.TaskIntent)
	}
	if ir.SourceDigests["intent_digest"] == "" {
		t.Error("expected non-empty intent digest")
	}

	// Test path mutation rules
	if !ir.CanMutatePath("internal/feature/x.go") {
		t.Error("expected internal/feature/x.go to be mutable")
	}
	if ir.CanMutatePath("internal/legacy/old.go") {
		t.Error("expected internal/legacy/old.go to be forbidden")
	}
	if ir.CanMutatePath("external/other.go") {
		t.Error("expected unlisted path external/other.go to be rejected when AllowedPaths is set")
	}

	// Test prompt formatting
	prompt := ir.FormatAgentPrompt()
	if !strings.Contains(prompt, "=== 1. HARD POLICIES ===") {
		t.Error("prompt missing tier 1")
	}
	if !strings.Contains(prompt, "=== 4. SCOPE & ACCEPTANCE ===") {
		t.Error("prompt missing tier 4")
	}
	if !strings.Contains(prompt, "DEC-1: Use Go Core") {
		t.Error("prompt missing decisions/invariants")
	}
}

func TestCompileDirective_MissingAuthority(t *testing.T) {
	input := CompilerInput{
		GoalID:        "goal-1",
		TaskID:        "task-1",
		TaskIntent:    "Delete canonical database schema",
		IsDestructive: true,
		HasAuthority:  false, // No authority!
	}

	_, err := CompileDirective(input)
	if err == nil {
		t.Fatal("expected error for missing authority on destructive task")
	}
	if !errors.Is(err, ErrMissingAuthority) {
		t.Errorf("expected ErrMissingAuthority, got: %v", err)
	}
}

func TestCompileDirective_ScopeFirewallOverlap(t *testing.T) {
	input := CompilerInput{
		GoalID:     "goal-1",
		TaskID:     "task-1",
		TaskIntent: "Refactor core",
		InScope:    []string{"internal/core"},
		OutOfScope: []string{"internal/core"}, // Conflict!
	}

	_, err := CompileDirective(input)
	if err == nil {
		t.Fatal("expected error for in-scope and out-of-scope conflict")
	}
	if !errors.Is(err, ErrScopeViolation) {
		t.Errorf("expected ErrScopeViolation, got: %v", err)
	}
}
