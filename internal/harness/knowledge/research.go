// Research ledger (GAP-026, page 27-G11): first-class research records
// with open/resolved lifecycle over the canonical store. Queries are
// views; mutation stays Delta-first like all agent writes.
package knowledge

import (
	"fmt"
	"sort"
	"time"
)

// AddResearch opens a research record.
func AddResearch(s *Store, id, title, body, provenance string) error {
	return s.Commit(Delta{ID: "research-" + id, Author: "harness-run", Upserts: []Record{{
		ID: id, Kind: KindResearch, Title: title, Body: body,
		Authority: "reference", Trust: "medium", Status: "active",
		Provenance: provenance, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}}})
}

// ResolveResearch marks a research record resolved with its finding.
func ResolveResearch(s *Store, id, finding string) error {
	r, ok := s.Get(id)
	if !ok || r.Kind != KindResearch {
		return fmt.Errorf("unknown research %s", id)
	}
	r.Status = "resolved"
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	upserts := []Record{r}
	links := []Relation{}
	if finding != "" {
		f := Record{ID: id + "-finding", Kind: KindFinding, Title: "finding for " + id,
			Body: finding, Authority: "reference", Trust: "medium", Status: "active",
			Provenance: r.Provenance, UpdatedAt: r.UpdatedAt}
		upserts = append(upserts, f)
		links = append(links, Relation{From: f.ID, Type: "evidences", To: id})
	}
	return s.Commit(Delta{ID: "resolve-" + id, Author: "harness-run", Upserts: upserts, Links: links})
}

// OpenResearch lists unresolved research records, oldest first.
func OpenResearch(s *Store) []Record {
	var out []Record
	for _, r := range s.Search("", []Kind{KindResearch}) {
		if r.Status == "active" {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt < out[j].UpdatedAt })
	return out
}
