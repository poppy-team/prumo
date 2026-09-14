package contextv2

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/knowledge"
)

func TestWorkspaceConsultsAtlas(t *testing.T) {
	dir := t.TempDir()
	s := knowledge.New()
	if err := knowledge.Remember(s, "m1", "flaky gateway retry", "retry thrice", "run:R", nil); err != nil {
		t.Fatal(err)
	}
	if err := knowledge.SaveAtlas(knowledge.AtlasPath(dir), knowledge.SnapshotAtlas("p", s)); err != nil {
		t.Fatal(err)
	}
	m := CompileWorkspace("R-atlas", "gateway retry policy", dir, 8000, "")
	found := false
	for _, it := range m.Included {
		if it.Ref == "memory:m1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("atlas memory must pack: %+v", m.Included)
	}
}
