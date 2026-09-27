package runlayer

import (
	"os"
	"path/filepath"
	"testing"
)

// The envelope was written on every run stop and never read back, so a run
// resumed after a restart began with a full budget. A ceiling that resets on
// restart is not a ceiling, and crashing is the cheapest way past one (GAP-001).

func TestARestoredBudgetCarriesWhatWasAlreadySpent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget-R1.json")

	spent := NewTracker(1_000, 10, 100)
	release, err := spent.Reserve(300)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	release(300, 4)
	if err := spent.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	resumed := NewTracker(1_000, 10, 100)
	if err := Load(path, resumed); err != nil {
		t.Fatalf("load: %v", err)
	}
	snapshot := resumed.Snapshot()
	if snapshot["tokens"] != 300 {
		t.Fatalf("the restored budget must show what was spent, got %v", snapshot)
	}
	if snapshot["cost_usd"] != 4 {
		t.Fatalf("the restored cost must show too, got %v", snapshot)
	}
	// And the ceiling must still bite: 300 of 1000 spent means 700 remain.
	if _, err := resumed.Reserve(701); err == nil {
		t.Fatal("a restored budget must not grant a fresh allowance")
	}
	if _, err := resumed.Reserve(700); err != nil {
		t.Fatalf("what is left must still be spendable: %v", err)
	}
}

func TestARestoredBudgetReportsExhaustionWhenItWasExhausted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget-R2.json")
	spent := NewTracker(100, 0, 0)
	release, _ := spent.Reserve(100)
	release(100, 0)
	if err := spent.Save(path); err != nil {
		t.Fatal(err)
	}

	resumed := NewTracker(100, 0, 0)
	if err := Load(path, resumed); err != nil {
		t.Fatal(err)
	}
	if err := resumed.Exhausted(); err == nil {
		t.Fatal("a budget that was exhausted must stay exhausted across a restart")
	}
}

func TestAMissingBudgetIsNotAnError(t *testing.T) {
	// A run that never had a tracker has nothing to restore. That is normal.
	tracker := NewTracker(100, 0, 0)
	if err := Load(filepath.Join(t.TempDir(), "absent.json"), tracker); err != nil {
		t.Fatalf("a missing budget must not fail: %v", err)
	}
	if err := tracker.Exhausted(); err != nil {
		t.Fatalf("the budget must be usable: %v", err)
	}
}

func TestACorruptBudgetIsRefusedRatherThanSilentlyReset(t *testing.T) {
	// Starting fresh on unreadable data is the same hole with an extra step.
	path := filepath.Join(t.TempDir(), "budget-R3.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	tracker := NewTracker(100, 0, 0)
	if err := Load(path, tracker); err == nil {
		t.Fatal("a corrupt budget must be reported, not ignored")
	}
}

func TestARestoredBudgetCarriesNoStaleReservation(t *testing.T) {
	// A held reservation belongs to a call that was in flight. Restoring it
	// would refuse calls that fit, with no call of ours outstanding.
	path := filepath.Join(t.TempDir(), "budget-R4.json")
	live := NewTracker(1_000, 0, 0)
	release, err := live.Reserve(900)
	if err != nil {
		t.Fatal(err)
	}
	defer release(0, 0)
	if err := live.Save(path); err != nil {
		t.Fatal(err)
	}

	resumed := NewTracker(1_000, 0, 0)
	if err := Load(path, resumed); err != nil {
		t.Fatal(err)
	}
	// The envelope on disk records usage, not reservations, so a full allowance
	// is available again — which is right: the in-flight call is gone.
	second, err := resumed.Reserve(1_000)
	if err != nil {
		t.Fatalf("a restored budget must not carry a stale hold: %v", err)
	}
	second(0, 0)
}

func TestToolCallsAreAlsoRestored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget-R5.json")
	spent := NewTracker(0, 0, 10)
	if err := spent.ConsumeTools(7); err != nil {
		t.Fatal(err)
	}
	if err := spent.Save(path); err != nil {
		t.Fatal(err)
	}
	resumed := NewTracker(0, 0, 10)
	if err := Load(path, resumed); err != nil {
		t.Fatal(err)
	}
	if got := resumed.Snapshot()["tool_calls"]; got != 7 {
		t.Fatalf("tool calls must be restored too, got %v", got)
	}
	if err := resumed.ConsumeTools(4); err == nil {
		t.Fatal("the restored tool ceiling must still bite")
	}
}
