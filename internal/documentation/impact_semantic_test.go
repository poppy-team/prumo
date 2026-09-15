package docengine

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSemanticFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"internal/demo/demo.go":         "package demo\n\n// Readiness reports state.\nfunc Readiness() bool { return true }\n",
		"schemas/goal.schema.json":      `{"$id":"goal"}`,
		"docs/ui-ux/design-tokens.json": `{"tokens":[{"id":"accent.primary"}]}`, "docs/contracts/builtin.json": `[{"id":"ui.command-palette","version":1,"role":"ui","required_knowledge":["palette states"]}]`,
		"docs/profiles/builtin.json":           `[{"id":"core-software","version":1,"capabilities":["core"],"contracts":["ui.command-palette"]}]`,
		"docs/contracts/bindings.json":         `[{"contract_id":"ui.command-palette","sources":["docs/ui-ux/command-palette.md"],"ownership":"human-maintained","authority":"canonical-documentation"}]`,
		"docs/ui-ux/command-palette.md":        "# Command palette\n\nKeyboard states and empty state.\n",
		"docs/decisions.md":                    "# Decisions\n\nADR 010 records the documentation control plane.\n",
		"docs/development/testing-strategy.md": "# Testing\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestSemanticTriggerMatchers(t *testing.T) {
	root := writeSemanticFixture(t)
	cases := []struct {
		name    string
		trigger string
		changed []string
		sources []string
		want    bool
	}{
		{"symbol present in changed source", "symbol:Readiness", []string{"internal/demo/demo.go"}, nil, true},
		{"symbol absent", "symbol:NothingHere", []string{"internal/demo/demo.go"}, nil, false},
		{"symbol ignores non-source files", "symbol:Readiness", []string{"docs/decisions.md"}, nil, false},
		{"schema file", "schema:goal", []string{"schemas/goal.schema.json"}, nil, true},
		{"schema extensionless", "schema:goal.schema", []string{"schemas/goal.schema.json"}, nil, true},
		{"schema unrelated", "schema:agent-event", []string{"schemas/goal.schema.json"}, nil, false},
		{"token declared in changed token file", "token:accent.primary", []string{"docs/ui-ux/design-tokens.json"}, nil, true},
		{"ui contract bound source", "ui:command-palette", []string{"docs/ui-ux/command-palette.md"},
			[]string{"docs/ui-ux/command-palette.md"}, true},
		{"ui contract unrelated path", "ui:command-palette", []string{"docs/development/testing-strategy.md"},
			[]string{"docs/ui-ux/command-palette.md"}, false},
		{"typed namespace in bound documents", "adr:010", []string{"docs/decisions.md"}, []string{"docs/decisions.md"}, true},
		{"unresolved identifier never matches", "adr:999", []string{"docs/decisions.md"}, []string{"docs/decisions.md"}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ok, reason := MatchTriggerIn(root, tc.trigger, tc.changed, tc.sources, nil)
			if ok != tc.want {
				t.Fatalf("MatchTriggerIn(%q) = %v (%s), want %v", tc.trigger, ok, reason, tc.want)
			}
			if ok && reason != "trigger:"+tc.trigger {
				t.Fatalf("unexpected reason %q", reason)
			}
		})
	}
}

func TestSemanticNamespaceListIsDocumented(t *testing.T) {
	want := []string{"api", "event", "permission", "locale", "media", "goal", "wave", "adr", "release", "evidence"}
	if len(SemanticTriggerKinds) != len(want) {
		t.Fatalf("namespace list changed: %v", SemanticTriggerKinds)
	}
	for _, w := range want {
		if !containsString(SemanticTriggerKinds, w) {
			t.Fatalf("missing namespace %q", w)
		}
	}
	for _, tc := range []struct{ trigger, want string }{
		{"symbol:Readiness", "high"},
		{"schema:goal", "high"},
		{"path:internal/", "medium"},
		{"adr:010", "medium"},
	} {
		if got := impactSeverity(tc.trigger); got != tc.want {
			t.Fatalf("severity(%q) = %s, want %s", tc.trigger, got, tc.want)
		}
	}
}
