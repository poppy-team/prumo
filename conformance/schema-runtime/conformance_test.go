// Package schemaruntime enforces the schema/runtime conformance gate (W1) and
// the canonical-format policy (W2). It is the `docs-schema` CI gate.
package schemaruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/gauntlet"
	"github.com/raillen/prumo/internal/knowledge"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("%s is not valid JSON: %v", path, err)
	}
	return out
}

func dig(t *testing.T, doc map[string]any, path ...string) any {
	t.Helper()
	var cur any = doc
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("path %v: %q is not an object", path, key)
		}
		cur, ok = m[key]
		if !ok {
			t.Fatalf("path %v: missing key %q", path, key)
		}
	}
	return cur
}

func stringsOf(t *testing.T, v any, path string) []string {
	t.Helper()
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("%s is not an array", path)
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("%s contains a non-string", path)
		}
		out = append(out, s)
	}
	return out
}

// TestSchemasConform verifies every schema parses, declares its identity and
// contains no defective enum.
func TestSchemasConform(t *testing.T) {
	root := repoRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "schemas", "*.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 70 {
		t.Fatalf("expected the canonical schema set, found %d", len(files))
	}
	seen := map[string]string{}
	for _, path := range files {
		name := filepath.Base(path)
		doc := readJSON(t, path)

		id, ok := doc["$id"].(string)
		if !ok || id == "" {
			t.Errorf("%s: missing $id", name)
		} else {
			if got := filepath.Base(id); got != name {
				t.Errorf("%s: $id basename %q does not match the filename", name, got)
			}
			if owner, dup := seen[id]; dup {
				t.Errorf("%s: duplicate $id also declared by %s", name, owner)
			}
			seen[id] = name
		}

		if schema, ok := doc["$schema"].(string); ok && !acceptedDraft(schema) {
			t.Errorf("%s: unsupported $schema %q (accepted: 2020-12, draft-07)", name, schema)
		}

		if !hasShape(doc) {
			t.Errorf("%s: must declare type, $ref, oneOf, anyOf or allOf", name)
		}
		for _, f := range enumDefects(doc, name) {
			t.Error(f)
		}
	}
}

// acceptedDraft is the explicit allowlist of supported JSON Schema drafts.
func acceptedDraft(schema string) bool {
	return strings.Contains(schema, "2020-12") || strings.Contains(schema, "draft-07")
}

func hasShape(doc map[string]any) bool {
	for _, key := range []string{"type", "$ref", "oneOf", "anyOf", "allOf"} {
		if _, ok := doc[key]; ok {
			return true
		}
	}
	return false
}

