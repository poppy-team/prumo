package knowledge

import (
	"path/filepath"
	"testing"
)

func TestAtlasRoundtripAndRecall(t *testing.T) {
	s := New()
	if err := Remember(s, "m1", "opencode abort endpoint", "POST abort returns true", "run:R", []string{"opencode"}); err != nil {
		t.Fatal(err)
	}
	if err := Remember(s, "m2", "unrelated cooking", "pasta", "run:R", nil); err != nil {
		t.Fatal(err)
	}
	got := Recall(s, "opencode abort", 5)
	if len(got) != 1 || got[0].ID != "m1" {
		t.Fatalf("recall failed: %+v", got)
	}
	path := filepath.Join(t.TempDir(), "atlas.json")
	if err := SaveAtlas(path, SnapshotAtlas("proj", s)); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadAtlas(path)
	if err != nil || len(loaded.Records) != 2 {
		t.Fatalf("atlas roundtrip failed: %+v %v", loaded, err)
	}
	if _, err := LoadAtlas(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal("missing atlas must yield empty, not error")
	}
}
