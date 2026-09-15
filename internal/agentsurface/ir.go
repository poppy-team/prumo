// Package agentsurface compiles provider-neutral agent instruction knowledge
// into verifiable per-tool surfaces.
//
// The core knows nothing about any vendor format. Adapters implement Adapter
// and depend on the IR; the IR never depends on an adapter (W16.13). A vendor
// surface is always a projection: it can be regenerated from canonical
// knowledge and it is never the source of truth.
package agentsurface

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IRPath is the canonical, provider-neutral instruction knowledge.
const IRPath = "docs/agents/instruction-ir.json"

// Budget caps instruction tokens per adapter scope (W16.11). Zero means no cap.
type Budget struct {
	Root  int `json:"root,omitempty"`
	Scope int `json:"scope,omitempty"`
}

// Rule is one instruction statement with its scope and provenance.
type Rule struct {
	ID         string   `json:"id"`
	Scope      string   `json:"scope,omitempty"`
	Text       string   `json:"text"`
	Precedence int      `json:"precedence,omitempty"`
	Adapters   []string `json:"adapters,omitempty"`
	Sources    []string `json:"sources,omitempty"`
}

// IR is the agent instruction intermediate representation.
type IR struct {
	Version     int    `json:"version"`
	TokenBudget Budget `json:"token_budget"`
	Rules       []Rule `json:"rules"`
}

// LoadIR reads the canonical instruction IR from the repository root.
func LoadIR(root string) (IR, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(IRPath)))
	if err != nil {
		return IR{}, err
	}
	var ir IR
	if err := json.Unmarshal(data, &ir); err != nil {
		return IR{}, err
	}
	if ir.Version < 1 {
		return IR{}, fmt.Errorf("agent instruction IR requires a version")
	}
	if len(ir.Rules) == 0 {
		return IR{}, fmt.Errorf("agent instruction IR has no rules")
	}
	return ir, nil
}

// Enabled reports whether a rule targets an adapter. An empty adapter list
// means every adapter.
func (r Rule) Enabled(adapter string) bool {
	if len(r.Adapters) == 0 {
		return true
	}
	for _, a := range r.Adapters {
		if a == adapter {
			return true
		}
	}
	return false
}

// NormalScope canonicalises a scope. The root scope is the empty string.
func NormalScope(scope string) string {
	scope = filepath.ToSlash(strings.TrimSpace(scope))
	scope = strings.TrimPrefix(scope, "./")
	scope = strings.Trim(scope, "/")
	if scope == "" || scope == "." {
		return ""
	}
	return scope
}

// ScopeContains reports whether rules declared in scope apply to target. The
// root scope applies everywhere; nested scopes take precedence by being more
// specific (W16.2).
func ScopeContains(scope, target string) bool {
	scope, target = NormalScope(scope), NormalScope(target)
	if scope == "" {
		return true
	}
	return target == scope || strings.HasPrefix(target, scope+"/")
}

// EffectiveRules returns the rules that apply to a target scope, in the
// deterministic compilation order: broadest scope first, then precedence, then
// rule ID. Ties are impossible because IDs are unique per scope.
func (ir IR) EffectiveRules(target string) []Rule {
	rules := make([]Rule, 0, len(ir.Rules))
	for _, r := range ir.Rules {
		if ScopeContains(r.Scope, target) {
			r.Scope = NormalScope(r.Scope)
			rules = append(rules, r)
		}
	}
	sort.SliceStable(rules, func(i, j int) bool {
		di, dj := scopeDepth(rules[i].Scope), scopeDepth(rules[j].Scope)
		if di != dj {
			return di < dj
		}
		if rules[i].Precedence != rules[j].Precedence {
			return rules[i].Precedence < rules[j].Precedence
		}
		return rules[i].ID < rules[j].ID
	})
	return rules
}

// Scopes lists every distinct declared scope plus the root scope, ordered
// broadest first.
func (ir IR) Scopes() []string {
	seen := map[string]bool{"": true}
	for _, r := range ir.Rules {
		seen[NormalScope(r.Scope)] = true
	}
	scopes := make([]string, 0, len(seen))
	for s := range seen {
		scopes = append(scopes, s)
	}
	sort.Slice(scopes, func(i, j int) bool {
		if di, dj := scopeDepth(scopes[i]), scopeDepth(scopes[j]); di != dj {
			return di < dj
		}
		return scopes[i] < scopes[j]
	})
	return scopes
}

