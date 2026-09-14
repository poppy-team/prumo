// Package humandocs implements the Human Documentation Runtime baseline
// (HD0–HD4): DocumentationSpec + profile composition, deterministic planner
// producing units/page graph, README + reference + tree generation, assisted
// prose briefs, and coverage/readiness gates. Generation policy per intent:
//
//	GENERATED: fully rebuilt from sources (deterministic, no-op writes)
//	ASSISTED:  scaffold + TODO(CURATED) briefs; prose stays human-owned
//	CURATED:   never touched by the compiler
//
// Missing knowledge is reported as gaps, never hallucinated into prose.
package humandocs

import (
	"fmt"
	"sort"

	docengine "github.com/raillen/prumo/internal/documentation"
)

// Intent distinguishes manual surfaces (page 36: they stay distinct).
type Intent string

const (
	IntentREADME      Intent = "readme"
	IntentReference   Intent = "reference"
	IntentTutorial    Intent = "tutorial"
	IntentHowTo       Intent = "how-to"
	IntentConcepts    Intent = "concepts"
	IntentManual      Intent = "manual"
	IntentExplanation Intent = "explanation"
)

// Policy is the generation policy for one intent.
type Policy string

const (
	PolicyGenerated Policy = "GENERATED"
	PolicyAssisted  Policy = "ASSISTED"
	PolicyCurated   Policy = "CURATED"
)

// Spec (HD0) binds audiences, profiles, intents and source pointers.
type Spec struct {
	ID       string              `json:"id"`
	Title    string              `json:"title"`
	Audience string              `json:"audience"`
	Profiles []string            `json:"profiles"`
	Policies map[Intent]Policy   `json:"policies,omitempty"`
	Sources  map[string][]string `json:"sources,omitempty"` // contract-id -> source refs
	OutDir   string              `json:"out_dir,omitempty"`
}

// PolicyFor returns the effective policy (default: GENERATED for
// readme/reference, ASSISTED for the rest, CURATED never defaulted).
func (s Spec) PolicyFor(intent Intent) Policy {
	if p, ok := s.Policies[intent]; ok {
		return p
	}
	switch intent {
	case IntentREADME, IntentReference:
		return PolicyGenerated
	default:
		return PolicyAssisted
	}
}

// ComposeProfile (HD0) merges a base profile with overlays deterministically:
// union of capabilities/contracts minus excluded, sorted.
func ComposeProfile(base docengine.Profile, overlays ...docengine.Profile) docengine.Profile {
	out := docengine.Profile{ID: base.ID, Description: base.Description}
	caps := map[string]bool{}
	contracts := map[string]bool{}
	excluded := map[string]bool{}
	add := func(p docengine.Profile) {
		for _, c := range p.Capabilities {
			caps[c] = true
		}
		for _, c := range p.Contracts {
			contracts[c] = true
		}
		for _, c := range p.ExcludedContracts {
			excluded[c] = true
		}
	}
	add(base)
	for _, o := range overlays {
		add(o)
		if o.ID != "" {
			out.Extends = append(out.Extends, o.ID)
		}
	}
	for c := range caps {
		out.Capabilities = append(out.Capabilities, c)
	}
	for c := range contracts {
		if !excluded[c] {
			out.Contracts = append(out.Contracts, c)
		}
	}
	sort.Strings(out.Capabilities)
	sort.Strings(out.Contracts)
	sort.Strings(out.Extends)
	return out
}

func shortDesc(s string) string {
	if i := len(s); i > 80 {
		// cut at word boundary
		cut := 80
		for cut > 60 && s[cut] != ' ' {
			cut--
		}
		return s[:cut] + "…"
	}
	return s
}

// Unit (HD1) is one plannable documentation node.
type Unit struct {
	ID       string   `json:"id"`
	Intent   Intent   `json:"intent"`
	Audience string   `json:"audience"`
	Title    string   `json:"title"`
	Sources  []string `json:"sources,omitempty"`
	Required bool     `json:"required"`
	// Status: ready | missing (no source) — never invented.
	Status string `json:"status"`
}

