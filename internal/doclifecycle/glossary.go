package doclifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// GlossaryPath is the canonical terminology registry.
const GlossaryPath = "docs/glossary.json"

// Term statuses. The vocabulary is closed: a term is either the wording the
// project wants, wording on its way out, or wording that must not be used.
const (
	TermPreferred  = "preferred"
	TermDeprecated = "deprecated"
	TermForbidden  = "forbidden"
)

// TermStatuses lists the closed status vocabulary.
func TermStatuses() []string { return []string{TermForbidden, TermPreferred, TermDeprecated} }

// Term is one glossary entry (W20.2).
type Term struct {
	Term       string   `json:"term"`
	Definition string   `json:"definition"`
	Status     string   `json:"status"`
	ReplacedBy string   `json:"replaced_by,omitempty"`
	Locale     string   `json:"locale,omitempty"`
	Sources    []string `json:"sources,omitempty"`
}

// Glossary is the canonical terminology registry.
type Glossary struct {
	Version int    `json:"version"`
	Terms   []Term `json:"terms"`
}

// LoadGlossary reads the terminology registry. A repository without one simply
// has no terminology policy; that is not an error.
func LoadGlossary(root string) (Glossary, error) {
	glossary := Glossary{Terms: []Term{}}
	found, err := loadJSON(filepath.Join(root, filepath.FromSlash(GlossaryPath)), &glossary)
	if err != nil {
		return Glossary{}, err
	}
	if !found {
		return Glossary{Terms: []Term{}}, nil
	}
	if glossary.Terms == nil {
		glossary.Terms = []Term{}
	}
	return glossary, nil
}

// ValidateGlossary checks the registry itself, before it is applied to any
// document: a glossary with duplicate, undefined or circular entries would
// enforce the wrong wording.
func ValidateGlossary(glossary Glossary) []Finding {
	findings := []Finding{}
	byTerm := map[string]Term{}
	for _, term := range glossary.Terms {
		name := strings.TrimSpace(term.Term)
		if name == "" {
			findings = append(findings, Finding{Kind: "term-unnamed", Target: GlossaryPath,
				Detail: "a glossary entry has no term"})
			continue
		}
		key := strings.ToLower(name)
		if _, exists := byTerm[key]; exists {
			findings = append(findings, Finding{Kind: "term-duplicate", Target: name,
				Detail: "the term is declared more than once"})
			continue
		}
		if strings.TrimSpace(term.Definition) == "" {
			findings = append(findings, Finding{Kind: "term-undefined", Target: name,
				Detail: "the term has no definition, so a reader cannot resolve it"})
		}
		if !containsString(TermStatuses(), term.Status) {
			findings = append(findings, Finding{Kind: "term-invalid-status", Target: name,
				Detail: fmt.Sprintf("status %q is not one of %s", term.Status, strings.Join(TermStatuses(), ", "))})
		}
		byTerm[key] = term
	}
	// A retired term must say what replaces it, and the replacement must be a
	// real preferred term rather than a dangling name.
	for _, term := range glossary.Terms {
		if term.Status != TermDeprecated {
			continue
		}
		replacement := strings.ToLower(strings.TrimSpace(term.ReplacedBy))
		if replacement == "" {
			findings = append(findings, Finding{Kind: "term-without-replacement", Target: term.Term,
				Detail: "a deprecated term must name its replacement"})
			continue
		}
		target, ok := byTerm[replacement]
		if !ok {
			findings = append(findings, Finding{Kind: "term-replacement-unknown", Target: term.Term,
				Detail: "replacement is not a glossary term: " + term.ReplacedBy})
			continue
		}
		if target.Status != TermPreferred {
			findings = append(findings, Finding{Kind: "term-replacement-not-preferred", Target: term.Term,
				Detail: "replacement " + term.ReplacedBy + " is itself " + target.Status})
		}
	}
	sortFindings(findings)
	return findings
}

// GlossaryFindings applies the glossary to the given documents. Matching is
// word-boundary and case-insensitive, so a term inside a longer word is not a
// violation, and the check never guesses at intent.
func GlossaryFindings(root string, glossary Glossary, documents []string) []Finding {
	findings := ValidateGlossary(glossary)
	for _, rel := range documents {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		data := string(raw)
		for _, term := range glossary.Terms {
			if term.Status == TermPreferred || strings.TrimSpace(term.Term) == "" {
				continue
			}
			pattern, err := termPattern(term.Term)
			if err != nil || !pattern.MatchString(data) {
				continue
			}
			kind := "term-forbidden"
			detail := fmt.Sprintf("%q is forbidden terminology", term.Term)
			if term.Status == TermDeprecated {
				kind = "term-deprecated"
				detail = fmt.Sprintf("%q is deprecated; use %q", term.Term, term.ReplacedBy)
			}
			findings = append(findings, Finding{Kind: kind, Target: rel, Detail: detail})
		}
	}
	sortFindings(findings)
	return findings
}

// termPattern builds a whole-word matcher for a term. Multi-word terms are
// matched as a phrase with flexible whitespace, which is how they appear in
// prose after line wrapping.
func termPattern(term string) (*regexp.Regexp, error) {
	words := strings.Fields(term)
	for i, w := range words {
		words[i] = regexp.QuoteMeta(w)
	}
	return regexp.Compile(`(?i)\b` + strings.Join(words, `\s+`) + `\b`)
}

// GlossaryStatus is the operator-facing view of the terminology registry.
type GlossaryStatus struct {
	Path       string    `json:"path"`
	Total      int       `json:"total"`
	Preferred  int       `json:"preferred"`
	Deprecated int       `json:"deprecated"`
	Forbidden  int       `json:"forbidden"`
	Terms      []Term    `json:"terms"`
	Findings   []Finding `json:"findings"`
	OK         bool      `json:"ok"`
}

// CheckGlossary validates the registry and reports it.
func CheckGlossary(root string) (GlossaryStatus, error) {
	glossary, err := LoadGlossary(root)
	if err != nil {
		return GlossaryStatus{}, err
	}
	status := GlossaryStatus{Path: GlossaryPath, Terms: glossary.Terms, Findings: []Finding{}}
	for _, term := range glossary.Terms {
		switch term.Status {
		case TermPreferred:
			status.Preferred++
		case TermDeprecated:
			status.Deprecated++
		case TermForbidden:
			status.Forbidden++
		}
	}
	status.Total = len(glossary.Terms)
	status.Findings = ValidateGlossary(glossary)
	status.OK = len(status.Findings) == 0
	return status, nil
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Target < findings[j].Target
	})
}
