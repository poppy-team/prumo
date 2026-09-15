package contextv2

import (
	"sort"
	"strconv"
	"strings"
)

// Exclusion reasons. They are a closed vocabulary so a caller can act on a
// reason instead of parsing prose (W4.5).
const (
	ReasonBudget         = "budget"
	ReasonDuplicate      = "duplicate-ref"
	ReasonProjection     = "prefer-canonical"
	ReasonIneligible     = "ineligible"
	ReasonNotDisclosed   = "empty-at-level"
	ReasonDependencyOnly = "dependency"
)

// ExclusionReasons lists the closed vocabulary in sorted order. It is the Go
// side of the manifest schema's `excluded[].reason` enum.
func ExclusionReasons() []string {
	return []string{
		ReasonBudget, ReasonDependencyOnly, ReasonDuplicate,
		ReasonIneligible, ReasonNotDisclosed, ReasonProjection,
	}
}

// Exclusion records why an item did not make it into a manifest.
type Exclusion struct {
	Ref    string `json:"ref"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// DeduplicatePreferCanonical removes two kinds of redundancy (W4.6):
//
//  1. repeated references, kept once;
//  2. a projection whose canonical source is also a candidate, which is dropped
//     so the context carries the canonical text rather than a derived copy.
//
// Ordering is preserved: the surviving items keep their input order, so the
// caller's ranking is not disturbed. Every drop is explained.
func DeduplicatePreferCanonical(items []Item) ([]Item, []Exclusion) {
	canonicalPresent := map[string]bool{}
	for _, it := range items {
		if !it.Projection && it.CanonicalSource == "" {
			canonicalPresent[it.Ref] = true
		}
	}
	kept := []Item{}
	excluded := []Exclusion{}
	seen := map[string]bool{}
	for _, it := range items {
		if seen[it.Ref] {
			excluded = append(excluded, Exclusion{Ref: it.Ref, Reason: ReasonDuplicate,
				Detail: "reference already included"})
			continue
		}
		if it.Projection && canonicalPresent[it.CanonicalSource] {
			excluded = append(excluded, Exclusion{Ref: it.Ref, Reason: ReasonProjection,
				Detail: "canonical source " + it.CanonicalSource + " is already included"})
			continue
		}
		seen[it.Ref] = true
		kept = append(kept, it)
	}
	sort.SliceStable(excluded, func(i, j int) bool {
		if excluded[i].Reason != excluded[j].Reason {
			return excluded[i].Reason < excluded[j].Reason
		}
		return excluded[i].Ref < excluded[j].Ref
	})
	return kept, excluded
}

// selectionReason explains, in one line, why an item is in the context. Every
// included item carries one; a missing reason is a bug, not an implicit "yes"
// (W4.5).
func selectionReason(it Item) string {
	reason := it.Reason
	if reason == "" {
		reason = "retrieved candidate"
	}
	parts := []string{reason}
	if it.Method != "" {
		parts = append(parts, "via "+it.Method)
	}
	if it.Score > 0 {
		parts = append(parts, "score "+formatScore(it.Score))
	}
	return strings.Join(parts, "; ")
}

// joinReason appends a signal to an item's reason without repeating it.
func joinReason(reason, signal string) string {
	if signal == "" || strings.Contains(reason, signal) {
		return reason
	}
	if reason == "" {
		return signal
	}
	return reason + "; " + signal
}

func formatScore(f float64) string {
	// Three decimals is enough to explain a ranking without implying precision
	// the scoring does not have.
	s := strings.TrimRight(strings.TrimRight(strconv.FormatFloat(f, 'f', 3, 64), "0"), ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}
