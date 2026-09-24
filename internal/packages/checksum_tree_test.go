package packages

import (
	"os"
	"path/filepath"
	"testing"
)

// The lock recorded the checksum of manifest.json alone. So a changed SKILL.md,
// a replaced script or an edited reference left the lock reporting exactly the
// same hash: the lock said "this is the version you asked for" about a package
// whose contents had changed underneath it (GAP-147).

func TestChangingAnyFileChangesTheTreeChecksum(t *testing.T) {
	root := t.TempDir()
	write(t, root, "manifest.json", `{"id":"a"}`)
	write(t, root, "SKILL.md", "# skill")
	write(t, root, "scripts/verify.sh", "echo ok")

	before, err := ChecksumTree(root)
	if err != nil {
		t.Fatal(err)
	}
	// The manifest alone would say nothing about this one.
	write(t, root, "scripts/verify.sh", "rm -rf /")
	afterScript, err := ChecksumTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if before == afterScript {
		t.Fatal("editing a script did not change the checksum; the lock cannot see the scripts")
	}

	// Restored to the same bytes, the checksum returns: the digest is over
	// content, not over "something changed".
	write(t, root, "scripts/verify.sh", "echo ok")
	restored, err := ChecksumTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if restored != before {
		t.Fatal("restoring a file did not restore the checksum")
	}
}

func TestMovingAFileChangesTheTreeChecksum(t *testing.T) {
	root := t.TempDir()
	write(t, root, "references/guide.md", "guide")
	before, err := ChecksumTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(
		filepath.Join(root, "references", "guide.md"),
		filepath.Join(root, "examples", "guide.md"),
	); err != nil {
		if mkErr := os.MkdirAll(filepath.Join(root, "examples"), 0o755); mkErr != nil {
			t.Fatal(mkErr)
		}
		if err := os.Rename(
			filepath.Join(root, "references", "guide.md"),
			filepath.Join(root, "examples", "guide.md"),
		); err != nil {
			t.Fatal(err)
		}
	}
	after, err := ChecksumTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("a file that moved produced the same checksum; the package is not the same package")
	}
}

func TestTheChecksumDoesNotDependOnDirectoryOrder(t *testing.T) {
	// Two trees with identical contents must hash identically regardless of the
	// order the filesystem returns their entries, or a lock entry is a lottery.
	build := func() string {
		root := t.TempDir()
		write(t, root, "manifest.json", "{}")
		write(t, root, "checks/a.md", "a")
		write(t, root, "checks/b.md", "b")
		write(t, root, "scripts/run.sh", "run")
		return root
	}
	first, err := ChecksumTree(build())
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		next, err := ChecksumTree(build())
		if err != nil {
			t.Fatal(err)
		}
		if next != first {
			t.Fatal("two identical trees hashed differently")
		}
	}
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
