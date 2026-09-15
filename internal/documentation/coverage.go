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

type Binding struct {
	ContractID        string   `json:"contract_id"`
	Sources           []string `json:"sources"`
	Ownership         string   `json:"ownership"`
	Authority         string   `json:"authority"`
	AnsweredQuestions []string `json:"answered_questions"`
	Evidence          []string `json:"evidence"`
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
	Profiles            []string   `json:"profiles"`
	ApplicableContracts []string   `json:"applicable_contracts"`
	SemanticContracts   int        `json:"semantic_contracts"`
	LexicalContracts    int        `json:"lexical_contracts"`
	Coverage            []Coverage `json:"coverage"`
	Warnings            []string   `json:"warnings"`
}
type ReadinessReport struct {
	Ready               bool       `json:"ready"`
	Goal                string     `json:"goal,omitempty"`
	BlockingContracts   []string   `json:"blocking_contracts"`
	UnverifiedContracts []string   `json:"unverified_contracts"`
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
func ResolveProfiles(root string, registry Registry) ([]string, map[string]bool) {
	cap := Capabilities(root)
	profiles := []string{"core-software"}
	for _, pair := range []struct{ cap, profile string }{{"cli", "cli"}, {"web-application", "web-application"}, {"desktop-gui", "desktop-gui"}, {"api-service", "api-service"}, {"compiler", "compiler"}, {"library", "library"}} {
		if cap[pair.cap] {
			profiles = append(profiles, pair.profile)
		}
	}
	return profiles, cap
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
	report := AuditReport{Profiles: profiles, Warnings: []string{}}
	for _, id := range unknown {
		report.Warnings = append(report.Warnings, "unknown contract: "+id)
	}
	for _, c := range contracts {
		report.ApplicableContracts = append(report.ApplicableContracts, c.ID)
		coverage := evaluate(root, c, bound[c.ID])
		if coverage.Authoritative {
			report.SemanticContracts++
		} else {
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
// present but semantically unproven (unverified) (W15.4).
func applyCoverage(report *ReadinessReport, c Coverage) {
	switch c.State {
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
