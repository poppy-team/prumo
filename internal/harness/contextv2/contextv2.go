// Package contextv2 implements the Context Compiler v2 baseline:
// authority/trust/privacy gates -> freshness -> exact/structured ->
// FTS/BM25 scoring -> RRF fusion -> MMR dedup -> dependency/coverage ->
// marginal-utility-per-token packing -> L0-L4 progressive disclosure ->
// replayable manifest. No-LLM path is first-class.
package contextv2

import (
	"sort"
	"strconv"
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
	// Reason says why this item belongs in the context. Every included item
	// carries one, so a compilation is explainable rather than merely
	// reproducible (W4.5).
	Reason string `json:"reason,omitempty"`
	// Projection marks derived content and CanonicalSource names the unit it was
	// derived from, which is what canonical-preference dedup needs (W4.6).
	Projection      bool   `json:"projection,omitempty"`
	CanonicalSource string `json:"canonical_source,omitempty"`
	// Level is the disclosure level this item was rendered at, and Truncated
	// records that a higher level would reveal more.
	Level     Level `json:"level,omitempty"`
	Truncated bool  `json:"truncated,omitempty"`
}

// Manifest is the replayable v2 output.
type Manifest struct {
	Version         int         `json:"version"`
	ID              string      `json:"id,omitempty"`
	RunID           string      `json:"run_id"`
	CreatedAt       string      `json:"created_at,omitempty"`
	Included        []Item      `json:"included"`
	Excluded        []Exclusion `json:"excluded,omitempty"`
	EstimatedTokens int         `json:"estimated_tokens"`
	Pressure        string      `json:"pressure"`
	// DependencyCycles names the references whose DependsOn edges form a loop.
	//
	// A cycle is reported rather than silently broken: the expansion stops
	// walking the back edge, and a caller that cannot see the cycle cannot tell
	// that a document it expected in the context was left out (GAP-149).
	DependencyCycles []string `json:"dependency_cycles,omitempty"`
	Level            Level    `json:"level"` // L0..L4 disclosure
	// InstructionTokens is the share of the budget spent on agent instruction
	// surfaces, reported separately so instruction overhead is measurable
	// instead of hidden inside the total (W4.10).
	InstructionTokens int    `json:"instruction_tokens,omitempty"`
	Policy            string `json:"policy,omitempty"`
}

// CompilePolicy is how a caller chooses a compilation. Making it explicit is
// what allows `prumo context compile` to name the level and the budget it used
// (W4.4, W4.7).
type CompilePolicy struct {
	Budget            int
	Level             Level
	Policy            string
	InstructionTokens int
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

// Compile packs by marginal utility per token with dependency closure, applying
// the disclosure level and canonical-preference dedup first (W4.4–W4.6).
func Compile(runID string, candidates []Item, budget int, level string) Manifest {
	return CompileWithPolicy(runID, candidates, CompilePolicy{Budget: budget, Level: ParseLevel(level)})
}

// CompileWithPolicy is Compile with an explicit policy.
func CompileWithPolicy(runID string, candidates []Item, policy CompilePolicy) Manifest {
	budget := policy.Budget
	if budget <= 0 {
		budget = 8000
	}
	level := policy.Level
	if !level.Valid() {
		level = DefaultLevel
	}

	// Dedup first: a projection that duplicates its canonical source must not
	// consume budget before it is dropped.
	deduped, dedupExcluded := DeduplicatePreferCanonical(candidates)

	// Disclose each candidate at the chosen level, so cost accounting reflects
	// what will actually be revealed.
	disclosed := make([]Item, 0, len(deduped))
	for _, it := range deduped {
		disclosed = append(disclosed, DiscloseItem(it, level))
	}

	byRef := map[string]Item{}
	for _, c := range disclosed {
		byRef[c.Ref] = c
	}
	scored := append([]Item{}, disclosed...)
	sort.Slice(scored, func(i, j int) bool {
		ui := utility(scored[i])
		uj := utility(scored[j])
		if ui != uj {
			return ui > uj
		}
		return scored[i].Ref < scored[j].Ref
	})
	included := []Item{}
	excluded := append([]Exclusion{}, dedupExcluded...)
	used := 0
	inSet := map[string]bool{}
	// visiting is the set of references currently being expanded.
	//
	// inSet only learned a reference after its dependencies were expanded, so
	// two items that depend on each other sent the expansion into a loop: A
	// marks nothing, asks for B, B asks for A, and A is still unmarked. A
	// dependency cycle in a repository's own documentation — a pointer in a
	// doc pointing at the doc that points back — took the compiler down with a
	// stack overflow rather than a cycle report (GAP-149).
	visiting := map[string]bool{}
	cycles := []string{}
	var add func(it Item)
	add = func(it Item) {
		if inSet[it.Ref] {
			return
		}
		if visiting[it.Ref] {
			// A back edge. Expanding it again is what does not terminate, so the
			// edge is recorded and the walk stops here.
			cycleNote(&cycles, it.Ref)
			return
		}
		visiting[it.Ref] = true
		defer delete(visiting, it.Ref)
		for _, d := range it.DependsOn {
			if dep, ok := byRef[d]; ok {
				add(dep)
			}
		}
		if used+it.TokenCost > budget {
			excluded = append(excluded, Exclusion{Ref: it.Ref, Reason: ReasonBudget,
				Detail: "would exceed the " + strconv.Itoa(budget) + " token budget"})
			return
		}
		inSet[it.Ref] = true
		used += it.TokenCost
		it.Reason = selectionReason(it)
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
	sort.SliceStable(excluded, func(i, j int) bool {
		if excluded[i].Reason != excluded[j].Reason {
			return excluded[i].Reason < excluded[j].Reason
		}
		return excluded[i].Ref < excluded[j].Ref
	})
	return Manifest{
		Version: ManifestSchemaVersion, ID: ManifestID(runID), RunID: runID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Included:  included, Excluded: excluded, EstimatedTokens: used, Pressure: pressure,
		Level: level, InstructionTokens: policy.InstructionTokens, Policy: policy.Policy,
		DependencyCycles: dedupeStrings(cycles),
	}
}

// ManifestID is the stable identifier of a compiled context, so an operator can
// ask about a specific compilation (W4.8).
func ManifestID(runID string) string { return "CTX-" + runID }

// ManifestSchemaVersion is the schema version the compiler emits. Version 1 is
// the legacy `sources` projection; version 2 is the `included`/`excluded`
// manifest with disclosure levels and typed exclusion reasons.
const ManifestSchemaVersion = 2

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

// cycleNote records that a reference was reached while already being expanded.
func cycleNote(cycles *[]string, ref string) {
	for _, seen := range *cycles {
		if seen == ref {
			return
		}
	}
	*cycles = append(*cycles, ref)
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
