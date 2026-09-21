package directive

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrMissingAuthority = errors.New("directive: missing canonical authority for required action")
	ErrScopeViolation   = errors.New("directive: scope firewall violation")
	ErrEmptyIntent      = errors.New("directive: intent is required")
	ErrInvalidIR        = errors.New("directive: invalid directive ir")
)

// ToolCapability describes an available tool and its constraints.
type ToolCapability struct {
	Name              string `json:"name"`
	SideEffectClass   string `json:"side_effect_class"` // read_only, idempotent_mutation, stateful_mutation, destructive
	TimeoutMs         int    `json:"timeout_ms"`
	ResultBudgetBytes int    `json:"result_budget_bytes"`
}

// ModelRouteSpec defines the routed provider and model tier.
type ModelRouteSpec struct {
	RouteClass string `json:"route_class"` // ROUTE-CHEAP, ROUTE-PRIMARY, ROUTE-FALLBACK
	Provider   string `json:"provider"`
	Model      string `json:"model"`
}

// WorkforceBindingSpec binds an actor to a role.
type WorkforceBindingSpec struct {
	Role  string `json:"role"`
	Actor string `json:"actor"`
}

// BudgetEnvelopeSpec defines budget limits and reserves.
type BudgetEnvelopeSpec struct {
	HardLimitCents     int    `json:"hard_limit_cents"`
	SoftLimitCents     int    `json:"soft_limit_cents"`
	ReviewReserveCents int    `json:"review_reserve_cents"`
	Currency           string `json:"currency"`
}

// ScopeFirewall defines explicit IN, OUT, and INCIDENTAL scopes.
type ScopeFirewall struct {
	InScope         []string `json:"in_scope"`
	OutOfScope      []string `json:"out_of_scope"`
	IncidentalScope []string `json:"incidental_scope,omitempty"`
}

// MutationBoundaries restricts allowed filesystem paths.
type MutationBoundaries struct {
	AllowedPaths   []string `json:"allowed_paths"`
	ForbiddenPaths []string `json:"forbidden_paths"`
}

// PermissionEnvelope defines permitted actions and approval requirements.
type PermissionEnvelope struct {
	AllowedActions   []string `json:"allowed_actions"`
	ForbiddenActions []string `json:"forbidden_actions"`
	RequiresApproval []string `json:"requires_approval,omitempty"`
}

// UnknownOrAssumption records unverified facts or explicit working assumptions.
type UnknownOrAssumption struct {
	Kind        string `json:"kind"` // unknown, assumption
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
}

// AuthoritySnapshot records the active governance decisions and constraints.
type AuthoritySnapshot struct {
	ActiveDecisions    []string `json:"active_decisions"`
	ActiveRequirements []string `json:"active_requirements"`
	ActiveConstraints  []string `json:"active_constraints"`
	SupersededRecords  []string `json:"superseded_records,omitempty"`
}

// DocDeltaContract defines required documentation updates before completion.
type DocDeltaContract struct {
	RequiredDocUpdates []string `json:"required_doc_updates,omitempty"`
}

// DirectiveIR is the canonical versioned intermediate representation.
type DirectiveIR struct {
	DirectiveIRVersion       string                `json:"directive_ir_version"`
	ProjectID                string                `json:"project_id"`
	WorkspaceRoot            string                `json:"workspace_root"`
	RepositoryRevision       string                `json:"repository_revision"`
	GoalID                   string                `json:"goal_id"`
	PlanID                   string                `json:"plan_id,omitempty"`
	TaskID                   string                `json:"task_id"`
	TaskIntent               string                `json:"task_intent"`
	AuthoritySnapshot        AuthoritySnapshot     `json:"authority_snapshot"`
	ScopeFirewall            ScopeFirewall         `json:"scope_firewall"`
	UnknownsAndAssumptions   []UnknownOrAssumption `json:"unknowns_and_assumptions,omitempty"`
	MutationBoundaries       MutationBoundaries    `json:"mutation_boundaries"`
	PermissionEnvelope       PermissionEnvelope    `json:"permission_envelope"`
	ToolCapabilities         []ToolCapability      `json:"tool_capabilities"`
	ModelRoute               ModelRouteSpec        `json:"model_route"`
	WorkforceBinding         WorkforceBindingSpec  `json:"workforce_binding"`
	BudgetEnvelope           BudgetEnvelopeSpec    `json:"budget_envelope"`
	RequiredSkills           []string              `json:"required_skills,omitempty"`
	RequiredEvidence         []string              `json:"required_evidence"`
	QualityGates             []string              `json:"quality_gates"`
	StopConditions           []string              `json:"stop_conditions"`
	DocumentationDelta       DocDeltaContract      `json:"documentation_delta_contract,omitempty"`
	SourceDigests            map[string]string     `json:"source_digests"`
}

