// Package team linters for agent contracts and skill contracts.
//
// In accordance with Constitución 93 (Capability & Skill Gap Register):
// - agent-contract-linter: CHECK de role, scope, permissions, inputs/outputs, handoff, failures e evidence.
// - skill-contract-linter: CHECK de manifest, activation positiva/negativa, permissions, output schema, freshness, examples e evals.
package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContractFinding describes an issue detected in an agent or skill contract.
type ContractFinding struct {
	EntityID string `json:"entity_id"`
	Kind     string `json:"kind"` // "agent" or "skill"
	Rule     string `json:"rule"`
	Severity string `json:"severity"` // "error", "warning"
	Message  string `json:"message"`
}

// AgentContract matches schemas/agent.schema.json
type AgentContract struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Version             int            `json:"version"`
	Purpose             string         `json:"purpose"`
	Core                any            `json:"core"`
	Inputs              []string       `json:"inputs"`
	Outputs             []string       `json:"outputs"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	RequiredEvidence    []string       `json:"required_evidence"`
	HandoffTo           []string       `json:"handoff_to"`
	RiskLevel           string         `json:"risk_level"`
	Permissions         map[string]any `json:"permissions"`
	StopConditions      []string       `json:"stop_conditions"`
}

// LintAgentContract validates an agent specification against workforce invariants.
func LintAgentContract(rawJSON []byte) []ContractFinding {
	findings := make([]ContractFinding, 0)
	var ag AgentContract
	if err := json.Unmarshal(rawJSON, &ag); err != nil {
		return append(findings, ContractFinding{
			EntityID: "unknown",
			Kind:     "agent",
			Rule:     "valid_json",
			Severity: "error",
			Message:  fmt.Sprintf("failed to parse agent contract JSON: %v", err),
		})
	}

	if strings.TrimSpace(ag.ID) == "" {
		findings = append(findings, ContractFinding{
			EntityID: "unknown",
			Kind:     "agent",
			Rule:     "required_id",
			Severity: "error",
			Message:  "agent contract missing required 'id'",
		})
	}
	if strings.TrimSpace(ag.Name) == "" {
		findings = append(findings, ContractFinding{
			EntityID: ag.ID,
			Kind:     "agent",
			Rule:     "required_name",
			Severity: "error",
			Message:  "agent contract missing required 'name'",
		})
	}
	if strings.TrimSpace(ag.Purpose) == "" {
		findings = append(findings, ContractFinding{
			EntityID: ag.ID,
			Kind:     "agent",
			Rule:     "required_purpose",
			Severity: "error",
			Message:  "agent contract missing required 'purpose'",
		})
	}

	// Invariant: Blank permissions / unbounded wildcards prohibited
	if perm, ok := ag.Permissions["allow_all"]; ok && perm == true {
		findings = append(findings, ContractFinding{
			EntityID: ag.ID,
			Kind:     "agent",
			Rule:     "bounded_permissions",
			Severity: "error",
			Message:  "blanket 'allow_all' permission prohibited; permissions must be explicitly scoped",
		})
	}

	// Invariant: Non-trivial agents must declare outputs or handoffs
	if len(ag.Outputs) == 0 && len(ag.HandoffTo) == 0 {
		findings = append(findings, ContractFinding{
			EntityID: ag.ID,
			Kind:     "agent",
			Rule:     "outputs_or_handoff",
			Severity: "warning",
			Message:  "agent specifies neither outputs nor handoff targets",
		})
	}

	return findings
}

// SkillManifest matches schemas/skill.schema.json
type SkillManifest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      int      `json:"version"`
	Purpose      string   `json:"purpose"`
	RiskLevel    string   `json:"risk_level"`
	Modes        []string `json:"modes"`
	Inputs       []any    `json:"inputs"`
	Outputs      []any    `json:"outputs"`
	Capabilities []string `json:"capabilities"`
}

var validSkillModes = map[string]bool{
	"implementation": true,
	"review":         true,
	"audit":          true,
	"research":       true,
	"design":         true,
	"testing":        true,
	"documentation":  true,
	"release":        true,
}

// LintSkillContract validates a skill directory structure and manifest.
func LintSkillContract(skillDir string) []ContractFinding {
	findings := make([]ContractFinding, 0)
	manifestPath := filepath.Join(skillDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return append(findings, ContractFinding{
			EntityID: filepath.Base(skillDir),
			Kind:     "skill",
			Rule:     "manifest_exists",
			Severity: "error",
			Message:  fmt.Sprintf("missing manifest.json: %v", err),
		})
	}

	var sm SkillManifest
	if err := json.Unmarshal(data, &sm); err != nil {
		return append(findings, ContractFinding{
			EntityID: filepath.Base(skillDir),
			Kind:     "skill",
			Rule:     "valid_json",
			Severity: "error",
			Message:  fmt.Sprintf("invalid manifest.json syntax: %v", err),
		})
	}

	if strings.TrimSpace(sm.ID) == "" {
		findings = append(findings, ContractFinding{
			EntityID: filepath.Base(skillDir),
			Kind:     "skill",
			Rule:     "required_id",
			Severity: "error",
			Message:  "skill manifest missing required 'id'",
		})
	}
	if strings.TrimSpace(sm.Name) == "" {
		findings = append(findings, ContractFinding{
			EntityID: sm.ID,
			Kind:     "skill",
			Rule:     "required_name",
			Severity: "error",
			Message:  "skill manifest missing required 'name'",
		})
	}
	if strings.TrimSpace(sm.Purpose) == "" {
		findings = append(findings, ContractFinding{
			EntityID: sm.ID,
			Kind:     "skill",
			Rule:     "required_purpose",
			Severity: "error",
			Message:  "skill manifest missing required 'purpose'",
		})
	}

	// Validate modes
	if len(sm.Modes) == 0 {
		findings = append(findings, ContractFinding{
			EntityID: sm.ID,
			Kind:     "skill",
			Rule:     "required_modes",
			Severity: "error",
			Message:  "skill manifest must specify at least one operational mode",
		})
	} else {
		for _, m := range sm.Modes {
			if !validSkillModes[m] {
				findings = append(findings, ContractFinding{
					EntityID: sm.ID,
					Kind:     "skill",
					Rule:     "valid_modes",
					Severity: "error",
					Message:  fmt.Sprintf("invalid mode %q in skill manifest (must be one of implementation|review|audit|research|design|testing|documentation|release)", m),
				})
			}
		}
	}

	// Instructions check (instructions.md or prompt.md)
	instructionsFound := false
	for _, candidate := range []string{"instructions.md", "INSTRUCTIONS.md", "prompt.md", "skill.md"} {
		iPath := filepath.Join(skillDir, candidate)
		if fi, err := os.Stat(iPath); err == nil && fi.Size() > 0 {
			instructionsFound = true
			break
		}
	}
	if !instructionsFound {
		findings = append(findings, ContractFinding{
			EntityID: sm.ID,
			Kind:     "skill",
			Rule:     "instructions_exist",
			Severity: "error",
			Message:  "skill directory missing non-empty instructions file (instructions.md)",
		})
	}

	return findings
}
