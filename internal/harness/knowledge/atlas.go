// Memory Atlas (GAP-018): a project-scoped, file-backed memory index
// feeding the Context Compiler, plus gated cross-project promotion.
// Records carry provenance/freshness/sensitivity; recall is lexical and
// bounded.
package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AtlasPath is the conventional project-local index location.
func AtlasPath(root string) string { return filepath.Join(root, ".prumo", "atlas.json") }

// Atlas is the serializable memory index.
type Atlas struct {
	Project string   `json:"project"`
	Records []Record `json:"records"`
	Updated string   `json:"updated"`
}

// LoadAtlas reads the index; missing file yields an empty atlas.
func LoadAtlas(path string) (Atlas, error) {
	var a Atlas
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return a, err
	}
	if err := json.Unmarshal(data, &a); err != nil {
		return a, err
	}
	return a, nil
}

// SaveAtlas writes atomically.
func SaveAtlas(path string, a Atlas) error {
	a.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	data, err := json.MarshalIndent(a, "", "  ")
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

// Remember upserts one memory record.
func Remember(s *Store, id, title, body, provenance string, tags []string) error {
	meta := map[string]any{}
	if len(tags) > 0 {
		joined := make([]any, 0, len(tags))
		for _, tg := range tags {
			joined = append(joined, tg)
		}
		meta["tags"] = joined
	}
	return s.Commit(Delta{ID: "remember-" + id, Author: "harness-run", Upserts: []Record{{
		ID: id, Kind: KindMemory, Title: title, Body: body,
		Authority: "reference", Trust: "medium", Status: "active",
		Provenance: provenance, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano), Meta: meta,
	}}})
}

// Recall returns up to n memory records overlapping query terms,
// freshest most-overlapping first.
func Recall(s *Store, query string, n int) []Record {
	return RecallSince(s, query, n, time.Time{})
}

// RecallSince additionally drops records updated before since (zero = no
// freshness floor). Stale memories stay stored; they just stop surfacing.
func RecallSince(s *Store, query string, n int, since time.Time) []Record {
	if n <= 0 {
		n = 5
	}
	terms := map[string]bool{}
	for _, tok := range strings.Fields(strings.ToLower(query)) {
		if len(tok) > 1 {
			terms[tok] = true
		}
	}
	type hit struct {
		r     Record
		score int
	}
	var hits []hit
	floor := !since.IsZero()
	for _, r := range s.Search("", []Kind{KindMemory}) {
		if r.Status != "active" {
			continue
		}
		if floor {
			if ts, err := time.Parse(time.RFC3339Nano, r.UpdatedAt); err != nil || ts.Before(since) {
				continue
			}
		}
		hay := strings.ToLower(r.Title + " " + r.Body)
		score := 0
		for term := range terms {
			if strings.Contains(hay, term) {
				score++
			}
		}
		if score > 0 {
			hits = append(hits, hit{r, score})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].r.UpdatedAt > hits[j].r.UpdatedAt
	})
	out := []Record{}
	for i := range hits {
		if len(out) >= n {
			break
		}
		out = append(out, hits[i].r)
	}
	return out
}

// SnapshotAtlas exports a store's memory records as an Atlas.
func SnapshotAtlas(project string, s *Store) Atlas {
	records, _ := s.Snapshot()
	mem := []Record{}
	for _, r := range records {
		if r.Kind == KindMemory {
			mem = append(mem, r)
		}
	}
	sort.Slice(mem, func(i, j int) bool { return mem[i].ID < mem[j].ID })
	return Atlas{Project: project, Records: mem}
}

// PromotionPolicy gates cross-project memory promotion.
type PromotionPolicy struct {
	// AllowProjects lists target projects; empty denies all cross-project
	// promotion (local recall unaffected).
	AllowProjects []string
	// MaxRecords caps promoted volume per call.
	MaxRecords int
}

// PromotionResult reports what crossed the boundary.
type PromotionResult struct {
	Promoted []string `json:"promoted"`
	Denied   []string `json:"denied"`
}

// Promote copies eligible memory records into the target atlas store:
// internal/public only — restricted and confidential never leave their
// project. Provenance is stamped; the source is untouched (copy, not move).
func Promote(s *Store, ids []string, targetProject string, policy PromotionPolicy, target *Store) (PromotionResult, error) {
	var res PromotionResult
	allowed := false
	for _, p := range policy.AllowProjects {
		if p == targetProject {
			allowed = true
		}
	}
	if !allowed {
		return res, fmt.Errorf("project %q not in promotion allowlist", targetProject)
	}
	max := policy.MaxRecords
	if max <= 0 {
		max = 25
	}
	for _, id := range ids {
		if len(res.Promoted) >= max {
			break
		}
		r, ok := s.Get(id)
		if !ok || r.Kind != KindMemory || r.Status != "active" {
			res.Denied = append(res.Denied, id+":not-found")
			continue
		}
		sens := r.Sensitivity
		if sens == "" {
			sens = "internal"
		}
		if sens == "restricted" || sens == "confidential" {
			res.Denied = append(res.Denied, id+":"+sens)
			continue
		}
		cp := r
		cp.Provenance = "promoted:" + provenanceProject(r) + "→" + targetProject + "|" + r.Provenance
		cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := target.Commit(Delta{ID: "promote-" + id, Author: "atlas-promotion", Upserts: []Record{cp}}); err != nil {
			res.Denied = append(res.Denied, id+":commit-failed")
			continue
		}
		res.Promoted = append(res.Promoted, id)
	}
	sort.Strings(res.Promoted)
	sort.Strings(res.Denied)
	return res, nil
}

func provenanceProject(r Record) string {
	if strings.HasPrefix(r.Provenance, "run:") {
		return "run"
	}
	if i := strings.Index(r.Provenance, "|"); i >= 0 {
		return r.Provenance
	}
	if r.Provenance != "" {
		return r.Provenance
	}
	return "unknown"
}
