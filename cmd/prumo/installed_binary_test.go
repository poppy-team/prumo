package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/resources"
)

// The assets were embedded into the binary and never wired to anything:
// resources.DefaultFS stayed nil, every read went to disk, and the repository
// root was found by walking up from the executable and then from the working
// directory. When neither had a checkout it fell through to one developer's
// home directory and then to ".". So an installed Prumo read whatever schemas
// happened to sit in the directory someone ran it from, and a version check
// could be satisfied by an unrelated checkout (GAP-140).
//
// The tests build the binary and run it somewhere that is not a repository.

func buildBinary(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("building the binary is too slow for -short")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "prumo")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Env = append(os.Environ(), "PRUMO_REPO_ROOT=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the binary: %v\n%s", err, out)
	}
	return binary
}

func TestTheInstalledBinaryFindsNoCheckoutByGuessing(t *testing.T) {
	// The hard-coded developer home is gone, and so is the "." fallback: a
	// binary with no checkout says so instead of pointing at the caller's
	// current directory.
	//
	// This test binary itself lives inside a checkout, so the executable walk
	// legitimately finds one. What must not happen is the home-directory guess
	// or the "." return; the installed-binary test below covers a binary that
	// is genuinely outside any checkout.
	t.Setenv("PRUMO_REPO_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	root := repoRoot()
	if root == "." {
		t.Fatal("repoRoot() returned the caller's working directory as a repository root")
	}
	if strings.HasSuffix(root, filepath.Join("Documentos", "Projetos", "prumo")) {
		t.Fatalf("repoRoot() still guesses one developer's home directory: %q", root)
	}
}

func TestTheEmbeddedSchemasAreReachableWithNoRepoRoot(t *testing.T) {
	// The whole point of embedding: resource reads work with an empty root,
	// because the answer is in the binary rather than on disk.
	root := ""
	schema, err := resources.OpenSchema(root, "prumo.schema.json")
	if err != nil {
		t.Fatalf("the embedded schemas are unreachable with no repo root: %v", err)
	}
	if len(schema) == 0 {
		t.Fatal("the embedded prumo.schema.json is empty")
	}
	if !strings.Contains(string(schema), `"$schema"`) {
		t.Fatal("what was read is not a JSON Schema")
	}
}

func TestTheEmbeddedCatalogIsReachableWithNoRepoRoot(t *testing.T) {
	if resources.AdaptersFS == nil {
		t.Fatal("the adapter assets were not installed")
	}
	// fs.FS has no ReadDir; the point is that a known adapter is readable.
	name, err := fs.ReadFile(resources.AdaptersFS, "claude.md")
	if err != nil {
		names := knownAdapter(t)
		t.Fatalf("the embedded adapters are unreadable (%v); the set is %v", err, names)
	}
	if len(name) == 0 {
		t.Fatal("an embedded adapter is empty")
	}
}

// knownAdapter reports what the source tree actually contains, so a failure
// says which name was tried rather than only that some name was tried.
func knownAdapter(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("..", "..", "src", "prumo", "resources", "adapters"))
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestAnInstalledBinaryDoesNotLookForAResourceOnDisk(t *testing.T) {
	if testing.Short() {
		t.Skip("building the binary is too slow for -short")
	}
	binary := buildBinary(t)
	// Somewhere with no go.mod, no schemas, and nothing Prumo-shaped.
	elsewhere := t.TempDir()
	cmd := exec.Command(binary, "docs", "verify", "--path", elsewhere, "--strict")
	cmd.Dir = elsewhere
	cmd.Env = append(os.Environ(),
		"PRUMO_REPO_ROOT=",
		"HOME="+elsewhere,
		"XDG_CONFIG_HOME="+filepath.Join(elsewhere, ".config"),
	)
	out, _ := cmd.CombinedOutput()
	text := string(out)

	// An empty directory has no documents, so this command is expected to
	// complain — about documents. What it must not do is go looking for a
	// resource on disk, which is what an installed binary used to do: no
	// checkout beside the executable, no checkout in the working directory, and
	// a guess at one developer's home.
	for _, missing := range []string{
		"no such file or directory: schemas",
		"no such file or directory: src/prumo/resources",
		"Documentos/Projetos/prumo",
	} {
		if strings.Contains(text, missing) {
			t.Fatalf("the installed binary reached for %q on disk:\n%s", missing, text)
		}
	}
	// And the complaint, if any, is about the empty project rather than about
	// Prumo being unable to find itself.
	if text != "" && !strings.Contains(text, "docs/") && !strings.Contains(text, "entrypoint") &&
		!strings.Contains(text, "ENTRYPOINT") {
		t.Fatalf("the installed binary failed for an unrelated reason:\n%s", text)
	}
}
