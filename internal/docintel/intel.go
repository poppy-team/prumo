// Package docintel turns documentation behaviour into measurable project
// intelligence and supports brownfield adoption (W21).
//
// Metrics are computed from repository state, never from tracked user content:
// there is no telemetry channel and no query log. Adoption proposals are
// proposals only; promoting an inferred authority stays a human decision.
package docintel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/doclifecycle"
	"github.com/raillen/prumo/internal/docpublish"
	docengine "github.com/raillen/prumo/internal/documentation"
)

// Metrics is the documentation intelligence projection.
type Metrics struct {
	Pages               int      `json:"pages"`
	HistoricalPages     int      `json:"historical_pages"`
	AgentSurfaces       int      `json:"agent_surfaces"`
	DocumentationTokens int      `json:"documentation_tokens"`
	InstructionTokens   int      `json:"instruction_tokens"`
	RetrievalEntries    int      `json:"retrieval_entries"`
	SemanticContracts   int      `json:"semantic_contracts"`
	LexicalContracts    int      `json:"lexical_contracts"`
	UnverifiedContracts []string `json:"unverified_contracts"`
	StaleTranslations   int      `json:"stale_translations"`
	StaleMedia          int      `json:"stale_media"`
	Deprecations        int      `json:"deprecations"`
	BrokenLinks         int      `json:"broken_links"`
	ClaimDrift          int      `json:"claim_drift"`
	DocumentationDebt   []string `json:"documentation_debt"`
	FreshnessScore      float64  `json:"freshness_score"`
}

// Collect computes documentation metrics from repository state.
func Collect(root string) (Metrics, error) {
	metrics := Metrics{UnverifiedContracts: []string{}, DocumentationDebt: []string{}}

	graph, err := docpublish.LoadWithOptions(root, docpublish.Options{Version: "", Locale: "en"})
	if err != nil {
		return Metrics{}, err
	}
	for _, page := range graph.Pages {
		metrics.Pages++
		metrics.DocumentationTokens += page.Tokens
		if page.Historical {
			metrics.HistoricalPages++
		}
		if page.Audience == docpublish.AudienceAgent {
			metrics.AgentSurfaces++
			metrics.InstructionTokens += page.Tokens
		}
	}
	entries, err := retrievalEntries(root)
	if err != nil {
		return Metrics{}, err
	}
	metrics.RetrievalEntries = entries

	audit, err := docengine.Audit(root)
	if err != nil {
		return Metrics{}, err
	}
	metrics.SemanticContracts = audit.SemanticContracts
	metrics.LexicalContracts = audit.LexicalContracts
	readiness, err := docengine.Readiness(root, "")
	if err != nil {
		return Metrics{}, err
	}
	metrics.UnverifiedContracts = append(metrics.UnverifiedContracts, readiness.UnverifiedContracts...)

	translations, err := doclifecycle.TranslationStatuses(root)
	if err != nil {
		return Metrics{}, err
	}
	for _, status := range translations {
		if status.Stale {
			metrics.StaleTranslations++
		}
	}
	media, err := doclifecycle.MediaStatuses(root)
	if err != nil {
		return Metrics{}, err
	}
	for _, status := range media {
		if status.Stale {
			metrics.StaleMedia++
		}
	}
	lifecycle, err := doclifecycle.LoadLifecycle(root)
	if err != nil {
		return Metrics{}, err
	}
	metrics.Deprecations = len(lifecycle.Deprecated)

	verification, err := docengine.VerifyDocs(root)
	if err != nil {
		return Metrics{}, err
	}
	for _, finding := range verification.Findings {
		switch finding.Kind {
		case "broken-link":
			metrics.BrokenLinks++
		case "claim-drift":
			metrics.ClaimDrift++
		}
	}

	metrics.DocumentationDebt = documentationDebt(metrics)
	metrics.FreshnessScore = freshnessScore(metrics)
	return metrics, nil
}

// documentationDebt names the outstanding documentation obligations, so the
// metrics feed project intelligence instead of being a vanity dashboard.
func documentationDebt(metrics Metrics) []string {
	debt := []string{}
	if len(metrics.UnverifiedContracts) > 0 {
		debt = append(debt, fmt.Sprintf("%d contract(s) have no semantic requirement→claim→evidence binding", len(metrics.UnverifiedContracts)))
	}
	if metrics.StaleTranslations > 0 {
		debt = append(debt, fmt.Sprintf("%d translation unit(s) need an update", metrics.StaleTranslations))
	}
	if metrics.StaleMedia > 0 {
		debt = append(debt, fmt.Sprintf("%d media record(s) are out of date", metrics.StaleMedia))
	}
	if metrics.BrokenLinks > 0 {
		debt = append(debt, fmt.Sprintf("%d broken documentation link(s)", metrics.BrokenLinks))
	}
	if metrics.ClaimDrift > 0 {
		debt = append(debt, fmt.Sprintf("%d claim(s) contradict the implementation", metrics.ClaimDrift))
	}
	sort.Strings(debt)
	return debt
}

// freshnessScore is a bounded 0..1 indicator: 1 means nothing known is stale.
func freshnessScore(metrics Metrics) float64 {
	penalties := 0
	penalties += len(metrics.UnverifiedContracts)
	penalties += metrics.StaleTranslations + metrics.StaleMedia
	penalties += metrics.BrokenLinks + metrics.ClaimDrift
	if penalties == 0 {
		return 1
	}
	denominator := penalties + metrics.Pages
	if denominator == 0 {
		return 0
	}
	score := 1 - float64(penalties)/float64(denominator)
	if score < 0 {
		return 0
	}
	return score
}

