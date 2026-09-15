package docengine

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CoverageMode records how a coverage state was derived. Only ModeSemantic is
// authoritative: lexical keyword matching can never assert readiness (W15.5).
type CoverageMode string

const (
	ModeSemantic CoverageMode = "semantic"
	ModeLexical  CoverageMode = "lexical"
)

// EvidenceRef is evidence bound to the requirement or claim it proves. Evidence
// that does not name its target can never satisfy a requirement (W15.6).
type EvidenceRef struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Target    string `json:"target"`
	Revision  string `json:"revision,omitempty"`
	Producer  string `json:"producer,omitempty"`
	Verified  bool   `json:"verified,omitempty"`
	Freshness string `json:"freshness,omitempty"`
	Artifact  string `json:"artifact,omitempty"`
	// ArtifactDigest makes an artifact's currency mechanically checkable. When
	// present, a mismatch with the artifact on disk is stale evidence. When
	// absent, currency rests on the recorded revision (W15.7).
	ArtifactDigest string `json:"artifact_digest,omitempty"`
}

// ClaimBinding is a durable claim that satisfies one or more requirements and
// carries the evidence proving it at a revision.
type ClaimBinding struct {
	ID        string        `json:"id"`
	Statement string        `json:"statement"`
	Status    string        `json:"status"`
	Authority string        `json:"authority,omitempty"`
	Revision  int           `json:"revision"`
	Freshness string        `json:"freshness,omitempty"`
	Requires  []string      `json:"requires"`
	Evidence  []EvidenceRef `json:"evidence,omitempty"`
	// SupersededBy names the accepted claim that replaced this one. A superseded
	// claim must never be used to promote readiness (W15.8).
	SupersededBy string `json:"superseded_by,omitempty"`
	// Contradicts names accepted claims this statement contradicts. An unresolved
	// contradiction blocks readiness (W15.8).
	Contradicts []string `json:"contradicts,omitempty"`
}

// RequirementCoverage is the per-requirement result.
type RequirementCoverage struct {
	RequirementID string   `json:"requirement_id"`
	Knowledge     string   `json:"knowledge,omitempty"`
	SatisfiedBy   []string `json:"satisfied_by,omitempty"`
	EvidenceCount int      `json:"evidence_count"`
	Findings      []string `json:"findings,omitempty"`
}

// SemanticFinding is a typed, blocking-or-informational result.
type SemanticFinding struct {
	Kind          string `json:"kind"`
	RequirementID string `json:"requirement_id,omitempty"`
	ClaimID       string `json:"claim_id,omitempty"`
	EvidenceID    string `json:"evidence_id,omitempty"`
	Detail        string `json:"detail"`
}

// SemanticCoverage is the deterministic result of the semantic evaluator.
type SemanticCoverage struct {
	Mode         CoverageMode
	State        CoverageState
	Requirements []RequirementCoverage
	Findings     []SemanticFinding
}

// Blocking reports whether any finding prevents an authoritative ready state.
func (s SemanticCoverage) Blocking() bool { return len(s.Findings) > 0 }

// MissingKnowledge lists the knowledge items whose requirements are unmet.
func (s SemanticCoverage) MissingKnowledge() []string {
	var out []string
	for _, r := range s.Requirements {
		if len(r.Findings) == 0 {
			continue
		}
		if r.Knowledge != "" {
			out = append(out, r.Knowledge)
		} else {
			out = append(out, r.RequirementID)
		}
	}
	return out
}

