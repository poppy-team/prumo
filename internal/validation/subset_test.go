package validation

import "testing"

// The validator recognised a strict subset of what the schemas actually use.
// A schema with `type: ["boolean", "string"]` — the form that says "a flag may
// be a bool, or a path to a value that overrides it" — had its type constraint
// skipped entirely, and `enum` was read only as []any, so an enum checked
// against a value that had not been through JSON enforced nothing (GAP-151).

func TestATypeUnionIsChecked(t *testing.T) {
	schema := map[string]any{"type": []any{"boolean", "string"}}
	for _, ok := range []any{true, false, "enabled", ""} {
		if problems := ValidateSchema(ok, schema, Registry{}); len(problems) > 0 {
			t.Errorf("%#v is allowed by the union but was rejected: %v", ok, problems)
		}
	}
	for _, bad := range []any{1, nil, []any{}, map[string]any{}} {
		if problems := ValidateSchema(bad, schema, Registry{}); len(problems) == 0 {
			t.Errorf("%#v is not a boolean or a string but the union accepted it", bad)
		}
	}
}

func TestAGoValueIsJudgedByItsJsonType(t *testing.T) {
	// A value built in memory has never been through json.Unmarshal, so its
	// arrays and objects are []string and map[string]string rather than []any
	// and map[string]any. The old type assertion saw those as "not an array" and
	// reported a failure that did not exist — which reads, to whoever hits it,
	// as "the schema is too strict" (GAP-151).
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"names": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"count": map[string]any{"type": "integer"},
		},
		"required": []any{"names"},
	}
	instance := map[string]any{"names": []string{"a", "b"}, "count": 3}
	if problems := ValidateSchema(instance, schema, Registry{}); len(problems) > 0 {
		t.Fatalf("a valid Go value was rejected: %v", problems)
	}

	// And the looseness is not blanket: a genuinely wrong value still fails.
	bad := map[string]any{"names": []string{"a"}, "count": "three"}
	if problems := ValidateSchema(bad, schema, Registry{}); len(problems) == 0 {
		t.Fatal("a string where the schema says integer was accepted")
	}
}

func TestAnEnumBuiltAsAGoSliceIsEnforced(t *testing.T) {
	schema := map[string]any{"enum": []string{"low", "medium", "high"}}
	if problems := ValidateSchema("medium", schema, Registry{}); len(problems) > 0 {
		t.Fatalf("a listed value was rejected: %v", problems)
	}
	if problems := ValidateSchema("extreme", schema, Registry{}); len(problems) == 0 {
		t.Fatal("an unlisted value was accepted because the enum was a Go slice")
	}
}
