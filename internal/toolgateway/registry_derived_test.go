package toolgateway

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/aci"
)

// The registry carried a hand-written list beside the executor and the two
// drifted: four descriptors were registered with kinds, scopes, timeouts and
// output limits and had no implementation behind them, so `tool list`
// advertised capabilities that could not be invoked (GAP-158).

func TestEveryRegisteredNativeToolIsImplemented(t *testing.T) {
	// The property that closes the gap rather than the four instances: a tool
	// described by the registry can be run.
	registry := NewRegistry()
	RegisterACI(registry)

	catalogue := map[string]bool{}
	for _, tool := range aci.Catalog() {
		catalogue[tool.Name] = true
	}
	if len(registry.List()) == 0 {
		t.Fatal("the registry must describe the native tools")
	}
	for _, descriptor := range registry.List() {
		if !catalogue[descriptor.ID] {
			t.Fatalf("the registry describes %q, which the catalogue does not implement", descriptor.ID)
		}
	}
}

func TestEveryCatalogueToolIsDescribed(t *testing.T) {
	// The other direction. RegisterACI panics on an unmapped entry, so this is
	// about the registry being complete rather than the mapping being total.
	registry := NewRegistry()
	RegisterACI(registry)
	for _, tool := range aci.Catalog() {
		descriptor, ok := registry.Get(tool.Name)
		if !ok {
			t.Fatalf("catalogue tool %q is not described by the registry", tool.Name)
		}
		if descriptor.Description == "" {
			t.Fatalf("tool %q is described with no description; a model is told nothing about it", tool.Name)
		}
		if descriptor.Kind == "" {
			t.Fatalf("tool %q has no kind, so policy cannot classify it", tool.Name)
		}
		if len(descriptor.FilesystemScope) == 0 {
			t.Fatalf("tool %q has no filesystem scope, so containment cannot be evaluated", tool.Name)
		}
	}
}

func TestTheKindsTheRegistryReportsAreTheKindsTheCatalogueHas(t *testing.T) {
	// A tool classified as read-only in the registry and side-effecting in the
	// executor is a permission decision made on the wrong information.
	registry := NewRegistry()
	RegisterACI(registry)
	for _, tool := range aci.Catalog() {
		descriptor, _ := registry.Get(tool.Name)
		if string(descriptor.Kind) != tool.Kind {
			t.Fatalf("tool %q: registry says %q, catalogue says %q", tool.Name, descriptor.Kind, tool.Kind)
		}
	}
}

func TestTheDestructiveToolIsRegisteredAsDestructive(t *testing.T) {
	// The one that must never be misclassified: the policy asks for it by name.
	registry := NewRegistry()
	RegisterACI(registry)
	descriptor, ok := registry.Get("edit.delete")
	if !ok {
		t.Fatal("edit.delete is not registered")
	}
	if descriptor.Kind != Destructive {
		t.Fatalf("edit.delete must be registered destructive, got %q", descriptor.Kind)
	}
	// And safe mode must refuse it, which is the decision that depends on it.
	decision := Evaluate(descriptor, "/repo", "/repo/a.txt", true)
	if decision.Allowed {
		t.Fatal("safe mode must refuse the destructive tool through the registry")
	}
}
