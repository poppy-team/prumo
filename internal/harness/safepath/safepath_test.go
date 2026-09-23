package safepath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A containment check that only compares path strings does not contain
// anything. These tests are the reason the four lexical implementations were
// replaced: each one of these cases passed the old check.

func TestResolveRejectsParentTraversal(t *testing.T) {
	root := t.TempDir()
	for _, candidate := range []string{
		"../outside.txt",
		"../../etc/passwd",
		"a/../../outside.txt",
		"./../outside.txt",
	} {
		if _, err := Resolve(root, candidate, false); err == nil {
			t.Errorf("%q must be rejected as escaping the root", candidate)
		}
	}
}

func TestResolveRejectsAbsolutePathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Resolve(root, outside, true); err == nil {
		t.Fatal("an absolute path outside the root must be rejected")
	}
}

func TestResolveRejectsSiblingWithSharedPrefix(t *testing.T) {
	// A prefix comparison on raw strings would accept "/root-evil" for "/root".
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	sibling := filepath.Join(parent, "root-evil")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatalf("mkdir sibling: %v", err)
	}
	if _, err := Resolve(root, sibling, true); err == nil {
		t.Fatal("a sibling directory sharing a name prefix must be rejected")
	}
}

func TestResolveRejectsSymlinkedFileLeavingRoot(t *testing.T) {
	// This is the case the lexical check missed: the path reads as inside the
	// root and resolves outside it.
	root := t.TempDir()
	outsideDir := t.TempDir()
	secret := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	link := filepath.Join(root, "innocent.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Resolve(root, "innocent.txt", true); err == nil {
		t.Fatal("a symlink pointing outside the root must be rejected, not followed")
	}
}

func TestResolveRejectsSymlinkedDirectoryLeavingRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	secret := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outsideDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Resolve(root, filepath.Join("escape", "secret.txt"), true); err == nil {
		t.Fatal("a path traversing a symlinked directory out of the root must be rejected")
	}
}

func TestResolveRejectsSymlinkedDirectoryInPathForNewFile(t *testing.T) {
	// edit.create follows the destination, so a missing leaf behind a symlinked
	// parent is the dangerous case.
	root := t.TempDir()
	outsideDir := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outsideDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Resolve(root, filepath.Join("escape", "new.txt"), false); err == nil {
		t.Fatal("creating a new file through a symlinked parent out of the root must be rejected")
	}
}

func TestResolveAllowsSymlinkStayingInsideRoot(t *testing.T) {
	// Containment is about leaving the root, not about links existing. A link
	// that stays inside is legitimate and must keep working.
	root := t.TempDir()
	target := filepath.Join(root, "real")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "file.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	resolved, err := Resolve(root, filepath.Join("alias", "file.txt"), true)
	if err != nil {
		t.Fatalf("a symlink that stays inside the root must resolve: %v", err)
	}
	if filepath.Base(resolved) != "file.txt" {
		t.Fatalf("unexpected resolution: %s", resolved)
	}
}

func TestResolveAllowsOrdinaryPaths(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "file.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "top.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	for _, candidate := range []string{"top.txt", "a/b/file.txt", "./a/./b/file.txt", ""} {
		if _, err := Resolve(root, candidate, true); err != nil {
			t.Errorf("%q must resolve inside the root: %v", candidate, err)
		}
	}
}

func TestResolveAllowsNewFileInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	resolved, err := Resolve(root, filepath.Join("a", "new.txt"), false)
	if err != nil {
		t.Fatalf("a new file inside the root must be allowed: %v", err)
	}
	if resolved != filepath.Join(root, "a", "new.txt") {
		t.Fatalf("unexpected resolution: %s", resolved)
	}
}

func TestResolveWithMustExistRejectsMissingFile(t *testing.T) {
	root := t.TempDir()
	if _, err := Resolve(root, "absent.txt", true); err == nil {
		t.Fatal("mustExist must reject a path that does not exist")
	}
}

func TestResolveRejectsEmptyRoot(t *testing.T) {
	if _, err := Resolve("", "file.txt", false); err == nil {
		t.Fatal("an empty root must be rejected: it would contain everything")
	}
}

func TestIsWithinDoesNotTouchTheFilesystem(t *testing.T) {
	root := t.TempDir()
	if !IsWithin(root, "absent.txt") {
		t.Error("a path inside the root must be within, existing or not")
	}
	if IsWithin(root, "../absent.txt") {
		t.Error("a path outside the root must not be within")
	}
}

func TestValidateIDAcceptsTheSchemaIdentifierSet(t *testing.T) {
	for _, value := range []string{
		"R-daemon-1", "ev-R-1-complete", "a", "A9", "with_underscore",
		"dotted.name", "0123456789", "a" + strings.Repeat("b", 190),
	} {
		if err := ValidateID("run_id", value); err != nil {
			t.Errorf("%q must be a valid id: %v", value, err)
		}
	}
}

func TestValidateIDRejectsPathEscapes(t *testing.T) {
	for _, value := range []string{
		"../escape", "..", ".", "a/b", "a\\b", "/absolute", "~/home",
		"has space", "has:colon", "has\x00nul", ".hidden", "tab\there",
		"trailing-",
	} {
		if value == "trailing-" {
			// A trailing dash is inside the permitted set; it is a valid id.
			if err := ValidateID("run_id", value); err != nil {
				t.Errorf("%q must be a valid id: %v", value, err)
			}
			continue
		}
		if err := ValidateID("run_id", value); err == nil {
			t.Errorf("%q must be rejected as a run id", value)
		}
	}
}

func TestValidateIDRejectsEmptyAndOverlong(t *testing.T) {
	if err := ValidateID("run_id", ""); err == nil {
		t.Error("an empty id must be rejected")
	}
	if err := ValidateID("run_id", strings.Repeat("a", 201)); err == nil {
		t.Error("an overlong id must be rejected")
	}
}

func TestValidateIDsReportsEveryOffender(t *testing.T) {
	err := ValidateIDs("run_id", []string{"good-1", "../bad", "also bad"})
	if err == nil {
		t.Fatal("a batch containing invalid ids must fail")
	}
	if !strings.Contains(err.Error(), "2 invalid") {
		t.Fatalf("the error must count the offenders, got %q", err.Error())
	}
}

func FuzzResolveNeverEscapes(f *testing.F) {
	f.Add("file.txt")
	f.Add("../escape")
	f.Add("a/b/c")
	f.Add(".")

	f.Fuzz(func(t *testing.T, candidate string) {
		root := t.TempDir()
		resolved, err := Resolve(root, candidate, false)
		if err != nil {
			// Rejection is always acceptable.
			return
		}
		// Acceptance is only acceptable if the result really is inside.
		if !IsWithin(root, resolved) {
			t.Fatalf("Resolve accepted %q and returned %q, which is outside %q", candidate, resolved, root)
		}
	})
}

func FuzzValidateIDNeverPanics(f *testing.F) {
	f.Add("R-1")
	f.Add("../x")
	f.Add("")

	f.Fuzz(func(t *testing.T, value string) {
		// The contract is only that it returns and that acceptance implies the
		// value is a single safe component.
		if err := ValidateID("run_id", value); err != nil {
			return
		}
		if strings.ContainsRune(value, filepath.Separator) || value == "." || value == ".." {
			t.Fatalf("ValidateID accepted an unsafe id %q", value)
		}
	})
}
