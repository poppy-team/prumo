// Generation (HD2) and assisted briefs (HD3): README + reference rebuilds
// go through doccompile (CAS/no-op atomic writes); missing units get index
// rows, never invented files. Briefs outline CURATED work with source
// pointers and blocking questions.
package humandocs

import (
	"fmt"
	"sort"
	"strings"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/harness/doccompile"
)

// GenerateREADME (HD2) writes the short entrypoint: title, one-liner,
// quickstart pointers and the unit index. Second run is a no-op.
func GenerateREADME(spec Spec, plan Plan, outPath string) (doccompile.Artifact, error) {
	var b strings.Builder
	b.WriteString("# " + spec.Title + "\n\n")
	b.WriteString("Audience: " + spec.Audience + "\n\n")
	b.WriteString("## Map\n\n")
	ordered, err := plan.Graph.Order()
	if err != nil {
		return doccompile.Artifact{}, err
	}
	for _, p := range ordered {
		mark := "ready"
		if p.Unit.Status != "ready" {
			mark = p.Unit.Status
		}
		fmt.Fprintf(&b, "- [%s](%s) — %s\n", p.Unit.Title, p.Path, mark)
	}
	b.WriteString("\n_Generated from spec " + spec.ID + "; curated pages stay human-owned._\n")
	return doccompile.Write(outPath, b.String())
}

// GenerateReference (HD2) renders contracts as tables (automation-first).
func GenerateReference(contracts []docengine.Contract, outPath string) (doccompile.Artifact, error) {
	ids := make([]string, 0, len(contracts))
	byID := map[string]docengine.Contract{}
	for _, c := range contracts {
		ids = append(ids, c.ID)
		byID[c.ID] = c
	}
	sort.Strings(ids)
	var b strings.Builder
	b.WriteString("# Reference\n\n")
	b.WriteString("| id | role | required knowledge | evidence |\n")
	b.WriteString("|----|------|--------------------|----------|\n")
	for _, id := range ids {
		c := byID[id]
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", id, c.Role,
			strings.Join(c.RequiredKnowledge, ", "), strings.Join(c.EvidenceRequirements, ", "))
	}
	return doccompile.Write(outPath, b.String())
}

// GenerateTree (HD2) writes README + reference + INDEX with unit statuses.
func GenerateTree(spec Spec, plan Plan, reg docengine.Registry, root string) ([]doccompile.Artifact, error) {
	var arts []doccompile.Artifact
	join := func(p string) string {
		if root == "" || root == "." {
			return p
		}
		return root + "/" + p
	}
	a, err := GenerateREADME(spec, plan, join("README.md"))
	if err != nil {
		return nil, err
	}
	arts = append(arts, a)
	var contracts []docengine.Contract
	for _, c := range reg.Contracts {
		contracts = append(contracts, c)
	}
	a, err = GenerateReference(contracts, join("docs/reference.md"))
	if err != nil {
		return nil, err
	}
	arts = append(arts, a)
	var b strings.Builder
	b.WriteString("# Index\n\n")
	units := append([]Unit{}, plan.Units...)
	sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })
	for _, u := range units {
		srcs := strings.Join(u.Sources, ", ")
		if srcs == "" {
			srcs = "—"
		}
		fmt.Fprintf(&b, "- %s [%s] policy=%s sources=%s\n", u.ID, u.Status, spec.PolicyFor(u.Intent), srcs)
	}
	a, err = doccompile.Write(join("docs/INDEX.md"), b.String())
	if err != nil {
		return nil, err
	}
	return append(arts, a), nil
}

// Brief (HD3) outlines one assisted/curated unit: sections derived from the
// contract's required knowledge, blocking questions as TODO(CURATED), and
// source pointers. It contains no claims beyond its inputs.
func Brief(u Unit, c docengine.Contract) string {
	var b strings.Builder
	b.WriteString("# Brief: " + u.Title + "\n\n")
	b.WriteString("Intent: " + string(u.Intent) + " | Audience: " + u.Audience + "\n\n")
	b.WriteString("## Sections (from required knowledge)\n\n")
	for _, k := range c.RequiredKnowledge {
		b.WriteString("- [ ] " + k + "\n")
	}
	if len(c.RequiredKnowledge) == 0 {
		b.WriteString("- [ ] scope (no required knowledge declared)\n")
	}
	b.WriteString("\n## Blocking questions → TODO(CURATED)\n\n")
	for _, q := range c.BlockingQuestions {
		b.WriteString("- TODO(CURATED): " + q + "\n")
	}
	if len(c.BlockingQuestions) == 0 {
		b.WriteString("- none declared\n")
	}
	b.WriteString("\n## Sources\n\n")
	for _, s := range u.Sources {
		b.WriteString("- " + s + "\n")
	}
	if len(u.Sources) == 0 {
		b.WriteString("- none: unit status is missing; do not write prose without sources\n")
	}
	b.WriteString("\n## Evidence required\n\n")
	for _, e := range c.EvidenceRequirements {
		b.WriteString("- " + e + "\n")
	}
	return b.String()
}

// Coverage (HD4) splits units into ready vs missing.
func Coverage(plan Plan) (ready, missing []string) {
	for _, u := range plan.Units {
		if u.Status == "ready" {
			ready = append(ready, u.ID)
		} else {
			missing = append(missing, u.ID)
		}
	}
	sort.Strings(ready)
	sort.Strings(missing)
	return ready, missing
}

// Readiness (HD4) gates publication: missing REQUIRED units and structural
// gaps (unknown profiles/contracts) block; optional-unit gaps are warnings.
func Readiness(plan Plan) (bool, []string) {
	required := map[string]bool{}
	for _, u := range plan.Units {
		if u.Required {
			required[u.ID] = true
		}
	}
	var blockers []string
	for _, u := range plan.Units {
		if u.Required && u.Status != "ready" {
			blockers = append(blockers, "missing-required:"+u.ID)
		}
	}
	for _, g := range plan.Gaps {
		if strings.HasPrefix(g, "profile:") || strings.HasPrefix(g, "contract:") || required[g] {
			blockers = append(blockers, "gap:"+g)
		}
	}
	sort.Strings(blockers)
	return len(blockers) == 0, blockers
}
