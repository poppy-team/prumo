package tui

import "testing"

func TestPaletteRanksPrefixAboveSubsequence(t *testing.T) {
	p := NewPalette(commands())
	p.SetQuery("run")

	first, ok := p.Selected()
	if !ok {
		t.Fatal("query 'run' matched nothing")
	}
	if first.ID != CommandRunGoal {
		t.Fatalf("best match for 'run' is %q; a prefix match must beat a mid-word one", first.ID)
	}
}

func TestPaletteMatchesKeywordsWithoutShowingThem(t *testing.T) {
	p := NewPalette(commands())
	// "exit" is a keyword of Quit, not part of its title.
	p.SetQuery("exit")
	selected, ok := p.Selected()
	if !ok {
		t.Fatal("keyword 'exit' matched nothing")
	}
	if selected.ID != CommandQuit {
		t.Fatalf("'exit' selected %q, want %q", selected.ID, CommandQuit)
	}
}

func TestPaletteNavigationWraps(t *testing.T) {
	p := NewPalette(commands())
	count := len(p.Matches())
	if count == 0 {
		t.Fatal("empty palette")
	}
	p.Move(-1)
	if p.Cursor() != count-1 {
		t.Fatalf("moving up from the top should wrap to %d, got %d", count-1, p.Cursor())
	}
	p.Move(1)
	if p.Cursor() != 0 {
		t.Fatalf("moving down from the bottom should wrap to 0, got %d", p.Cursor())
	}
}

func TestPaletteEmptyQueryListsEverything(t *testing.T) {
	p := NewPalette(commands())
	if len(p.Matches()) != len(commands()) {
		t.Fatalf("empty query listed %d of %d commands", len(p.Matches()), len(commands()))
	}
}

func TestPaletteEmptyStateIsExplicit(t *testing.T) {
	p := NewPalette(commands())
	p.SetQuery("zzzzzz")
	if !p.Empty() {
		t.Fatal("a query matching nothing must report Empty")
	}
	if _, ok := p.Selected(); ok {
		t.Fatal("an empty palette must not report a selection")
	}
	// Backspacing out of the empty state must restore a selection, otherwise
	// the user is stuck with enter doing nothing.
	p.SetQuery("")
	if p.Empty() {
		t.Fatal("clearing the query should restore matches")
	}
	if _, ok := p.Selected(); !ok {
		t.Fatal("clearing the query should restore a selection")
	}
}

func TestPaletteBackspaceOnEmptyQueryIsSafe(t *testing.T) {
	p := NewPalette(commands())
	p.Backspace()
	if p.Query() != "" {
		t.Fatalf("query = %q, want empty", p.Query())
	}
}

func TestPaletteIsDeterministic(t *testing.T) {
	a, b := NewPalette(commands()), NewPalette(commands())
	for _, q := range []string{"", "r", "run", "sh", "e"} {
		a.SetQuery(q)
		b.SetQuery(q)
		am, bm := a.Matches(), b.Matches()
		if len(am) != len(bm) {
			t.Fatalf("query %q: %d vs %d matches", q, len(am), len(bm))
		}
		for i := range am {
			if am[i].Command.ID != bm[i].Command.ID || am[i].Score != bm[i].Score {
				t.Fatalf("query %q ranked differently at %d: %v vs %v", q, i, am[i], bm[i])
			}
		}
	}
}