func retrievalEntries(root string) (int, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(docpublish.DefaultOutDir), "search-index.json"))
	if err != nil {
		return 0, nil // the index is derived; a missing index is 0, not an error
	}
	var index struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return 0, nil
	}
	return len(index.Entries), nil
}

// AdoptionReport is the brownfield discovery result for a foreign project.
type AdoptionReport struct {
	Root       string          `json:"root"`
	Signals    map[string]bool `json:"signals"`
	DocFiles   []string        `json:"doc_files"`
	Proposals  []Proposal      `json:"proposals"`
	Conflicts  []string        `json:"conflicts"`
	Duplicates []string        `json:"duplicates"`
	Findings   []string        `json:"findings"`
}

// Proposal is one inferred contract/binding with its evidence and promotion
// requirement. Inferred authority is never applied automatically (W21.10).
type Proposal struct {
	ContractID string   `json:"contract_id"`
	Kind       string   `json:"kind"`
	Sources    []string `json:"sources"`
	Evidence   []string `json:"evidence"`
	Confidence string   `json:"confidence"`
	Promotion  string   `json:"promotion"`
}

// Inspect discovers documentation signals in a repository without writing
// anything.
func Inspect(root string) (AdoptionReport, error) {
	report := AdoptionReport{Root: root, Signals: map[string]bool{}, DocFiles: []string{},
		Proposals: []Proposal{}, Conflicts: []string{}, Duplicates: []string{}, Findings: []string{}}
	if _, err := os.Stat(root); err != nil {
		return AdoptionReport{}, fmt.Errorf("adoption target does not exist: %w", err)
	}
	for _, signal := range []struct{ name, path string }{
		{"docs-directory", "docs"},
		{"readme", "README.md"},
		{"agents-file", "AGENTS.md"},
		{"schema-directory", "schemas"},
		{"ci-workflows", ".github/workflows"},
		{"go-module", "go.mod"},
		{"package-json", "package.json"},
		{"python-project", "pyproject.toml"},
		{"tests", "tests"},
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(signal.path))); err == nil {
			report.Signals[signal.name] = true
		}
	}
	// One walk from the root: walking docs/ separately would report every
	// document twice and invent duplicates that do not exist.
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".mdx") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		report.DocFiles = append(report.DocFiles, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(report.DocFiles)
	report.Duplicates = duplicateBasenames(report.DocFiles)
	report.Proposals = propose(report)
	return report, nil
}

// duplicateBasenames finds documents that describe the same topic twice, the
// documented failure mode that adoption must surface rather than copy (W21.9).
func duplicateBasenames(files []string) []string {
	byBase := map[string][]string{}
	for _, file := range files {
		base := strings.ToLower(filepath.Base(file))
		byBase[base] = append(byBase[base], file)
	}
	duplicates := []string{}
	for base, group := range byBase {
		if len(group) < 2 {
			continue
		}
		sort.Strings(group)
		duplicates = append(duplicates, base+" → "+strings.Join(group, ", "))
	}
	sort.Strings(duplicates)
	return duplicates
}

// propose infers candidate documentation contracts from the discovered files.
func propose(report AdoptionReport) []Proposal {
	proposals := []Proposal{}
	hasMatch := func(fragment string) []string {
		matches := []string{}
		for _, file := range report.DocFiles {
			if strings.Contains(strings.ToLower(file), fragment) {
				matches = append(matches, file)
			}
		}
		sort.Strings(matches)
		return matches
	}
	rules := []struct {
		contract string
		fragment string
		evidence string
	}{
		{"product.vision", "vision", "a vision document exists"},
		{"architecture.system", "architecture", "an architecture document exists"},
		{"testing.strategy", "test", "a testing document exists"},
		{"security.trust", "security", "a security document exists"},
		{"installation.lifecycle", "install", "an installation document exists"},
		{"cli.reference", "cli", "a CLI reference document exists"},
	}
	for _, rule := range rules {
		sources := hasMatch(rule.fragment)
		if len(sources) == 0 {
			continue
		}
		proposals = append(proposals, Proposal{
			ContractID: rule.contract, Kind: "binding", Sources: sources,
			Evidence: []string{rule.evidence}, Confidence: "inferred",
			Promotion: "requires maintainer review before the binding becomes canonical",
		})
	}
	if len(report.DocFiles) > 0 && !report.Signals["docs-directory"] {
		report.Conflicts = append(report.Conflicts, "documents exist outside a docs/ directory")
	}
	return proposals
}

// Propose renders the adoption plan as actionable proposals plus the explicit
// review requirement.
func Propose(root string) (map[string]any, error) {
	report, err := Inspect(root)
	if err != nil {
		return nil, err
	}
	actions := []string{}
	for _, proposal := range report.Proposals {
		actions = append(actions, fmt.Sprintf("add binding %s → %s", proposal.ContractID, strings.Join(proposal.Sources, ", ")))
	}
	for _, duplicate := range report.Duplicates {
		actions = append(actions, "reconcile duplicate documents: "+duplicate)
	}
	sort.Strings(actions)
	return map[string]any{
		"root": report.Root, "signals": report.Signals, "doc_files": report.DocFiles,
		"proposals": report.Proposals, "conflicts": report.Conflicts, "duplicates": report.Duplicates,
		"actions":            actions,
		"promotion_required": len(report.Proposals) > 0,
		"note":               "no inferred authority is applied automatically; review proposals before promoting them",
	}, nil
}
