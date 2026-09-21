// Package evidence derived completion state machine.
//
// In accordance with Constitución 84.A & 85.A:
// - Universal Surface Coverage: documented, implemented, reachable, exercised, evidenced,
//   independently-verified, regression-protected, release-accepted.
// - Acceptance Coverage: uncovered, planned, implemented, evidenced, verified, accepted,
//   waived, blocked_external.
// - Fundamental Rule: "implemented != reachable != exercised != evidenced != verified != accepted != released."
//   Completion is ALWAYS derived by the system from verified evidence records, never accepted from
//   agent natural language statements ("I have finished", "all done", etc.).
package evidence

import (
	"fmt"
	"regexp"
	"strings"
)

// AcceptanceStatus represents the progression of an acceptance criterion.
type AcceptanceStatus string

const (
	AcceptanceUncovered       AcceptanceStatus = "uncovered"
	AcceptancePlanned         AcceptanceStatus = "planned"
	AcceptanceImplemented     AcceptanceStatus = "implemented"
	AcceptanceEvidenced       AcceptanceStatus = "evidenced"
	AcceptanceVerified        AcceptanceStatus = "verified"
	AcceptanceAccepted        AcceptanceStatus = "accepted"
	AcceptanceWaived          AcceptanceStatus = "waived"
	AcceptanceBlockedExternal AcceptanceStatus = "blocked_external"
)

// CoverageState represents independent surface coverage axes per Constitución 84.A.
type CoverageState string

const (
	CoverageDocumented            CoverageState = "documented"
	CoverageImplemented           CoverageState = "implemented"
	CoverageReachable             CoverageState = "reachable"
	CoverageExercised             CoverageState = "exercised"
	CoverageEvidenced             CoverageState = "evidenced"
	CoverageIndependentlyVerified CoverageState = "independently-verified"
	CoverageRegressionProtected   CoverageState = "regression-protected"
	CoverageReleaseAccepted       CoverageState = "release-accepted"
)

// AcceptanceCriterion is an individual requirement that must be evidenced before completion.
type AcceptanceCriterion struct {
	ID           string           `json:"id"`
	Description  string           `json:"description"`
	Required     bool             `json:"required"`
	Status       AcceptanceStatus `json:"status"`
	EvidenceIDs  []string         `json:"evidence_ids,omitempty"`
	WaivedReason string           `json:"waived_reason,omitempty"`
}

// CompletionEvaluation holds the deterministic machine verdict on completion.
type CompletionEvaluation struct {
	GoalID               string   `json:"goal_id"`
	CanComplete          bool     `json:"can_complete"`
	DerivedState         string   `json:"derived_state"`
	UnsatisfiedCriteria  []string `json:"unsatisfied_criteria,omitempty"`
	StaleEvidenceCount   int      `json:"stale_evidence_count"`
	MissingEvidenceCount int      `json:"missing_evidence_count"`
	ReasonCodes          []string `json:"reason_codes"`
	Explanation          string   `json:"explanation"`
}

var selfDeclarationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(i('?m| am) (done|finished))\b`),
	regexp.MustCompile(`(?i)\b(task (is )?(done|finished|complete(d)?))\b`),
	regexp.MustCompile(`(?i)\b(all requirements are (met|implemented|done|complete))\b`),
	regexp.MustCompile(`(?i)\b(work (is )?complete)\b`),
	regexp.MustCompile(`(?i)\b(successfully (implemented|completed|finished))\b`),
}

// IsAgentSelfDeclaration identifies natural language claims in agent output that attempt
// to unilaterally assert completion without backing evidence.
func IsAgentSelfDeclaration(text string) bool {
	for _, pat := range selfDeclarationPatterns {
		if pat.MatchString(text) {
			return true
		}
	}
	return false
}

// EvaluateCompletion computes the derived state of a goal or task given its acceptance criteria
// and registry of available evidence records.
//
// Rules enforced:
// 1. "Goal DONE exige zero required criteria em uncovered/planned/implemented-only."
// 2. An evidenced or verified criterion MUST point to at least one valid, non-stale evidence record.
// 3. Waived criteria require an explicit waiver reason.
// 4. Blocked external criteria block completion unless formally waived.
// 5. Completion is derived; if criteria are missing or unsatisfied, CanComplete is strictly false.
func EvaluateCompletion(goalID string, criteria []AcceptanceCriterion, evidenceRegistry []Record) CompletionEvaluation {
	eval := CompletionEvaluation{
		GoalID:      goalID,
		CanComplete: false,
		ReasonCodes: make([]string, 0),
	}

	if len(criteria) == 0 {
		eval.DerivedState = "uncovered"
		eval.ReasonCodes = append(eval.ReasonCodes, "REASON-NO-CRITERIA-DEFINED")
		eval.Explanation = "Goal cannot be completed: no acceptance criteria defined."
		return eval
	}

	// Index evidence by ID
	evidenceMap := make(map[string]Record)
	for _, rec := range evidenceRegistry {
		evidenceMap[rec.ID] = rec
	}

	allAccepted := true
	allEvidencedOrVerified := true

	for _, c := range criteria {
		if !c.Required {
			continue
		}

		switch c.Status {
		case AcceptanceUncovered, AcceptancePlanned:
			eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
			eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-CRITERION-UNCOVERED")
			allAccepted = false
			allEvidencedOrVerified = false

		case AcceptanceImplemented:
			// Constitución 84.A / 85.A: implemented != evidenced != verified
			eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
			eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-IMPLEMENTED-ONLY-REJECTED")
			allAccepted = false
			allEvidencedOrVerified = false

		case AcceptanceBlockedExternal:
			eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
			eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-CRITERION-BLOCKED-EXTERNAL")
			allAccepted = false
			allEvidencedOrVerified = false

		case AcceptanceWaived:
			if strings.TrimSpace(c.WaivedReason) == "" {
				eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
				eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-UNJUSTIFIED-WAIVER")
				allAccepted = false
				allEvidencedOrVerified = false
			}

		case AcceptanceEvidenced, AcceptanceVerified:
			allAccepted = false
			// Verify evidence presence and freshness
			if len(c.EvidenceIDs) == 0 {
				eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
				eval.MissingEvidenceCount++
				eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-EVIDENCE-MISSING")
				allEvidencedOrVerified = false
			} else {
				hasValidEvidence := false
				for _, eid := range c.EvidenceIDs {
					rec, ok := evidenceMap[eid]
					if !ok {
						eval.MissingEvidenceCount++
						eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-EVIDENCE-NOT-FOUND")
						continue
					}
					if rec.Stale {
						eval.StaleEvidenceCount++
						eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-EVIDENCE-STALE")
						continue
					}
					hasValidEvidence = true
				}
				if !hasValidEvidence {
					eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
					allEvidencedOrVerified = false
				}
			}

		case AcceptanceAccepted:
			// Accepted requires valid evidence
			if len(c.EvidenceIDs) == 0 {
				eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
				eval.MissingEvidenceCount++
				eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-EVIDENCE-MISSING")
				allAccepted = false
				allEvidencedOrVerified = false
			} else {
				hasValidEvidence := false
				for _, eid := range c.EvidenceIDs {
					rec, ok := evidenceMap[eid]
					if !ok || rec.Stale {
						continue
					}
					hasValidEvidence = true
				}
				if !hasValidEvidence {
					eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
					allAccepted = false
					allEvidencedOrVerified = false
				}
			}

		default:
			eval.UnsatisfiedCriteria = append(eval.UnsatisfiedCriteria, c.ID)
			eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-UNKNOWN-ACCEPTANCE-STATUS")
			allAccepted = false
			allEvidencedOrVerified = false
		}
	}

	if len(eval.UnsatisfiedCriteria) > 0 {
		eval.CanComplete = false
		eval.DerivedState = "incomplete"
		eval.Explanation = fmt.Sprintf("Goal cannot complete: %d criteria unsatisfied (%s)",
			len(eval.UnsatisfiedCriteria), strings.Join(eval.UnsatisfiedCriteria, ", "))
		return eval
	}

	eval.CanComplete = true
	if allAccepted {
		eval.DerivedState = "accepted"
		eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-RELEASE-ACCEPTED")
		eval.Explanation = "All criteria verified and formally accepted."
	} else if allEvidencedOrVerified {
		eval.DerivedState = "verified"
		eval.ReasonCodes = appendUnique(eval.ReasonCodes, "REASON-TOTAL-ASSURANCE-SATISFIED")
		eval.Explanation = "All criteria evidenced with fresh, verified evidence records."
	}

	return eval
}
