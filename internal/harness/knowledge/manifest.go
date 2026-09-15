package knowledge

import (
	"sort"
	"strings"

	coreknowledge "github.com/raillen/prumo/internal/knowledge"
)

// relationVocabulary maps harness edge types to the canonical typed
// relationship vocabulary, so a manifest speaks the same language as the
// schema instead of exposing harness-internal spellings (W3.5).
var relationVocabulary = map[string]coreknowledge.Relationship{
	"supports":    coreknowledge.RelRequires,
	"contradicts": coreknowledge.RelContradicts,
	"refines":     coreknowledge.RelExplains,
	"supersedes":  coreknowledge.RelSupersedes,
	"evidences":   coreknowledge.RelVerifies,
	"covers":      coreknowledge.RelDocuments,
}

// CanonicalRelation reports the schema relationship for a harness edge type.
func CanonicalRelation(edge string) (coreknowledge.Relationship, bool) {
	rel, ok := relationVocabulary[strings.ToLower(strings.TrimSpace(edge))]
	return rel, ok
}

// Manifest builds the derived knowledge index for the store (W3.6). It is a
// projection: it may be regenerated at will and is never canonical. Locators
// combine the record's stable id, its provenance and every registered alias, so
// a lookup by an old path still finds the unit.
func (s *Store) Manifest(generatedAt, sourceRevision string) (coreknowledge.Manifest, []error) {
	records, rels := s.Snapshot()
	byID := map[string][]coreknowledge.Relationship{}
	for _, rel := range rels {
		canonical, ok := CanonicalRelation(rel.Type)
		if !ok {
			continue
		}
		byID[rel.From] = append(byID[rel.From], canonical)
	}
	locatorsByID := map[string][]string{}
	for locator, id := range s.aliases {
		locatorsByID[id] = append(locatorsByID[id], locator)
	}

	inputs := make([]coreknowledge.UnitInput, 0, len(records))
	for _, r := range records {
		locators := []string{r.ID}
		if r.Provenance != "" {
			locators = append(locators, r.Provenance)
		}
		locators = append(locators, locatorsByID[r.ID]...)
		visibility := visibilityOf(r)
		inputs = append(inputs, coreknowledge.UnitInput{
			ID:            coreknowledge.ID(r.ID),
			Kind:          kindOf(r),
			Title:         r.Title,
			Authority:     r.Authority,
			Visibility:    visibility,
			Revision:      1,
			Locators:      locators,
			Relationships: byID[r.ID],
		})
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].ID < inputs[j].ID })
	return coreknowledge.BuildManifest(generatedAt, sourceRevision, inputs)
}

// visibilityOf maps a record's sensitivity to the projection vocabulary.
// A record that does not declare sensitivity is internal, never public: the
// mapping fails closed (W15.25).
func visibilityOf(r Record) coreknowledge.Visibility {
	switch strings.ToLower(strings.TrimSpace(r.Sensitivity)) {
	case "public":
		return coreknowledge.VisibilityPublic
	case "internal", "":
		return coreknowledge.VisibilityInternal
	case "confidential":
		return coreknowledge.VisibilityConfidential
	case "restricted":
		return coreknowledge.VisibilityRestricted
	default:
		// An unknown sensitivity is treated as the most restrictive known class
		// rather than silently leaking into a public projection.
		return coreknowledge.VisibilityRestricted
	}
}

// kindOf maps a harness record kind to the canonical vocabulary.
func kindOf(r Record) coreknowledge.Kind {
	if mapped, ok := CanonicalKind(r.Kind); ok {
		return mapped
	}
	return coreknowledge.Kind(r.Kind)
}
