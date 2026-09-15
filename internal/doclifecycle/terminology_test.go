package doclifecycle

import (
	"path/filepath"
	"strings"
	"testing"
)

// authorityMap is a minimal authority map so version-policy tests can exercise
// the historical exemption without depending on the real repository map. It
// classifies docs/archived.md as historical and everything else under docs/ as
// canonical.
func authorityMap() map[string]any {
	return map[string]any{
		"version":         1,
		"current_version": "0.6.0",
		"documents": []map[string]any{
			{"path": "docs/archived.md", "role": "historical"},
			{"path": "docs/**", "role": "canonical"},
		},
	}
}

func TestGlossaryRequiresDefinedPreferredReplacement(t *testing.T) {
	glossary := Glossary{Version: 1, Terms: []Term{
		{Term: "Goal", Definition: "declared intent", Status: TermPreferred},
		{Term: "goal", Definition: "duplicate, different case", Status: TermPreferred},
		{Term: "undocumented", Status: TermPreferred},
		{Term: "mystery", Definition: "no status"},
		{Term: "old word", Definition: "outgoing wording", Status: TermDeprecated},
		{Term: "stale word", Definition: "outgoing wording", Status: TermDeprecated, ReplacedBy: "Goal"},
		{Term: "dead word", Definition: "outgoing", Status: TermDeprecated, ReplacedBy: "phantom"},
	}}
	kinds := map[string]int{}
	for _, finding := range ValidateGlossary(glossary) {
		kinds[finding.Kind]++
	}
	for _, want := range []string{"term-duplicate", "term-undefined", "term-invalid-status",
		"term-without-replacement", "term-replacement-unknown"} {
		if kinds[want] == 0 {
			t.Fatalf("expected a %s finding, got %#v", want, kinds)
		}
	}
	if kinds["term-duplicate"] != 1 {
		t.Fatalf("duplicate detection must be case-insensitive, got %d", kinds["term-duplicate"])
	}
}

// A replacement that is itself deprecated would send a writer from one retired
// term to another, so it is rejected.
func TestGlossaryRejectsDeprecatedReplacement(t *testing.T) {
	glossary := Glossary{Version: 1, Terms: []Term{
		{Term: "old", Definition: "outgoing", Status: TermDeprecated, ReplacedBy: "older"},
		{Term: "older", Definition: "also outgoing", Status: TermDeprecated, ReplacedBy: "current"},
		{Term: "current", Definition: "the wording we want", Status: TermPreferred},
	}}
	kinds := map[string]bool{}
	for _, finding := range ValidateGlossary(glossary) {
		kinds[finding.Kind] = true
	}
	if !kinds["term-replacement-not-preferred"] {
		t.Fatalf("a deprecated replacement must be rejected: %#v", kinds)
	}
}

func TestGlossaryMatchesWholeWordsOnly(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "docs/guide.md", "The goalie stopped it.\nThe Goal is locked.\n")
	writeText(t, root, "docs/other.md", "Nothing to see here.\n")
	glossary := Glossary{Version: 1, Terms: []Term{
		{Term: "Goal", Definition: "declared intent", Status: TermForbidden},
	}}
	findings := GlossaryFindings(root, glossary, []string{"docs/guide.md", "docs/other.md"})
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding, got %#v", findings)
	}
	if findings[0].Target != "docs/guide.md" || findings[0].Kind != "term-forbidden" {
		t.Fatalf("unexpected finding: %#v", findings[0])
	}
}

func TestGlossaryReportsDeprecatedWithItsReplacement(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "docs/guide.md", "We use the legacy gatekeeper role.\n")
	glossary := Glossary{Version: 1, Terms: []Term{
		{Term: "gatekeeper", Definition: "retired role", Status: TermDeprecated, ReplacedBy: "evidence gate"},
		{Term: "evidence gate", Definition: "current role", Status: TermPreferred},
	}}
	findings := GlossaryFindings(root, glossary, []string{"docs/guide.md"})
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %#v", findings)
	}
	if !strings.Contains(findings[0].Detail, "evidence gate") {
		t.Fatalf("a deprecated finding must name its replacement: %q", findings[0].Detail)
	}
}