// CompilerInput contains the raw inputs to be compiled into a DirectiveIR.
type CompilerInput struct {
	ProjectID          string
	WorkspaceRoot      string
	RepositoryRevision string
	GoalID             string
	PlanID             string
	TaskID             string
	UserIntent         string
	TaskIntent         string
	Decisions          []string
	Requirements       []string
	Constraints        []string
	Superseded         []string
	InScope            []string
	OutOfScope         []string
	IncidentalScope    []string
	AllowedPaths       []string
	ForbiddenPaths     []string
	AllowedActions     []string
	ForbiddenActions   []string
	RequiresApproval   []string
	ToolCapabilities   []ToolCapability
	ModelRoute         ModelRouteSpec
	WorkforceRole      string
	WorkforceActor     string
	HardLimitCents     int
	SoftLimitCents     int
	ReviewReserveCents int
	Currency           string
	RequiredSkills     []string
	RequiredEvidence   []string
	QualityGates       []string
	StopConditions     []string
	RequiredDocUpdates []string
	Unknowns           []string
	Assumptions        []string
	IsDestructive      bool
	HasAuthority       bool
}

// CompileDirective compiles raw input into a validated DirectiveIR.
func CompileDirective(input CompilerInput) (*DirectiveIR, error) {
	intent := strings.TrimSpace(input.TaskIntent)
	if intent == "" {
		intent = strings.TrimSpace(input.UserIntent)
	}
	if intent == "" {
		return nil, ErrEmptyIntent
	}

	// 1. Missing Authority check
	if input.IsDestructive && !input.HasAuthority {
		return nil, fmt.Errorf("%w: destructive action requires explicit canonical authority", ErrMissingAuthority)
	}

	// 2. Scope Firewall: ensure no overlap between in-scope and out-of-scope
	for _, in := range input.InScope {
		for _, out := range input.OutOfScope {
			if strings.EqualFold(strings.TrimSpace(in), strings.TrimSpace(out)) {
				return nil, fmt.Errorf("%w: item '%s' is declared both in-scope and out-of-scope", ErrScopeViolation, in)
			}
		}
	}

	// 3. Build unknowns and assumptions
	unknownsAssumptions := make([]UnknownOrAssumption, 0)
	for _, u := range input.Unknowns {
		unknownsAssumptions = append(unknownsAssumptions, UnknownOrAssumption{
			Kind:        "unknown",
			Description: u,
		})
	}
	for _, a := range input.Assumptions {
		unknownsAssumptions = append(unknownsAssumptions, UnknownOrAssumption{
			Kind:        "assumption",
			Description: a,
		})
	}

	curr := input.Currency
	if curr == "" {
		curr = "BRL"
	}

	ir := &DirectiveIR{
		DirectiveIRVersion: "1.0.0",
		ProjectID:          input.ProjectID,
		WorkspaceRoot:      input.WorkspaceRoot,
		RepositoryRevision: input.RepositoryRevision,
		GoalID:             input.GoalID,
		PlanID:             input.PlanID,
		TaskID:             input.TaskID,
		TaskIntent:         intent,
		AuthoritySnapshot: AuthoritySnapshot{
			ActiveDecisions:    input.Decisions,
			ActiveRequirements: input.Requirements,
			ActiveConstraints:  input.Constraints,
			SupersededRecords:  input.Superseded,
		},
		ScopeFirewall: ScopeFirewall{
			InScope:         input.InScope,
			OutOfScope:      input.OutOfScope,
			IncidentalScope: input.IncidentalScope,
		},
		UnknownsAndAssumptions: unknownsAssumptions,
		MutationBoundaries: MutationBoundaries{
			AllowedPaths:   input.AllowedPaths,
			ForbiddenPaths: input.ForbiddenPaths,
		},
		PermissionEnvelope: PermissionEnvelope{
			AllowedActions:   input.AllowedActions,
			ForbiddenActions: input.ForbiddenActions,
			RequiresApproval: input.RequiresApproval,
		},
		ToolCapabilities: input.ToolCapabilities,
		ModelRoute:       input.ModelRoute,
		WorkforceBinding: WorkforceBindingSpec{
			Role:  input.WorkforceRole,
			Actor: input.WorkforceActor,
		},
		BudgetEnvelope: BudgetEnvelopeSpec{
			HardLimitCents:     input.HardLimitCents,
			SoftLimitCents:     input.SoftLimitCents,
			ReviewReserveCents: input.ReviewReserveCents,
			Currency:           curr,
		},
		RequiredSkills:   input.RequiredSkills,
		RequiredEvidence: input.RequiredEvidence,
		QualityGates:     input.QualityGates,
		StopConditions:   input.StopConditions,
		DocumentationDelta: DocDeltaContract{
			RequiredDocUpdates: input.RequiredDocUpdates,
		},
		SourceDigests: make(map[string]string),
	}

	// Calculate intent digest
	h := sha256.New()
	h.Write([]byte(intent))
	ir.SourceDigests["intent_digest"] = hex.EncodeToString(h.Sum(nil))

	if err := ir.Validate(); err != nil {
		return nil, err
	}

	return ir, nil
}

