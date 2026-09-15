// Documentation planning (W17.5–W17.11): a behavior-changing Goal gets a
// replayable preflight plan before implementation and is reconciled against the
// real impact afterwards. Plans are derived runtime state, never canonical
// documents.
package docengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PlanRoot is the derived directory holding Goal documentation plans.
const PlanRoot = ".prumo/runtime/docs/plans"

// Obligation kinds. A plan states what a change owes to documentation and to
// every downstream projection.
const (
	ObligationDocumentUpdate = "document-update"
	ObligationSchema         = "schema"
	ObligationAgentSurface   = "agent-surface"
	ObligationTranslation    = "translation"
	ObligationMedia          = "media"
	ObligationEvidence       = "evidence"
)

// PlanObligation is one predicted documentation obligation.
type PlanObligation struct {
	ID                  string   `json:"id"`
	ContractID          string   `json:"contract_id,omitempty"`
	Kind                string   `json:"kind"`
	Target              []string `json:"target"`
	Applicable          bool     `json:"applicable"`
	NotApplicableReason string   `json:"not_applicable_reason,omitempty"`
	Reason              string   `json:"reason"`
}

// DocumentationPlan is the preflight result for one Goal.
type DocumentationPlan struct {
	Goal         string            `json:"goal"`
	CreatedAt    string            `json:"created_at"`
	Changed      []string          `json:"changed"`
	Obligations  []PlanObligation  `json:"obligations"`
	Findings     []SemanticFinding `json:"findings"`
	PredictedSet []string          `json:"predicted_contracts"`
}

// PlanReconciliation diffs the preflight prediction against the real impact.
type PlanReconciliation struct {
	Goal        string            `json:"goal"`
	Predicted   []string          `json:"predicted"`
	Actual      []string          `json:"actual"`
	Missing     []string          `json:"missing"`
	Unpredicted []string          `json:"unpredicted"`
	DriftRatio  float64           `json:"drift_ratio"`
	Findings    []SemanticFinding `json:"findings"`
}

// BuildPlan computes the preflight plan for a Goal from the semantic impact of
// the changed paths plus the projections the change implies.
func BuildPlan(root, goal string, changed []string, now string) (DocumentationPlan, error) {
	registry, err := LoadRegistry(root)
	if err != nil {
		return DocumentationPlan{}, err
	}
	bindings, err := LoadBindings(root)
	if err != nil {
		return DocumentationPlan{}, err
	}
	impacts := AnalyzeImpactsIn(root, registry, bindings, changed, nil)
	plan := DocumentationPlan{Goal: goal, CreatedAt: now, Changed: append([]string{}, changed...),
		Obligations: []PlanObligation{}, Findings: []SemanticFinding{}, PredictedSet: []string{}}

	for _, impact := range impacts {
		plan.PredictedSet = append(plan.PredictedSet, impact.ContractID)
		plan.Obligations = append(plan.Obligations, PlanObligation{
			ID:         "doc:" + impact.ContractID,
			ContractID: impact.ContractID,
			Kind:       ObligationDocumentUpdate,
			Target:     append([]string{}, impact.Documents...),
			Applicable: true,
			Reason:     impact.Reason,
		})
	}
	plan.Obligations = append(plan.Obligations, impliedObligations(root, changed)...)
	plan.Obligations = append(plan.Obligations, notApplicableObligations(changed)...)
	sort.Slice(plan.Obligations, func(i, j int) bool { return plan.Obligations[i].ID < plan.Obligations[j].ID })
	sort.Strings(plan.PredictedSet)
	plan.Findings = plan.Findings[:0]
	plan.Findings = append(plan.Findings, plan.Validate()...)
	return plan, nil
}

