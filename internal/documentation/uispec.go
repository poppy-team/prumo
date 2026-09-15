// UI specification checking (W13.3). A UI specification is only as good as its
// weakest obligation: a contract with no required knowledge, a state nobody
// mapped, or a token that resolves to nothing all produce a UI that cannot be
// verified. This checker is deterministic and reads only declared data.
//
// Scope: the checks apply to UI-bearing contracts (`ui.*`, `tui.*`,
// `accessibility.*`). Documentation contracts are verified by the semantic
// readiness model (W15), where evidence is bound at the claim level instead of
// on the contract.
package docengine

import (
	"fmt"
	"sort"
	"strings"
)

// UIStates is the canonical state vocabulary every surface must address. It is
// the Go side of the ui-state-matrix schema enum and is asserted equal to it by
// the conformance gate.
func UIStates() []string {
	return []string{
		"default", "hover", "focus", "active", "selected", "disabled", "empty",
		"loading", "partial", "success", "warning", "error", "offline",
		"reconnecting", "permission-requested", "permission-denied", "read-only",
		"destructive-confirmation", "narrow-viewport", "alternate-theme",
		"high-contrast", "alternate-locale",
	}
}

// MinimumContrast is the WCAG 2.2 AA ratio for normal text. Below it, a token
// cannot be used for text at all.
const MinimumContrast = 4.5

// AAAContrast is the ratio a token must reach to claim the AAA level.
const AAAContrast = 7.0

// ExceptionEvidence is the evidence class for an obligation that automation
// cannot settle.
const ExceptionEvidence = "exception-with-rationale"

// uicapabilities are the applicability capabilities that make a contract UI
// bearing.
var uicapabilities = []string{"ui", "tui", "desktop-gui", "web-application"}

// UISpecContract is one UI contract as the checker sees it.
type UISpecContract struct {
	ID                   string   `json:"id"`
	Role                 string   `json:"role"`
	Description          string   `json:"description"`
	RequiredKnowledge    []string `json:"required_knowledge"`
	BlockingQuestions    []string `json:"blocking_questions"`
	EvidenceRequirements []string `json:"evidence_requirements"`
	States               []string `json:"states"`
	Applicability        struct {
		CapabilitiesAny []string `json:"capabilities_any"`
	} `json:"applicability"`
}

// UISpecProfile composes contracts into a surface profile.
type UISpecProfile struct {
	ID        string   `json:"id"`
	Version   int      `json:"version"`
	Contracts []string `json:"contracts"`
}

// UISpecContrast is a measured contrast claim.
type UISpecContrast struct {
	Against string  `json:"against"`
	Ratio   float64 `json:"ratio"`
	WCAG    string  `json:"wcag"`
}

// UISpecToken is one design or component token.
type UISpecToken struct {
	ID                 string          `json:"id"`
	Tier               string          `json:"tier"`
	Type               string          `json:"type"`
	Value              string          `json:"value"`
	Platforms          []string        `json:"platforms"`
	CapabilityFallback string          `json:"capability_fallback"`
	Contrast           *UISpecContrast `json:"contrast"`
}

// UISpecTheme is a presentation variant.
type UISpecTheme struct {
	ID            string            `json:"id"`
	Extends       string            `json:"extends"`
	HighContrast  bool              `json:"high_contrast"`
	ReducedMotion bool              `json:"reduced_motion"`
	Overrides     map[string]string `json:"overrides"`
}

// UISpec is the checkable view of a UI specification.
type UISpec struct {
	Contracts []UISpecContract `json:"contracts"`
	Profiles  []UISpecProfile  `json:"profiles"`
	Tokens    []UISpecToken    `json:"tokens"`
	Themes    []UISpecTheme    `json:"themes"`
}

// SpecFinding is one UI specification defect.
type SpecFinding struct {
	Kind    string `json:"kind"`
	Subject string `json:"subject"`
	Detail  string `json:"detail"`
}

// UISpecChecks lists the deterministic checks by name, so a report can state
// exactly what ran.
func UISpecChecks() []string {
	return []string{
		"accessibility-evidence", "contract-blocking-questions", "contract-composition",
		"contract-knowledge", "contract-uniqueness", "state-coverage",
		"theme-required-variants", "token-contrast", "token-integrity", "token-tui-fallback",
	}
}

// IsUIContract reports whether a contract is UI bearing: its id is namespaced
// under ui./tui./accessibility., or it applies to a UI capability.
func IsUIContract(contract UISpecContract) bool {
	for _, prefix := range []string{"ui.", "tui.", "accessibility."} {
		if strings.HasPrefix(contract.ID, prefix) {
			return true
		}
	}
	for _, capability := range contract.Applicability.CapabilitiesAny {
		if containsString(uicapabilities, capability) {
			return true
		}
	}
	return false
}

// IsDeprecated reports a contract that is retained for read compatibility only.
// A superseded contract is not required to carry live obligations.
func IsDeprecated(contract UISpecContract) bool {
	return strings.Contains(strings.ToUpper(contract.ID+" "+contract.Description), "DEPRECATED")
}

