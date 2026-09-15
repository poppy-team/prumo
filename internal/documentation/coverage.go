package docengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CoverageState string

const (
	Missing             CoverageState = "missing"
	Partial             CoverageState = "partial"
	ImplementationReady CoverageState = "implementation-ready"
	Verified            CoverageState = "verified"
	Stale               CoverageState = "stale"
	NotApplicable       CoverageState = "not-applicable"
	// Unverified means the obligation is structurally present, but only a
	// lexical word match backs it. It is never a passing state (W15.5).
	Unverified CoverageState = "unverified"
)

// Applicability is the closed vocabulary for whether a contract is in force.
// There is deliberately no third value: a contract is either owed, or declared
// not owed with a reason. "Unknown" would be indistinguishable from forgotten.
const (
	ApplicabilityApplicable    = "applicable"
	ApplicabilityNotApplicable = "not-applicable"
)

type Binding struct {
	ContractID        string   `json:"contract_id"`
	Sources           []string `json:"sources"`
	Ownership         string   `json:"ownership"`
	Authority         string   `json:"authority"`
	AnsweredQuestions []string `json:"answered_questions"`
	Evidence          []string `json:"evidence"`
	// Applicability declares whether this contract is owed by the project right
	// now. Empty means applicable, so an existing binding cannot become
	// inapplicable by omission.
	Applicability string `json:"applicability,omitempty"`
	// NotApplicableReason is required when Applicability is not-applicable. An
	// obligation may be waived, but never silently: the reason is what a
	// reviewer reads instead of re-deriving why a gate went quiet.
	NotApplicableReason string `json:"not_applicable_reason,omitempty"`
	// RequirementMap maps a required-knowledge phrase to its stable requirement
	// ID. Claims reference those IDs, so readiness resolves knowledge through
	// typed relations instead of substring matching (W15.3).
	RequirementMap map[string]string `json:"requirement_map,omitempty"`
	// Claims carry the durable statements and their bound evidence that prove a
	// requirement at a revision (W15.3, W15.6).
	Claims []ClaimBinding `json:"claims,omitempty"`
}

type Coverage struct {
	ContractID        string                `json:"contract_id"`
	State             CoverageState         `json:"state"`
	Mode              CoverageMode          `json:"mode"`
	Authoritative     bool                  `json:"authoritative"`
	Sources           []string              `json:"sources"`
	MissingKnowledge  []string              `json:"missing_knowledge"`
	BlockingQuestions []string              `json:"blocking_questions"`
	Requirements      []RequirementCoverage `json:"requirements,omitempty"`
	Findings          []SemanticFinding     `json:"findings"`
	Warnings          []string              `json:"warnings"`
}
type AuditReport struct {
	Profiles               []string   `json:"profiles"`
	ApplicableContracts    []string   `json:"applicable_contracts"`
	SemanticContracts      int        `json:"semantic_contracts"`
	LexicalContracts       int        `json:"lexical_contracts"`
	NotApplicableContracts []string   `json:"not_applicable_contracts"`
	Coverage               []Coverage `json:"coverage"`
	Warnings               []string   `json:"warnings"`
}
type ReadinessReport struct {
	Ready               bool       `json:"ready"`
	Goal                string     `json:"goal,omitempty"`
	BlockingContracts   []string   `json:"blocking_contracts"`
	UnverifiedContracts []string   `json:"unverified_contracts"`
	NotApplicable       []string   `json:"not_applicable_contracts"`
	BlockingQuestions   []string   `json:"blocking_questions"`
	Warnings            []string   `json:"warnings"`
	ApplicableContracts []string   `json:"applicable_contracts"`
	Coverage            []Coverage `json:"coverage"`
}

func LoadBindings(root string) ([]Binding, error) {
	data, err := os.ReadFile(filepath.Join(root, "docs", "contracts", "bindings.json"))
	if err != nil {
		return nil, err
	}
	var bindings []Binding
	if err := json.Unmarshal(data, &bindings); err != nil {
		return nil, err
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].ContractID < bindings[j].ContractID })
	return bindings, nil
}
func Capabilities(root string) map[string]bool {
	capabilities := map[string]bool{"core": true}
	if data, err := os.ReadFile(filepath.Join(root, "prumo.json")); err == nil {
		var project map[string]any
		if json.Unmarshal(data, &project) == nil {
			if features, ok := project["features"].([]any); ok {
				for _, f := range features {
					capabilities[fmt.Sprint(f)] = true
				}
			}
			if info, ok := project["project"].(map[string]any); ok {
				if types, ok := info["type"].([]any); ok {
					for _, v := range types {
						capabilities[fmt.Sprint(v)] = true
					}
				}
			}
		}
	}
	for _, pair := range []struct{ path, cap string }{{"cmd", "cli"}, {"schemas", "schema"}, {"go.mod", "go"}} {
		if _, err := os.Stat(filepath.Join(root, pair.path)); err == nil {
			capabilities[pair.cap] = true
		}
	}
	return capabilities
}

