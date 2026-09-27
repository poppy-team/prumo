//go:build ignore

package tui

import (
	"strings"
	"testing"

	"github.com/raillen/prumo/tui/theme"
)

func TestEveryContractualTokenResolves(t *testing.T) {
	for _, id := range theme.Names() {
		styles, err := NewStyles(id)
		if err != nil {
			t.Fatalf("theme %s does not resolve: %v", id, err)
		}
		palette, err := theme.Resolve(id)
		if err != nil {
			t.Fatalf("theme %s does not resolve: %v", id, err)
		}
		for _, token := range theme.ContractualTokens() {
			if _, ok := palette.Value(token); !ok {
				t.Errorf("theme %s is missing contractual token %s", styles.ThemeID(), token)
			}
		}
	}
}

func TestNoColorThemePaintsNoColour(t *testing.T) {
	styles, err := NewStyles(theme.NoColorTheme)
	if err != nil {
		t.Fatalf("no-color theme does not resolve: %v", err)
	}
	if !styles.NoColor() {
		t.Fatal("theme.no-color must report NoColor")
	}
	// Emphasis (bold, borders) is deliberately *kept* in this theme: with no
	// colour left, it is the only readability signal remaining. What must
	// disappear is every colour token.
	if painted := styles.PaintedTokens(); len(painted) != 0 {
		t.Fatalf("no-color theme paints %d colour tokens: %v", len(painted), painted)
	}
	// The literal string "none" must never leak into the view — that is the
	// failure mode of applying an unresolved value.
	if rendered := styles.Highlight("Run goal", "run"); rendered != "Run goal" {
		t.Fatalf("no-color highlight changed the text: %q", rendered)
	}
}

func TestDefaultThemePaintsContractualColour(t *testing.T) {
	styles, err := NewStyles(theme.DefaultTheme)
	if err != nil {
		t.Fatalf("default theme does not resolve: %v", err)
	}
	painted := styles.PaintedTokens()
	if len(painted) != len(theme.ContractualTokens()) {
		t.Fatalf("default theme painted %d of %d contractual tokens: %v",
			len(painted), len(theme.ContractualTokens()), painted)
	}
}

func TestHighlightNeverCorruptsTheText(t *testing.T) {
	for _, id := range theme.Names() {
		styles, err := NewStyles(id)
		if err != nil {
			t.Fatalf("theme %s: %v", id, err)
		}
		// Colour emission depends on the terminal profile, so the invariant
		// worth asserting is the one that holds everywhere: whatever the
		// profile, the visible characters are never altered.
		if got := stripANSI(styles.Highlight("Run goal", "Run")); got != "Run goal" {
			t.Fatalf("theme %s: highlighting %q corrupted the text: %q", id, "Run goal", got)
		}
	}
}

// stripANSI removes escape sequences so a test can compare visible text without
// depending on the colour profile of the machine running it.
func stripANSI(s string) string {
	var b strings.Builder
	escaping := false
	for _, r := range s {
		switch {
		case escaping:
			if r == 'm' {
				escaping = false
			} else if r == 0x1b {
				escaping = true
			}
		case r == 0x1b:
			escaping = true
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestHighlightLeavesFuzzyMatchesAlone(t *testing.T) {
	styles, _ := NewStyles(theme.DefaultTheme)
	// "rg" matches "Run goal" as a subsequence but is not a contiguous span;
	// highlighting an invented span would misreport why the row matched.
	if got := styles.Highlight("Run goal", "rg"); got != "Run goal" {
		t.Fatalf("fuzzy match was rewritten: %q", got)
	}
}

func TestContrastEvidenceExistsForTextAndFocus(t *testing.T) {
	palette, err := theme.Resolve(theme.DefaultTheme)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for _, token := range []string{theme.Text, theme.TextMuted, theme.Focus} {
		against, ratio, wcag, ok := palette.Contrast(token)
		if !ok {
			t.Errorf("token %s carries no contrast evidence", token)
			continue
		}
		if against == "" || ratio <= 1 || wcag == "" {
			t.Errorf("token %s has incomplete evidence: against=%q ratio=%v wcag=%q", token, against, ratio, wcag)
		}
	}
}