func scopeDepth(scope string) int {
	scope = NormalScope(scope)
	if scope == "" {
		return 0
	}
	return len(strings.Split(scope, "/"))
}

// Conflict is a duplicate or contradictory instruction across scopes.
type Conflict struct {
	Kind    string `json:"kind"`
	RuleID  string `json:"rule_id"`
	OtherID string `json:"other_id,omitempty"`
	Detail  string `json:"detail"`
}

// Conflicts detects duplicate instructions and same-ID rules with different
// text inside one effective scope (W16.12). Semantic contradiction between two
// different statements is not decidable deterministically and is out of scope.
func (ir IR) Conflicts(target string) []Conflict {
	rules := ir.EffectiveRules(target)
	var conflicts []Conflict
	byText := map[string]Rule{}
	byID := map[string]Rule{}
	for _, r := range rules {
		text := normalizeText(r.Text)
		if prev, ok := byText[text]; ok && prev.ID != r.ID {
			conflicts = append(conflicts, Conflict{
				Kind: "duplicate-instruction", RuleID: r.ID, OtherID: prev.ID,
				Detail: fmt.Sprintf("rules %s (%s) and %s (%s) carry the same instruction", prev.ID, prev.Scope, r.ID, r.Scope),
			})
		}
		byText[text] = r
		if prev, ok := byID[r.ID]; ok && normalizeText(prev.Text) != text {
			conflicts = append(conflicts, Conflict{
				Kind: "conflicting-rule", RuleID: r.ID, OtherID: prev.ID,
				Detail: fmt.Sprintf("rule %s is declared twice with different text (%s, %s)", r.ID, prev.Scope, r.Scope),
			})
		}
		byID[r.ID] = r
	}
	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Kind != conflicts[j].Kind {
			return conflicts[i].Kind < conflicts[j].Kind
		}
		return conflicts[i].RuleID < conflicts[j].RuleID
	})
	return conflicts
}

// Validate checks the IR's structural invariants before any rendering.
func (ir IR) Validate() []Conflict {
	var problems []Conflict
	seen := map[string]bool{}
	for _, r := range ir.Rules {
		scope := NormalScope(r.Scope)
		key := scope + "\x00" + r.ID
		switch {
		case strings.TrimSpace(r.ID) == "":
			problems = append(problems, Conflict{Kind: "missing-rule-id", Detail: "every rule needs a stable id"})
			continue
		case seen[key]:
			problems = append(problems, Conflict{Kind: "duplicate-rule-id", RuleID: r.ID, Detail: "duplicate rule id in scope " + describeScope(scope)})
		case strings.TrimSpace(r.Text) == "":
			problems = append(problems, Conflict{Kind: "empty-rule-text", RuleID: r.ID, Detail: "rule has no instruction text"})
		}
		seen[key] = true
		if strings.Contains(scope, "..") {
			problems = append(problems, Conflict{Kind: "invalid-scope", RuleID: r.ID, Detail: "scope may not contain .."})
		}
		for _, a := range r.Adapters {
			if _, ok := Adapters()[a]; !ok {
				problems = append(problems, Conflict{Kind: "unknown-adapter", RuleID: r.ID, Detail: "unknown adapter " + a})
			}
		}
		for _, s := range r.Sources {
			if strings.TrimSpace(s) == "" {
				problems = append(problems, Conflict{Kind: "empty-source", RuleID: r.ID, Detail: "source pointer must not be empty"})
			}
		}
	}
	sort.Slice(problems, func(i, j int) bool {
		if problems[i].Kind != problems[j].Kind {
			return problems[i].Kind < problems[j].Kind
		}
		return problems[i].RuleID < problems[j].RuleID
	})
	return problems
}

func describeScope(scope string) string {
	if scope == "" {
		return "root"
	}
	return scope
}

func normalizeText(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}
