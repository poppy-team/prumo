package doclifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/protocol"
)

// The version policy gate checked the documentation lifecycle against the build
// and nothing else. So the installers could pin v0.5.0 while the tree built 0.6.0
// and the release workflow could trigger on v0.5.* — meaning a `curl | sh`
// installed a two-minor-old binary and tagging the current version published
// nothing, with every existing gate green (GAP-138).
//
// These tests are about the gate, not the repository: a repository can be
// consistent today and drift tomorrow, and what has to hold is that the drift
// becomes a finding.

func writeSurface(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// consistentRoot writes three surfaces that all name the built version.
func consistentRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	v := "v" + minorOf(protocol.CLIVersion)
	writeSurface(t, root, "scripts/install.sh", `VERSION="${PRUMO_VERSION:-`+v+`.0}"`)
	writeSurface(t, root, "scripts/install.ps1", `{ "v`+minorOf(protocol.CLIVersion)+`.0" }`)
	writeSurface(t, root, ".github/workflows/release.yml", "on:\n  push:\n    tags:\n      - \""+v+".*\"\n")
	return root
}

func TestAConsistentRepositoryHasNoReleaseSurfaceFinding(t *testing.T) {
	if findings := checkReleaseSurfaces(consistentRoot(t)); len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
}

func TestAnInstallerOnAnOlderMinorIsReported(t *testing.T) {
	root := consistentRoot(t)
	writeSurface(t, root, "scripts/install.sh", `VERSION="${PRUMO_VERSION:-v0.1.0}"`)
	findings := checkReleaseSurfaces(root)
	if len(findings) != 1 || findings[0].Target != "scripts/install.sh" {
		t.Fatalf("findings = %v, want one against install.sh", findings)
	}
	if findings[0].Kind != "release-surface-drift" {
		t.Fatalf("kind = %s", findings[0].Kind)
	}
}

func TestAReleaseWorkflowOnTheWrongTagIsReported(t *testing.T) {
	// The failure with the worst consequences and the least visibility: tagging
	// the current version produced no release at all.
	root := consistentRoot(t)
	writeSurface(t, root, ".github/workflows/release.yml", "on:\n  push:\n    tags:\n      - \"v0.1.*\"\n")
	findings := checkReleaseSurfaces(root)
	if len(findings) != 1 || findings[0].Target != ".github/workflows/release.yml" {
		t.Fatalf("findings = %v, want one against the release workflow", findings)
	}
}

func TestACommentMentioningTheRightVersionDoesNotSatisfyTheGate(t *testing.T) {
	// The first version of this check searched the file for the version string,
	// and a comment mentioning the right value satisfied it — so the release
	// workflow passed while still triggering on the wrong tag. A gate a comment
	// can satisfy is not a gate, and this is the shape of bug that produces
	// exactly the silent drift the gate was added to stop.
	root := consistentRoot(t)
	writeSurface(t, root, ".github/workflows/release.yml",
		"on:\n  push:\n    tags:\n"+
			"      # this workflow is about to be updated to v"+minorOf(protocol.CLIVersion)+".*\n"+
			"      - \"v0.1.*\"\n")
	if findings := checkReleaseSurfaces(root); len(findings) != 1 {
		t.Fatalf("a comment satisfied the gate: %v", findings)
	}
}

func TestASurfaceWithNoReadableVersionIsReportedRatherThanAssumedFine(t *testing.T) {
	// "I could not tell" is not "it is correct". A file whose version cannot be
	// read is a file that might ship anything.
	root := consistentRoot(t)
	writeSurface(t, root, "scripts/install.sh", "echo installing\n")
	findings := checkReleaseSurfaces(root)
	if len(findings) != 1 || findings[0].Kind != "release-surface-unreadable" {
		t.Fatalf("findings = %v, want an unreadable finding", findings)
	}
}

func TestAMissingSurfaceIsNotAVersionFinding(t *testing.T) {
	// Whether a surface should exist is not a version question, and reporting it
	// here would train people to ignore the gate.
	root := t.TempDir()
	if findings := checkReleaseSurfaces(root); len(findings) != 0 {
		t.Fatalf("findings = %v, want none for an empty tree", findings)
	}
}

func TestTheRealRepositorySurfacesAgreeWithTheBuild(t *testing.T) {
	// The check that would have caught GAP-138, run against this repository.
	// Version numbers in the fixtures above are synthetic; this one is not.
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if findings := checkReleaseSurfaces(root); len(findings) != 0 {
		t.Fatalf("this repository's release surfaces have drifted: %v", findings)
	}
}

func TestTheReleasePolicyNamesTheSameTagTheWorkflowDoes(t *testing.T) {
	// docs/governance/release-policy.md states the trigger in prose, and the
	// workflow drifted from it. Checking the workflow against the build catches
	// the version number; this catches the policy having been changed under it.
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := os.ReadFile(filepath.Join(root, "docs", "governance", "release-policy.md"))
	if err != nil {
		t.Skip("release policy is not present in this tree")
	}
	want := "v" + minorOf(protocol.CLIVersion) + ".*"
	if !strings.Contains(string(policy), want) {
		t.Fatalf("the release policy no longer names %s; the workflow trigger needs a decision, not a guess", want)
	}
}
