package schemaruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/uimap"
)

// TestInterfaceMapVocabulariesMatchSchema is the W1 rule applied to the
// interface map: a closed vocabulary written twice drifts, and the drift is
// invisible until a renderer meets a kind the validator never allowed.
func TestInterfaceMapVocabulariesMatchSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "schemas", "ui-interface-map.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties  map[string]json.RawMessage `json:"properties"`
		Definitions map[string]struct {
			Properties map[string]struct {
				Enum  []string `json:"enum"`
				Items struct {
					Enum []string `json:"enum"`
				} `json:"items"`
			} `json:"properties"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	node := schema.Definitions["node"].Properties
	slot := schema.Definitions["slot"].Properties
	edge := schema.Definitions["edge"].Properties

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"node.kind", uimap.NodeKinds(), node["kind"].Enum},
		{"node.size_classes", uimap.SizeClasses(), node["size_classes"].Items.Enum},
		{"edge.kind", uimap.EdgeKinds(), edge["kind"].Enum},
		{"slot.region", uimap.Regions(), slot["region"].Enum},
		{"slot.align", uimap.Alignments(), slot["align"].Enum},
		{"slot.stack", uimap.Stacks(), slot["stack"].Enum},
	}
	for _, tc := range cases {
		got, want := sortedCopy(tc.got), sortedCopy(tc.want)
		if len(want) == 0 {
			t.Errorf("%s: schema declares no enum, so the vocabulary is unenforced", tc.name)
			continue
		}
		if !equal(got, want) {
			t.Errorf("%s drift:\n  Go:     %v\n  schema: %v", tc.name, got, want)
		}
	}
}

// TestInterfaceMapSchemaForbidsUnknownFields asserts the schema and the loader
// agree that an unknown field is an error. A loader that rejects what the schema
// allows, or the reverse, means the contract in the repository is not the one the
// compiler enforces.
func TestInterfaceMapSchemaForbidsUnknownFields(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "schemas", "ui-interface-map.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		AdditionalProperties *bool `json:"additionalProperties"`
		Definitions          map[string]struct {
			AdditionalProperties *bool `json:"additionalProperties"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if schema.AdditionalProperties == nil || *schema.AdditionalProperties {
		t.Error("the map itself must not accept unknown fields")
	}
	for _, def := range []string{"node", "slot", "edge", "implementation"} {
		entry, ok := schema.Definitions[def]
		if !ok {
			t.Errorf("schema has no %q definition", def)
			continue
		}
		if entry.AdditionalProperties == nil || *entry.AdditionalProperties {
			t.Errorf("%q accepts unknown fields, but the loader rejects them", def)
		}
	}
}

// TestProjectMapIsValid is the dogfood gate: this repository's own interface map
// must satisfy the rules it ships. A framework that exempts its own repository
// from its contract has no contract.
//
// It deliberately says nothing about projection freshness. Projections live only
// under `.prumo/runtime/`, so a clean checkout has none and a freshness check here
// would either fail for the wrong reason or pass because someone happened to run
// `--write` locally. The digest round trip is covered hermetically in
// `internal/uimap`, where a temporary root makes the write and the stale case
// deterministic; `prumo ui verify` is the check that runs where the artifacts exist.
func TestProjectMapIsValid(t *testing.T) {
	root := repoRoot(t)
	cfg, err := uimap.ResolveConfig(root)
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("this repository declares a tui project type, so the interface map must be on: %s", cfg.Reason)
	}
	result, err := uimap.Compile(root, uimap.Options{})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !result.Report.OK() {
		for _, f := range result.Report.Findings {
			if f.Severity == uimap.SeverityError {
				t.Errorf("%s %s: %s", f.Code, f.Node, f.Message)
			}
		}
		t.FailNow()
	}

	// Derived artifacts must stay in runtime: a projection under `docs/` would be
	// a regenerable summary sitting where readers look for canonical sources.
	rel, err := filepath.Rel(root, uimap.ProjectionRoot(root))
	if err != nil {
		t.Fatalf("locate the projection root: %v", err)
	}
	slash := filepath.ToSlash(rel)
	if slash == "docs" || strings.HasPrefix(slash, "docs/") {
		t.Errorf("projections are rooted at %q; derived artifacts must not become canonical files", slash)
	}
	if len(result.Projections) == 0 {
		t.Error("the map compiled but produced no projections, so no reader is served")
	}
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
