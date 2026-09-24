package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckJSON(t *testing.T) {
	if err := CheckJSON([]byte(`{"valid": true}`)); err != nil {
		t.Fatalf("expected valid json, got error: %v", err)
	}
	if err := CheckJSON([]byte(`invalid json`)); err == nil {
		t.Fatalf("expected invalid json to return error")
	}
	if err := CheckJSON([]byte(`null`)); err == nil {
		t.Fatalf("expected null json to return error")
	}
	if err := CheckJSONObject([]byte(`{"valid": true}`)); err != nil {
		t.Fatalf("expected valid object, got error: %v", err)
	}
	if err := CheckJSONObject([]byte(`[1, 2, 3]`)); err == nil {
		t.Fatalf("expected array to fail object check")
	}
}

func schemaDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return filepath.Join(root, "schemas")
}

func TestLoadRegistry(t *testing.T) {
	registry, err := LoadRegistry(schemaDir(t))
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if _, ok := registry.Schema("goal.schema.json"); !ok {
		t.Fatalf("goal schema missing")
	}
}

func TestValidateGoalFixture(t *testing.T) {
	dir := schemaDir(t)
	registry, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "..", "conformance", "goals", "valid_locked_goal.json"))
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		t.Fatalf("fixture parse: %v", err)
	}
	schema, _ := registry.Schema("goal.schema.json")
	if errors := ValidateSchema(instance, schema, registry); len(errors) != 0 {
		t.Fatalf("expected conformance goal valid, got %v", errors)
	}
}

func TestValidateGoalRejectsInvalid(t *testing.T) {
	dir := schemaDir(t)
	registry, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	schema, _ := registry.Schema("goal.schema.json")
	errors := ValidateSchema(map[string]any{"id": "BAD ID!", "state": "UNKNOWN"}, schema, registry)
	if len(errors) == 0 {
		t.Fatalf("expected invalid goal to fail")
	}
}

func TestValidateWithNestedRef(t *testing.T) {
	dir := schemaDir(t)
	registry, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	schema, _ := registry.Schema("plan.schema.json")
	errors := ValidateSchema(map[string]any{}, schema, registry)
	if len(errors) == 0 {
		t.Fatalf("expected empty plan to fail")
	}
}

// A schema-valued additionalProperties is the only way JSON Schema can say
// "every value in this map must look like this". It used to be read as a bool,
// found not to be one, and skipped — so a contract written that way passed every
// instance it was given. A constraint that looks written and enforces nothing is
// worse than one never written, because it is read as coverage (GAP-151).

func TestASchemaValuedAdditionalPropertiesConstrainsEveryValue(t *testing.T) {
	schema := mustSchema(t, `{
		"type": "object",
		"properties": {
			"roles": {
				"type": "object",
				"additionalProperties": {
					"oneOf": [
						{"type": "string", "minLength": 1},
						{"type": "object", "properties": {"preferred": {"type": "array", "items": {"type": "string"}}}}
					]
				}
			}
		}
	}`)
	if got := ValidateSchema(mustInstance(t, `{"roles":{"architect":42}}`), schema, Registry{}); len(got) == 0 {
		t.Fatal("a numeric role value must not satisfy a oneOf that admits only a string or an object")
	}
}

func TestAValueThatSatisfiesTheConstraintPasses(t *testing.T) {
	schema := mustSchema(t, `{
		"type": "object",
		"properties": {
			"roles": {
				"type": "object",
				"additionalProperties": {"oneOf": [{"type": "string"}, {"type": "object"}]}
			}
		}
	}`)
	for _, instance := range []string{
		`{"roles":{"architect":"fast"}}`,
		`{"roles":{"architect":{"preferred":["m1"]}}}`,
		`{"roles":{"a":"x","b":"y","c":{"preferred":[]}}}`,
	} {
		if got := ValidateSchema(mustInstance(t, instance), schema, Registry{}); len(got) != 0 {
			t.Errorf("%s must pass, got %v", instance, got)
		}
	}
}

func TestTheConstraintIsNotSkippedWhenNoNamedPropertiesAreDeclared(t *testing.T) {
	// The handling used to live inside the named-properties block, so a schema
	// with only additionalProperties — which is the normal way to constrain a
	// map's values — was never reached at all.
	schema := mustSchema(t, `{
		"type": "object",
		"additionalProperties": {"type": "string"}
	}`)
	if got := ValidateSchema(mustInstance(t, `{"anything":42}`), schema, Registry{}); len(got) == 0 {
		t.Fatal("a map with no named properties must still have its values constrained")
	}
}

func TestABooleanAdditionalPropertiesStillForbidsExtras(t *testing.T) {
	schema := mustSchema(t, `{
		"type": "object",
		"properties": {"known": {"type": "string"}},
		"additionalProperties": false
	}`)
	if got := ValidateSchema(mustInstance(t, `{"known":"x","extra":1}`), schema, Registry{}); len(got) == 0 {
		t.Fatal("additionalProperties:false must still reject an undeclared property")
	}
	if got := ValidateSchema(mustInstance(t, `{"known":"x"}`), schema, Registry{}); len(got) != 0 {
		t.Fatalf("a declared property must pass, got %v", got)
	}
}

func mustSchema(t *testing.T, raw string) map[string]any {
	t.Helper()
	var schema map[string]any
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func mustInstance(t *testing.T, raw string) any {
	t.Helper()
	var instance any
	if err := json.Unmarshal([]byte(raw), &instance); err != nil {
		t.Fatal(err)
	}
	return instance
}