// Validate ensures all required invariants in DirectiveIR are met.
func (d *DirectiveIR) Validate() error {
	if d.DirectiveIRVersion != "1.0.0" {
		return fmt.Errorf("%w: unsupported version %s", ErrInvalidIR, d.DirectiveIRVersion)
	}
	if d.TaskID == "" {
		return fmt.Errorf("%w: task_id is required", ErrInvalidIR)
	}
	if d.GoalID == "" {
		return fmt.Errorf("%w: goal_id is required", ErrInvalidIR)
	}
	if d.BudgetEnvelope.HardLimitCents <= 0 {
		return fmt.Errorf("%w: budget hard_limit_cents must be positive", ErrInvalidIR)
	}
	return nil
}

// CanMutatePath checks whether a path mutation is allowed by MutationBoundaries and ScopeFirewall.
func (d *DirectiveIR) CanMutatePath(relPath string) bool {
	norm := filepath.ToSlash(filepath.Clean(relPath))
	// Check forbidden paths
	for _, f := range d.MutationBoundaries.ForbiddenPaths {
		fnorm := filepath.ToSlash(filepath.Clean(f))
		if strings.HasPrefix(norm, fnorm) || norm == fnorm {
			return false
		}
	}
	// Check out of scope
	for _, out := range d.ScopeFirewall.OutOfScope {
		outnorm := filepath.ToSlash(filepath.Clean(out))
		if strings.HasPrefix(norm, outnorm) || norm == outnorm {
			return false
		}
	}
	// If allowed paths is non-empty, path must match at least one
	if len(d.MutationBoundaries.AllowedPaths) > 0 {
		allowed := false
		for _, a := range d.MutationBoundaries.AllowedPaths {
			anorm := filepath.ToSlash(filepath.Clean(a))
			if strings.HasPrefix(norm, anorm) || norm == anorm {
				allowed = true
				break
			}
		}
		return allowed
	}
	return true
}

