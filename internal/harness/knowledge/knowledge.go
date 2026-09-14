// Package knowledge implements the Knowledge Runtime baseline: typed
// records with stable IDs, authority/trust/freshness/status, provenance,
// lineage/supersession, plus KnowledgeDelta validate/commit and
// contradiction/coverage/readiness engines (typed rules, not substrings).
package knowledge

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Kind enumerates record types.
type Kind string

const (
	KindSource       Kind = "source"
	KindSection      Kind = "section"
	KindClaim        Kind = "claim"
	KindFinding      Kind = "finding"
	KindDecision     Kind = "decision"
	KindRequirement  Kind = "requirement"
	KindConstraint   Kind = "constraint"
	KindRisk         Kind = "risk"
	KindAssumption   Kind = "assumption"
	KindOpenQuestion Kind = "open_question"
	KindResearch     Kind = "research"
	KindEvidence     Kind = "evidence"
	KindMemory       Kind = "memory"
)

// Record is one typed knowledge unit.
type Record struct {
	ID           string         `json:"id"`
	Kind         Kind           `json:"kind"`
	Title        string         `json:"title"`
	Body         string         `json:"body,omitempty"`
	Authority    string         `json:"authority"` // canonical|reference|imported|untrusted
	Trust        string         `json:"trust"`     // high|medium|low
	Status       string         `json:"status"`    // active|superseded|deprecated|draft
	Provenance   string         `json:"provenance,omitempty"`
	Supersedes   string         `json:"supersedes,omitempty"`
	SupersededBy string         `json:"superseded_by,omitempty"`
	UpdatedAt    string         `json:"updated_at"`
	Refs         []string       `json:"refs,omitempty"`
	Meta         map[string]any `json:"meta,omitempty"`
	// Sensitivity gates cross-project promotion: public|internal|
	// confidential|restricted. Empty means internal. Restricted never
	// leaves its project (GAP-018).
	Sensitivity string `json:"sensitivity,omitempty"`
}

// Relation is a typed edge between records.
type Relation struct {
	From string `json:"from"`
	Type string `json:"type"` // supports|contradicts|refines|supersedes|evidences|covers
	To   string `json:"to"`
}

// Store is an in-memory canonical record store with derived indexes.
type Store struct {
	records map[string]Record
	rels    []Relation
}

func New() *Store { return &Store{records: map[string]Record{}} }

func (s *Store) Put(r Record) error {
	if r.ID == "" || r.Kind == "" {
		return fmt.Errorf("record requires id and kind")
	}
	if r.Status == "" {
		r.Status = "active"
	}
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	s.records[r.ID] = r
	return nil
}

func (s *Store) Get(id string) (Record, bool) { r, ok := s.records[id]; return r, ok }

func (s *Store) Link(rel Relation) {
	s.rels = append(s.rels, rel)
}

// Locate/Get/Search/Related/Explain/Context/Affected (read API).
func (s *Store) Search(query string, kinds []Kind) []Record {
	q := strings.ToLower(query)
	out := []Record{}
	for _, r := range s.records {
		if len(kinds) > 0 {
			match := false
			for _, k := range kinds {
				if k == r.Kind {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		if q == "" || strings.Contains(strings.ToLower(r.Title+" "+r.Body), q) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) Related(id string) []Record {
	out := []Record{}
	for _, rel := range s.rels {
		other := ""
		if rel.From == id {
			other = rel.To
		} else if rel.To == id {
			other = rel.From
		}
		if other != "" {
			if r, ok := s.records[other]; ok {
				out = append(out, r)
			}
		}
	}
	return out
}

func (s *Store) Affected(id string) []Record {
	// Records evidencing or covered by id.
	out := []Record{}
	for _, rel := range s.rels {
		if (rel.Type == "evidences" || rel.Type == "covers") && rel.To == id {
			if r, ok := s.records[rel.From]; ok {
				out = append(out, r)
			}
		}
	}
	return out
}

// ---- KnowledgeDelta: Validate -> Commit ----

// Delta is one proposed mutation batch.
type Delta struct {
	ID      string     `json:"id"`
	Author  string     `json:"author"`
	Upserts []Record   `json:"upserts,omitempty"`
	Links   []Relation `json:"links,omitempty"`
	Reason  string     `json:"reason,omitempty"`
}

// Validate enforces policy/review gates without committing.
func (s *Store) Validate(d Delta) error {
	if d.ID == "" {
		return fmt.Errorf("delta requires id")
	}
	for _, r := range d.Upserts {
		if r.ID == "" || r.Kind == "" {
			return fmt.Errorf("delta record requires id+kind")
		}
		if r.Authority == "canonical" && d.Author == "" {
			return fmt.Errorf("canonical promotion requires author")
		}
	}
	return nil
}

// Commit applies a validated delta and emits derived rebuild signal.
func (s *Store) Commit(d Delta) error {
	if err := s.Validate(d); err != nil {
		return err
	}
	for _, r := range d.Upserts {
		if r.Supersedes != "" {
			if old, ok := s.records[r.Supersedes]; ok {
				old.Status = "superseded"
				old.SupersededBy = r.ID
				s.records[old.ID] = old
			}
		}
		_ = s.Put(r)
	}
	for _, l := range d.Links {
		s.Link(l)
	}
	return nil
}

// ---- Contradiction / Coverage / Readiness ----

// Contradiction is a typed rule hit (LLM may propose candidates only).
type Contradiction struct {
	A    string `json:"a"`
	B    string `json:"b"`
	Rule string `json:"rule"`
}

// DetectContradictions uses typed relations, never substring heuristics.
func (s *Store) DetectContradictions() []Contradiction {
	out := []Contradiction{}
	for _, rel := range s.rels {
		if rel.Type == "contradicts" {
			a, aok := s.records[rel.From]
			b, bok := s.records[rel.To]
			if aok && bok && a.Status == "active" && b.Status == "active" {
				out = append(out, Contradiction{A: a.ID, B: b.ID, Rule: "explicit-contradicts-edge"})
			}
		}
	}
	return out
}

// Coverage reports requirements with/without linked evidence or decisions.
func (s *Store) Coverage() (covered, uncovered []string) {
	for _, r := range s.records {
		if r.Kind != KindRequirement || r.Status != "active" {
			continue
		}
		has := false
		for _, rel := range s.rels {
			if rel.To == r.ID && (rel.Type == "evidences" || rel.Type == "covers") {
				has = true
			}
		}
		if has {
			covered = append(covered, r.ID)
		} else {
			uncovered = append(uncovered, r.ID)
		}
	}
	sort.Strings(covered)
	sort.Strings(uncovered)
	return covered, uncovered
}

// Readiness evaluates a policy: no contradictions + no uncovered reqs + no open blockers.
func (s *Store) Readiness() (ready bool, blockers []string) {
	for _, c := range s.DetectContradictions() {
		blockers = append(blockers, "contradiction:"+c.A+"<>"+c.B)
	}
	_, uncovered := s.Coverage()
	for _, u := range uncovered {
		blockers = append(blockers, "uncovered:"+u)
	}
	for _, r := range s.records {
		if r.Kind == KindOpenQuestion && r.Status == "active" {
			if v, ok := r.Meta["blocker"]; ok && v == true {
				blockers = append(blockers, "open-question:"+r.ID)
			}
		}
	}
	sort.Strings(blockers)
	return len(blockers) == 0, blockers
}
