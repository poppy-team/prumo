package resolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Profile struct {
	Raw map[string]any
}

func LoadProfile(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Profile{}, err
	}
	return Profile{Raw: raw}, nil
}

func (p Profile) ProjectTypes() map[string]bool { return stringSet(nested(p.Raw, "project", "type")) }
func (p Profile) Features() map[string]bool     { return stringSet(p.Raw["features"]) }
func (p Profile) Risks() map[string]bool {
	out := stringSet(p.Raw["risk"])
	quality, _ := p.Raw["quality"].(map[string]any)
	for key, value := range quality {
		if value == true || value == "high" {
			out[strings.ToLower(key)] = true
		}
	}
	return out
}
func (p Profile) StackTokens() map[string]bool { return flattenSet(p.Raw["stack"]) }

// Mode is the project's current mode, defaulting to "implementation" because a
// profile that names no mode is a project doing the work, not one only reading
// about it.
func (p Profile) Mode() string {
	if mode, ok := p.Raw["mode"].(string); ok && mode != "" {
		return mode
	}
	return "implementation"
}

// InMode reports whether the project is in one of the named modes. A skill that
// names no modes is not restricted; a skill that names modes is restricted to
// them.
func (p Profile) InMode(mode string) bool {
	current := p.Mode()
	return strings.EqualFold(current, mode)
}

// Capabilities is everything the project says it has: declared features, stack
// tokens and project types, which together are what a skill's `requires` is
// written against.
func (p Profile) Capabilities() map[string]bool {
	out := map[string]bool{}
	for _, set := range []map[string]bool{p.Features(), p.StackTokens(), p.ProjectTypes()} {
		for k := range set {
			out[k] = true
		}
	}
	if raw, ok := p.Raw["capabilities"].([]any); ok {
		for _, v := range raw {
			out[fmt.Sprint(v)] = true
		}
	}
	return out
}

func (p Profile) Orchestrator() string {
	ai, _ := p.Raw["ai"].(map[string]any)
	if value, ok := ai["orchestrator"].(string); ok && value != "" {
		return value
	}
	return "native"
}

func (p Profile) Autonomy() string {
	ai, _ := p.Raw["ai"].(map[string]any)
	if value, ok := ai["autonomy"].(string); ok && value != "" {
		return value
	}
	return "agentic"
}

