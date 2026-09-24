package agent

import (
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/safepath"
)

// A generated run id was the constant "R-agent-1" in one place, so every run
// that did not name itself was the same run and the second overwrote the first's
// checkpoint, log and permission trail. In another it was a counter from zero
// that restarted at one on every boot, so today's daemon named a run exactly as
// last week's did. Both collide with records already on disk, which is the one
// thing a run id has to avoid (GAP-136).

func TestGeneratedRunIDsAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	for range 2000 {
		id := NewRunID("agent")
		if seen[id] {
			t.Fatalf("two generated ids collided: %s", id)
		}
		seen[id] = true
	}
}

func TestThePrefixDistinguishesProducers(t *testing.T) {
	// A client run and a daemon run must not be able to produce the same id,
	// because they write into the same store.
	for range 50 {
		if NewRunID("agent") == NewRunID("daemon") {
			t.Fatal("an agent run and a daemon run produced the same id")
		}
	}
}

func TestAGeneratedIDPassesTheIDValidator(t *testing.T) {
	// The daemon validates every run id before touching a path with it, so an
	// id the generator produces that the validator rejects is an id that cannot
	// be used.
	for range 50 {
		if err := safepath.ValidateID("run_id", NewRunID("daemon")); err != nil {
			t.Fatalf("the generator produces an id the validator refuses: %v", err)
		}
	}
}

func TestAGeneratedIDSaysWhenAndWhichProcess(t *testing.T) {
	// An operator reading a directory of runs should be able to tell when one
	// happened without opening it. The old ids said nothing at all.
	id := NewRunID("daemon")
	if !strings.HasPrefix(id, "R-daemon-") {
		t.Fatalf("the producer is not in the id: %s", id)
	}
	// YYYYMMDDTHHMMSS
	parts := strings.Split(id, "-")
	if len(parts) < 4 || len(parts[2]) != 15 {
		t.Fatalf("the id carries no sortable timestamp: %s", id)
	}
}