// BuildPlanForContracts is the Goal-lock preflight (W17.7): it states the
// documentation obligations implied by the contracts a scope's accepted
// decisions touch, before any implementation path exists.
func BuildPlanForContracts(root, goal string, contracts []string, now string) (DocumentationPlan, error) {
	bindings, err := LoadBindings(root)
	if err != nil {
		return DocumentationPlan{}, err
	}
	sources := map[string][]string{}
	for _, b := range bindings {
		sources[b.ContractID] = append(sources[b.ContractID], b.Sources...)
	}
	seen := map[string]bool{}
	plan := DocumentationPlan{Goal: goal, CreatedAt: now, Changed: []string{},
		Obligations: []PlanObligation{}, Findings: []SemanticFinding{}, PredictedSet: []string{}}
	for _, contract := range contracts {
		contract = strings.TrimSpace(contract)
		if contract == "" || seen[contract] {
			continue
		}
		seen[contract] = true
		targets := append([]string{}, sources[contract]...)
		sort.Strings(targets)
		kind, reason := ObligationDocumentUpdate, "accepted decisions affect contract "+contract
		if strings.HasPrefix(contract, "ui.") || strings.HasPrefix(contract, "tui.") || strings.HasPrefix(contract, "accessibility.") {
			kind = ObligationDocumentUpdate
		}
		plan.Obligations = append(plan.Obligations, PlanObligation{
			ID: "doc:" + contract, ContractID: contract, Kind: kind, Target: targets,
			Applicable: true, Reason: reason,
		})
		plan.PredictedSet = append(plan.PredictedSet, contract)
	}
	if len(plan.Obligations) == 0 {
		plan.Obligations = append(plan.Obligations, PlanObligation{
			ID: "doc:none", Kind: ObligationDocumentUpdate, Applicable: false,
			NotApplicableReason: "no accepted decision affects a documented contract",
			Reason:              "empty affected-contract set",
		})
	}
	sort.Strings(plan.PredictedSet)
	sort.Slice(plan.Obligations, func(i, j int) bool { return plan.Obligations[i].ID < plan.Obligations[j].ID })
	plan.Findings = append(plan.Findings, plan.Validate()...)
	return plan, nil
}

// impliedObligations derives non-document obligations from the shape of the
// change itself: schemas, agent instruction sources, locales and media.
func impliedObligations(root string, changed []string) []PlanObligation {
	var out []PlanObligation
	schemaPaths, localePaths, mediaPaths, agentPaths := []string{}, []string{}, []string{}, []string{}
	for _, path := range changed {
		switch {
		case strings.HasPrefix(path, "schemas/") && strings.HasSuffix(path, ".schema.json"):
			schemaPaths = append(schemaPaths, path)
		case strings.Contains(strings.ToLower(path), "translations") || strings.HasSuffix(path, ".i18n.json"):
			localePaths = append(localePaths, path)
		case strings.HasPrefix(path, "docs/media/") || strings.HasSuffix(path, ".media.json"):
			mediaPaths = append(mediaPaths, path)
		case path == "docs/agents/instruction-ir.json" || strings.HasSuffix(path, "AGENTS.md"):
			agentPaths = append(agentPaths, path)
		}
	}
	add := func(id, kind, reason string, targets []string) {
		if len(targets) == 0 {
			return
		}
		out = append(out, PlanObligation{ID: id, Kind: kind, Target: targets, Applicable: true, Reason: reason})
	}
	add("schema:contracts", ObligationSchema, "machine contract changed", schemaPaths)
	add("translation:delta", ObligationTranslation, "localized surfaces follow their source", localePaths)
	add("media:delta", ObligationMedia, "media records follow the UI state they document", mediaPaths)
	add("agent:surfaces", ObligationAgentSurface, "agent instruction knowledge changed", agentPaths)
	return out
}

// notApplicableObligations records the obligations a change explicitly does not
// owe, each with a reason (W17.11). Silent N/A is forbidden.
func notApplicableObligations(changed []string) []PlanObligation {
	if len(changed) == 0 {
		return []PlanObligation{{
			ID: "doc:none", Kind: ObligationDocumentUpdate, Applicable: false,
			NotApplicableReason: "no behavior-changing paths were supplied to the preflight",
			Reason:              "empty change set",
		}}
	}
	return nil
}

// Validate enforces the plan invariants: unique IDs, a reason for every
// obligation, and an explicit reason for every N/A.
func (p DocumentationPlan) Validate() []SemanticFinding {
	seen := map[string]bool{}
	var findings []SemanticFinding
	for _, o := range p.Obligations {
		if strings.TrimSpace(o.ID) == "" {
			findings = append(findings, SemanticFinding{Kind: "plan-missing-id", Detail: "obligation has no id"})
			continue
		}
		if seen[o.ID] {
			findings = append(findings, SemanticFinding{Kind: "plan-duplicate-obligation", Detail: "duplicate obligation " + o.ID})
		}
		seen[o.ID] = true
		if strings.TrimSpace(o.Reason) == "" && o.Applicable {
			findings = append(findings, SemanticFinding{Kind: "plan-missing-reason", Detail: "applicable obligation " + o.ID + " has no reason"})
		}
		if !o.Applicable && strings.TrimSpace(o.NotApplicableReason) == "" {
			findings = append(findings, SemanticFinding{Kind: "plan-silent-not-applicable", Detail: "obligation " + o.ID + " is marked not applicable without a reason"})
		}
	}
	if strings.TrimSpace(p.Goal) == "" {
		findings = append(findings, SemanticFinding{Kind: "plan-missing-goal", Detail: "a documentation plan requires a goal"})
	}
	return findings
}

