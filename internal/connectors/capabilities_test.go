package connectors_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/connectors/antigravity"
	"github.com/raillen/prumo/internal/connectors/claudecode"
	"github.com/raillen/prumo/internal/connectors/codex"
	"github.com/raillen/prumo/internal/connectors/gemini"
	"github.com/raillen/prumo/internal/connectors/opencode"
)

// A connector declares what it can do, and that declaration is what a caller
// plans against. The list was maintained by hand next to the code that generates
// the artifacts, so it drifted: a connector said it could verify after a tool
// ran, or isolate subagents, and the compile wrote nothing that did either
// (GAP-142).
//
// The compile now reports what it actually generated, and this test requires the
// two to be the same. A capability that cannot be derived from an artifact
// cannot be declared.

func TestEveryConnectorDeclaresWhatItGenerates(t *testing.T) {
	all := connectors.List()
	if len(all) == 0 {
		t.Fatal("no connectors are registered")
	}
	for _, c := range all {
		t.Run(c.ID(), func(t *testing.T) {
			root := t.TempDir()
			result, err := c.Compile(root, connectors.CompileOptions{})
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			declared := append([]string{}, c.Contract().Capabilities...)
			generated := append([]string{}, result.Implemented...)
			sort.Strings(declared)
			sort.Strings(generated)

			var missing, extra []string
			for _, want := range declared {
				if !inList(generated, want) {
					missing = append(missing, want)
				}
			}
			for _, got := range generated {
				if !inList(declared, got) {
					extra = append(extra, got)
				}
			}
			if len(missing) > 0 {
				t.Errorf("declares capabilities the compile did not generate: %v", missing)
			}
			if len(extra) > 0 {
				t.Errorf("generates capabilities it does not declare: %v", extra)
			}
		})
	}
}

func TestEveryConnectorDeclaresOnlyTheHooksItWires(t *testing.T) {
	// A hook declared in a contract and a hook present in the generated hooks
	// file are two claims, and only one of them runs. This is the same check as
	// capabilities, applied to the other half of the contract.
	for _, c := range connectors.List() {
		t.Run(c.ID(), func(t *testing.T) {
			root := t.TempDir()
			result, err := c.Compile(root, connectors.CompileOptions{})
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			declared := append([]string{}, c.Contract().Hooks...)
			sort.Strings(declared)
			generated := append([]string{}, result.ImplementedHooks...)
			sort.Strings(generated)

			var missing, extra []string
			for _, want := range declared {
				if !inList(generated, want) {
					missing = append(missing, want)
				}
			}
			for _, got := range generated {
				if !inList(declared, got) {
					extra = append(extra, got)
				}
			}
			if len(missing) > 0 {
				t.Errorf("declares hooks the compile did not wire: %v", missing)
			}
			if len(extra) > 0 {
				t.Errorf("wires hooks it does not declare: %v", extra)
			}
		})
	}
}

func TestNoConnectorDeclaresNothingItCannotBack(t *testing.T) {
	// A connector that writes no hooks must not declare a hook capability, and
	// the common failure — declaring everything because the contract is a
	// template — shows up here first.
	for _, c := range connectors.List() {
		declared := c.Contract().Capabilities
		if len(declared) == 0 {
			t.Errorf("%s declares no capabilities at all", c.ID())
		}
		seen := map[string]bool{}
		for _, capability := range declared {
			if seen[capability] {
				t.Errorf("%s declares %q twice", c.ID(), capability)
			}
			seen[capability] = true
		}
	}
}

func TestAConnectorsPluginCapabilityMeansAPluginWasWritten(t *testing.T) {
	// The one capability whose evidence is a single file rather than a config
	// key, checked on its own because a plugin that does not exist is the most
	// visible way to over-declare.
	root := t.TempDir()
	result, err := opencode.NewConnector().Compile(root, connectors.CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !inList(result.Implemented, connectors.CapNativePlugin) {
		t.Fatalf("opencode generated no plugin capability: %v", result.Implemented)
	}
	found := false
	for _, path := range result.CreatedPaths {
		if filepath.Base(path) == "prumo.ts" {
			found = true
		}
	}
	if !found {
		t.Fatal("opencode claims a native plugin and wrote no prumo.ts")
	}
}

// The connectors are reachable through their constructors, and the test above
// goes through the registry. These references keep the imports honest if the
// registry ever stops registering one of them.
var _ = []connectors.Connector{
	opencode.NewConnector(), codex.NewConnector(), claudecode.NewConnector(),
	antigravity.NewConnector(), gemini.NewConnector(),
}

func inList(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
