package docengine

import (
	"testing"

	"github.com/raillen/prumo/internal/traceability"
)

func typedRegistry() Registry {
	return Registry{Contracts: map[string]Contract{
		"c1": {ID: "c1", UpdateTriggers: []string{"path:src/harness", "ext:.md"}},
		"c2": {ID: "c2", UpdateTriggers: []string{"contract:c2"}},
		"c3": {ID: "c3", UpdateTriggers: []string{"rel:documents:guide"}},
		"c4": {ID: "c4", UpdateTriggers: []string{"harness"}},
	}}
}

func typedBindings() []Binding {
	return []Binding{{ContractID: "c2", Sources: []string{"docs/c2.md"}}}
}

func typedGraph() *traceability.Graph {
	g := traceability.NewGraph()
	_ = g.AddNode(traceability.Node{ID: "n-code", Kind: traceability.NodeCode, Ref: "src/harness/x.go"})
	_ = g.AddNode(traceability.Node{ID: "n-doc", Kind: traceability.NodeDoc, Ref: "guide"})
	_ = g.AddEdge(traceability.Edge{From: "n-code", To: "n-doc", Kind: string(traceability.EdgeDocuments)})
	return g
}

func TestTypedMatchers(t *testing.T) {
	g := typedGraph()
	cases := []struct {
		trigger string
		changed []string
		want    bool
		prefix  string
	}{
		{"path:src/harness", []string{"src/harness/x.go"}, true, "trigger:path:"},
		{"path:src/harness", []string{"src/other/y.go"}, false, ""},
		{"ext:.md", []string{"docs/a.MD"}, true, "trigger:ext:"},
		{"rel:documents:guide", []string{"src/harness/x.go"}, true, "trigger:rel:"},
		{"rel:verifies:guide", []string{"src/harness/x.go"}, false, ""},
		{"harness", []string{"src/harness/x.go"}, true, "legacy-substring:"},
	}
	for _, tc := range cases {
		got, reason := MatchTrigger(tc.trigger, tc.changed, nil, g)
		if got != tc.want {
			t.Fatalf("%q matched=%v want %v", tc.trigger, got, tc.want)
		}
		if tc.want && reason[:len(tc.prefix)] != tc.prefix {
			t.Fatalf("reason %q must start with %q", reason, tc.prefix)
		}
	}
	if _, reason := MatchTrigger("rel:documents:guide", []string{"src/harness/x.go"}, nil, nil); reason != "" {
		t.Fatal("rel: without graph must miss cleanly")
	}
}

func TestAnalyzeImpactsWithGraph(t *testing.T) {
	got := AnalyzeImpactsWithGraph(typedRegistry(), typedBindings(), []string{"src/harness/x.go", "docs/c2.md"}, typedGraph())
	byID := map[string]string{}
	for _, im := range got {
		byID[im.ContractID] = im.Reason
	}
	if len(got) != 4 {
		t.Fatalf("expected all 4 contracts hit: %+v", got)
	}
	if byID["c2"] != "trigger:contract:c2" {
		t.Fatalf("contract matcher wrong: %v", byID)
	}
	// Legacy path preserved: old entry point still works on bare tokens.
	legacy := AnalyzeImpacts(typedRegistry(), typedBindings(), []string{"src/harness/x.go"})
	if len(legacy) == 0 {
		t.Fatal("legacy entry must keep working")
	}
}
