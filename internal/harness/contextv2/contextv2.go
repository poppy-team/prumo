// Package contextv2 implements the Context Compiler v2 baseline:
// authority/trust/privacy gates -> freshness -> exact/structured ->
// FTS/BM25 scoring -> RRF fusion -> MMR dedup -> dependency/coverage ->
// marginal-utility-per-token packing -> L0-L4 progressive disclosure ->
// replayable manifest. No-LLM path is first-class.
package contextv2

import (
	"sort"
	"strings"
	"time"
)

// Item is one retrieval candidate.
type Item struct {
	Ref       string   `json:"ref"`
	Authority string   `json:"authority"` // canonical|reference|imported|untrusted
	Trust     string   `json:"trust"`     // high|medium|low
	Privacy   string   `json:"privacy"`   // public|internal|confidential|restricted
	Freshness string   `json:"freshness"` // ISO time or "current"
	Rev       string   `json:"rev,omitempty"`
	Score     float64  `json:"score"`
	Method    string   `json:"method"` // exact|fts|semantic|graph
	TokenCost int      `json:"token_cost"`
	Content   string   `json:"content,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

// Manifest is the replayable v2 output.
type Manifest struct {
	Version         int      `json:"version"`
	RunID           string   `json:"run_id"`
	Included        []Item   `json:"included"`
	Excluded        []string `json:"excluded,omitempty"`
	EstimatedTokens int      `json:"estimated_tokens"`
	Pressure        string   `json:"pressure"`
	Level           string   `json:"level"` // L0..L4 disclosure
}

var authorityRank = map[string]int{"canonical": 0, "reference": 1, "imported": 2, "untrusted": 3}

// Eligible enforces authority/trust/privacy gates before relevance.
func Eligible(it Item, minAuthority string, allowRestricted bool) bool {
	if it.Privacy == "restricted" && !allowRestricted {
		return false
	}
	if it.Trust == "low" && it.Authority == "untrusted" {
		return false
	}
	maxRank, ok := authorityRank[minAuthority]
	if !ok {
		maxRank = 3
	}
	rank, ok := authorityRank[it.Authority]
	if !ok {
		rank = 3
	}
	return rank <= maxRank
}

// RRF fuses ranked lists: score = sum(1/(k+rank)).
func RRF(lists [][]Item, k float64) []Item {
	acc := map[string]*Item{}
	for _, l := range lists {
		for rank, it := range l {
			e, ok := acc[it.Ref]
			if !ok {
				cp := it
				cp.Score = 0
				e = &cp
				acc[it.Ref] = e
			}
			e.Score += 1.0 / (k + float64(rank+1))
		}
	}
	out := make([]Item, 0, len(acc))
	for _, v := range acc {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// MMRDedup drops near-duplicates by ref-prefix/content-prefix overlap.
func MMRDedup(items []Item) []Item {
	seen := map[string]bool{}
	out := []Item{}
	for _, it := range items {
		key := it.Ref
		if len(it.Content) > 64 {
			key += ":" + it.Content[:64]
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, it)
	}
	return out
}

// Compile packs by marginal utility per token with dependency closure.
func Compile(runID string, candidates []Item, budget int, level string) Manifest {
	byRef := map[string]Item{}
	for _, c := range candidates {
		byRef[c.Ref] = c
	}
	scored := append([]Item{}, candidates...)
	sort.Slice(scored, func(i, j int) bool {
		ui := utility(scored[i])
		uj := utility(scored[j])
		if ui != uj {
			return ui > uj
		}
		return scored[i].Ref < scored[j].Ref
	})
	included := []Item{}
	excluded := []string{}
	used := 0
	inSet := map[string]bool{}
	var add func(it Item)
	add = func(it Item) {
		if inSet[it.Ref] {
			return
		}
		for _, d := range it.DependsOn {
			if dep, ok := byRef[d]; ok {
				add(dep)
			}
		}
		if used+it.TokenCost > budget {
			excluded = append(excluded, it.Ref)
			return
		}
		inSet[it.Ref] = true
		used += it.TokenCost
		included = append(included, it)
	}
	for _, it := range scored {
		add(it)
	}
	pressure := "healthy"
	if budget > 0 && used > budget*80/100 {
		pressure = "pressure"
	}
	if budget > 0 && used >= budget {
		pressure = "critical"
	}
	if level == "" {
		level = "L1"
	}
	return Manifest{Version: 2, RunID: runID, Included: included, Excluded: excluded, EstimatedTokens: used, Pressure: pressure, Level: level}
}

func utility(it Item) float64 {
	if it.TokenCost <= 0 {
		return it.Score
	}
	return it.Score / float64(it.TokenCost)
}

// Fresh reports staleness vs a revision timestamp.
func Fresh(freshness string, maxAge time.Duration) bool {
	if freshness == "" || freshness == "current" {
		return true
	}
	t, err := time.Parse(time.RFC3339, freshness)
	if err != nil {
		return !strings.Contains(freshness, "stale")
	}
	return time.Since(t) <= maxAge
}
