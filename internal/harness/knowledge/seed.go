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

// RestoreWithAliases also reinstates the locator bindings, so a store written
// before the stable-identity migration keeps resolving (W3.8).
func (s *Store) RestoreWithAliases(records []Record, rels []Relation, aliases map[string]string) {
	s.Restore(records, rels)
	s.aliases = map[string]string{}
	for locator, id := range aliases {
		s.aliases[locator] = id
	}
}

// Aliases returns a copy of the locator→stable-id bindings.
func (s *Store) Aliases() map[string]string {
	out := map[string]string{}
	for locator, id := range s.aliases {
		out[locator] = id
	}
	return out
}

type persisted struct {
	Records []Record          `json:"records"`
	Rels    []Relation        `json:"rels"`
	Aliases map[string]string `json:"aliases,omitempty"`
}

// Save writes the store atomically (tmp + rename).
func (s *Store) Save(path string) error {
	records, rels := s.Snapshot()
	data, err := json.MarshalIndent(persisted{Records: records, Rels: rels, Aliases: s.Aliases()}, "", "  ")
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
	s.RestoreWithAliases(p.Records, p.Rels, p.Aliases)
	return s, nil
}

// RequirementID is the stable id of the requirement a run seeds (W3.8). It is
// derived from the run's domain identity, never from a path or a title.
func RequirementID(runID string) string { return StableIDFor(KindRequirement, "run-"+runID) }

// EvidenceID is the stable id of the evidence a run closes with.
func EvidenceID(runID string) string { return StableIDFor(KindEvidence, "run-"+runID+"-final") }

// LegacyRequirementID / LegacyEvidenceID reproduce the pre-migration ids so a
// store persisted before W3.8 can still be read through Store.Resolve.
func LegacyRequirementID(runID string) string { return legacyRequirementPrefix + runID }

func LegacyEvidenceID(runID string) string { return legacyEvidencePrefix + runID + "-final" }

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
	_ = s.Commit(Delta{ID: StableIDFor(KindDecision, "seed-"+runID), Author: "harness-run", Upserts: []Record{r}})
	// Keep the pre-migration identifier resolvable: a store written before
	// W3.8 names this same requirement `req-<runID>`.
	s.Alias(r.ID, LegacyRequirementID(runID))
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
	_ = s.Commit(Delta{ID: StableIDFor(KindDecision, "seed-ev-"+runID), Author: "harness-run",
		Upserts: []Record{r},
		Links:   []Relation{{From: r.ID, Type: "evidences", To: RequirementID(runID)}}})
	s.Alias(r.ID, LegacyEvidenceID(runID))
	out, _ := s.Get(r.ID)
	return out
}
