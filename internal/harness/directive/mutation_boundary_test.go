package directive

import "testing"

// CanMutatePath took a workspace-relative path, cleaned it, and matched it
// against the boundary lists. A path was never checked for being relative, so
// "../etc/passwd" matched no forbidden entry and, with an empty AllowedPaths,
// was answered "allowed".

func TestCanMutatePathRejectsParentTraversal(t *testing.T) {
	d := &DirectiveIR{}
	if d.CanMutatePath("../etc/passwd") {
		t.Fatal("a parent-relative path must not be allowed to mutate")
	}
	if d.CanMutatePath("../../outside.txt") {
		t.Fatal("a deeper parent-relative path must not be allowed to mutate")
	}
	if d.CanMutatePath("pkg/../../outside.txt") {
		t.Fatal("traversal hidden behind a real segment must not be allowed to mutate")
	}
}

func TestCanMutatePathRejectsAbsolutePaths(t *testing.T) {
	d := &DirectiveIR{}
	if d.CanMutatePath("/etc/passwd") {
		t.Fatal("an absolute path must not be allowed to mutate")
	}
}

func TestCanMutatePathStillAllowsOrdinaryPaths(t *testing.T) {
	d := &DirectiveIR{}
	if !d.CanMutatePath("pkg/file.txt") {
		t.Fatal("an ordinary workspace-relative path must be allowed when no boundary says otherwise")
	}
	if !d.CanMutatePath(".") {
		t.Fatal("the workspace root itself must be allowed when no boundary says otherwise")
	}
}

func TestCanMutatePathHonoursForbiddenPaths(t *testing.T) {
	d := &DirectiveIR{}
	d.MutationBoundaries.ForbiddenPaths = []string{".git", "node_modules"}
	for _, p := range []string{".git/config", "node_modules/pkg/index.js"} {
		if d.CanMutatePath(p) {
			t.Fatalf("%s is forbidden and must be denied", p)
		}
	}
	if !d.CanMutatePath("src/app.go") {
		t.Fatal("a path outside the forbidden list must be allowed")
	}
}

func TestCanMutatePathMatchesByPathComponentNotStringPrefix(t *testing.T) {
	// "src-secrets" shares a string prefix with "src" but is a different
	// directory. Treating it as inside "src" both over-denies forbidden roots
	// and, for an allowlist, admits a sibling tree.
	d := &DirectiveIR{}
	d.MutationBoundaries.AllowedPaths = []string{"src"}
	if !d.CanMutatePath("src/app.go") {
		t.Fatal("src/app.go is inside src and must be allowed")
	}
	if d.CanMutatePath("src-secrets/key.pem") {
		t.Fatal("src-secrets is a sibling of src, not a child, and must be denied")
	}
}

func TestCanMutatePathHonoursAllowedPaths(t *testing.T) {
	d := &DirectiveIR{}
	d.MutationBoundaries.AllowedPaths = []string{"src", "docs"}
	if !d.CanMutatePath("docs/guide.md") {
		t.Fatal("docs/guide.md is in the allowlist and must be allowed")
	}
	if d.CanMutatePath("test/file_test.go") {
		t.Fatal("a path outside the allowlist must be denied")
	}
}

func TestCanMutatePathHonoursScopeFirewall(t *testing.T) {
	d := &DirectiveIR{}
	d.ScopeFirewall.OutOfScope = []string{"legacy"}
	if d.CanMutatePath("legacy/old.go") {
		t.Fatal("an out-of-scope path must be denied")
	}
	if !d.CanMutatePath("src/new.go") {
		t.Fatal("an in-scope path must be allowed")
	}
}