// Page binds a unit to a tree path with navigation links.
type Page struct {
	Unit  Unit     `json:"unit"`
	Path  string   `json:"path"`
	Links []string `json:"links,omitempty"` // unit ids, deterministic
}

// Graph is the navigable page DAG.
type Graph struct {
	Pages []Page `json:"pages"`
}

// Order topologically sorts pages by Links; cycles are errors.
func (g Graph) Order() ([]Page, error) {
	index := map[string]int{}
	for i, p := range g.Pages {
		index[p.Unit.ID] = i
	}
	state := map[string]int{} // 0 unvisited, 1 in stack, 2 done
	var out []Page
	var visit func(id string) error
	visit = func(id string) error {
		switch state[id] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("page graph cycle at %s", id)
		}
		state[id] = 1
		i, ok := index[id]
		if !ok {
			return fmt.Errorf("unknown unit %s", id)
		}
		links := append([]string{}, g.Pages[i].Links...)
		sort.Strings(links)
		for _, dep := range links {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		out = append(out, g.Pages[i])
		return nil
	}
	ids := make([]string, 0, len(g.Pages))
	for _, p := range g.Pages {
		ids = append(ids, p.Unit.ID)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Plan is the planner output: units, graph, gaps.
type Plan struct {
	Units []Unit   `json:"units"`
	Graph Graph    `json:"graph"`
	Gaps  []string `json:"gaps,omitempty"`
}

// Planner (HD1) derives units from spec profiles + registry contracts.
// A contract without sources becomes a missing gap, never prose.
func Planner(spec Spec, reg docengine.Registry) (Plan, error) {
	units := []Unit{}
	seen := map[string]bool{}
	add := func(u Unit) {
		if seen[u.ID] {
			return
		}
		seen[u.ID] = true
		units = append(units, u)
	}
	add(Unit{ID: spec.ID + ":readme", Intent: IntentREADME, Audience: spec.Audience,
		Title: spec.Title, Required: true, Status: "ready", Sources: []string{"spec:" + spec.ID}})
	add(Unit{ID: spec.ID + ":reference", Intent: IntentReference, Audience: spec.Audience,
		Title: spec.Title + " reference", Required: true, Status: "ready", Sources: []string{"contracts"}})
	intents := []Intent{IntentTutorial, IntentHowTo, IntentConcepts, IntentManual}
	profiles := append([]string{}, spec.Profiles...)
	sort.Strings(profiles)
	var gaps []string
	for _, pid := range profiles {
		prof, ok := reg.Profiles[pid]
		if !ok {
			gaps = append(gaps, "profile:"+pid)
			continue
		}
		contracts := append([]string{}, prof.Contracts...)
		sort.Strings(contracts)
		for _, cid := range contracts {
			c, ok := reg.Contracts[cid]
			if !ok {
				gaps = append(gaps, "contract:"+cid)
				continue
			}
			for _, intent := range intents {
				uid := spec.ID + ":" + string(intent) + ":" + cid
				srcs := append([]string{}, spec.Sources[cid]...)
				status := "ready"
				if len(srcs) == 0 {
					status = "missing"
					gaps = append(gaps, uid)
				}
				add(Unit{ID: uid, Intent: intent, Audience: spec.Audience,
					Title: string(intent) + ": " + shortDesc(c.Description), Required: false,
					Status: status, Sources: srcs})
			}
		}
	}
	sort.Strings(gaps)
	pages := make([]Page, 0, len(units))
	for _, u := range units {
		path := "docs/" + string(u.Intent) + "/" + u.ID + ".md"
		if u.Intent == IntentREADME {
			path = "README.md"
		} else if u.Intent == IntentReference {
			path = "docs/reference.md"
		}
		pages = append(pages, Page{Unit: u, Path: path})
	}
	// Deterministic nav: every page links back to the readme unit.
	for i := range pages {
		if pages[i].Unit.Intent != IntentREADME {
			pages[i].Links = []string{spec.ID + ":readme"}
		}
	}
	plan := Plan{Units: units, Graph: Graph{Pages: pages}, Gaps: gaps}
	if _, err := plan.Graph.Order(); err != nil {
		return Plan{}, err
	}
	return plan, nil
}