// FormatAgentPrompt generates the governed prompt projection according to Constitución 92.
func (d *DirectiveIR) FormatAgentPrompt() string {
	var sb strings.Builder

	// Tier 1: Hard Policies
	sb.WriteString("=== 1. HARD POLICIES ===\n")
	sb.WriteString("- Grounding & Anti-Invention: You must inspect before claiming existence; never invent APIs or facts.\n")
	sb.WriteString("- Least Privilege & Safe Stop: Mutations must remain strictly within allowed paths and permissions.\n")
	sb.WriteString("- Derived Completion: You cannot declare DONE; quality gates and derived evidence determine completion.\n\n")

	// Tier 2: Identity & Role
	sb.WriteString("=== 2. IDENTITY & ROLE ===\n")
	role := d.WorkforceBinding.Role
	if role == "" {
		role = "Native Implementer"
	}
	sb.WriteString(fmt.Sprintf("Role: %s\nTask ID: %s | Goal ID: %s\n\n", role, d.TaskID, d.GoalID))

	// Tier 3: Canonical Invariants
	sb.WriteString("=== 3. CANONICAL INVARIANTS ===\n")
	for _, d := range d.AuthoritySnapshot.ActiveDecisions {
		sb.WriteString(fmt.Sprintf("- Decision: %s\n", d))
	}
	for _, r := range d.AuthoritySnapshot.ActiveRequirements {
		sb.WriteString(fmt.Sprintf("- Requirement: %s\n", r))
	}
	for _, c := range d.AuthoritySnapshot.ActiveConstraints {
		sb.WriteString(fmt.Sprintf("- Constraint: %s\n", c))
	}
	if len(d.AuthoritySnapshot.ActiveDecisions) == 0 && len(d.AuthoritySnapshot.ActiveRequirements) == 0 && len(d.AuthoritySnapshot.ActiveConstraints) == 0 {
		sb.WriteString("- Standard repository governance applies.\n")
	}
	sb.WriteString("\n")

	// Tier 4: Scope & Acceptance
	sb.WriteString("=== 4. SCOPE & ACCEPTANCE ===\n")
	sb.WriteString(fmt.Sprintf("Intent: %s\n", d.TaskIntent))
	sb.WriteString("In-Scope:\n")
	for _, in := range d.ScopeFirewall.InScope {
		sb.WriteString(fmt.Sprintf("  + %s\n", in))
	}
	sb.WriteString("Out-of-Scope (Non-Goals):\n")
	for _, out := range d.ScopeFirewall.OutOfScope {
		sb.WriteString(fmt.Sprintf("  - %s\n", out))
	}
	sb.WriteString("\n")

	// Tier 5: Implementation Facts & Unknowns
	sb.WriteString("=== 5. FACTS & UNKNOWNS ===\n")
	for _, ua := range d.UnknownsAndAssumptions {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", strings.ToUpper(ua.Kind), ua.Description))
	}
	sb.WriteString("\n")

	// Tier 6: Skills & Procedures
	sb.WriteString("=== 6. SKILLS & PROCEDURES ===\n")
	for _, s := range d.RequiredSkills {
		sb.WriteString(fmt.Sprintf("- Skill: %s\n", s))
	}
	sb.WriteString("\n")

	// Tier 7: Tools & Permissions
	sb.WriteString("=== 7. TOOLS & PERMISSIONS ===\n")
	sb.WriteString("Available Tools:\n")
	for _, tc := range d.ToolCapabilities {
		sb.WriteString(fmt.Sprintf("  - %s (side-effect: %s)\n", tc.Name, tc.SideEffectClass))
	}
	sb.WriteString("\n")

	// Tier 8: Required Evidence & Quality Gates
	sb.WriteString("=== 8. REQUIRED EVIDENCE & GATES ===\n")
	for _, ev := range d.RequiredEvidence {
		sb.WriteString(fmt.Sprintf("- Evidence: %s\n", ev))
	}
	for _, g := range d.QualityGates {
		sb.WriteString(fmt.Sprintf("- Gate: %s\n", g))
	}
	sb.WriteString("\n")

	// Tier 9: Budget & Stop Rules
	sb.WriteString("=== 9. BUDGET & STOP RULES ===\n")
	sb.WriteString(fmt.Sprintf("Hard Limit: %d %s (Review Reserve: %d %s)\n",
		d.BudgetEnvelope.HardLimitCents, d.BudgetEnvelope.Currency,
		d.BudgetEnvelope.ReviewReserveCents, d.BudgetEnvelope.Currency))
	for _, sc := range d.StopConditions {
		sb.WriteString(fmt.Sprintf("- Stop if: %s\n", sc))
	}
	sb.WriteString("\n")

	// Tier 10: Output Contract
	sb.WriteString("=== 10. OUTPUT CONTRACT ===\n")
	sb.WriteString("Format final output with status, inspected reality, applied changes, and produced evidence.\n")

	return sb.String()
}

// ToJSON serializes the DirectiveIR into formatted JSON.
func (d *DirectiveIR) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}
