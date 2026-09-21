package team

import (
	"fmt"
)

// Reason codes for workforce explainability per Constitution 93.
const (
	ReasonLeastWorkforceSolo        = "LEAST_WORKFORCE_SOLO"
	ReasonIndependentReviewRequired = "INDEPENDENT_REVIEW_REQUIRED"
	ReasonSpecialistRequired        = "SPECIALIST_REQUIRED"
	ReasonModelDiversityRequired    = "MODEL_DIVERSITY_REQUIRED"
	ReasonExcessAgentsRejected      = "EXCESS_AGENTS_REJECTED"
)

// WorkforceExplanation provides structured rationale for workforce composition.
type WorkforceExplanation struct {
	DelegationMode   Delegation `json:"delegation_mode"`
	PrimaryReason    string     `json:"primary_reason"`
	AssignedRoles    []string   `json:"assigned_roles"`
	RejectedRoles    []string   `json:"rejected_roles,omitempty"`
	RequiresReviewer bool       `json:"requires_reviewer"`
	ModelDiversity   bool       `json:"model_diversity"`
}

// WorkforceSpec inputs the task criteria for workforce resolution.
type WorkforceSpec struct {
	TaskID               string   `json:"task_id"`
	RiskLevel            string   `json:"risk_level"` // low, medium, high, critical
	RequiredCapabilities []string `json:"required_capabilities"`
	AllowMultiAgent      bool     `json:"allow_multi_agent"`
}

// ResolveLeastWorkforce derives the minimum sufficient workforce for a task.
func ResolveLeastWorkforce(spec WorkforceSpec) (Team, WorkforceExplanation) {
	roles := []Role{
		{
			Name:    "implementer",
			Binding: "prumo-native-implementer",
			Budget: map[string]float64{
				"cost_usd": 0.80,
			},
		},
	}

	explanation := WorkforceExplanation{
		DelegationMode:   DelegationSolo,
		PrimaryReason:    ReasonLeastWorkforceSolo,
		AssignedRoles:    []string{"implementer"},
		RequiresReviewer: false,
		ModelDiversity:   false,
	}

	// If low risk or multi-agent disabled, solo is strictly enforced
	if spec.RiskLevel == "low" || !spec.AllowMultiAgent {
		return Team{
			ID:         "team-" + spec.TaskID,
			Delegation: DelegationSolo,
			Roles:      roles,
		}, explanation
	}

	// Medium / High / Critical risk requires an independent reviewer
	if spec.RiskLevel == "medium" || spec.RiskLevel == "high" || spec.RiskLevel == "critical" {
		roles = append(roles, Role{
			Name:    "reviewer",
			Binding: "prumo-native-reviewer",
			Budget: map[string]float64{
				"cost_usd": 0.20,
			},
		})
		explanation.DelegationMode = DelegationSuggested
		explanation.PrimaryReason = ReasonIndependentReviewRequired
		explanation.AssignedRoles = append(explanation.AssignedRoles, "reviewer")
		explanation.RequiresReviewer = true
		explanation.ModelDiversity = true
	}

	// For critical risk with specialist capabilities, attach a specialist role
	hasSpecialistCap := false
	for _, cap := range spec.RequiredCapabilities {
		if cap == "security" || cap == "kernel" || cap == "compiler" {
			hasSpecialistCap = true
			break
		}
	}

	if spec.RiskLevel == "critical" && hasSpecialistCap {
		roles = append(roles, Role{
			Name:    "specialist",
			Binding: "prumo-domain-specialist",
			Budget: map[string]float64{
				"cost_usd": 0.50,
			},
		})
		explanation.DelegationMode = DelegationBoundedAuto
		explanation.PrimaryReason = fmt.Sprintf("%s; %s", ReasonSpecialistRequired, ReasonIndependentReviewRequired)
		explanation.AssignedRoles = append(explanation.AssignedRoles, "specialist")
	}

	return Team{
		ID:         "team-" + spec.TaskID,
		Delegation: explanation.DelegationMode,
		Roles:      roles,
	}, explanation
}
