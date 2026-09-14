// Run seeding: every headless run leaves typed Knowledge behind — a
// Requirement seeded from the goal at start, Evidence linked at finish.
// Per-run stores persist as JSON so restarts and reviews can read them;
// global promotion still goes through KnowledgeDelta Validate→Commit.
package knowledge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Snapshot exports records and relations in deterministic order.
func (s *Store) Snapshot() ([]Record, []Relation) {
	records := make([]Record, 0, len(s.records))
	for _, r := range s.records {
		records = append(records, r)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	rels := append([]Relation{}, s.rels...)
	sort.Slice(rels, func(i, j int) bool {
		if rels[i].From != rels[j].From {
			return rels[i].From < rels[j].From
		}
		return rels[i].To < rels[j].To
	})
	return records, rels
}

// Restore replaces store contents (used by Load).
func (s *Store) Restore(records []Record, rels []Relation) {
	s.records = map[string]Record{}
	for _, r := range records {
		s.records[r.ID] = r
	}
	s.rels = append([]Relation{}, rels...)
}

type persisted struct {
	Records []Record   `json:"records"`
	Rels    []Relation `json:"rels"`
}

// Save writes the store atomically (tmp + rename).
func (s *Store) Save(path string) error {
	records, rels := s.Snapshot()
	data, err := json.MarshalIndent(persisted{Records: records, Rels: rels}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads a persisted store; a missing file yields an empty store.
func Load(path string) (*Store, error) {
	s := New()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	s.Restore(p.Records, p.Rels)
	return s, nil
}

// RequirementID is the stable per-run requirement id.
func RequirementID(runID string) string { return "req-" + runID }

// EvidenceID is the stable per-run evidence id.
func EvidenceID(runID string) string { return "ev-" + runID + "-final" }

// SeedRequirement records the run goal via KnowledgeDelta (GAP-019:
// agent writes are Delta-first). Idempotent per run id.
func SeedRequirement(s *Store, runID, goal string) Record {
	title := goal
	if len(title) > 120 {
		title = title[:120] + "…"
	}
	r := Record{
		ID: RequirementID(runID), Kind: KindRequirement, Title: title,
		Authority: "canonical", Trust: "high", Status: "active",
		Provenance: "run:" + runID + ":goal",
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	_ = s.Commit(Delta{ID: "seed-" + runID, Author: "harness-run", Upserts: []Record{r}})
	out, _ := s.Get(r.ID)
	return out
}

// SeedEvidence records the run outcome linked to its requirement.
func SeedEvidence(s *Store, runID, phase, stopReason, checkpointID string) Record {
	body := "phase=" + phase
	if stopReason != "" {
		body += " stop=" + stopReason
	}
	if checkpointID != "" {
		body += " checkpoint=" + checkpointID
	}
	r := Record{
		ID: EvidenceID(runID), Kind: KindEvidence, Title: "run " + runID + " " + phase,
		Body: body, Authority: "reference", Trust: "high", Status: "active",
		Provenance: "run:" + runID + ":finish",
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	_ = s.Commit(Delta{ID: "seed-ev-" + runID, Author: "harness-run",
		Upserts: []Record{r},
		Links:   []Relation{{From: r.ID, Type: "evidences", To: RequirementID(runID)}}})
	out, _ := s.Get(r.ID)
	return out
}
