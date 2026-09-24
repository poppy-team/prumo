package install

import (
	"os"
	"path/filepath"
	"testing"
)

// There was one cleanup file per connector, with no project in it. Installing a
// connector into a second project overwrote the first project's record, so
// uninstalling cleaned up whichever project was installed last and left the
// other's files behind with nothing pointing at them. Fifty projects meant
// forty-nine sets of orphaned files (GAP-137).

func TestTwoProjectsGetTwoCleanupRecords(t *testing.T) {
	home := t.TempDir()
	projectA := t.TempDir()
	projectB := t.TempDir()

	pathA := CleanupPath(home, "opencode", projectA)
	pathB := CleanupPath(home, "opencode", projectB)
	if pathA == pathB {
		t.Fatalf("two projects share one cleanup record: %s", pathA)
	}

	if err := os.MkdirAll(filepath.Dir(pathA), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathA, []byte(`{"connector":"opencode","created_paths":["/a"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordCleanup(home, "opencode", projectA); err != nil {
		t.Fatal(err)
	}
	// The second install is the one that used to destroy the first record.
	if err := os.MkdirAll(filepath.Dir(pathB), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte(`{"connector":"opencode","created_paths":["/b"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordCleanup(home, "opencode", projectB); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{pathA, pathB} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("a project's cleanup record is gone: %s (%v)", path, err)
		}
	}

	index, err := LoadCleanupIndex(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Entries) != 2 {
		t.Fatalf("the index names %d records, want 2: %+v", len(index.Entries), index.Entries)
	}
}

func TestTheSameProjectRecordedTwiceIsOneEntry(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	for range 3 {
		if err := RecordCleanup(home, "opencode", project); err != nil {
			t.Fatal(err)
		}
	}
	index, err := LoadCleanupIndex(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Entries) != 1 {
		t.Fatalf("reinstalling into the same project produced %d index entries", len(index.Entries))
	}
}

func TestForgettingOneProjectLeavesTheOthers(t *testing.T) {
	home := t.TempDir()
	projectA, projectB := t.TempDir(), t.TempDir()
	for _, project := range []string{projectA, projectB} {
		if err := RecordCleanup(home, "opencode", project); err != nil {
			t.Fatal(err)
		}
	}
	if err := ForgetCleanup(home, "opencode", projectA); err != nil {
		t.Fatal(err)
	}
	index, err := LoadCleanupIndex(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Entries) != 1 {
		t.Fatalf("uninstalling one project left %d entries, want 1", len(index.Entries))
	}
	if index.Entries[0].ProjectRoot != projectB {
		t.Fatalf("the wrong project's record was kept: %q", index.Entries[0].ProjectRoot)
	}
}

func TestTheProjectNameDoesNotLeakAPath(t *testing.T) {
	// A readable directory name would have to be escaped and would put a user's
	// directory layout into a filename.
	project := filepath.Join(t.TempDir(), "cliente", "projeto secreto")
	dir := ScopeDir(project)
	if dir == "global" {
		t.Fatal("a real project was filed as global")
	}
	if len(dir) < 8 || dir[:8] != "project-" {
		t.Fatalf("the scope directory is %q, want a project digest", dir)
	}
	for _, secret := range []string{"cliente", "projeto", "secreto"} {
		if filepathContains(dir, secret) {
			t.Fatalf("the scope directory %q leaks %q", dir, secret)
		}
	}
}

func filepathContains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}()
}

func TestTheSameProjectAlwaysGetsTheSameDirectory(t *testing.T) {
	// Two installs of the same project must land on the same record, or the
	// "one file per connector" problem returns with extra steps.
	_ = t.TempDir()
	a := ScopeDir("/home/someone/project")
	b := ScopeDir("/home/someone/project/")
	if a != b {
		t.Fatalf("a trailing slash produced a different record: %q vs %q", a, b)
	}
}
