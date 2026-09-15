package contextv2

import (
	"strings"
	"testing"
)

const sampleDoc = "# Title\n\nintro line\n\n## Section A\n\nbody a\n\n## Section B\n\nbody b\n"

func TestParseLevelFallsBackWithoutInventingALevel(t *testing.T) {
	for _, in := range []string{"L0", "l1", " l2 ", "L4"} {
		if got := ParseLevel(in); !got.Valid() {
			t.Fatalf("ParseLevel(%q) produced an invalid level %q", in, got)
		}
	}
	if got := ParseLevel("L9"); got != DefaultLevel {
		t.Fatalf("an unknown level must fall back to %s, got %s", DefaultLevel, got)
	}
	if got := ParseLevel(""); got != DefaultLevel {
		t.Fatalf("an empty level must fall back to %s, got %s", DefaultLevel, got)
	}
}

func TestDisclosureIsMonotonicAndHonestAboutTruncation(t *testing.T) {
	previous := -1
	for _, level := range Levels() {
		content, full := level.Disclose(sampleDoc)
		size := len(content)
		if size < previous {
			t.Fatalf("level %s revealed less than the level below it (%d < %d)", level, size, previous)
		}
		previous = size
		switch level {
		case LevelPointer:
			if content != "" {
				t.Fatalf("L0 must reveal a pointer only, got %q", content)
			}
			if full {
				t.Fatal("L0 cannot claim to be complete")
			}
		case LevelFull:
			if content != sampleDoc || !full {
				t.Fatalf("L4 must reveal everything and say so: full=%v", full)
			}
		}
		// Completeness may only be claimed when the disclosed text is the whole
		// document. A short document makes L3 complete, and saying so is honest.
		if full && content != sampleDoc {
			t.Fatalf("level %s claimed completeness after disclosing a prefix", level)
		}
	}

	// On a document longer than the excerpt window, L3 must truncate and admit it.
	long := sampleDoc + strings.Repeat("paragraph line\n", excerptLines*2)
	excerpt, full := LevelExcerpt.Disclose(long)
	if full {
		t.Fatal("L3 must not claim completeness on a document longer than its window")
	}
	if len(excerpt) >= len(long) {
		t.Fatalf("L3 must truncate: %d >= %d", len(excerpt), len(long))
	}

	// L2 is an outline: headings survive, body text does not.
	outline, _ := LevelOutline.Disclose(sampleDoc)
	if !strings.Contains(outline, "## Section A") || strings.Contains(outline, "body a") {
		t.Fatalf("L2 must be an outline, got %q", outline)
	}
}

func TestDiscloseItemKeepsProducerEstimateWhenNothingIsRevealed(t *testing.T) {
	// A candidate with no content cannot be disclosed less than it already is,
	// so its producer's estimate must survive: otherwise the budget accounting
	// silently under-counts and the compiler packs far too much.
	opaque := Item{Ref: "file:big.go", TokenCost: 900}
	got := DiscloseItem(opaque, LevelSummary)
	if got.TokenCost != 900 {
		t.Fatalf("an item without content must keep its producer estimate, got %d", got.TokenCost)
	}
	if got.Truncated {
		t.Fatal("an item without content is not truncated by a level")
	}
	// At L0 everything is a pointer and pays for its reference alone.
	pointer := DiscloseItem(opaque, LevelPointer)
	if pointer.TokenCost >= 900 {
		t.Fatalf("L0 must reduce every item to its reference cost, got %d", pointer.TokenCost)
	}
}

func TestDiscloseItemRecountsCostFromRevealedText(t *testing.T) {
	withContent := Item{Ref: "docs/x.md", Content: sampleDoc, TokenCost: 9999}
	full := DiscloseItem(withContent, LevelFull)
	summary := DiscloseItem(withContent, LevelSummary)
	if summary.TokenCost >= full.TokenCost {
		t.Fatalf("a summary must cost less than the full item: %d >= %d", summary.TokenCost, full.TokenCost)
	}
	if !summary.Truncated || full.Truncated {
		t.Fatalf("truncation must be reported: summary=%v full=%v", summary.Truncated, full.Truncated)
	}
}
