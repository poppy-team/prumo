package knowledge

import (
	"testing"
	"time"
)

func TestRecallFreshnessFloor(t *testing.T) {
	s := New()
	old := Record{ID: "old", Kind: KindMemory, Title: "ancient gateway lore", Status: "active",
		Authority: "reference", Trust: "medium", UpdatedAt: time.Now().Add(-400 * 24 * time.Hour).Format(time.RFC3339Nano)}
	// Restore preserves stored timestamps (as Load does); Put stamps now.
	s.Restore([]Record{old}, nil)
	if got := Recall(s, "gateway lore", 5); len(got) != 1 {
		t.Fatalf("no floor must recall: %+v", got)
	}
	if got := RecallSince(s, "gateway lore", 5, time.Now().Add(-30*24*time.Hour)); len(got) != 0 {
		t.Fatalf("stale must not surface: %+v", got)
	}
	if got := RecallSince(s, "gateway lore", 5, time.Now().Add(-500*24*time.Hour)); len(got) != 1 {
		t.Fatalf("wide floor must recall: %+v", got)
	}
}
