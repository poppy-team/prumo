package docengine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Impact struct {
	ContractID string   `json:"contract_id"`
	Documents  []string `json:"documents"`
	Reason     string   `json:"reason"`
	Severity   string   `json:"severity"`
}
type Delta struct {
	ID        string   `json:"id,omitempty"`
	Version   int      `json:"version,omitempty"`
	State     string   `json:"state"`
	Source    string   `json:"source"`
	Contracts []string `json:"contracts"`
	Documents []string `json:"documents"`
	Reason    string   `json:"reason"`
	Evidence  []string `json:"evidence"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}
type Finding struct {
	ID          string   `json:"id"`
	Sources     []string `json:"sources"`
	Contract    string   `json:"contract,omitempty"`
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Authority   string   `json:"authority"`
	Description string   `json:"description"`
	State       string   `json:"state"`
}

func AnalyzeImpact(root string, changed []string) ([]Impact, error) {
	registry, err := LoadRegistry(root)
	if err != nil {
		return nil, err
	}
	bindings, err := LoadBindings(root)
	if err != nil {
		return nil, err
	}
	return AnalyzeImpactsIn(root, registry, bindings, changed, nil), nil
}

func AnalyzeImpacts(registry Registry, bindings []Binding, changed []string) []Impact {
	return AnalyzeImpactsWithGraph(registry, bindings, changed, nil)
}
func triggerMatches(trigger string, changed []string) bool {
	raw := trigger
	if idx := strings.Index(raw, ":"); idx != -1 {
		raw = raw[idx+1:]
	}
	token := strings.ToLower(strings.TrimSuffix(raw, "/"))
	if token == "" {
		return false
	}
	for _, path := range changed {
		norm := strings.ToLower(filepath.ToSlash(path))
		if strings.Contains(norm, token) {
			return true
		}
	}
	return false
}
func MakeDelta(goal string, impacts []Impact) Delta {
	contracts := []string{}
	docs := []string{}
	for _, impact := range impacts {
		contracts = append(contracts, impact.ContractID)
		docs = append(docs, impact.Documents...)
	}
	sort.Strings(contracts)
	sort.Strings(docs)
	return Delta{State: "proposed", Source: goal, Contracts: unique(contracts), Documents: unique(docs), Reason: "documentation impact analysis", Evidence: []string{}}
}
func unique(values []string) []string {
	out := []string{}
	for _, v := range values {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

// numericFact is a statement of the shape <key: value | key=value | "key": value>
// for the same subject key. Detecting the same key with materially different
// numeric/config values across canonical sources is a deterministic
// contradiction, and it replaces a hard-coded "port only, four files" regex.
var numericClaim = regexp.MustCompile(`(?:"([a-zA-Z0-9_.\-]+)"\s*:\s*([0-9][0-9.]*)|([a-zA-Z0-9_.\-]+)\s*[=:]\s*([0-9][0-9.]*))`)

// genericIndicators and intentionally shape the contradiction detector toward
// keys that plausibly carry comparable meaning across sources. Keys like
// "version" carry a different value per context (a profile's schema version
// vs the project protocol version), so flagging them as contradictions was a
// false-positive path, not a correctness signal.
var genericIndicators = map[string]bool{
	"version": true, "id": true, "index": true, "size": true, "count": true,
	"length": true, "len": true, "width": true, "height": true, "rank": true,
}

// DetectContradictions finds canonical sources that assert different numeric
// values for the same declarative key (config or SLA). The previous version
// only inspected `port`/`default_port` in four hard-coded files, so a real
// conflict between `prumo.json`, an ADR and a deployment doc passed silently.
// This general detector compares numeric-claim keys across every canonical
// document except the ambiguous scaffolding ones above.
func DetectContradictions(root string) ([]Finding, error) {
	sources, err := contradictionSources(root)
	if err != nil {
		return nil, err
	}
	occ := map[string]map[string]map[string]bool{} // key -> source -> set of values
	for _, source := range sources {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source)))
		if err != nil {
			continue
		}
		for _, m := range numericClaim.FindAllStringSubmatch(string(data), -1) {
			key := strings.ToLower(m[1])
			if key == "" {
				key = strings.ToLower(m[3])
			}
			value := m[2]
			if value == "" {
				value = m[4]
			}
			if key == "" || value == "" {
				continue
			}
			if occ[key] == nil {
				occ[key] = map[string]map[string]bool{}
			}
			if occ[key][source] == nil {
				occ[key][source] = map[string]bool{}
			}
			occ[key][source][value] = true
		}
	}
	findings := []Finding{}
	for key, bySource := range occ {
		if genericIndicators[key] {
			continue
		}
		// A conflict requires the same key to be set with different values
		// across distinct sources. A single file that legitimately nests the
		// key several times (for example the per-profile context budgets in
		// prumo.json) is not a cross-source contradiction.
		if len(bySource) < 2 {
			continue
		}
		valueSet := map[string]string{}
		for source, values := range bySource {
			for value := range values {
				valueSet[value] = source
			}
		}
		if len(valueSet) < 2 {
			continue
		}
		vals := []string{}
		paths := []string{}
		for value, source := range valueSet {
			vals = append(vals, value)
			paths = append(paths, source)
		}
		sort.Strings(vals)
		sort.Strings(paths)
		findings = append(findings, Finding{
			ID:          "DOC_CONTRADICTION",
			Sources:     unique(paths),
			Type:        "numeric-fact contradiction",
			Severity:    "medium",
			Authority:   "canonical-documentation",
			Description: "conflicting values for " + key + ": " + strings.Join(vals, ", "),
			State:       "open",
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Description < findings[j].Description })
	return findings, nil
}

// contradictionSources returns every canonical doc that can contradict another:
// prumo.json plus all documents bound to contracts. Falling back to all docs
// when no bindings are declared keeps the check fair for a project that has not
// wired bindings yet.
func contradictionSources(root string) ([]string, error) {
	seen := map[string]bool{"prumo.json": true}
	out := []string{"prumo.json"}
	for _, doc := range walkAllDocs(root)[1:] {
		if seen[doc] {
			continue
		}
		seen[doc] = true
		out = append(out, doc)
	}
	// Documents can also be bound from outside the docs/ tree; include them so a
	// conflict in an operations or product doc is still a cross-file contradiction.
	bindings, err := LoadBindings(root)
	if err == nil {
		for _, b := range bindings {
			for _, s := range b.Sources {
				if seen[s] {
					continue
				}
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out, nil
}

func walkAllDocs(root string) []string {
	out := []string{"prumo.json"}
	_ = filepath.Walk(filepath.Join(root, "docs"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".json") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out
}
func DetectStaleness(root string, changed []string) ([]Finding, error) {
	impacts, err := AnalyzeImpact(root, changed)
	if err != nil {
		return nil, err
	}
	out := []Finding{}
	for _, impact := range impacts {
		out = append(out, Finding{ID: "DOC_SOURCE_STALE", Sources: impact.Documents, Contract: impact.ContractID, Type: "trigger-based", Severity: impact.Severity, Authority: "canonical-documentation", Description: impact.Reason, State: "open"})
	}
	return out, nil
}
func (d Delta) Validate() error {
	valid := map[string]bool{"proposed": true, "reviewed": true, "accepted": true, "rejected": true, "applied": true}
	if !valid[d.State] || d.Source == "" {
		return fmt.Errorf("documentation delta requires valid state and source")
	}
	if d.State == "applied" && len(d.Evidence) == 0 {
		return fmt.Errorf("applied documentation delta requires evidence")
	}
	return nil
}