// evaluateSemantic resolves required knowledge to requirements, requires to
// accepted claims, and claims to verified evidence bound to the current
// revision. It is the authoritative readiness path (W15.4).
func evaluateSemantic(root string, c Contract, b Binding) SemanticCoverage {
	sc := SemanticCoverage{Mode: ModeSemantic}
	claimsByRequirement := map[string][]ClaimBinding{}
	acceptedClaims := map[string]bool{}
	for _, claim := range b.Claims {
		if claim.Status == "accepted" {
			acceptedClaims[claim.ID] = true
		}
	}
	for _, claim := range b.Claims {
		for _, requirement := range claim.Requires {
			claimsByRequirement[requirement] = append(claimsByRequirement[requirement], claim)
		}
	}

	for _, knowledge := range c.RequiredKnowledge {
		requirementID := strings.TrimSpace(b.RequirementMap[knowledge])
		result := RequirementCoverage{RequirementID: requirementID, Knowledge: knowledge}
		if requirementID == "" {
			result.Findings = []string{"unmapped-requirement"}
			sc.Findings = append(sc.Findings, SemanticFinding{
				Kind: "unmapped-requirement", Detail: "no requirement ID mapped for required knowledge " + strconv.Quote(knowledge),
			})
			sc.Requirements = append(sc.Requirements, result)
			continue
		}

		claims := claimsByRequirement[requirementID]
		if len(claims) == 0 {
			result.Findings = append(result.Findings, "missing-claim")
			sc.Findings = append(sc.Findings, SemanticFinding{
				Kind: "missing-claim", RequirementID: requirementID,
				Detail: "no claim satisfies requirement " + requirementID,
			})
		}

		verifiedClaims := 0
		for _, claim := range claims {
			if claim.Status != "accepted" {
				result.Findings = append(result.Findings, "claim-not-accepted")
				sc.Findings = append(sc.Findings, SemanticFinding{
					Kind: "claim-not-accepted", RequirementID: requirementID, ClaimID: claim.ID,
					Detail: "claim status is " + strconv.Quote(claim.Status),
				})
				continue
			}
			if claim.SupersededBy != "" && acceptedClaims[claim.SupersededBy] {
				result.Findings = append(result.Findings, "superseded-claim")
				sc.Findings = append(sc.Findings, SemanticFinding{
					Kind: "superseded-claim", RequirementID: requirementID, ClaimID: claim.ID,
					Detail: "claim was superseded by " + claim.SupersededBy,
				})
				continue
			}
			contradiction := ""
			for _, other := range claim.Contradicts {
				if acceptedClaims[other] {
					contradiction = other
					break
				}
			}
			if contradiction != "" {
				result.Findings = append(result.Findings, "contradiction")
				sc.Findings = append(sc.Findings, SemanticFinding{
					Kind: "contradiction", RequirementID: requirementID, ClaimID: claim.ID,
					Detail: "claim contradicts accepted claim " + contradiction,
				})
				continue
			}
			if claim.Authority == "" || claim.Authority == "external" {
				result.Findings = append(result.Findings, "unauthorised-claim")
				sc.Findings = append(sc.Findings, SemanticFinding{
					Kind: "unauthorised-claim", RequirementID: requirementID, ClaimID: claim.ID,
					Detail: "claim authority " + strconv.Quote(claim.Authority) + " is not promotable to readiness",
				})
				continue
			}

			verified := 0
			seenEvidence := map[string]bool{}
			for _, ev := range claim.Evidence {
				evidenceID := strings.TrimSpace(ev.ID)
				if evidenceID == "" {
					// Evidence without a stable id can never prove a claim (W15.6).
					result.Findings = append(result.Findings, "invalid-evidence")
					sc.Findings = append(sc.Findings, SemanticFinding{
						Kind: "invalid-evidence", RequirementID: requirementID, ClaimID: claim.ID,
						Detail: "evidence has no stable id",
					})
					continue
				}
				if ev.Target != claim.ID {
					continue // evidence must name the claim it proves (W15.6)
				}
				if seenEvidence[evidenceID] {
					continue // duplicates never double-count (W15.12)
				}
				if ev.Freshness == "stale" || revisionDrift(ev.Revision, claim.Revision) {
					result.Findings = append(result.Findings, "stale-evidence")
					sc.Findings = append(sc.Findings, SemanticFinding{
						Kind: "stale-evidence", RequirementID: requirementID, ClaimID: claim.ID, EvidenceID: ev.ID,
						Detail: "evidence does not prove revision " + strconv.Itoa(claim.Revision),
					})
					continue
				}
				if !ev.Verified {
					result.Findings = append(result.Findings, "unverified-claim")
					sc.Findings = append(sc.Findings, SemanticFinding{
						Kind: "unverified-claim", RequirementID: requirementID, ClaimID: claim.ID, EvidenceID: ev.ID,
						Detail: "evidence is not marked verified",
					})
					continue
				}
				if artifact := strings.TrimSpace(ev.Artifact); artifact != "" && !artifactExists(root, artifact) {
					result.Findings = append(result.Findings, "missing-artifact")
					sc.Findings = append(sc.Findings, SemanticFinding{
						Kind: "missing-artifact", RequirementID: requirementID, ClaimID: claim.ID, EvidenceID: ev.ID,
						Detail: "evidence artifact does not exist: " + artifact,
					})
					continue
				}
				if artifact := strings.TrimSpace(ev.Artifact); artifact != "" {
					if ev.ArtifactDigest != "" && artifactDigest(root, artifact) != ev.ArtifactDigest {
						result.Findings = append(result.Findings, "stale-evidence")
						sc.Findings = append(sc.Findings, SemanticFinding{
							Kind: "stale-evidence", RequirementID: requirementID, ClaimID: claim.ID, EvidenceID: ev.ID,
							Detail: "artifact digest changed since the evidence was recorded: " + artifact,
						})
						continue
					}
					if ev.Type == "canonical-document" && !documentationRole(root, artifact) {
						result.Findings = append(result.Findings, "non-canonical-evidence")
						sc.Findings = append(sc.Findings, SemanticFinding{
							Kind: "non-canonical-evidence", RequirementID: requirementID, ClaimID: claim.ID, EvidenceID: ev.ID,
							Detail: "documentation evidence must be canonical or a projection, not an archived record: " + artifact,
						})
						continue
					}
				}
				seenEvidence[evidenceID] = true
				verified++
			}
			if verified > 0 {
				verifiedClaims++
				result.EvidenceCount += verified
			} else if !contains(result.Findings, "unverified-claim") && !contains(result.Findings, "stale-evidence") {
				result.Findings = append(result.Findings, "unverified-claim")
				sc.Findings = append(sc.Findings, SemanticFinding{
					Kind: "unverified-claim", RequirementID: requirementID, ClaimID: claim.ID,
					Detail: "claim has no verified evidence bound to it",
				})
			}
		}

		if verifiedClaims > 0 {
			result.SatisfiedBy = append(result.SatisfiedBy, claims[0].ID)
		}
		sort.Strings(result.Findings)
		sc.Requirements = append(sc.Requirements, result)
	}

	sc.State = semanticState(sc.Requirements)
	sort.Slice(sc.Findings, func(i, j int) bool {
		if sc.Findings[i].RequirementID != sc.Findings[j].RequirementID {
			return sc.Findings[i].RequirementID < sc.Findings[j].RequirementID
		}
		return sc.Findings[i].Kind < sc.Findings[j].Kind
	})
	return sc
}

