package tui

import (
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/config"
)

// Reduced motion drops the movement and keeps the information the movement was
// carrying: a user who asked for no animation still has to be able to tell that
// the run is thinking.
func TestReducedMotionStatesProgressWithoutAnimatingIt(t *testing.T) {
	previous := config.Get().ReducedMotion
	t.Cleanup(func() {
		cfg := *config.Get()
		cfg.ReducedMotion = previous
		config.Set(cfg)
	})

	animated := frameWhileWorking(t, false)
	still := frameWhileWorking(t, true)

	verb := ""
	for _, candidate := range []string{"Thinking", "Waiting for tool response", "Building tool call", "Generating"} {
		if strings.Contains(animated, candidate) && strings.Contains(still, candidate) {
			verb = candidate
		}
	}
	if verb == "" {
		t.Fatalf("the run's state is what that line is for, and it went missing:\n%q\n%q", animated, still)
	}

	// The animation is the whole of the difference: every other line — the
	// transcript, the composer, the statusline — is the same frame.
	changed := differingLines(animated, still)
	if len(changed) != 1 || !strings.Contains(changed[0], verb) {
		t.Fatalf("reduced motion changed %d line(s) (%v), want only the one that animates", len(changed), changed)
	}
}

// differingLines returns the lines that are not identical in both frames.
func differingLines(a, b string) []string {
	first := strings.Split(a, "\n")
	second := strings.Split(b, "\n")
	var out []string
	for i := 0; i < len(first) || i < len(second); i++ {
		var x, y string
		if i < len(first) {
			x = first[i]
		}
		if i < len(second) {
			y = second[i]
		}
		if x != y {
			out = append(out, strings.TrimSpace(x)+" → "+strings.TrimSpace(y))
		}
	}
	return out
}

// frameWhileWorking is the transcript while a run is in flight.
func frameWhileWorking(t *testing.T, reduced bool) string {
	t.Helper()
	cfg := *config.Get()
	cfg.ReducedMotion = reduced
	config.Set(cfg)
	return plain(streamingShell(t).View().Content)
}
