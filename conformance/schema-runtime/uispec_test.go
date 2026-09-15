package schemaruntime

import (
	"path/filepath"
	"testing"

	docengine "github.com/raillen/prumo/internal/documentation"
)

// TestUIStateVocabularyMatchesSchema keeps the state matrix honest: the states
// the UI specification is checked against are exactly the states the schema
// allows. Adding a state in one place without the other makes the contract
// meaningless, so it fails here.
func TestUIStateVocabularyMatchesSchema(t *testing.T) {
	root := repoRoot(t)
	matrix := readJSON(t, filepath.Join(root, "schemas", "ui-state-matrix.schema.json"))
	assertEnumEquals(t, "ui-state-matrix.states.state",
		stringsOf(t, dig(t, matrix, "properties", "states", "items", "properties", "state", "enum"), "states.state"),
		docengine.UIStates())
}
