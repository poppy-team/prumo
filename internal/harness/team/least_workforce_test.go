package team

import (
	"strings"
	"testing"
)

func TestResolveLeastWorkforce(t *testing.T) {
	// 1. Low risk task -> solo
	teamSolo, expSolo := ResolveLeastWorkforce(WorkforceSpec{
		TaskID:          "task-low",
		RiskLevel:       "low",
		AllowMultiAgent: true,
	})

	if teamSolo.Delegation != DelegationSolo {
		t.Errorf("expected DelegationSolo, got %s", teamSolo.Delegation)
	}
	if len(teamSolo.Roles) != 1 || teamSolo.Roles[0].Name != "implementer" {
		t.Errorf("expected single implementer role, got: %#v", teamSolo.Roles)
	}
	if expSolo.PrimaryReason != ReasonLeastWorkforceSolo {
		t.Errorf("expected reason %s, got %s", ReasonLeastWorkforceSolo, expSolo.PrimaryReason)
	}

	// 2. Medium risk task -> implementer + independent reviewer
	teamMed, expMed := ResolveLeastWorkforce(WorkforceSpec{
		TaskID:          "task-med",
		RiskLevel:       "medium",
		AllowMultiAgent: true,
	})

	if teamMed.Delegation != DelegationSuggested {
		t.Errorf("expected DelegationSuggested, got %s", teamMed.Delegation)
	}
	if len(teamMed.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(teamMed.Roles))
	}
	if !expMed.RequiresReviewer || !expMed.ModelDiversity {
		t.Error("expected independent reviewer and model diversity to be true")
	}

	// 3. Critical risk with security specialist -> implementer + specialist + reviewer
	teamCrit, expCrit := ResolveLeastWorkforce(WorkforceSpec{
		TaskID:               "task-crit",
		RiskLevel:            "critical",
		RequiredCapabilities: []string{"security"},
		AllowMultiAgent:      true,
	})

	if len(teamCrit.Roles) != 3 {
		t.Fatalf("expected 3 roles for critical security task, got %d", len(teamCrit.Roles))
	}
	if !strings.Contains(expCrit.PrimaryReason, ReasonSpecialistRequired) {
		t.Errorf("expected reason to contain %s, got %s", ReasonSpecialistRequired, expCrit.PrimaryReason)
	}
}
