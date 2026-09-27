package theme

import (
	"fmt"
	"image/color"
	"os"
	"sync"

	"charm.land/lipgloss/v2"
)

// AdaptiveColor is one colour per background mode.
//
// lipgloss v2 removed its own AdaptiveColor: a colour is an image/color.Color,
// and adaptivity is the caller's job (lipgloss.LightDark). The theme still has
// to say "this colour, per mode", so it says it here — and implements
// color.Color, which is the only thing every consumer hands to lipgloss.
// Resolution therefore happens where it did before: when the colour is read.
type AdaptiveColor struct {
	// Light and Dark hold the colour for each background mode. They accept the
	// forms the theme data is written in — a color.Color, or a "#rrggbb"
	// string from a palette package — and resolved in one place below, so no
	// theme has to know which form lipgloss wants.
	Light any
	Dark  any
}

// String returns the resolved colour as the text the theme declared it in: a
// hex literal when it wrote one, otherwise the colour's own "#rrggbb".
//
// Syntax highlighting and diff rendering need a literal rather than a colour,
// and asking the colour for it keeps the light/dark decision in one place.
func (c AdaptiveColor) String() string {
	raw := c.Light
	if darkBackground() {
		raw = c.Dark
	}
	if literal, ok := raw.(string); ok {
		return literal
	}
	resolved := toColor(raw)
	if resolved == nil {
		return ""
	}
	r, g, b, _ := resolved.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// RGBA resolves the colour for this terminal and returns its channels, which is
// what makes AdaptiveColor satisfiable anywhere a color.Color is.
func (c AdaptiveColor) RGBA() (uint32, uint32, uint32, uint32) {
	return c.resolve().RGBA()
}

func (c AdaptiveColor) resolve() color.Color {
	if darkBackground() {
		if resolved := toColor(c.Dark); resolved != nil {
			return resolved
		}
	}
	if resolved := toColor(c.Light); resolved != nil {
		return resolved
	}
	if resolved := toColor(c.Dark); resolved != nil {
		return resolved
	}
	return color.Black
}

// toColor normalises a theme colour to something lipgloss will accept.
func toColor(v any) color.Color {
	switch value := v.(type) {
	case nil:
		return nil
	case color.Color:
		return value
	case string:
		if value == "" {
			return nil
		}
		return lipgloss.Color(value)
	}
	return nil
}

var (
	darkOnce sync.Once
	darkMode bool
)

// darkBackground asks the terminal once and remembers the answer.
//
// v1 resolved adaptivity inside lipgloss on every render; v2 hands the decision
// to the caller, so the caller decides once. A client process has one terminal,
// and asking it on every colour lookup would be an escape sequence per style.
// DarkBackground reports whether the terminal has a dark background.
func DarkBackground() bool { return darkBackground() }

func darkBackground() bool {
	darkOnce.Do(func() {
		darkMode = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	})
	return darkMode
}