func TestVersionPolicyRejectsOverlapAndMissingCurrent(t *testing.T) {
	if findings := ValidateVersionPolicy(VersionPolicy{Current: "", Supported: []string{"0.4"}}); len(findings) != 1 ||
		findings[0].Kind != "version-policy-incomplete" {
		t.Fatalf("a policy without a current version must be rejected: %#v", findings)
	}
	overlap := VersionPolicy{Current: "0.6.0", Supported: []string{"0.6"}, Removed: []string{"0.6.1"}}
	findings := ValidateVersionPolicy(overlap)
	if len(findings) != 2 || findings[0].Kind != "version-policy-overlap" || findings[1].Kind != "version-policy-overlap" {
		t.Fatalf("every overlapping bucket must be reported: %#v", findings)
	}
	bad := VersionPolicy{Current: "0.6.0", Deprecated: []string{"not-a-version"}}
	if findings := ValidateVersionPolicy(bad); len(findings) != 1 || findings[0].Kind != "version-policy-invalid" {
		t.Fatalf("a malformed version must be reported: %#v", findings)
	}
}

func TestVersionPolicyFlagsRemovedReferencesOnlyInCurrentDocs(t *testing.T) {
	root := t.TempDir()
	writeJSON(t, root, LifecyclePath, map[string]any{
		"version":        "0.6.0",
		"deprecated":     []any{},
		"version_policy": VersionPolicy{Current: "0.6.0", Supported: []string{"0.5"}, Removed: []string{"0.2"}, HistoricalExempt: true},
	})
	writeJSON(t, root, "docs/AUTHORITY_MAP.json", authorityMap())
	writeText(t, root, "docs/current.md", "Install v0.2 of the tool.\n")
	writeText(t, root, "docs/archived.md", "Install v0.2 of the tool.\n")
	writeText(t, root, "docs/narrating.md", "The legacy v0.2 line has been superseded.\n")

	report, err := CheckVersionPolicy(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatal("a current document presenting a removed version must fail")
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected one finding, got %#v", report.Findings)
	}
	if report.Findings[0].Target != "docs/current.md" || report.Findings[0].Kind != "removed-version-reference" {
		t.Fatalf("unexpected finding: %#v", report.Findings[0])
	}
}

// A reference with a patch component is a component version, not a release line,
// and a policy that disagrees with the shipped CLI version is drift.
func TestVersionPolicyIgnoresComponentVersionsAndDetectsDrift(t *testing.T) {
	root := t.TempDir()
	writeJSON(t, root, LifecyclePath, map[string]any{
		"version":        "0.6.0",
		"deprecated":     []any{},
		"version_policy": VersionPolicy{Current: "0.6.0", Removed: []string{"0.1"}},
	})
	writeJSON(t, root, "docs/AUTHORITY_MAP.json", authorityMap())
	writeText(t, root, "docs/current.md", "The negotiation kernel is v0.1.0; the product is v0.6.\n")
	report, err := CheckVersionPolicy(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("v0.1.0 is a component version, not a removed release: %#v", report.Findings)
	}

	drift := t.TempDir()
	writeJSON(t, drift, LifecyclePath, map[string]any{
		"version":        "0.4.0",
		"deprecated":     []any{},
		"version_policy": VersionPolicy{Current: "0.4.0"},
	})
	driftReport, err := CheckVersionPolicy(drift)
	if err != nil {
		t.Fatal(err)
	}
	if driftReport.OK || driftReport.Findings[0].Kind != "version-policy-drift" {
		t.Fatalf("policy must agree with the shipped version: %#v", driftReport.Findings)
	}
}

func TestRepositoryTerminologyAndVersionPolicyAreConsistent(t *testing.T) {
	root, _ := filepath.Abs("../..")
	status, err := CheckGlossary(root)
	if err != nil {
		t.Fatal(err)
	}
	if !status.OK {
		t.Fatalf("the canonical glossary must be internally consistent: %#v", status.Findings)
	}
	if status.Total == 0 {
		t.Fatal("the repository must declare its terminology")
	}
	report, err := CheckVersionPolicy(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("current documents must not present a removed version: %#v", report.Findings)
	}
}
