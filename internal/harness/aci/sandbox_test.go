package aci

import "testing"

func TestSandboxChainHonest(t *testing.T) {
	chain := DefaultChain(t.TempDir())
	if len(chain) != 4 {
		t.Fatalf("expected 4-rung ladder, got %d", len(chain))
	}
	if !chain[0].Available() || !chain[1].Available() {
		t.Fatal("local + worktree must be available")
	}
	bogus := ContainerProvider{Runtime: "prumo-no-such-runtime", Image: "x"}
	if bogus.Available() {
		t.Fatal("bogus runtime must be unavailable")
	}
	rt := DetectContainerRuntime()
	if rt != "" && rt != "podman" && rt != "docker" {
		t.Fatalf("unexpected runtime %q", rt)
	}
	if strong := DetectStrongRuntime(); strong != "" && strong != "runsc" {
		t.Fatalf("unexpected strong runtime %q", strong)
	}
	if (StrongProvider{Runtime: "prumo-no-such-runtime"}.Available()) {
		t.Fatal("bogus strong runtime must be unavailable")
	}
}
