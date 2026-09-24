package doclifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A nested Go module is invisible to `go test ./...` from the root: the command
// walks packages, skips a directory that has its own go.mod, and says nothing.
// That is a Go fact rather than a defect, and it is why prumo-tui has its own CI
// job (GAP-148).
//
// The real risk is not the module that has a job. It is the next one: a nested
// module added by someone who does not know the root command will not reach it,
// and the omission is silent. This test is the reminder — it fails when a nested
// module exists that no CI workflow runs tests in.
func TestEveryNestedModuleHasACITestJob(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", ".."))
	root, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}

	modules := nestedModules(t, root)
	if len(modules) == 0 {
		t.Skip("no nested modules in this checkout")
	}

	workflows := readWorkflows(t, root)
	if len(workflows) == 0 {
		t.Fatalf("there are nested modules (%s) but no workflow files to check against", strings.Join(modules, ", "))
	}

	for _, module := range modules {
		if !anyWorkflowRunsIn(t, workflows, module) {
			t.Errorf("nested module %q is not tested by any CI workflow; "+
				"`go test ./...` from the root skips it, so nothing runs it", module)
		}
	}
}

// nestedModules finds every go.mod below the root that is not the root's.
func nestedModules(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "go.mod" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		if rel == "go.mod" {
			return nil
		}
		// A module under testdata/ or a vendored tree is a fixture or a
		// dependency, not something this repository ships and tests. A fixture
		// with its own go.mod is exactly what a migration test needs, and
		// requiring a CI job for it would be requiring the gate to be wrong.
		if strings.HasPrefix(filepath.ToSlash(rel), "testdata/") || strings.Contains(rel, string(filepath.Separator)+"vendor"+string(filepath.Separator)) {
			return nil
		}
		// Only a module's own root counts. A directory inside a module that has a
		// go.mod of its own is a different module, and one that does not is part
		// of its parent.
		found = append(found, filepath.ToSlash(filepath.Dir(rel)))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// readWorkflows returns every workflow file's contents, keyed by path.
func readWorkflows(t *testing.T, root string) map[string]string {
	t.Helper()
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("no workflow directory: %v", err)
	}
	out := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			t.Fatal(readErr)
		}
		out[entry.Name()] = string(data)
	}
	return out
}

// anyWorkflowRunsIn reports whether some workflow runs go test inside the module.
func anyWorkflowRunsIn(t *testing.T, workflows map[string]string, module string) bool {
	t.Helper()
	for name, body := range workflows {
		// A job that sets its working-directory to the module runs there. The
		// check is deliberately textual: a workflow is YAML this repository does
		// not parse anywhere, and a false positive here means an extra job, while
		// a false negative means an untested module.
		if strings.Contains(body, "working-directory: "+module) ||
			strings.Contains(body, "working-directory: ./"+module) {
			if strings.Contains(body, "go test") {
				return true
			}
			t.Logf("workflow %s runs in %s but has no go test step", name, module)
		}
	}
	return false
}

// A gate that never fails is decoration. This proves it fails for a module
// nobody tests, and that the fixture exclusion is what excludes it — not an
// accident that would hide a real module too.
func TestTheGateWouldCatchAModuleNobodyTests(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".github", "workflows", "ci.yml"),
		"jobs:\n  build:\n    steps:\n      - run: go test ./...\n")
	mustWrite(t, filepath.Join(root, "prumo-agent", "go.mod"), "module example.com/agent\n\ngo 1.22\n")

	modules := nestedModules(t, root)
	if len(modules) != 1 || modules[0] != "prumo-agent" {
		t.Fatalf("the module was not found: %v", modules)
	}
	workflows := readWorkflows(t, root)
	if anyWorkflowRunsIn(t, workflows, "prumo-agent") {
		t.Fatal("the gate passed a module no workflow tests")
	}

	// And a fixture is excluded on purpose, which is the difference between a
	// gate and a gate that cries wolf on every test fixture.
	mustWrite(t, filepath.Join(root, "testdata", "sample", "go.mod"), "module example.com/sample\n\ngo 1.22\n")
	if found := nestedModules(t, root); len(found) != 1 {
		t.Fatalf("a testdata fixture was counted as a module: %v", found)
	}
}

func TestTheGatePassesAModuleWithAJob(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".github", "workflows", "ci.yml"),
		"jobs:\n  agent-test:\n    defaults:\n      run:\n        working-directory: prumo-agent\n    steps:\n      - run: go test -race ./...\n")
	mustWrite(t, filepath.Join(root, "prumo-agent", "go.mod"), "module example.com/agent\n\ngo 1.22\n")

	workflows := readWorkflows(t, root)
	for _, module := range nestedModules(t, root) {
		if !anyWorkflowRunsIn(t, workflows, module) {
			t.Errorf("a module with a CI job was reported as untested: %s", module)
		}
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
