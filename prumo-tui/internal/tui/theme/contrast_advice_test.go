package theme

import (
	"flag"
	"fmt"
	"image/color"
	"sort"
	"testing"
)

// contrastAdvice makes the test print the value each failing token would need,
// instead of only reporting that it failed.
//
// It exists because the alternative is a hand-picked colour per theme per mode:
// forty numbers nobody can reproduce or review. The advice is computed from the
// theme's own foreground, so the replacement is that palette's colour carried to
// the floor rather than a new one.
var contrastAdvice = flag.Bool("contrast-advice", false, "print the value each failing token would need")

// adviceFloor is the floor plus a margin, so a value that lands exactly on the
// boundary does not fail on the next rounding.
const adviceFloor = 4.6

// tokenPairs are the tokens measured against the surface they are drawn on.
func tokenPairs(t Theme) []struct {
	token string
	fg    AdaptiveColor
} {
	return []struct {
		token string
		fg    AdaptiveColor
	}{
		{"TextMutedColor", t.TextMuted()},
		{"TextEmphasizedColor", t.TextEmphasized()},
		{"PrimaryColor", t.Primary()},
		{"SecondaryColor", t.Secondary()},
		{"AccentColor", t.Accent()},
		{"ErrorColor", t.Error()},
		{"WarningColor", t.Warning()},
		{"SuccessColor", t.Success()},
		{"InfoColor", t.Info()},
	}
}

func TestContrastAdvice(t *testing.T) {
	if !*contrastAdvice {
		t.Skip("run with -contrast-advice to print the values each failing token would need")
	}
	names := AvailableThemes()
	sort.Strings(names)

	for _, name := range names {
		theme := GetTheme(name)
		for _, pair := range tokenPairs(theme) {
			recommendations := advice(pair.fg, theme.Text(), theme.Background())
			if recommendations[0] == "" && recommendations[1] == "" {
				continue
			}
			fmt.Printf("ADVICE|%s|%s|%s|%s\n", name, pair.token, recommendations[0], recommendations[1])
		}
	}
}

// advice returns, per mode, the hex a colour needs to reach the floor, or an
// empty string when it already does.
func advice(fg, toward, back AdaptiveColor) [2]string {
	var out [2]string
	for which, dark := range map[int]bool{0: true, 1: false} {
		value, target, surface := modeColor(fg, dark), modeColor(toward, dark), modeColor(back, dark)
		if value == nil || target == nil || surface == nil {
			continue
		}
		if contrast(value, surface) >= adviceFloor {
			continue
		}
		for step := 1; step <= 100; step++ {
			candidate := mix(value, target, float64(step)/100)
			if contrast(candidate, surface) >= adviceFloor {
				out[which] = hexOf(candidate)
				break
			}
		}
	}
	return out
}

// modeColor reads one mode of an adaptive colour, falling back to the other so a
// token defined only once is still measured.
func modeColor(c AdaptiveColor, dark bool) color.Color {
	if dark {
		if value := toColor(c.Dark); value != nil {
			return value
		}
	}
	if value := toColor(c.Light); value != nil {
		return value
	}
	return toColor(c.Dark)
}

func mix(from, toward color.Color, t float64) color.Color {
	fr, fg, fb, _ := from.RGBA()
	tr, tg, tb, _ := toward.RGBA()
	mixed := func(a, b uint32) uint8 {
		return uint8(float64(a>>8)*(1-t) + float64(b>>8)*t)
	}
	return color.RGBA{mixed(fr, tr), mixed(fg, tg), mixed(fb, tb), 255}
}

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
