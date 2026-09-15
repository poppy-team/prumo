package schemaruntime

import (
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/harness/contextv2"
)

func disclosureLevels() []string {
	var out []string
	for _, level := range contextv2.Levels() {
		out = append(out, string(level))
	}
	return out
}

// TestContextManifestVocabularyMatchesSchema is the W4 half of the W1 gate: the
// disclosure levels and exclusion reasons the compiler emits are exactly the
// values the manifest schema allows. A new reason cannot be invented in Go
// without updating the contract.
func TestContextManifestVocabularyMatchesSchema(t *testing.T) {
	root := repoRoot(t)
	manifest := readJSON(t, filepath.Join(root, "schemas", "context-manifest.schema.json"))

	assertEnumEquals(t, "context-manifest.level",
		stringsOf(t, dig(t, manifest, "properties", "level", "enum"), "level"),
		disclosureLevels())
	assertEnumEquals(t, "context-manifest.included.level",
		stringsOf(t, dig(t, manifest, "properties", "included", "items", "properties", "level", "enum"), "included.level"),
		disclosureLevels())
	assertEnumEquals(t, "context-manifest.excluded.reason",
		stringsOf(t, dig(t, manifest, "properties", "excluded", "items", "properties", "reason", "enum"), "excluded.reason"),
		contextv2.ExclusionReasons())
	// A manifest declares which schema versions it may carry, and the runtime
	// must not emit a version outside that set.
	assertContains(t, "context-manifest.version",
		dig(t, manifest, "properties", "version", "enum"), contextv2.ManifestSchemaVersion)
}

func assertContains(t *testing.T, label string, value any, want int) {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("%s is not an array", label)
	}
	for _, v := range values {
		if n, ok := v.(float64); ok && int(n) == want {
			return
		}
	}
	t.Errorf("%s does not allow version %d: %#v", label, want, values)
}