// Reconcile compares the preflight prediction with the impact actually
// observed after implementation (W17.8, W17.9).
func Reconcile(plan DocumentationPlan, actual []Impact) PlanReconciliation {
	predicted := map[string]bool{}
	for _, id := range plan.PredictedSet {
		predicted[id] = true
	}
	actualSet := map[string]bool{}
	for _, impact := range actual {
		actualSet[impact.ContractID] = true
	}
	out := PlanReconciliation{Goal: plan.Goal, Predicted: append([]string{}, plan.PredictedSet...), Actual: []string{},
		Missing: []string{}, Unpredicted: []string{}, Findings: []SemanticFinding{}}
	for id := range actualSet {
		out.Actual = append(out.Actual, id)
	}
	sort.Strings(out.Actual)
	for id := range predicted {
		if !actualSet[id] {
			out.Missing = append(out.Missing, id)
		}
	}
	for id := range actualSet {
		if !predicted[id] {
			out.Unpredicted = append(out.Unpredicted, id)
		}
	}
	sort.Strings(out.Missing)
	sort.Strings(out.Unpredicted)
	if len(out.Actual) > 0 {
		out.DriftRatio = float64(len(out.Missing)+len(out.Unpredicted)) / float64(len(out.Actual))
	}
	for _, id := range out.Missing {
		out.Findings = append(out.Findings, SemanticFinding{
			Kind: "plan-underpredicted", Detail: "contract " + id + " was impacted but not planned",
		})
	}
	for _, id := range out.Unpredicted {
		out.Findings = append(out.Findings, SemanticFinding{
			Kind: "plan-overpredicted", Detail: "contract " + id + " was planned but not impacted",
		})
	}
	return out
}

// PlanPath is where a Goal's plan is stored.
func PlanPath(root, goal string) string {
	return filepath.Join(root, filepath.FromSlash(PlanRoot), slugify(goal)+".json")
}

// SavePlan writes the plan as derived runtime state.
func SavePlan(root string, plan DocumentationPlan) error {
	path := PlanPath(root, plan.Goal)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// LoadPlan reads a stored Goal plan.
func LoadPlan(root, goal string) (DocumentationPlan, error) {
	data, err := os.ReadFile(PlanPath(root, goal))
	if err != nil {
		return DocumentationPlan{}, err
	}
	var plan DocumentationPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return DocumentationPlan{}, err
	}
	return plan, nil
}

// PlanImpacts converts the plan's document obligations into impacts so the
// Goal/Plan integration can report the documentation gap (W17.6).
func (p DocumentationPlan) PlanImpacts() []Impact {
	var out []Impact
	for _, o := range p.Obligations {
		if !o.Applicable || o.Kind != ObligationDocumentUpdate {
			continue
		}
		out = append(out, Impact{ContractID: o.ContractID, Documents: append([]string{}, o.Target...),
			Reason: "documentation plan: " + o.Reason, Severity: "medium"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ContractID < out[j].ContractID })
	return out
}

// Explain renders a human-readable explanation of one plan obligation.
func (p DocumentationPlan) Explain(id string) (string, error) {
	for _, o := range p.Obligations {
		if o.ID != id && o.ContractID != id {
			continue
		}
		lines := []string{
			"obligation: " + o.ID,
			"kind: " + o.Kind,
			"contract: " + o.ContractID,
			"applicable: " + fmt.Sprint(o.Applicable),
			"reason: " + o.Reason,
			"targets: " + strings.Join(o.Target, ", "),
		}
		if !o.Applicable {
			lines = append(lines, "not-applicable reason: "+o.NotApplicableReason)
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("no obligation matches %q", id)
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "goal"
	}
	return out
}