// capabilityProfiles maps an auto-detected capability to the documentation
// profile it composes. Every profile declared in the builtin registry must
// appear here: a capability that resolves to no profile silently drops every
// contract that profile carries, which is invisible until someone asks why a
// gate never ran. `tui` was missing exactly that way.
var capabilityProfiles = []struct{ capability, profile string }{
	{"cli", "cli"},
	{"web-application", "web-application"},
	{"desktop-gui", "desktop-gui"},
	{"tui", "tui"},
	{"api-service", "api-service"},
	{"compiler", "compiler"},
	{"library", "library"},
}

// ResolveProfiles reports which documentation profiles a repository composes and
// the effective capability set. A composed profile contributes its declared
// capabilities: the `tui` profile declares both `tui` and `ui`, and a contract
// whose applicability names `ui` has to become applicable once that profile is
// selected — otherwise the profile selects its contracts and the capability
// filter removes them again, which looks identical to having no UI gate at all.
func ResolveProfiles(root string, registry Registry) ([]string, map[string]bool) {
	capabilities := Capabilities(root)
	profiles := []string{"core-software"}
	for _, pair := range capabilityProfiles {
		if capabilities[pair.capability] {
			profiles = append(profiles, pair.profile)
		}
	}
	for _, id := range profiles {
		profile, ok := registry.Profiles[id]
		if !ok {
			continue
		}
		for _, capability := range profile.Capabilities {
			capabilities[capability] = true
		}
	}
	return profiles, capabilities
}
func Audit(root string) (AuditReport, error) {
	registry, err := LoadRegistry(root)
	if err != nil {
		return AuditReport{}, err
	}
	bindings, err := LoadBindings(root)
	if err != nil {
		return AuditReport{}, err
	}
	profiles, cap := ResolveProfiles(root, registry)
	contracts, unknown, err := registry.ResolveProfiles(profiles, cap)
	if err != nil {
		return AuditReport{}, err
	}
	bound := map[string]Binding{}
	for _, b := range bindings {
		bound[b.ContractID] = b
	}
	report := AuditReport{Profiles: profiles, NotApplicableContracts: []string{}, Warnings: []string{}}
	for _, id := range unknown {
		report.Warnings = append(report.Warnings, "unknown contract: "+id)
	}
	for _, c := range contracts {
		report.ApplicableContracts = append(report.ApplicableContracts, c.ID)
		coverage := evaluate(root, c, bound[c.ID])
		switch {
		case coverage.State == NotApplicable:
			// A waived obligation is neither semantic nor lexical: counting it
			// as lexical would inflate the number of contracts a reviewer still
			// has to bind.
			report.NotApplicableContracts = append(report.NotApplicableContracts, c.ID)
		case coverage.Authoritative:
			report.SemanticContracts++
		default:
			report.LexicalContracts++
		}
		report.Coverage = append(report.Coverage, coverage)
	}
	return report, nil
}

// evaluate computes coverage for one contract. A binding is authoritative only
// when it is semantic: required knowledge mapped to requirement IDs and claims
// carrying bound evidence. Legacy lexical bindings are still evaluated, but can
// never report a ready state — a word match is not readiness (W15.4, W15.5).
func evaluate(root string, c Contract, b Binding) Coverage {
	if coverage, ok := evaluateNotApplicable(c, b); ok {
		return coverage
	}
	if b.semantic() {
		return evaluateSemanticContract(root, c, b)
	}
	coverage := evaluateLexical(root, c, b)
	coverage.Mode = ModeLexical
	coverage.Authoritative = false
	coverage.Requirements = []RequirementCoverage{}
	switch coverage.State {
	case ImplementationReady, Verified:
		coverage.State = Unverified
		coverage.Warnings = append(coverage.Warnings,
			"lexical match only: bind requirement_map and claims with bound evidence to make this contract authoritative (W15)")
	}
	return coverage
}

// evaluateNotApplicable handles a contract the project has declared it does not
// owe yet. It fails closed on a missing reason: an unexplained waiver is a
// forgotten gate, and reporting it as satisfied would hide exactly that.
//
// ok is false when the contract is applicable (the caller continues normally).
func evaluateNotApplicable(c Contract, b Binding) (Coverage, bool) {
	if b.Applicability != ApplicabilityNotApplicable {
		return Coverage{}, false
	}
	coverage := Coverage{
		ContractID: c.ID, State: NotApplicable, Mode: ModeLexical,
		Sources:  append([]string{}, b.Sources...),
		Findings: []SemanticFinding{}, Warnings: []string{},
	}
	reason := strings.TrimSpace(b.NotApplicableReason)
	if reason == "" {
		coverage.State = Missing
		coverage.BlockingQuestions = unresolvedQuestions(c, b)
		coverage.Findings = append(coverage.Findings, SemanticFinding{
			Kind: "not-applicable-without-reason",
			Detail: "a contract declared not-applicable must state why, so a waiver " +
				"can be reviewed instead of re-derived",
		})
		return coverage, true
	}
	coverage.Warnings = append(coverage.Warnings, "not applicable: "+reason)
	return coverage, true
}

