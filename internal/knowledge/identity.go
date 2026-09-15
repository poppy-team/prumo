// Package knowledge defines stable identity and vocabulary for durable
// semantic entities. Identity is independent of file path and title so that
// renames and moves never create a new entity (W3).
package knowledge

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Kind classifies a KnowledgeUnit. Mirrors schemas/knowledge-unit.schema.json.
type Kind string

const (
	KindRequirement  Kind = "requirement"
	KindClaim        Kind = "claim"
	KindDecision     Kind = "decision"
	KindConstraint   Kind = "constraint"
	KindRisk         Kind = "risk"
	KindAssumption   Kind = "assumption"
	KindOpenQuestion Kind = "open-question"
	KindSource       Kind = "source"
	KindFinding      Kind = "finding"
	KindEvidence     Kind = "evidence"
	KindResearch     Kind = "research"
	KindMemory       Kind = "memory"
)

// Kinds returns every valid Kind in sorted order.
func Kinds() []Kind {
	return []Kind{KindAssumption, KindClaim, KindConstraint, KindDecision, KindEvidence, KindFinding, KindMemory, KindOpenQuestion, KindRequirement, KindResearch, KindRisk, KindSource}
}

// Relationship names a typed edge. Free-form relationship strings are invalid.
type Relationship string

const (
	RelDocuments     Relationship = "documents"
	RelExplains      Relationship = "explains"
	RelReferences    Relationship = "references"
	RelImplements    Relationship = "implements"
	RelVerifies      Relationship = "verifies"
	RelSupersedes    Relationship = "supersedes"
	RelContradicts   Relationship = "contradicts"
	RelProjectsTo    Relationship = "projects_to"
	RelTranslatedAs  Relationship = "translated_as"
	RelIllustratedBy Relationship = "illustrated_by"
	RelAffects       Relationship = "affects"
	RelRequires      Relationship = "requires"
	RelSupportedBy   Relationship = "supported_by"
)

// Relationships returns every valid Relationship in sorted order.
func Relationships() []Relationship {
	return []Relationship{RelAffects, RelContradicts, RelDocuments, RelExplains, RelIllustratedBy, RelImplements, RelProjectsTo, RelReferences, RelRequires, RelSupersedes, RelSupportedBy, RelTranslatedAs, RelVerifies}
}

// Visibility mirrors the projection policy (W15.25): a projection must never
// leak content classified above its own visibility.
type Visibility string

const (
	VisibilityPublic       Visibility = "public"
	VisibilityProject      Visibility = "project"
	VisibilityInternal     Visibility = "internal"
	VisibilityRestricted   Visibility = "restricted"
	VisibilityConfidential Visibility = "confidential"
)

var visibilityRank = map[Visibility]int{
	VisibilityPublic: 0, VisibilityProject: 1, VisibilityInternal: 2,
	VisibilityRestricted: 3, VisibilityConfidential: 4,
}

// VisibleTo reports whether content at visibility v may appear in a surface
// rendered at ceiling. Unknown visibilities are not visible (fail-closed).
func VisibleTo(v, ceiling Visibility) bool {
	vr, ok := visibilityRank[v]
	if !ok {
		return false
	}
	cr, ok := visibilityRank[ceiling]
	if !ok {
		return false
	}
	return vr <= cr
}

// ID is a stable identifier for a durable semantic entity.
type ID string

var idPattern = regexp.MustCompile(`^ku:[a-z0-9][a-z0-9._-]*$`)
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// NewID builds a stable ID from a domain slug.
func NewID(slug string) (ID, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if !slugPattern.MatchString(slug) {
		return "", fmt.Errorf("invalid knowledge slug %q: use [a-z0-9._-] starting alphanumeric", slug)
	}
	return ID("ku:" + slug), nil
}

// ParseID validates an existing ID string.
func ParseID(s string) (ID, error) {
	s = strings.TrimSpace(s)
	if !idPattern.MatchString(s) {
		return "", fmt.Errorf("invalid knowledge id %q: expected ku:<slug>", s)
	}
	return ID(s), nil
}

func (id ID) String() string { return string(id) }

func (id ID) Valid() bool { return idPattern.MatchString(string(id)) }

// AliasTable maps historical locators (paths, anchors, old slugs) to stable
// IDs so identity survives renames and moves (W3.7, audit DOC-GAP-011).
type AliasTable struct {
	byLocator map[string]ID
}

// NewAliasTable builds a table from id -> locators.
func NewAliasTable(entries map[ID][]string) *AliasTable {
	t := &AliasTable{byLocator: map[string]ID{}}
	for id, locators := range entries {
		if !id.Valid() {
			continue
		}
		for _, loc := range locators {
			loc = strings.TrimSpace(loc)
			if loc == "" {
				continue
			}
			t.byLocator[loc] = id
		}
	}
	return t
}

// Resolve returns the stable ID for a locator, if known.
func (t *AliasTable) Resolve(locator string) (ID, bool) {
	if t == nil {
		return "", false
	}
	id, ok := t.byLocator[strings.TrimSpace(locator)]
	return id, ok
}

// Add registers a new locator for an ID (a rename appends; it never replaces).
func (t *AliasTable) Add(id ID, locator string) {
	if t.byLocator == nil {
		t.byLocator = map[string]ID{}
	}
	if !id.Valid() {
		return
	}
	locator = strings.TrimSpace(locator)
	if locator != "" {
		t.byLocator[locator] = id
	}
}

// Locators lists every locator bound to an ID, sorted.
func (t *AliasTable) Locators(id ID) []string {
	var out []string
	if t == nil {
		return out
	}
	for loc, bound := range t.byLocator {
		if bound == id {
			out = append(out, loc)
		}
	}
	sort.Strings(out)
	return out
}
