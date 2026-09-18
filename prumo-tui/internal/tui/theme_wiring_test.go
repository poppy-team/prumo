package tui

import (
	"testing"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/tui/theme"
)

// The configured palette has to be the one the interface is built with. It was
// not: nothing read the setting, so `--theme` and the choice a user made last
// session were both accepted and ignored.
func TestTheConfiguredThemeIsTheOneDrawn(t *testing.T) {
	previous := theme.CurrentThemeName()
	t.Cleanup(func() { _ = theme.SetTheme(previous) })

	config.Set(config.Config{Theme: "gruvbox", Provider: "fake", MaxTurns: 5})
	New(app.New(app.Options{Client: &stubHarness{}}))

	if got := theme.CurrentThemeName(); got != "gruvbox" {
		t.Fatalf("the interface was built with %q, want the configured palette", got)
	}
}

// A palette the client does not know is reported rather than silently ignored,
// and the one in use is kept: a typo in a flag must not leave the terminal
// unstyled.
func TestAnUnknownThemeKeepsTheCurrentOne(t *testing.T) {
	previous := theme.CurrentThemeName()
	t.Cleanup(func() { _ = theme.SetTheme(previous) })
	if err := theme.SetTheme("prumo"); err != nil {
		t.Fatalf("cannot select the default palette: %v", err)
	}

	config.Set(config.Config{Theme: "no-such-palette", Provider: "fake", MaxTurns: 5})
	model := New(app.New(app.Options{Client: &stubHarness{}}))

	if got := theme.CurrentThemeName(); got != "prumo" {
		t.Fatalf("an unknown palette replaced the working one with %q", got)
	}
	// The client still draws, which is the point of keeping the previous one.
	if frame := model.View().Content; frame == "" {
		t.Fatal("the interface drew nothing after an unknown palette")
	}
}
