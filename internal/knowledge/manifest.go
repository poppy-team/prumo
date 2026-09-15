package knowledge

import (
	"fmt"
	"sort"
	"strings"
)

// UnitInput is the minimal description a manifest builder needs about one
// durable semantic entity.
type UnitInput struct {
	ID            ID
	Kind          Kind
	Title         string
	Authority     string
	Visibility    Visibility
	Revision      int
	Locators      []string
	Relationships []Relationship
}

// Unit is one entry of a derived knowledge manifest.
type Unit struct {
	ID            ID             `json:"id"`
	Kind          Kind           `json:"kind"`
	Title         string         `json:"title,omitempty"`
	Authority     string         `json:"authority"`
	Visibility    Visibility     `json:"visibility"`
	Revision      int            `json:"revision,omitempty"`
	Locators      []string       `json:"locators"`
	Relationships []Relationship `json:"relationships,omitempty"`
}

// ManifestCounts is the summary block of a manifest.
type ManifestCounts struct {
	Total        int            `json:"total"`
	ByKind       map[string]int `json:"by_kind,omitempty"`
	ByVisibility map[string]int `json:"by_visibility,omitempty"`
}

// Manifest is the derived index over durable entities (W3.6). It is a runtime
// projection: it may be regenerated at any time and must never be promoted to a
// canonical artifact.
type Manifest struct {
	Version        int            `json:"version"`
	GeneratedAt    string         `json:"generated_at"`
	SourceRevision string         `json:"source_revision"`
	Derived        bool           `json:"derived"`
	Units          []Unit         `json:"units"`
	Counts         ManifestCounts `json:"counts"`
}

// ManifestVersion is the schema version emitted by BuildManifest.
const ManifestVersion = 1

// BuildManifest assembles a deterministic manifest from units. It returns the
// manifest together with every validation problem found, so a caller may report
// a partially valid index instead of silently dropping bad entries.
//
// Determinism: units are ordered by ID, locators and relationships are sorted
// and de-duplicated, and counts are derived from the emitted units only.
func BuildManifest(generatedAt, sourceRevision string, inputs []UnitInput) (Manifest, []error) {
	manifest := Manifest{
		Version:        ManifestVersion,
		GeneratedAt:    generatedAt,
		SourceRevision: sourceRevision,
		Derived:        true,
		Units:          []Unit{},
		Counts: ManifestCounts{
			ByKind:       map[string]int{},
			ByVisibility: map[string]int{},
		},
	}
	problems := []error{}
	seenID := map[ID]bool{}
	locatorOwner := map[string]ID{}

	for _, in := range inputs {
		if !in.ID.Valid() {
			problems = append(problems, fmt.Errorf("unit %q has an invalid stable id", in.ID))
			continue
		}
		if seenID[in.ID] {
			problems = append(problems, fmt.Errorf("duplicate stable id %s", in.ID))
			continue
		}
		if strings.TrimSpace(string(in.Kind)) == "" {
			problems = append(problems, fmt.Errorf("unit %s has no kind", in.ID))
			continue
		}
		if strings.TrimSpace(in.Authority) == "" {
			problems = append(problems, fmt.Errorf("unit %s has no authority", in.ID))
			continue
		}
		if _, ok := visibilityRank[in.Visibility]; !ok {
			problems = append(problems, fmt.Errorf("unit %s has an unknown visibility %q", in.ID, in.Visibility))
			continue
		}
		if in.Revision < 1 {
			problems = append(problems, fmt.Errorf("unit %s has no revision", in.ID))
			continue
		}
		locators := normalizeLocators(in.Locators)
		for _, loc := range locators {
			if owner, taken := locatorOwner[loc]; taken && owner != in.ID {
				problems = append(problems, fmt.Errorf("locator %q is bound to both %s and %s", loc, owner, in.ID))
			}
			locatorOwner[loc] = in.ID
		}
		seenID[in.ID] = true
		manifest.Units = append(manifest.Units, Unit{
			ID: in.ID, Kind: in.Kind, Title: in.Title, Authority: in.Authority,
			Visibility: in.Visibility, Revision: in.Revision, Locators: locators,
			Relationships: normalizeRelationships(in.Relationships),
		})
	}

	sort.Slice(manifest.Units, func(i, j int) bool { return manifest.Units[i].ID < manifest.Units[j].ID })
	manifest.Counts.Total = len(manifest.Units)
	for _, unit := range manifest.Units {
		manifest.Counts.ByKind[string(unit.Kind)]++
		manifest.Counts.ByVisibility[string(unit.Visibility)]++
	}
	if len(manifest.Counts.ByKind) == 0 {
		manifest.Counts.ByKind = nil
	}
	if len(manifest.Counts.ByVisibility) == 0 {
		manifest.Counts.ByVisibility = nil
	}
	return manifest, problems
}

// Lookup resolves a locator (path, anchor or historical slug) to its unit,
// which is how identity survives a rename or a move (W3.7).
func (m Manifest) Lookup(locator string) (Unit, bool) {
	for _, unit := range m.Units {
		for _, loc := range unit.Locators {
			if loc == locator {
				return unit, true
			}
		}
	}
	return Unit{}, false
}

// Locators lists every locator declared by a unit of the same manifest.
func (m Manifest) Locators(id ID) []string {
	for _, unit := range m.Units {
		if unit.ID == id {
			return append([]string{}, unit.Locators...)
		}
	}
	return nil
}

func normalizeLocators(locators []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, loc := range locators {
		loc = strings.TrimSpace(loc)
		if loc == "" || seen[loc] {
			continue
		}
		seen[loc] = true
		out = append(out, loc)
	}
	sort.Strings(out)
	return out
}

func normalizeRelationships(rels []Relationship) []Relationship {
	seen := map[Relationship]bool{}
	out := []Relationship{}
	for _, rel := range rels {
		if !rel.Valid() || seen[rel] {
			continue
		}
		seen[rel] = true
		out = append(out, rel)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Valid reports whether a relationship is part of the typed vocabulary.
func (r Relationship) Valid() bool {
	for _, known := range Relationships() {
		if r == known {
			return true
		}
	}
	return false
}