// semanticState maps findings to exactly one state. Unbound obligations are
// Partial; bound-but-unproven obligations are Unverified. Neither passes.
func semanticState(requirements []RequirementCoverage) CoverageState {
	if len(requirements) == 0 {
		return NotApplicable
	}
	unsatisfied := 0
	unbound := 0
	for _, r := range requirements {
		if len(r.Findings) == 0 {
			continue
		}
		unsatisfied++
		if contains(r.Findings, "missing-claim") || contains(r.Findings, "unmapped-requirement") {
			unbound++
		}
	}
	if unsatisfied == 0 {
		return Verified
	}
	if unbound > 0 {
		return Partial
	}
	return Unverified
}

// evaluateSemanticContract adapts the semantic evaluator to the public coverage
// model used by audit, readiness and the CLI.
func evaluateSemanticContract(root string, c Contract, b Binding) Coverage {
	sc := evaluateSemantic(root, c, b)
	return Coverage{
		ContractID:        c.ID,
		State:             sc.State,
		Mode:              ModeSemantic,
		Authoritative:     true,
		Sources:           append([]string{}, b.Sources...),
		MissingKnowledge:  sc.MissingKnowledge(),
		BlockingQuestions: unresolvedQuestions(c, b),
		Requirements:      sc.Requirements,
		Findings:          sc.Findings,
		Warnings:          []string{},
	}
}

// revisionDrift reports evidence that targets an older revision. An empty
// evidence revision is accepted (the producer did not record one).
func revisionDrift(evidenceRevision string, claimRevision int) bool {
	evidenceRevision = strings.TrimSpace(evidenceRevision)
	if evidenceRevision == "" || claimRevision <= 0 {
		return false
	}
	return evidenceRevision != strconv.Itoa(claimRevision)
}

// artifactDigest is the content digest used to detect evidence drift.
func artifactDigest(root, artifact string) string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact)))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])[:32]
}

// documentationRole reports whether a documentation artifact can prove current
// state. Historical records cannot (W15.9), so they are rejected; an artifact
// the authority map does not classify is left to the existence and digest
// checks rather than being condemned by inference.
func documentationRole(root, artifact string) bool {
	switch strings.ToLower(filepath.Ext(artifact)) {
	case ".md", ".mdx", ".json":
	default:
		return true // non-documentation evidence keeps its own verification path
	}
	role, ok := RoleOf(root, artifact)
	if !ok {
		return true
	}
	return role == RoleCanonical || role == RoleProjection
}

func artifactExists(root, artifact string) bool {
	if strings.Contains(artifact, "://") {
		return true
	}
	_, err := os.Stat(filepath.Join(root, artifact))
	return err == nil
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// semantic reports whether the binding carries semantic data. Without it a
// binding can only be evaluated lexically, which is never authoritative.
func (b Binding) semantic() bool {
	return len(b.RequirementMap) > 0 || len(b.Claims) > 0
}
