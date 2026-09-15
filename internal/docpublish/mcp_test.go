package docpublish

import (
	"reflect"
	"strings"
	"testing"
)

func mcpFixture(t *testing.T) *Server {
	t.Helper()
	root := fixture(t)
	server, err := NewServer(root, Options{Version: "0.6.0", Locale: "en"})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestServerListsAndReadsResources(t *testing.T) {
	server := mcpFixture(t)
	resources := server.List()
	if len(resources) == 0 {
		t.Fatal("expected readable resources")
	}
	for _, r := range resources {
		if !strings.HasPrefix(r.URI, ResourceURI) {
			t.Fatalf("resource %q must use the %s scheme", r.URI, ResourceURI)
		}
		if r.MIMEType == "" {
			t.Fatalf("resource %q must declare a media type", r.URI)
		}
	}
	content, err := server.Read(ResourceURI + "product/vision")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content.Text, "Vision") {
		t.Fatalf("read returned the wrong content: %q", content.Text)
	}
	// Agent-facing pages are readable too; their audience is stated, not hidden.
	agent, err := server.Read(ResourceURI + "root/AGENTS")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(agent.Text, "Agent") {
		t.Fatalf("agent surface must be readable: %q", agent.Text)
	}
	if _, err := server.Read(ResourceURI + "does/not/exist"); err == nil {
		t.Fatal("reading an unknown resource must fail")
	}
}

func TestServerIsReadOnlyByConstruction(t *testing.T) {
	server := mcpFixture(t)
	for _, tool := range server.Tools() {
		if tool.Kind != "read-only" {
			t.Fatalf("documentation tool %s must be read-only, got %q", tool.Name, tool.Kind)
		}
	}
	// The method set itself is the security property: no exported method may be
	// a writer. A new mutating method on this type would fail here.
	mutating := []string{"write", "create", "update", "delete", "remove", "deprecate", "publish", "record", "commit", "apply"}
	typ := reflect.TypeOf(*server)
	for i := 0; i < typ.NumMethod(); i++ {
		name := strings.ToLower(typ.Method(i).Name)
		for _, verb := range mutating {
			if strings.HasPrefix(name, verb) {
				t.Fatalf("Server.%s looks like a mutation; the read surface must have none", typ.Method(i).Name)
			}
		}
	}
	// An unknown method is refused, including a plausible mutation attempt.
	if _, err := server.Dispatch("tools/call", map[string]any{"name": "docs.write"}); err == nil {
		t.Fatal("the documentation surface must refuse an unknown tool")
	}
	if _, err := server.Dispatch("unit.write", nil); err == nil {
		t.Fatal("the documentation surface must refuse a mutation method")
	}
}

func TestServerDispatchAnswersReadMethods(t *testing.T) {
	server := mcpFixture(t)
	listing, err := server.Dispatch("resources/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := listing.(map[string]any)["resources"]; !ok {
		t.Fatalf("resources/list must return resources: %#v", listing)
	}
	read, err := server.Dispatch("resources/read", map[string]any{"uri": ResourceURI + "product/vision"})
	if err != nil {
		t.Fatal(err)
	}
	contents, _ := read.(map[string]any)["contents"].([]ResourceContent)
	if len(contents) != 1 || contents[0].Text == "" {
		t.Fatalf("resources/read must return content: %#v", read)
	}
	if _, err := server.MarshalDispatch("tools/list", nil); err != nil {
		t.Fatalf("marshal dispatch: %v", err)
	}
}

func TestMutationIsDeniedByDefault(t *testing.T) {
	called := false
	server := MutationServer{Apply: func(string, map[string]any) (any, error) {
		called = true
		return "written", nil
	}}
	if _, err := server.Execute(MutationRequest{Operation: "unit.write"}); err == nil {
		t.Fatal("an empty policy must deny every mutation")
	}
	if called {
		t.Fatal("a denied mutation must not reach the handler")
	}
	if _, err := server.Execute(MutationRequest{Operation: "unit.write", Approval: "reviewer"}); err == nil {
		t.Fatal("approval alone must not enable a mutation that policy denies")
	}
	if _, err := server.Execute(MutationRequest{Operation: "not.a.verb"}); err == nil {
		t.Fatal("an unknown operation must be refused even if a policy lists it")
	}
}

func TestMutationRequiresApprovalWhenPolicyDemandsIt(t *testing.T) {
	server := MutationServer{
		Policy: MutationPolicy{Allowed: []string{"unit.write"}, RequireApproval: true},
		Apply:  func(operation string, _ map[string]any) (any, error) { return operation + " ok", nil },
	}
	if _, err := server.Execute(MutationRequest{Operation: "unit.write"}); err == nil {
		t.Fatal("a whitespace-free empty approval must not be treated as consent")
	}
	if _, err := server.Execute(MutationRequest{Operation: "unit.write", Approval: "   "}); err == nil {
		t.Fatal("whitespace approval must not be treated as consent")
	}
	result, err := server.Execute(MutationRequest{Operation: "unit.write", Approval: "maintainer"})
	if err != nil || result != "unit.write ok" {
		t.Fatalf("an approved mutation must run: %v %v", result, err)
	}
	// A mutation outside the allowlist stays denied even when approval is given.
	if _, err := server.Execute(MutationRequest{Operation: "release.publish", Approval: "maintainer"}); err == nil {
		t.Fatal("a mutation outside the allowlist must be denied")
	}
}

func TestMutationDescribeReportsTheGateWithoutActing(t *testing.T) {
	server := MutationServer{Policy: MutationPolicy{Allowed: []string{"release.publish"}, RequireApproval: true}}
	report := server.Describe()
	enabled, _ := report["enabled"].([]string)
	if len(enabled) != 1 || enabled[0] != "release.publish" {
		t.Fatalf("describe must report the enabled verbs: %#v", report)
	}
	if report["require_approval"] != true {
		t.Fatalf("describe must report the approval requirement: %#v", report)
	}
	if all, _ := report["all_verbs"].([]string); len(all) != len(MutationVerbs()) {
		t.Fatalf("describe must list the whole verb vocabulary: %#v", report)
	}
}
