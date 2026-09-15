package docengine

import (
	"os"
	"path/filepath"
	"testing"
)

func testRegistry(t *testing.T) Registry {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

// writeManifest declares a project whose capabilities are exactly what the test
// needs, so profile resolution is exercised without depending on the
// repository's own prumo.json.
func writeManifest(t *testing.T, root string, types, features []string) {
	t.Helper()
	manifest := `{"version":3,"protocol":{"version":3},"project":{"name":"t","type":[`
	for i, value := range types {
		if i > 0 {
			manifest += ","
		}
		manifest += `"` + value + `"`
	}
	manifest += `]},"features":[`
	for i, value := range features {
		if i > 0 {
			manifest += ","
		}
		manifest += `"` + value + `"`
	}
	manifest += `]}`
	if err := os.WriteFile(filepath.Join(root, "prumo.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestEveryBuiltinProfileIsReachable is the guard for the defect this test was
// written for: `tui` was declared in the builtin registry with 26 contracts and
// never composed, because the capability→profile map did not know it. A profile
// nobody can select is a gate nobody runs.
func TestEveryBuiltinProfileIsReachable(t *testing.T) {
	registry := testRegistry(t)
	for id := range registry.Profiles {
		if id == "core-software" {
			continue // always composed, by definition
		}
		reachable := false
		for _, pair := range capabilityProfiles {
			if pair.profile == id {
				reachable = true
				break
			}
		}
		if !reachable {
			t.Errorf("profile %q is declared but no capability composes it", id)
		}
	}
}

// TestTUIImpliesUICapability covers the second half of the same defect: selecting
// the tui profile must also enable the `ui` capability, otherwise the profile
// selects its contracts and the applicability filter removes them again.
func TestTUIImpliesUICapability(t *testing.T) {
	registry := testRegistry(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, root, []string{"cli", "tui"}, nil)

	profiles, capabilities := ResolveProfiles(root, registry)
	if !contains(profiles, "tui") {
		t.Fatalf("a tui project must compose the tui profile: %v", profiles)
	}
	if !capabilities["tui"] || !capabilities["ui"] {
		t.Fatalf("the tui profile declares tui and ui; got tui=%v ui=%v",
			capabilities["tui"], capabilities["ui"])
	}

	contracts, unknown, err := registry.ResolveProfiles(profiles, capabilities)
	if err != nil {
		t.Fatal(err)
	}
	if len(unknown) != 0 {
		t.Fatalf("profile references unknown contracts: %v", unknown)
	}
	ids := map[string]bool{}
	for _, contract := range contracts {
		ids[contract.ID] = true
	}
	for _, want := range []string{"ui.state-model", "ui.interaction", "tui.interaction", "tui.accessibility"} {
		if !ids[want] {
			t.Errorf("%s must be selected and applicable for a tui project", want)
		}
	}
}

// TestProfileCapabilitiesDoNotLeak keeps a CLI-only project from inheriting the
// UI obligations: the union must be driven by the composed profiles, not by
// whatever the registry happens to declare.
func TestProfileCapabilitiesDoNotLeak(t *testing.T) {
	registry := testRegistry(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, root, []string{"cli"}, nil)

	profiles, capabilities := ResolveProfiles(root, registry)
	if contains(profiles, "tui") {
		t.Fatalf("a cli project must not compose the tui profile: %v", profiles)
	}
	if capabilities["tui"] || capabilities["ui"] {
		t.Fatalf("a cli project must not gain UI capabilities: %v", capabilities)
	}
}
