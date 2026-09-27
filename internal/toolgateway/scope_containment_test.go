package toolgateway

import (
	"path/filepath"
	"testing"
)

// The scope test compared strings with HasPrefix. A sibling directory sharing
// the root's name prefix is not a child of the root, and it passed.

func TestEvaluateRejectsSiblingSharingNamePrefix(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	sibling := filepath.Join(parent, "repo-private")

	d := Descriptor{
		ID:              "edit.create",
		Kind:            SideEffecting,
		FilesystemScope: []string{"project-root"},
	}
	decision := Evaluate(d, root, sibling, false)
	if decision.Allowed {
		t.Fatal("a sibling directory sharing the root's name prefix must be outside the scope")
	}
}

func TestEvaluateRejectsParentTraversal(t *testing.T) {
	root := t.TempDir()
	d := Descriptor{ID: "edit.create", Kind: SideEffecting, FilesystemScope: []string{"project-root"}}
	for _, target := range []string{
		filepath.Join(root, "..", "outside.txt"),
		filepath.Join(root, "a", "..", "..", "outside.txt"),
	} {
		if Evaluate(d, root, target, false).Allowed {
			t.Fatalf("%s escapes the scope and must be denied", target)
		}
	}
}

func TestEvaluateAllowsPathsInsideTheScope(t *testing.T) {
	root := t.TempDir()
	d := Descriptor{ID: "edit.create", Kind: SideEffecting, FilesystemScope: []string{"project-root"}}
	for _, target := range []string{
		filepath.Join(root, "file.txt"),
		filepath.Join(root, "pkg", "deep", "file.txt"),
		root,
	} {
		if !Evaluate(d, root, target, false).Allowed {
			t.Fatalf("%s is inside the scope and must be allowed", target)
		}
	}
}

func TestEvaluateMCPRootScopeRejectsSiblingSharingNamePrefix(t *testing.T) {
	root := t.TempDir()
	sibling := root + "-other"

	server := MCPServerDescriptor{ID: "srv", Trust: "trusted", RootScopes: []string{root}}
	tool := Descriptor{ID: "fs.read", Kind: SideEffecting, FilesystemScope: []string{"project-root"}}
	if EvaluateMCP(server, tool, sibling, false).Allowed {
		t.Fatal("a sibling sharing the root scope's name prefix must be denied")
	}
}

func TestEvaluateMCPWildcardRootScopeStillWorks(t *testing.T) {
	server := MCPServerDescriptor{ID: "srv", Trust: "trusted", RootScopes: []string{"*"}}
	tool := Descriptor{ID: "fs.read", Kind: SideEffecting, FilesystemScope: []string{"project-root"}}
	if !EvaluateMCP(server, tool, "/anywhere/at/all", false).Allowed {
		t.Fatal("the explicit wildcard must still allow any target")
	}
}
