package layout

import (
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/tui/theme"
)

func TestPlaceOverlayFillsShortRowsWithThemeBackground(t *testing.T) {
	previous := theme.CurrentThemeName()
	if err := theme.SetTheme("prumo"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = theme.SetTheme(previous) })

	got := PlaceOverlay(1, 0, "X\nY", "012345\nabcdef", false)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 || strings.Contains(lines[1], "bcdef") {
		t.Fatalf("overlay left underlying cells visible: %q", got)
	}
	if width := len([]rune(lines[1])); width != 6 {
		t.Fatalf("overlay row width = %d, want 6: %q", width, lines[1])
	}
}
