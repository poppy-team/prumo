package resolver

import (
	"strings"
	"testing"
)

// Four fields across 156 catalogue entries were read by nobody: `requires`,
// `requires_any`, `conflicts` and `modes`. A skill that says it needs another
// skill, or that it does not apply in audit mode, was selected anyway — so the
// plan advertised a workflow whose first step did not exist (GAP-146).
//
// The resolution also listed what it chose and said nothing about the rest, so a
// missing skill looked the same whether the resolver decided that or nobody
// looked.

func testCatalog(skills ...map[string]any) Catalog {
	return Catalog{Sections: map[string][]map[string]any{
		"skills":     skills,
		"agents":     {},
		"recipes":    {},
		"bundles":    {},
		"risk_rules": {},
	}}
}

// selectorFor builds the value of a skill's `select` block: the keys are the
// profile attributes it matches on and the values are what must be present.
func selectorFor(on map[string]bool) map[string]any {
	if on == nil {
		on = map[string]bool{"always": true}
	}
	return map[string]any{"features_any": keysOf(on)}
}

// always is a skill that matches whatever the profile is.
func always() map[string]any {
	return map[string]any{"select": selectorFor(nil)}
}

// onFeature is a skill that matches when the profile declares the feature.
func onFeature(feature string) map[string]any {
	return map[string]any{"select": selectorFor(map[string]bool{feature: true})}
}

func keysOf(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

func TestASkillIsPulledInToSatisfyARequirement(t *testing.T) {
	// `lang-zig` requires clean-code, testing-quality and secure-coding. If the
	// selector picks the language but not its prerequisites, dropping the
	// language means no language skill ever resolves.
	c := testCatalog(
		map[string]any{
			"id": "lang-thing", "select": selectorFor(map[string]bool{"go": true}),
			"requires": []any{"secure-coding"},
		},
		map[string]any{"id": "secure-coding", "select": selectorFor(nil)},
	)
	res := c.Resolve(Profile{Raw: map[string]any{"features": []any{"go"}}})
	if !contains(res.Skills, "lang-thing") {
		t.Fatalf("the dependent skill was dropped: %v", res.Skills)
	}
	if !contains(res.Skills, "secure-coding") {
		t.Fatalf("the prerequisite was not pulled in: %v", res.Skills)
	}
}

func TestADependencyIsClosedTransitively(t *testing.T) {
	c := testCatalog(
		map[string]any{"id": "a", "select": selectorFor(map[string]bool{"x": true}), "requires": []any{"b"}},
		map[string]any{"id": "b", "select": selectorFor(nil), "requires": []any{"c"}},
		map[string]any{"id": "c", "select": selectorFor(nil)},
	)
	res := c.Resolve(Profile{Raw: map[string]any{"features": []any{"x"}}})
	for _, want := range []string{"a", "b", "c"} {
		if !contains(res.Skills, want) {
			t.Errorf("%s is missing from the closed plan: %v", want, res.Skills)
		}
	}
}

func TestASkillRequiringSomethingUndefinedIsDroppedWithAReason(t *testing.T) {
	// There is nothing to pull in, so the dependent cannot be made coherent and
	// says so rather than vanishing.
	c := testCatalog(
		map[string]any{"id": "a", "select": selectorFor(map[string]bool{"x": true}), "requires": []any{"nope"}},
	)
	res := c.Resolve(Profile{Raw: map[string]any{"features": []any{"x"}}})
	if contains(res.Skills, "a") {
		t.Fatalf("a skill requiring an undefined skill was selected: %v", res.Skills)
	}
	reason, ok := res.Rejected["a"]
	if !ok {
		t.Fatalf("the drop was silent: %+v", res.Rejected)
	}
	if !strings.Contains(reason, "nope") {
		t.Fatalf("the reason does not name what is missing: %q", reason)
	}
}

func TestConflictingSkillsAreNotBothSelected(t *testing.T) {
	c := testCatalog(
		map[string]any{"id": "keep", "select": selectorFor(map[string]bool{"x": true}), "conflicts": []any{"drop"}},
		map[string]any{"id": "drop", "select": selectorFor(map[string]bool{"x": true})},
	)
	res := c.Resolve(Profile{Raw: map[string]any{"features": []any{"x"}}})
	if contains(res.Skills, "keep") && contains(res.Skills, "drop") {
		t.Fatalf("both sides of a conflict were selected: %v", res.Skills)
	}
	if len(res.Rejected) == 0 {
		t.Fatal("the conflict was resolved silently")
	}
}

func TestASkillOutsideTheProjectsModeIsNotSelected(t *testing.T) {
	c := testCatalog(
		map[string]any{
			"id": "audit-only", "select": selectorFor(map[string]bool{"x": true}),
			"modes": []any{"audit"},
		},
		map[string]any{"id": "build-it", "select": selectorFor(map[string]bool{"x": true})},
	)
	res := c.Resolve(Profile{Raw: map[string]any{"features": []any{"x"}, "mode": "implementation"}})
	if contains(res.Skills, "audit-only") {
		t.Fatalf("an audit-mode skill was selected for an implementation project: %v", res.Skills)
	}
	if !contains(res.Skills, "build-it") {
		t.Fatalf("the mode check took an unrelated skill with it: %v", res.Skills)
	}
	reason := res.Rejected["audit-only"]
	if !strings.Contains(reason, "audit") {
		t.Fatalf("the rejection does not say which mode was needed: %q", reason)
	}
}

func TestASkillWithNoModesIsNotRestricted(t *testing.T) {
	// An absent list means "no restriction", not "applies to no mode".
	c := testCatalog(map[string]any{"id": "any-mode", "select": selectorFor(map[string]bool{"x": true})})
	res := c.Resolve(Profile{Raw: map[string]any{"features": []any{"x"}, "mode": "audit"}})
	if !contains(res.Skills, "any-mode") {
		t.Fatalf("a skill declaring no modes was restricted anyway: %v", res.Skills)
	}
}

func TestEveryRejectionHasAReason(t *testing.T) {
	// "Why is this skill not in my plan" has to have an answer that is not
	// silence.
	c := testCatalog(
		map[string]any{"id": "a", "select": selectorFor(map[string]bool{"x": true})},
		map[string]any{"id": "b", "select": selectorFor(map[string]bool{"y": true})},
	)
	res := c.Resolve(Profile{Raw: map[string]any{"features": []any{"x"}}})
	if _, ok := res.Rejected["b"]; !ok {
		t.Fatalf("an unselected skill has no recorded reason: %+v", res.Rejected)
	}
	if !contains(res.Skills, "a") {
		t.Fatalf("the matching skill was not selected: %v", res.Skills)
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
