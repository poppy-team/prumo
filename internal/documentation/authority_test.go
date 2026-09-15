package docengine

import (
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/protocol"
)

func TestAuthorityMapMatchesProtocolVersion(t *testing.T) {
	root, _ := filepath.Abs("../..")
	m, err := LoadAuthorityMap(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.CurrentVersion != protocol.CLIVersion {
		t.Fatalf("authority map current_version %q != protocol.CLIVersion %q", m.CurrentVersion, protocol.CLIVersion)
	}
}

// TestAuthorityGateClean is the permanent W0.12 gate: no unclassified document,
// no version drift in active docs, no broken routing link, no projection cycle.
func TestAuthorityGateClean(t *testing.T) {
	root, _ := filepath.Abs("../..")
	report, err := CheckAuthority(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Files == 0 {
		t.Fatal("expected documents to be scanned")
	}
	for _, f := range report.Findings {
		t.Errorf("authority finding [%s] %s: %s", f.Kind, f.Path, f.Detail)
	}
}

func TestVersionDriftDetection(t *testing.T) {
	if got := versionDriftFindings("docs/active.md", "Prumo v0.4 is the release line", 6); len(got) != 1 {
		t.Fatalf("expected one drift finding, got %d", len(got))
	}
	if got := versionDriftFindings("docs/active.md", "historical v0.4 boundary", 6); len(got) != 0 {
		t.Fatalf("annotated line must be exempt, got %d", len(got))
	}
	if got := versionDriftFindings("docs/active.md", "current v0.6 line", 6); len(got) != 0 {
		t.Fatalf("current version must not drift, got %d", len(got))
	}
	if got := versionDriftFindings("docs/active.md", "negotiation kernel v0.1.0 pinned", 6); len(got) != 0 {
		t.Fatalf("component versions must not drift, got %d", len(got))
	}
}

func TestProjectionCycleDetection(t *testing.T) {
	cyclic := AuthorityMap{Documents: []AuthorityEntry{
		{Path: "docs/a.md", Role: RoleProjection, CanonicalSource: "docs/b.md"},
		{Path: "docs/b.md", Role: RoleProjection, CanonicalSource: "docs/a.md"},
	}}
	if got := projectionCycleFindings(cyclic); len(got) != 2 {
		t.Fatalf("expected two cycle findings, got %d", len(got))
	}
	valid := AuthorityMap{Documents: []AuthorityEntry{
		{Path: "docs/a.md", Role: RoleProjection, CanonicalSource: "docs/c.md"},
		{Path: "docs/c.md", Role: RoleCanonical},
	}}
	if got := projectionCycleFindings(valid); len(got) != 0 {
		t.Fatalf("projection to canonical is valid, got %d", len(got))
	}
}

func TestPatternMatches(t *testing.T) {
	cases := []struct {
		pattern, rel string
		want         bool
	}{
		{"docs/**", "docs/foo.md", true},
		{"docs/adr/**", "docs/adr/001-go-core.md", true},
		{"docs/adr/**", "docs/other.md", false},
		{"docs/runtime/d*-progress.md", "docs/runtime/d1-progress.md", true},
		{"docs/runtime/d*-progress.md", "docs/runtime/living-plan.md", false},
		{"AGENTS.md", "AGENTS.md", true},
		{"AGENTS.md", "docs/AGENTS.md", false},
	}
	for _, c := range cases {
		if got := patternMatches(c.pattern, c.rel); got != c.want {
			t.Errorf("patternMatches(%q, %q) = %v, want %v", c.pattern, c.rel, got, c.want)
		}
	}
}
