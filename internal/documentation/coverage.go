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
)

type Binding struct {
	ContractID        string   `json:"contract_id"`
	Sources           []string `json:"sources"`
	Ownership         string   `json:"ownership"`
	Authority         string   `json:"authority"`
	AnsweredQuestions []string `json:"answered_questions"`
	Evidence          []string `json:"evidence"`
}
type Coverage struct {
	ContractID        string        `json:"contract_id"`
	State             CoverageState `json:"state"`
	Sources           []string      `json:"sources"`
	MissingKnowledge  []string      `json:"missing_knowledge"`
	BlockingQuestions []string      `json:"blocking_questions"`
	Warnings          []string      `json:"warnings"`
}
type AuditReport struct {
	Profiles            []string   `json:"profiles"`
	ApplicableContracts []string   `json:"applicable_contracts"`
	Coverage            []Coverage `json:"coverage"`
	Warnings            []string   `json:"warnings"`
}
type ReadinessReport struct {
	Ready               bool       `json:"ready"`
	Goal                string     `json:"goal,omitempty"`
	BlockingContracts   []string   `json:"blocking_contracts"`
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
		report.Coverage = append(report.Coverage, evaluate(root, c, bound[c.ID]))
	}
	return report, nil
}
func evaluate(root string, c Contract, b Binding) Coverage {
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
	coverage := Coverage{ContractID: c.ID, Sources: append([]string{}, b.Sources...), BlockingQuestions: questions}
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
func Readiness(root, goal string) (ReadinessReport, error) {
	audit, err := Audit(root)
	if err != nil {
		return ReadinessReport{}, err
	}
	report := ReadinessReport{Ready: true, Goal: goal, ApplicableContracts: audit.ApplicableContracts, Coverage: audit.Coverage, Warnings: audit.Warnings}
	for _, c := range audit.Coverage {
		if c.State == Missing || c.State == Partial || c.State == Stale {
			report.Ready = false
			report.BlockingContracts = append(report.BlockingContracts, c.ContractID)
			report.BlockingQuestions = append(report.BlockingQuestions, c.BlockingQuestions...)
		}
	}
	return report, nil
}
