package resolver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The resolver now reads four fields it used to ignore. If the catalogue's own
// graph is not closed — a skill requiring something that does not exist — every
// project that selects it silently loses it (GAP-146).

func TestTheRealCatalogueIsSelfConsistent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "src", "prumo", "resources", "catalog", "skills.json"))
	if err != nil {
		t.Skipf("catalogue not available: %v", err)
	}
	var doc struct {
		Skills []map[string]any `json:"skills"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		// The file is a bare list in some revisions.
		if err := json.Unmarshal(raw, &doc.Skills); err != nil {
			t.Skipf("catalogue shape not understood: %v", err)
		}
	}
	if len(doc.Skills) == 0 {
		t.Skip("catalogue is empty in this checkout")
	}
	ids := map[string]bool{}
	for _, skill := range doc.Skills {
		if id, _ := skill["id"].(string); id != "" {
			ids[id] = true
		}
	}
	for _, skill := range doc.Skills {
		id, _ := skill["id"].(string)
		for _, need := range stringList(skill["requires"]) {
			if !ids[need] {
				t.Errorf("skill %q requires %q, which the catalogue does not define", id, need)
			}
		}
		for _, need := range stringList(skill["conflicts"]) {
			if !ids[need] {
				t.Errorf("skill %q conflicts with %q, which the catalogue does not define", id, need)
			}
		}
		for _, raw := range anyList(skill["requires_any"]) {
			for _, need := range stringList(raw) {
				if !ids[need] {
					t.Errorf("skill %q requires_any %q, which the catalogue does not define", id, need)
				}
			}
		}
	}
}
