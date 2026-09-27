package runlayer

import (
	"sync"
	"testing"
)

// NewTracker(0, 0, 0) means "unlimited" in the envelope, and that is exactly
// what the daemon built for every run. A run started through the daemon had no
// ceiling on tokens, cost or tool calls until the provider declined (GAP-098).

func TestZeroLimitsMeanUnlimitedAndThatIsTheTrap(t *testing.T) {
	tracker := NewTracker(0, 0, 0)
	release, err := tracker.Reserve(1_000_000_000)
	if err != nil {
		t.Fatalf("a zero limit is documented as unlimited, so this must be allowed: %v", err)
	}
	release(1_000_000_000, 0)
	if err := tracker.Exhausted(); err != nil {
		t.Fatalf("an unlimited tracker must never report exhaustion: %v", err)
	}
	// This is why the daemon must never build one. Documented here so the
	// behaviour is a stated contract rather than a surprise.
}

func TestAReservationLargerThanTheLimitIsRefused(t *testing.T) {
	tracker := NewTracker(1_000, 0, 0)
	if _, err := tracker.Reserve(1_001); err == nil {
		t.Fatal("a reservation past the limit must be refused before the call")
	}
}

func TestReservationsAccumulateAgainstTheSameLimit(t *testing.T) {
	// Two calls each within the limit, together past it. Without accounting for
	// what is held, both would be allowed and the run would land over.
	tracker := NewTracker(1_000, 0, 0)
	first, err := tracker.Reserve(600)
	if err != nil {
		t.Fatalf("first reservation: %v", err)
	}
	defer first(600, 0)
	if _, err := tracker.Reserve(600); err == nil {
		t.Fatal("the second reservation must see the first one still held")
	}
	if _, err := tracker.Reserve(300); err != nil {
		t.Fatalf("a reservation that fits alongside the first must be allowed: %v", err)
	}
}

func TestReleasingAReservationIsIdempotent(t *testing.T) {
	// The runtime settles a call from two places: the usage event and the end of
	// the stream. A double release would return the same allowance twice.
	tracker := NewTracker(1_000, 0, 0)
	release, err := tracker.Reserve(900)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	release(100, 0)
	release(100, 0)
	if err := tracker.Exhausted(); err != nil {
		t.Fatalf("a double release must not distort the accounting: %v", err)
	}
}

func TestSettlingAgainstRealUsageMovesTheLimitForward(t *testing.T) {
	tracker := NewTracker(1_000, 0, 0)
	release, err := tracker.Reserve(400)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	release(300, 0)
	// 300 spent, so 700 remain.
	second, err := tracker.Reserve(700)
	if err != nil {
		t.Fatalf("the settled cost must be what the next call sees: %v", err)
	}
	second(700, 0)
	if err := tracker.Exhausted(); err == nil {
		t.Fatal("the run is at its limit and must report it")
	}
}

func TestCostIsSettledAlongsideTokens(t *testing.T) {
	tracker := NewTracker(0, 10, 0)
	release, err := tracker.Reserve(1_000)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	release(500, 4)
	if err := tracker.Exhausted(); err != nil {
		t.Fatalf("half the cost spent must leave headroom: %v", err)
	}
	release2, err := tracker.Reserve(1_000)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	release2(100, 7) // 4 + 7 over a limit of 10
	if err := tracker.Exhausted(); err == nil {
		t.Fatal("the cost ceiling must bind, not only the token one")
	}
}

func TestExhaustedCountsWhatIsHeld(t *testing.T) {
	tracker := NewTracker(1_000, 0, 0)
	release, err := tracker.Reserve(900)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	defer release(900, 0)
	// 100 tokens of headroom remain, so this is not exhausted — but a call the
	// size the runtime actually reserves cannot fit, which is what stops it.
	if err := tracker.Exhausted(); err != nil {
		t.Fatalf("100 tokens of headroom remain: %v", err)
	}
	if _, err := tracker.Reserve(1_000); err == nil {
		t.Fatal("a call that does not fit alongside what is held must be refused")
	}
}

func TestExhaustedAtExactlyTheLimit(t *testing.T) {
	// At the boundary there is no free turn. A preflight that reported
	// headroom here would invite the call that crosses it.
	tracker := NewTracker(1_000, 0, 0)
	release, err := tracker.Reserve(1_000)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	release(1_000, 0)
	if err := tracker.Exhausted(); err == nil {
		t.Fatal("a run sitting exactly on its limit must report exhausted")
	}
}

func TestConcurrentReservationsDoNotOverCommit(t *testing.T) {
	tracker := NewTracker(1_000, 0, 0)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := tracker.Reserve(100)
			if err != nil {
				return
			}
			mu.Lock()
			allowed++
			mu.Unlock()
			release(100, 0)
		}()
	}
	wg.Wait()
	if allowed == 0 {
		t.Fatal("at least one reservation must fit")
	}
	if allowed > 10 {
		t.Fatalf("%d reservations of 100 against a limit of 1000 over-committed", allowed)
	}
}

func TestLimitsCopyReportsTheConfiguredCeiling(t *testing.T) {
	tracker := NewTracker(1_000, 2.5, 50)
	limits := tracker.LimitsCopy()
	if limits["tokens"] != 1_000 || limits["cost_usd"] != 2.5 || limits["tool_calls"] != 50 {
		t.Fatalf("the configured ceiling must be readable: %v", limits)
	}
	// The copy is a copy: a caller must not be able to raise the limit.
	limits["tokens"] = 9_999_999
	if tracker.LimitsCopy()["tokens"] != 1_000 {
		t.Fatal("LimitsCopy must not expose the tracker's own map")
	}
}