// evaluateLexical is the legacy diagnostic matcher. It is retained for
// migration visibility and never asserts readiness on its own.
func evaluateLexical(root string, c Contract, b Binding) Coverage {
	coverage := Coverage{
		ContractID:        c.ID,
		Sources:           append([]string{}, b.Sources...),
		BlockingQuestions: unresolvedQuestions(c, b),
		Findings:          []SemanticFinding{}, Warnings: []string{},
	}
	if len(b.Sources) == 0 {
		coverage.State = Missing
		return coverage
	}
	content := ""
	for _, s := range b.Sources {
		if data, err := os.ReadFile(filepath.Join(root, s)); err == nil {
			content += "\n" + string(data)
		}
	}
	content = strings.ToLower(content)
	for _, k := range c.RequiredKnowledge {
		if !matchesKnowledgeRequirement(content, k) {
			coverage.MissingKnowledge = append(coverage.MissingKnowledge, k)
		}
	}
	if len(coverage.MissingKnowledge) == 0 {
		coverage.State = ImplementationReady
		if len(c.EvidenceRequirements) > 0 {
			nonEmpty := 0
			seen := make(map[string]bool)
			for _, ev := range b.Evidence {
				trimmed := strings.TrimSpace(ev)
				if trimmed != "" && !seen[trimmed] {
					seen[trimmed] = true
					nonEmpty++
				}
			}
			if nonEmpty >= len(c.EvidenceRequirements) {
				coverage.State = Verified
			}
		}
	} else {
		coverage.State = Partial
	}
	return coverage
}

// unresolvedQuestions lists contract questions the binding has not answered.
// It applies to both the lexical and the semantic path.
func unresolvedQuestions(c Contract, b Binding) []string {
	answered := map[string]bool{}
	for _, question := range b.AnsweredQuestions {
		answered[question] = true
	}
	questions := []string{}
	for _, question := range c.BlockingQuestions {
		if !answered[question] {
			questions = append(questions, question)
		}
	}
	return questions
}

var defaultStopwords = map[string]bool{
	"and": true, "or": true, "in": true, "of": true,
	"to": true, "for": true, "with": true, "a": true,
	"an": true, "the": true, "by": true, "on": true,
}

func matchesKnowledgeRequirement(content, req string) bool {
	reqLower := strings.ToLower(strings.TrimSpace(req))
	if reqLower == "" {
		return true
	}
	if strings.Contains(content, reqLower) {
		return true
	}
	words := strings.Fields(reqLower)
	keyWords := make([]string, 0, len(words))
	for _, w := range words {
		cleaned := strings.Trim(w, ".,:;()[]\"'/-")
		if cleaned != "" && !defaultStopwords[cleaned] {
			keyWords = append(keyWords, cleaned)
		}
	}
	if len(keyWords) == 0 {
		return strings.Contains(content, reqLower)
	}
	for _, kw := range keyWords {
		if !strings.Contains(content, kw) {
			return false
		}
	}
	return true
}

// applyCoverage folds one coverage result into a readiness report. Blocking
// states are those where an obligation is unmet (missing/partial/stale) or
// present but semantically unproven (unverified) (W15.4). NotApplicable is
// deliberately absent: a waived contract with a stated reason is not a blocker,
// and the waiver is reported separately so it stays reviewable.
func applyCoverage(report *ReadinessReport, c Coverage) {
	switch c.State {
	case NotApplicable:
		return
	case Missing, Partial, Stale:
		report.Ready = false
		report.BlockingContracts = append(report.BlockingContracts, c.ContractID)
		report.BlockingQuestions = append(report.BlockingQuestions, c.BlockingQuestions...)
	case Unverified:
		report.Ready = false
		report.UnverifiedContracts = append(report.UnverifiedContracts, c.ContractID)
	}
}

func Readiness(root, goal string) (ReadinessReport, error) {
	audit, err := Audit(root)
	if err != nil {
		return ReadinessReport{}, err
	}
	report := ReadinessReport{
		Ready:               true,
		Goal:                goal,
		ApplicableContracts: audit.ApplicableContracts,
		Coverage:            audit.Coverage,
		Warnings:            audit.Warnings,
		UnverifiedContracts: []string{},
		NotApplicable:       audit.NotApplicableContracts,
	}
	for _, c := range audit.Coverage {
		applyCoverage(&report, c)
	}
	if len(report.UnverifiedContracts) > 0 {
		report.Warnings = append(report.Warnings,
			"contracts backed by lexical match only are not authoritative; bind requirement_map and claims with verified evidence (W15)")
	}
	return report, nil
}