// ValidateUISpec runs every deterministic UI specification check and returns the
// findings in a stable order.
func ValidateUISpec(spec UISpec) []SpecFinding {
	findings := []SpecFinding{}

	composed := map[string]bool{}
	for _, profile := range spec.Profiles {
		for _, id := range profile.Contracts {
			composed[id] = true
		}
	}
	stateVocabulary := map[string]bool{}
	for _, state := range UIStates() {
		stateVocabulary[state] = true
	}

	seen := map[string]bool{}
	for _, contract := range spec.Contracts {
		if contract.ID == "" {
			findings = append(findings, SpecFinding{Kind: "contract-uniqueness", Subject: "<unnamed>",
				Detail: "a UI contract has no id"})
			continue
		}
		if seen[contract.ID] {
			findings = append(findings, SpecFinding{Kind: "contract-uniqueness", Subject: contract.ID,
				Detail: "the contract id is declared more than once"})
			continue
		}
		seen[contract.ID] = true

		if !IsUIContract(contract) || IsDeprecated(contract) {
			continue
		}
		if len(contract.RequiredKnowledge) == 0 {
			findings = append(findings, SpecFinding{Kind: "contract-knowledge", Subject: contract.ID,
				Detail: "the contract states no required knowledge, so nothing is ever checked"})
		}
		// A contract with obligations but no blocking question can never
		// surface an unknown: it silently degrades into "nothing to ask".
		if len(contract.BlockingQuestions) == 0 {
			findings = append(findings, SpecFinding{Kind: "contract-blocking-questions", Subject: contract.ID,
				Detail: "the contract has obligations but no blocking question"})
		}
		if !composed[contract.ID] {
			findings = append(findings, SpecFinding{Kind: "contract-composition", Subject: contract.ID,
				Detail: "no profile composes this contract, so nothing ever evaluates it"})
		}
		for _, state := range contract.States {
			if !stateVocabulary[state] {
				findings = append(findings, SpecFinding{Kind: "state-coverage", Subject: contract.ID,
					Detail: fmt.Sprintf("state %q is not part of the canonical state vocabulary", state)})
			}
		}
		// An automated check can measure a threshold but cannot conclude that an
		// interface is usable. An accessibility contract whose evidence set is
		// exclusively automated has no path to a human judgement (W11).
		if strings.HasPrefix(contract.ID, "accessibility.") && len(contract.EvidenceRequirements) > 0 {
			allAutomated := true
			for _, req := range contract.EvidenceRequirements {
				if req != "automated" {
					allAutomated = false
					break
				}
			}
			if allAutomated {
				findings = append(findings, SpecFinding{Kind: "accessibility-evidence", Subject: contract.ID,
					Detail: "evidence is automated only; an accessibility obligation needs a human-judged class (" +
						"manual, visual or " + ExceptionEvidence + ")"})
			}
		}
	}

	// The TUI capability needs a theme that survives a terminal with no colour
	// and a theme with measured high contrast; without them the capability is
	// declared but unsupported.
	if len(spec.Themes) > 0 {
		hasNoColor, hasHighContrast := false, false
		for _, theme := range spec.Themes {
			if strings.Contains(theme.ID, "no-color") {
				hasNoColor = true
			}
			if theme.HighContrast || strings.Contains(theme.ID, "high-contrast") {
				hasHighContrast = true
			}
		}
		if !hasNoColor {
			findings = append(findings, SpecFinding{Kind: "theme-required-variants", Subject: "themes",
				Detail: "no no-color theme: a monochrome terminal would be asked to render colours it does not have"})
		}
		if !hasHighContrast {
			findings = append(findings, SpecFinding{Kind: "theme-required-variants", Subject: "themes",
				Detail: "no high-contrast theme declared"})
		}
	}

	tokenIDs := map[string]bool{}
	for _, token := range spec.Tokens {
		tokenIDs[token.ID] = true
	}
	for _, token := range spec.Tokens {
		if target, ok := strings.CutPrefix(token.Value, "token:"); ok && !tokenIDs["token:"+target] {
			findings = append(findings, SpecFinding{Kind: "token-integrity", Subject: token.ID,
				Detail: "value references an unknown token: " + token.Value})
		}
		// A non-colour token on a text terminal must say what it degrades to
		// when the terminal cannot draw what it wants.
		if token.Type != "color" && containsString(token.Platforms, "tui") && token.CapabilityFallback == "" {
			findings = append(findings, SpecFinding{Kind: "token-tui-fallback", Subject: token.ID,
				Detail: "a non-colour token used on the TUI declares no capability fallback"})
		}
		if token.Contrast == nil {
			continue
		}
		if token.Contrast.Against != "" && !tokenIDs[token.Contrast.Against] {
			findings = append(findings, SpecFinding{Kind: "token-integrity", Subject: token.ID,
				Detail: "contrast is measured against an unknown token: " + token.Contrast.Against})
		}
		if token.Contrast.Ratio < MinimumContrast {
			findings = append(findings, SpecFinding{Kind: "token-contrast", Subject: token.ID,
				Detail: fmt.Sprintf("contrast ratio %.2f is below the WCAG AA minimum %.1f", token.Contrast.Ratio, MinimumContrast)})
			continue
		}
		if strings.EqualFold(token.Contrast.WCAG, "AAA") && token.Contrast.Ratio < AAAContrast {
			findings = append(findings, SpecFinding{Kind: "token-contrast", Subject: token.ID,
				Detail: fmt.Sprintf("claims WCAG AAA but the measured ratio is %.2f, below %.1f", token.Contrast.Ratio, AAAContrast)})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].Subject != findings[j].Subject {
			return findings[i].Subject < findings[j].Subject
		}
		return findings[i].Detail < findings[j].Detail
	})
	return findings
}