// enumDefects walks a schema and reports empty or non-unique enums.
func enumDefects(node any, path string) []string {
	var out []string
	switch v := node.(type) {
	case map[string]any:
		if raw, ok := v["enum"]; ok {
			arr, ok := raw.([]any)
			if !ok || len(arr) == 0 {
				out = append(out, fmt.Sprintf("%s: enum must be a non-empty array", path))
			} else {
				seen := map[string]bool{}
				for _, item := range arr {
					key := fmt.Sprintf("%v", item)
					if seen[key] {
						out = append(out, fmt.Sprintf("%s: duplicate enum value %q", path, key))
					}
					seen[key] = true
				}
			}
		}
		for key, child := range v {
			out = append(out, enumDefects(child, path+"."+key)...)
		}
	case []any:
		for i, child := range v {
			out = append(out, enumDefects(child, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	sort.Strings(out)
	return out
}

// TestRequiredSchemaSetExists turns the wave plan into a gate: the canonical
// schemas promised by W3/W4/W5/W6/W7/W9/W10 must exist.
func TestRequiredSchemaSetExists(t *testing.T) {
	required := []string{
		"knowledge-unit", "knowledge-claim", "knowledge-manifest",
		"context-policy", "context-manifest",
		"ui-component-contract", "ui-state-matrix", "ui-interface-map", "design-token-set",
		"translation-record",
		"gauntlet-policy", "gauntlet-run",
		"documentation-spec", "documentation-plan", "documentation-unit",
		"documentation-brief", "documentation-surface", "documentation-projection",
		"documentation-quality-policy", "documentation-evidence-requirement",
		"documentation-manifest", "media-record", "agent-instruction-surface",
	}
	root := repoRoot(t)
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(root, "schemas", name+".schema.json")); err != nil {
			t.Errorf("missing canonical schema %s.schema.json", name)
		}
	}
}

// TestVocabularyMatchesSchemas is the enum↔Go conformance check: the Go
// vocabulary must equal the schema enum exactly (W1.2/W1.4).
func TestVocabularyMatchesSchemas(t *testing.T) {
	root := repoRoot(t)

	unit := readJSON(t, filepath.Join(root, "schemas", "knowledge-unit.schema.json"))
	assertEnumEquals(t, "knowledge-unit.kind", stringsOf(t, dig(t, unit, "properties", "kind", "enum"), "kind"), kindsOf())
	assertEnumEquals(t, "knowledge-unit.relationships.kind",
		stringsOf(t, dig(t, unit, "properties", "relationships", "items", "properties", "kind", "enum"), "relationships.kind"),
		relationshipsOf())
	assertEnumEquals(t, "knowledge-unit.visibility",
		stringsOf(t, dig(t, unit, "properties", "visibility", "enum"), "visibility"), visibilities())

	quality := readJSON(t, filepath.Join(root, "schemas", "documentation-quality-policy.schema.json"))
	assertEnumEquals(t, "documentation-quality-policy.dimensions",
		stringsOf(t, dig(t, quality, "properties", "dimensions", "items", "enum"), "dimensions"), gauntlet.Dimensions())
}

func assertEnumEquals(t *testing.T, label string, schema, goValues []string) {
	t.Helper()
	schemaCopy := append([]string{}, schema...)
	goCopy := append([]string{}, goValues...)
	sort.Strings(schemaCopy)
	sort.Strings(goCopy)
	if strings.Join(schemaCopy, "|") != strings.Join(goCopy, "|") {
		t.Errorf("%s drift:\n  schema: %v\n  go:     %v", label, schemaCopy, goCopy)
	}
}

func kindsOf() []string {
	var out []string
	for _, k := range knowledge.Kinds() {
		out = append(out, string(k))
	}
	return out
}

func relationshipsOf() []string {
	var out []string
	for _, r := range knowledge.Relationships() {
		out = append(out, string(r))
	}
	return out
}

func visibilities() []string {
	return []string{"public", "project", "internal", "restricted", "confidential"}
}

// TestContractRegistryConformance validates the documentation contract and
// profile registries and their referential integrity.
func TestContractRegistryConformance(t *testing.T) {
	root := repoRoot(t)

	var contracts []map[string]any
	loadArray(t, filepath.Join(root, "docs", "contracts", "builtin.json"), &contracts)
	if len(contracts) == 0 {
		t.Fatal("contract registry is empty")
	}
	required := []string{"id", "version", "role", "description", "required_knowledge", "blocking_questions", "update_triggers", "staleness"}
	ids := map[string]bool{}
	for _, c := range contracts {
		for _, key := range required {
			if _, ok := c[key]; !ok {
				t.Errorf("contract %v: missing required field %q", c["id"], key)
			}
		}
		id, _ := c["id"].(string)
		if id == "" {
			t.Error("contract without id")
			continue
		}
		if ids[id] {
			t.Errorf("duplicate contract id %q", id)
		}
		ids[id] = true
	}

	var profiles []map[string]any
	loadArray(t, filepath.Join(root, "docs", "profiles", "builtin.json"), &profiles)
	if len(profiles) == 0 {
		t.Fatal("profile registry is empty")
	}
	profileIDs := map[string]bool{}
	for _, p := range profiles {
		id, _ := p["id"].(string)
		profileIDs[id] = true
	}
	for _, p := range profiles {
		id, _ := p["id"].(string)
		for _, ref := range contractRefs(t, p) {
			if !ids[ref] {
				t.Errorf("profile %q references unknown contract %q", id, ref)
			}
		}
		if extends, ok := p["extends"].([]any); ok {
			for _, e := range extends {
				name, _ := e.(string)
				if !profileIDs[name] {
					t.Errorf("profile %q extends unknown profile %q", id, name)
				}
			}
		}
	}
	if !ids["tui.interaction"] || !ids["tui.accessibility"] {
		t.Error("TUI contracts (tui.interaction, tui.accessibility) must be registered")
	}
	if !profileIDs["tui"] {
		t.Error("the tui profile must be composable")
	}
}

func contractRefs(t *testing.T, profile map[string]any) []string {
	t.Helper()
	raw, ok := profile["contracts"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func loadArray(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("%s is not a valid JSON array: %v", path, err)
	}
}

// TestCanonicalPathsHaveNoYAML enforces the Markdown+JSON canonical policy
// (W2.5). Legacy read-compatibility fixtures are allowlisted explicitly.
func TestCanonicalPathsHaveNoYAML(t *testing.T) {
	root := repoRoot(t)
	canonicalRoots := []string{"docs", "schemas", ".prumo", filepath.Join("src", "prumo", "resources", "catalog")}
	allow := []string{
		filepath.Join("testdata", "brownfield"),                                        // migration fixtures
		filepath.Join("src", "prumo", "resources", "workforce", "skills", "lang-yaml"), // language skill examples
	}
	for _, rel := range canonicalRoots {
		base := filepath.Join(root, rel)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}
			repoRel, _ := filepath.Rel(root, path)
			for _, a := range allow {
				if strings.HasPrefix(repoRel, a) {
					return nil
				}
			}
			t.Errorf("canonical path %s contains YAML (%s): use Markdown or JSON", rel, repoRel)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestCoverageStatesMatchSchema keeps the readiness vocabulary aligned.
func TestCoverageStatesMatchSchema(t *testing.T) {
	states := []string{
		string(docengine.Missing), string(docengine.Partial),
		string(docengine.ImplementationReady), string(docengine.Verified),
		string(docengine.Stale), string(docengine.NotApplicable),
	}
	if len(states) != 6 {
		t.Fatalf("expected 6 coverage states, got %d", len(states))
	}
	for i := 1; i < len(states); i++ {
		if states[i-1] == states[i] {
			t.Fatalf("duplicate coverage state %q", states[i])
		}
	}
}