func (p Profile) PreferredModels() []map[string]any {
	ai, _ := p.Raw["ai"].(map[string]any)
	raw, _ := ai["preferred_models"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func nested(root map[string]any, keys ...string) any {
	var value any = root
	for _, key := range keys {
		m, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = m[key]
	}
	return value
}

// firstOfAnyGroup takes the first entry of the first `requires_any` group,
// which is the one to pull in when none of them is already available.
func firstOfAnyGroup(value any) []string {
	groups := anyList(value)
	if len(groups) == 0 {
		return nil
	}
	return stringList(groups[0])
}

// pullPrerequisites closes the requirement graph over the candidates.
//
// It works from a snapshot of the ids rather than ranging the map it is adding
// to: an entry added while ranging a Go map may or may not be visited, which
// would make whether a two-level dependency resolved depend on map iteration
// order. The snapshot makes it depend on the graph.
func pullPrerequisites(candidates map[string]map[string]any, all map[string]map[string]any, preselected map[string]bool, rejected map[string]string) {
	for frontier := true; frontier; {
		frontier = false
		ids := make([]string, 0, len(candidates))
		for id := range candidates {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			item, ok := candidates[id]
			if !ok {
				continue
			}
			needs := append(stringList(item["requires"]), firstOfAnyGroup(item["requires_any"])...)
			for _, need := range needs {
				if need == "" || preselected[need] || candidates[need] != nil {
					continue
				}
				prerequisite, exists := all[need]
				if !exists {
					continue
				}
				// The prerequisite is being added because something selected
				// asked for it, not because its own selector matched, so its
				// earlier rejection is no longer the reason it is absent.
				candidates[need] = prerequisite
				delete(rejected, need)
				frontier = true
			}
		}
	}
}

// anyList reads a list whose members may themselves be lists, which is how a
// grouped `requires_any` arrives from JSON.
func anyList(value any) []any {
	if values, ok := value.([]any); ok {
		return values
	}
	return nil
}

// stringList reads a list of strings whether it arrived as []any (from JSON)
// or as []string (built in memory). Only the first form was read, so a
// constraint built by a caller rather than parsed was silently no constraint.
func stringList(value any) []string {
	switch values := value.(type) {
	case []any:
		out := make([]string, 0, len(values))
		for _, v := range values {
			if s := fmt.Sprint(v); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return values
	default:
		return nil
	}
}

func stringSet(value any) map[string]bool {
	out := map[string]bool{}
	switch v := value.(type) {
	case string:
		out[strings.ToLower(v)] = true
	case []any:
		for _, x := range v {
			out[strings.ToLower(fmt.Sprint(x))] = true
		}
	case []string:
		for _, x := range v {
			out[strings.ToLower(x)] = true
		}
	}
	return out
}
func flattenSet(value any) map[string]bool {
	out := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, child := range x {
				out[strings.ToLower(k)] = true
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		case []string:
			for _, child := range x {
				out[strings.ToLower(child)] = true
			}
		case string:
			out[strings.ToLower(x)] = true
		case float64, bool:
			out[strings.ToLower(fmt.Sprint(x))] = true
		}
	}
	walk(value)
	return out
}
func anyMatch(values any, actual map[string]bool) bool {
	for key := range stringSet(values) {
		if actual[key] {
			return true
		}
	}
	return false
}

type Catalog struct{ Sections map[string][]map[string]any }

func LoadCatalog(repoRoot string) (Catalog, error) {
	base := filepath.Join(repoRoot, "src", "prumo", "resources", "catalog")
	manifestBytes, err := os.ReadFile(filepath.Join(base, "catalog.json"))
	if err != nil {
		return Catalog{}, err
	}
	var manifest struct {
		Sections map[string]string `json:"sections"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return Catalog{}, err
	}
	catalog := Catalog{Sections: map[string][]map[string]any{}}
	for section, file := range manifest.Sections {
		data, err := os.ReadFile(filepath.Join(base, file))
		if err != nil {
			return Catalog{}, err
		}
		var body map[string]any
		if err := json.Unmarshal(data, &body); err != nil {
			return Catalog{}, err
		}
		raw, _ := body[section].([]any)
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				catalog.Sections[section] = append(catalog.Sections[section], m)
			}
		}
	}
	return catalog, nil
}
func byID(items []map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, item := range items {
		if id, ok := item["id"].(string); ok {
			out[id] = item
		}
	}
	return out
}
func sortedKeys(values map[string]bool) []string {
	out := []string{}
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

type Resolution struct {
	Agents  []string                    `json:"agents"`
	Skills  []string                    `json:"skills"`
	Recipes []string                    `json:"recipes"`
	Reasons map[string][]string         `json:"reasons"`
	Traces  map[string][]map[string]any `json:"traces"`
	// Rejected says why each catalogue item that could have been selected was
	// not.
	//
	// The resolution listed what it chose and stayed silent about the rest, so a
	// skill that declares it needs a capability the project lacks, or that
	// conflicts with one already chosen, simply did not appear — and a plan
	// missing a skill looks the same whether the resolver decided that or
	// nobody looked (GAP-146).
	Rejected map[string]string `json:"rejected,omitempty"`
}

// unmetRequirements checks a catalogue item's declared prerequisites against
// the project's mode.
//
// Only `modes` is a project-level constraint. `requires`, `requires_any` and
// `conflicts` are relations *between skills* — every value in all 156 entries
// is another skill's id — so they are resolved against the selection, not
// against the profile. Reading them as capabilities is the obvious mistake and
// it drops core skills: `clean-code` declares `requires: [testing-quality]`,
// which is a skill, not something a project declares (GAP-146).
func unmetRequirements(item map[string]any, p Profile) (string, bool) {
	// `modes`: the skill only applies in the named modes. An empty list means
	// the skill declares no restriction, not that it applies to none.
	if modes := stringList(item["modes"]); len(modes) > 0 {
		matched := false
		for _, mode := range modes {
			if p.InMode(mode) {
				matched = true
				break
			}
		}
		if !matched {
			return "applies only in modes [" + strings.Join(modes, ", ") +
				"]; this project is in mode " + p.Mode(), true
		}
	}

	return "", false
}

// closeDependencies pulls a selected skill's prerequisites into the plan.
//
// A skill that names another is not usable alone, and shipping it without its
// prerequisite produces a plan that advertises a workflow whose first step does
// not exist. So `requires` is read as a dependency to satisfy, not a condition
// to fail: `lang-zig` requires `clean-code`, `testing-quality` and
// `secure-coding`, and dropping `lang-zig` because the selector happened not to
// pick `secure-coding` would mean no language skill ever resolved.
//
// A requirement that names a skill the catalogue does not contain is different:
// there is nothing to pull in, so the dependent is dropped and says why.
func closeDependencies(candidates map[string]map[string]any, all map[string]map[string]any, preselected map[string]bool, rejected map[string]string) {
	// conflicts is checked once the closure is complete, because a prerequisite
	// pulled in here can itself conflict with something already selected.
	// `changed` also guards the walk against a requirement cycle in the
	// catalogue's own graph.
	changed := true
	for changed {
		changed = false
		for id, item := range candidates {
			for _, need := range stringList(item["requires"]) {
				if preselected[need] || candidates[need] != nil {
					continue
				}
				prerequisite, exists := all[need]
				if !exists {
					rejected[id] = "requires " + need + ", which the catalogue does not define"
					delete(candidates, id)
					changed = true
					continue
				}
				candidates[need] = prerequisite
				changed = true
			}
			for _, raw := range anyList(item["requires_any"]) {
				group := stringList(raw)
				if len(group) == 0 {
					continue
				}
				satisfied := false
				for _, need := range group {
					if preselected[need] || candidates[need] != nil {
						satisfied = true
						break
					}
				}
				if satisfied {
					continue
				}
				pulled := false
				for _, need := range group {
					if prerequisite, exists := all[need]; exists {
						candidates[need] = prerequisite
						pulled = true
						changed = true
						break
					}
				}
				if !pulled {
					rejected[id] = "requires at least one of [" + strings.Join(group, ", ") +
						"], and the catalogue defines none of them"
					delete(candidates, id)
					changed = true
				}
			}
			if _, still := candidates[id]; !still {
				break
			}
		}
	}
	applyConflicts(candidates, preselected, rejected)
}

// unmetSkillRequirements reports why one skill cannot be selected yet.
func unmetSkillRequirements(id string, item map[string]any, available map[string]bool, candidates map[string]map[string]any) (string, bool) {
	// `requires`: all of them.
	if raw := stringList(item["requires"]); len(raw) > 0 {
		var missing []string
		for _, need := range raw {
			if !available[need] {
				missing = append(missing, need)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return "requires " + strings.Join(missing, ", ") + ", which the plan does not select", true
		}
	}
	// `requires_any`: at least one.
	if raw := stringList(item["requires_any"]); len(raw) > 0 {
		found := false
		for _, need := range raw {
			if available[need] {
				found = true
				break
			}
		}
		if !found {
			return "requires at least one of [" + strings.Join(raw, ", ") +
				"] and the plan selects none of them", true
		}
	}
	// `conflicts`: none of them.
	for _, other := range stringList(item["conflicts"]) {
		if available[other] && other != id {
			return "conflicts with " + other + ", which the plan also selects", true
		}
	}
	return "", false
}

// applyConflicts drops mutually exclusive skills.
//
// A conflict is symmetric, so both sides are removed rather than the one that
// happened to be seen second. Keeping either would mean the plan contains a
// skill whose own declaration says it cannot coexist with something else in the
// same plan.
func applyConflicts(candidates map[string]map[string]any, preselected map[string]bool, rejected map[string]string) {
	// A conflict names two skills, so it has to be found from both sides: only
	// checking whether X lists Y misses a pair where Y lists X. A single forward
	// pass over a map would also decide the outcome from whichever id the
	// iteration happened to reach first.
	for id, item := range candidates {
		for _, other := range stringList(item["conflicts"]) {
			switch {
			case preselected[other]:
				// The other side is a bundle, which is an explicit decision.
				// Silently unselecting it would be the resolver overruling
				// somebody, so the candidate is what goes.
				rejected[id] = "conflicts with " + other + ", which a bundle selects"
				delete(candidates, id)
			case candidates[other] != nil:
				// Both are candidates chosen by their own selectors, so neither
				// is more authoritative than the other and neither is kept.
				reason := "conflicts with " + other + "; both were selected and a conflict drops both"
				rejected[id] = reason
				rejected[other] = reason
				delete(candidates, id)
				delete(candidates, other)
			}
		}
	}
}

func selectorMatches(item map[string]any, p Profile) (bool, []string, []map[string]any) {
	if item["core"] == true {
		return true, []string{"core"}, []map[string]any{{"type": "core", "value": "core"}}
	}
	selector, _ := item["select"].(map[string]any)
	checks := []bool{}
	reasons := []string{}
	traces := []map[string]any{}
	checksFor := []struct {
		name   string
		key    string
		actual map[string]bool
	}{{"project-type", "project_types", p.ProjectTypes()}, {"stack", "stack_any", p.StackTokens()}, {"feature", "features_any", p.Features()}, {"risk", "risk_any", p.Risks()}}
	for _, check := range checksFor {
		if raw, ok := selector[check.key]; ok {
			matched := anyMatch(raw, check.actual)
			checks = append(checks, matched)
			if matched {
				reasons = append(reasons, check.name)
				traces = append(traces, map[string]any{"type": check.name, "value": raw})
			}
		}
	}
	if len(checks) == 0 {
		return false, nil, nil
	}
	for _, ok := range checks {
		if !ok {
			return false, nil, nil
		}
	}
	return true, reasons, traces
}
func intersectRequired(item map[string]any, selected map[string]bool) []string {
	raw, _ := item["requires_skills_any"].([]any)
	out := []string{}
	for _, v := range raw {
		if selected[fmt.Sprint(v)] {
			out = append(out, fmt.Sprint(v))
		}
	}
	sort.Strings(out)
	return out
}
func (c Catalog) Resolve(p Profile) Resolution {
	skills := byID(c.Sections["skills"])
	agents := byID(c.Sections["agents"])
	recipes := byID(c.Sections["recipes"])
	selectedSkills, selectedAgents, selectedRecipes := map[string]bool{}, map[string]bool{}, map[string]bool{}
	reasons := map[string][]string{}
	traces := map[string][]map[string]any{}
	for _, bundle := range c.Sections["bundles"] {
		if strings.ToLower(fmt.Sprint(bundle["project_type"])) != "" && p.ProjectTypes()[strings.ToLower(fmt.Sprint(bundle["project_type"]))] {
			bt := map[string]any{"type": "bundle", "value": bundle["id"]}
			for _, pair := range []struct {
				key string
				out map[string]bool
			}{{"skills", selectedSkills}, {"agents", selectedAgents}, {"recipes", selectedRecipes}} {
				if raw, ok := bundle[pair.key].([]any); ok {
					for _, v := range raw {
						id := fmt.Sprint(v)
						pair.out[id] = true
						reasons[id] = append(reasons[id], "bundle:"+fmt.Sprint(bundle["id"]))
						traces[id] = append(traces[id], bt)
					}
				}
			}
		}
	}
	// Reasons for rejection, so "why is this skill not in my plan" has an
	// answer that is not silence (GAP-146).
	rejected := map[string]string{}

	// Constraints are evaluated in two passes because `conflicts` is about the
	// set, not about one item: whether X conflicts with Y depends on both being
	// selected, and a single forward pass would decide that from whichever came
	// first in map order.
	candidate := map[string]map[string]any{}
	candidateReason := map[string][]string{}
	candidateTrace := map[string][]map[string]any{}
	for id, item := range skills {
		ok, why, tr := selectorMatches(item, p)
		if !ok {
			rejected[id] = "selector did not match this project"
			continue
		}
		if reason, blocked := unmetRequirements(item, p); blocked {
			rejected[id] = reason
			continue
		}
		candidate[id] = item
		candidateReason[id] = why
		candidateTrace[id] = tr
	}
	// Skills a bundle already selected are part of the plan before this point,
	// so the dependency and conflict passes see them.
	closeDependencies(candidate, skills, selectedSkills, rejected)
	// A prerequisite is pulled in even when its own selector did not match: the
	// dependent is what asked for it, and dropping the dependent because a
	// general-purpose skill was not independently selected is how a language
	// skill ends up unresolvable for every project.
	pullPrerequisites(candidate, skills, selectedSkills, rejected)
	for id := range candidate {
		selectedSkills[id] = true
		if len(candidateReason[id]) > 0 {
			reasons[id] = append(reasons[id], candidateReason[id]...)
		}
		if len(candidateTrace[id]) > 0 {
			traces[id] = append(traces[id], candidateTrace[id]...)
		}
	}
	for _, rule := range c.Sections["risk_rules"] {
		trigger := stringSet(rule["when_any"])
		universe := p.Risks()
		for k := range p.Features() {
			universe[k] = true
		}
		for k := range p.ProjectTypes() {
			universe[k] = true
		}
		hit := false
		for k := range trigger {
			if universe[k] {
				hit = true
			}
		}
		if hit {
			trace := map[string]any{"type": "risk-rule", "value": rule["id"]}
			if raw, ok := rule["require_skills"].([]any); ok {
				for _, v := range raw {
					id := fmt.Sprint(v)
					selectedSkills[id] = true
					reasons[id] = append(reasons[id], "risk-rule:"+fmt.Sprint(rule["id"]))
					traces[id] = append(traces[id], trace)
				}
			}
		}
	}
	for id, item := range agents {
		ok, why, tr := selectorMatches(item, p)
		required := intersectRequired(item, selectedSkills)
		if ok || len(required) > 0 {
			selectedAgents[id] = true
			if len(why) == 0 {
				why = []string{"skill-match"}
				tr = []map[string]any{{"type": "skill-match", "value": required}}
			}
			reasons[id] = append(reasons[id], why...)
			traces[id] = append(traces[id], tr...)
		}
	}
	for id, item := range recipes {
		ok, why, tr := selectorMatches(item, p)
		required := intersectRequired(item, selectedSkills)
		if ok || len(required) > 0 {
			selectedRecipes[id] = true
			if len(why) == 0 {
				why = []string{"skill-match"}
				tr = []map[string]any{{"type": "skill-match", "value": required}}
			}
			reasons[id] = append(reasons[id], why...)
			traces[id] = append(traces[id], tr...)
		}
	}
	changed := true
	for changed {
		changed = false
		for id := range selectedSkills {
			raw, _ := skills[id]["requires"].([]any)
			for _, v := range raw {
				dep := fmt.Sprint(v)
				if !selectedSkills[dep] {
					selectedSkills[dep] = true
					reasons[dep] = append(reasons[dep], "dependency:"+id)
					traces[dep] = append(traces[dep], map[string]any{"type": "dependency", "value": id})
					changed = true
				}
			}
		}
	}
	clean := func(values map[string]bool) []string { return sortedKeys(values) }
	for id, values := range reasons {
		set := map[string]bool{}
		for _, v := range values {
			set[v] = true
		}
		reasons[id] = sortedKeys(set)
	}
	resolution := Resolution{
		Agents: clean(selectedAgents), Skills: clean(selectedSkills), Recipes: clean(selectedRecipes),
		Reasons: reasons, Traces: traces,
	}
	// The reasons for rejection are the point: without them a skill that was
	// dropped for a declared conflict is indistinguishable from one nobody
	// looked at (GAP-146).
	if len(rejected) > 0 {
		resolution.Rejected = rejected
	}
	return resolution
}
