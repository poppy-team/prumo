package budget

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBudgetManager_HierarchyAndReviewReserve(t *testing.T) {
	mgr := NewManager(DefaultPricingRates())

	// Set a run envelope: Hard limit 100 cents (R$ 1.00), Review reserve 20 cents (R$ 0.20)
	// Effective implementation limit: 80 cents
	mgr.SetEnvelope("run-123", ScopeRun, ScopeGoal, 100, 70, 20)

	// Consume 50k input tokens during implementation phase:
	// Cost = 50 * 0.30 cents = 15.0 cents -> OK
	cost, err := mgr.RecordConsumption("run-123", false, 50000, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v (cost: %.2f)", err, cost)
	}

	// Try to consume 250k input tokens in implementation phase:
	// Cost = 250 * 0.30 = 75.0 cents + 15.0 = 90.0 cents
	// Exceeds implementation limit (80.0 cents) -> should fail with ErrReviewReserveProtected
	_, err = mgr.RecordConsumption("run-123", false, 250000, 0, 0)
	if err == nil || !errors.Is(err, ErrReviewReserveProtected) {
		t.Fatalf("expected ErrReviewReserveProtected, got: %v", err)
	}

	// But in review phase, dipping into the review reserve is allowed!
	_, err = mgr.RecordConsumption("run-123", true, 20000, 0, 0) // 20 * 0.30 = 6.0 cents -> total 21.0 < 100
	if err != nil {
		t.Fatalf("unexpected error during review phase: %v", err)
	}
}

func TestBudgetManager_HardStopBlockedBudget(t *testing.T) {
	mgr := NewManager(DefaultPricingRates())

	// Hard limit 50 cents, review reserve 0
	mgr.SetEnvelope("run-hard", ScopeRun, ScopeGoal, 50, 40, 0)

	// Consume 200k input tokens: 200 * 0.30 = 60.0 cents > 50 cents -> triggers ErrBlockedBudget
	_, err := mgr.RecordConsumption("run-hard", false, 200000, 0, 0)
	if err == nil || !errors.Is(err, ErrBlockedBudget) {
		t.Fatalf("expected ErrBlockedBudget, got: %v", err)
	}

	// Create blocked_budget checkpoint
	tmpDir, err := os.MkdirTemp("", "budget-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	chk, err := mgr.CreateBlockedCheckpoint(tmpDir, "run-hard", "task-99", "run-hard")
	if err != nil {
		t.Fatalf("failed to create blocked checkpoint: %v", err)
	}

	if chk.Status != "blocked_budget" {
		t.Errorf("expected status 'blocked_budget', got %s", chk.Status)
	}

	// Verify file was written to disk
	files, err := os.ReadDir(tmpDir)
	if err != nil || len(files) != 1 {
		t.Fatalf("expected 1 checkpoint file on disk, got: %v (len: %d)", err, len(files))
	}
	if filepath.Ext(files[0].Name()) != ".json" {
		t.Errorf("expected .json extension, got %s", files[0].Name())
	}
}
