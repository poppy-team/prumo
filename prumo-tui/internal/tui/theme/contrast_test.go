package theme

import (
	"image/color"
	"math"
	"sort"
	"testing"
)

// The evidence the contrast obligation asks for is measured, not intended: a
// token pair either reaches its ratio or it does not, and a theme that only
// claims to be readable is a theme nobody checked.
//
// The thresholds are the ones the accessibility contract cites: 4.5:1 for text
// against the surface it sits on, 3:1 for the lines and shapes that carry
// information without being text.

const (
	textFloor    = 4.5
	nonTextFloor = 3.0
)

// pair is one token against the surface it is drawn on.
type pair struct {
	name  string
	fore  color.Color
	back  color.Color
	floor float64
	// exempt states why a pair may fall below its floor and still be correct.
	// The contract allows an obligation to be waived with a rationale; what it
	// does not allow is a shortfall nobody wrote down.
	exempt string
}

func themePairs(t Theme) []pair {
	return []pair{
		{"text on surface", t.Text(), t.Background(), textFloor, ""},
		{"muted text on surface", t.TextMuted(), t.Background(), textFloor, ""},
		{"emphasized text on surface", t.TextEmphasized(), t.Background(), textFloor, ""},
		{"primary on surface", t.Primary(), t.Background(), textFloor, ""},
		{"accent on surface", t.Accent(), t.Background(), textFloor, ""},
		{"success on surface", t.Success(), t.Background(), textFloor, ""},
		{"warning on surface", t.Warning(), t.Background(), textFloor, ""},
		{"error on surface", t.Error(), t.Background(), textFloor, ""},
		{"focus border on surface", t.BorderFocused(), t.Background(), nonTextFloor, ""},
		{"normal border on surface", t.BorderNormal(), t.Background(), nonTextFloor,
			"the unfocused border separates rather than identifies. The focused/unfocused distinction is the border glyph — heavy against light, asserted by TestTheComposerShowsWhetherItHoldsTheKeyboard — and the state it marks is carried by the focused border, which reaches the floor in every theme. A decorative separation is what the non-text rule exempts."},
	}
}

// luminance is the WCAG relative luminance of a colour.
func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	channel := func(v uint32) float64 {
		s := float64(v) / 65535
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}

// contrast is the WCAG contrast ratio between two colours.
func contrast(fore, back color.Color) float64 {
	lighter, darker := luminance(fore), luminance(back)
	if lighter < darker {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

// A theme either reaches the measured floor for every pair it declares, or it
// carries a stated exception. Silence is what this test refuses: the obligation
// was previously met by a ratio nobody computed.
func TestEveryThemeMeasuresItsContrast(t *testing.T) {
	for _, name := range AvailableThemes() {
		theme := GetTheme(name)
		if theme == nil {
			t.Fatalf("theme %q is advertised and cannot be retrieved", name)
		}
		for _, p := range themePairs(theme) {
			got := contrast(p.fore, p.back)
			switch {
			case got >= p.floor:
			case p.exempt != "":
				t.Logf("theme %q: %s is %.2f:1, below %.1f:1 — exempt: %s", name, p.name, got, p.floor, p.exempt)
			default:
				t.Errorf("theme %q: %s is %.2f:1, below the %.1f:1 this contract asks for",
					name, p.name, got, p.floor)
			}
		}
	}
}

// The measurement is the evidence, so it is reported for every pair rather than
// only for the ones that fail: a number nobody can read is not evidence.
func TestContrastEvidenceIsReported(t *testing.T) {
	names := AvailableThemes()
	sort.Strings(names)
	for _, name := range names {
		theme := GetTheme(name)
		for _, p := range themePairs(theme) {
			t.Logf("%-12s %-28s %5.2f:1 (floor %.1f)", name, p.name, contrast(p.fore, p.back), p.floor)
		}
	}
}
