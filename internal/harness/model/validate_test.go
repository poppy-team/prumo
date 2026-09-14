package model

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestValidateValue(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"path"},
		"properties": map[string]any{
			"path":  map[string]any{"type": "string"},
			"count": map[string]any{"type": "integer"},
		},
	}
	if err := ValidateValue(schema, map[string]any{"path": "a.go", "count": 1.0}); err != nil {
		t.Fatalf("valid must pass: %v", err)
	}
	if err := ValidateValue(schema, map[string]any{"count": 1}); err == nil {
		t.Fatal("missing required must fail")
	}
	if err := ValidateValue(schema, map[string]any{"path": 42}); err == nil {
		t.Fatal("wrong type must fail")
	}
	if err := ValidateValue(map[string]any{"type": "bogus"}, 1); err == nil {
		t.Fatal("unsupported type must error loudly")
	}
	if err := ValidateValue(map[string]any{"enum": []any{"a", "b"}}, "c"); err == nil {
		t.Fatal("enum violation must fail")
	}
}

func TestValidateArgsEnforced(t *testing.T) {
	specs := []agent.ToolSpec{{Name: "fs.read", Schema: map[string]any{
		"type": "object", "required": []any{"path"},
		"properties": map[string]any{"path": map[string]any{"type": "string"}},
	}}}
	if err := ValidateArgs(specs, "fs.read", map[string]any{"path": "a"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArgs(specs, "fs.read", map[string]any{}); err == nil {
		t.Fatal("missing path must fail")
	}
	if err := ValidateArgs(specs, "unknown-tool", nil); err != nil {
		t.Fatalf("unknown tools pass (registry owns catalog): %v", err)
	}
	if err := UnknownToolError(specs, "nope"); err == nil {
		t.Fatal("helper must error")
	}
}
