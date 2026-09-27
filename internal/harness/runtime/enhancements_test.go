package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// Enhancement 1: Real-Time Budget Tracking (budget.tick & cumulative stats)
func TestBudgetTickAndCumulativeTracking(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{
		"*": {
			{Kind: "usage", Usage: &agent.Usage{
				InputTokens:      1000,
				OutputTokens:     200,
				CacheReadTokens:  500,
				CacheWriteTokens: 100,
				CostUSD:          0.015,
			}},
			{Kind: "complete"},
		},
	})

	var emittedEvents []agent.AgentEvent
	dir := t.TempDir()
	store := checkpoint.New(dir)

	runner := NewRunner(Services{
		Models:      fake,
		Tools:       &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
		Events: func(ev agent.AgentEvent) {
			emittedEvents = append(emittedEvents, ev)
		},
	}, "R-budget-tick", "S1")

	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hello"}}
	if err := runner.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Verify cumulative counters on Runner
	if runner.PromptTokens != 1000 {
		t.Errorf("expected 1000 prompt tokens, got %d", runner.PromptTokens)
	}
	if runner.CompletionTokens != 200 {
		t.Errorf("expected 200 completion tokens, got %d", runner.CompletionTokens)
	}
	if runner.CacheReadTokens != 500 {
		t.Errorf("expected 500 cache read tokens, got %d", runner.CacheReadTokens)
	}
	if runner.TotalTokens != 1800 {
		t.Errorf("expected 1800 total tokens, got %d", runner.TotalTokens)
	}
	if runner.TotalCostUSD != 0.015 {
		t.Errorf("expected 0.015 cost USD, got %f", runner.TotalCostUSD)
	}

	// Verify State.Budget
	if runner.State.Budget == nil {
		t.Fatal("expected State.Budget to be populated")
	}
	if runner.State.Budget["total_tokens"] != 1800 {
		t.Errorf("expected State.Budget total_tokens = 1800, got %v", runner.State.Budget["total_tokens"])
	}
	if runner.State.Budget["cost_usd"] != 0.015 {
		t.Errorf("expected State.Budget cost_usd = 0.015, got %v", runner.State.Budget["cost_usd"])
	}

	// Verify budget.tick event was emitted
	foundTick := false
	for _, ev := range emittedEvents {
		if ev.Kind == "budget.tick" {
			foundTick = true
			if ev.Payload["total_tokens"] != 1800 {
				t.Errorf("budget.tick payload total_tokens want 1800, got %v", ev.Payload["total_tokens"])
			}
			if ev.Payload["total_cost_usd"] != 0.015 {
				t.Errorf("budget.tick payload total_cost_usd want 0.015, got %v", ev.Payload["total_cost_usd"])
			}
		}
	}
	if !foundTick {
		t.Fatal("budget.tick event was not emitted")
	}

	// Verify checkpoint carries and restores budget
	latest, err := store.Latest("R-budget-tick")
	if err != nil {
		t.Fatalf("store.Latest failed: %v", err)
	}
	resumedRunner := NewRunner(Services{}, "R-budget-tick", "S1")
	resumedRunner.RestoreFrom(latest)
	if resumedRunner.PromptTokens != 1000 || resumedRunner.TotalTokens != 1800 || resumedRunner.TotalCostUSD != 0.015 {
		t.Errorf("RestoreFrom failed to restore budget counters: %+v", resumedRunner)
	}
}

// Enhancement 2: Micro-Compacting at 70% Context Window
func TestMicroCompactObservationsAtContextWindowRatio(t *testing.T) {
	var events []agent.AgentEvent
	runner := NewRunner(Services{
		Events: func(ev agent.AgentEvent) {
			events = append(events, ev)
		},
	}, "R-micro-compact", "S1")

	// Set context window to 600 tokens with 70% threshold (420 tokens)
	runner.ContextWindow = 600
	runner.CompactThresholdRatio = 0.70

	// Add system prompt and user prompt
	runner.Messages = []agent.Message{
		{ID: "sys", Role: agent.RoleSystem, Content: "You are a compiler engineer."},
		{ID: "usr", Role: agent.RoleUser, Content: "Optimize this IR."},
	}

	// Add historical tool observations (verbose)
	largeOutput := strings.Repeat("0x7fff deadbeef call function_trampoline\n", 30) // ~1200 chars (~300 tokens each)
	runner.Messages = append(runner.Messages, agent.Message{
		ID:      "tool-1",
		Role:    agent.RoleTool,
		Content: largeOutput,
	})
	runner.Messages = append(runner.Messages, agent.Message{
		ID:      "tool-2",
		Role:    agent.RoleTool,
		Content: largeOutput,
	})

	// Add recent turn messages (which should be protected by the active turn window)
	runner.Messages = append(runner.Messages,
		agent.Message{ID: "req-active", Role: agent.RoleAgent, Content: "Checking latest register allocation"},
		agent.Message{ID: "tool-active-1", Role: agent.RoleTool, Content: "Reg RAX allocated"},
		agent.Message{ID: "tool-active-2", Role: agent.RoleTool, Content: "Reg RBX allocated"},
		agent.Message{ID: "active-resp", Role: agent.RoleAgent, Content: "Done with registers."},
	)

	runner.mu.Lock()
	runner.maybeCompactLocked()
	runner.mu.Unlock()

	// Tool-1 and Tool-2 should be micro-compacted
	m1 := runner.Messages[2]
	if m1.ID != "tool-1" {
		t.Fatalf("expected tool-1 at index 2, got %s", m1.ID)
	}
	if !strings.HasPrefix(m1.Content, "[Observation micro-compacted:") {
		t.Fatalf("tool-1 should have been micro-compacted, got content: %s", m1.Content)
	}
	if m1.Metadata == nil || m1.Metadata["compacted"] != true {
		t.Fatalf("tool-1 metadata['compacted'] must be true")
	}
	if m1.Metadata["sha256"] == nil || m1.Metadata["sha256"] == "" {
		t.Fatalf("tool-1 metadata['sha256'] must be populated")
	}

	// Active turn messages must NOT be compacted
	activeTool := runner.Messages[len(runner.Messages)-3]
	if activeTool.ID != "tool-active-1" || strings.HasPrefix(activeTool.Content, "[Observation micro-compacted:") {
		t.Fatalf("active tool should not be compacted: %+v", activeTool)
	}

	// Verify context.micro_compacted event was emitted
	foundMicroCompactEvent := false
	for _, ev := range events {
		if ev.Kind == "context.micro_compacted" {
			foundMicroCompactEvent = true
			if ev.Payload["message_id"] == "tool-1" {
				if ev.Payload["sha256"] == nil || ev.Payload["original_bytes"] == nil {
					t.Errorf("micro_compacted payload missing required fields: %v", ev.Payload)
				}
			}
		}
	}
	if !foundMicroCompactEvent {
		t.Fatal("context.micro_compacted event was not emitted")
	}
}

// Enhancement 3: Verification-Guided Repair Loop
func TestVerificationGuidedRepairLoopSuccess(t *testing.T) {
	// Model returns completed on every step
	fake := model.NewFake(map[string][]model.ScriptStep{
		"*": {
			{Kind: "text", Text: "Attempting solution"},
			{Kind: "complete"},
		},
	})

	var events []agent.AgentEvent
	dir := t.TempDir()
	store := checkpoint.New(dir)

	runner := NewRunner(Services{
		Models:      fake,
		Tools:       &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
		Events: func(ev agent.AgentEvent) {
			events = append(events, ev)
		},
	}, "R-repair-loop", "S1")

	// Gate fails twice, passes on 3rd attempt
	gateAttempts := 0
	runner.QualityGate = func() error {
		gateAttempts++
		if gateAttempts < 3 {
			return fmt.Errorf("compiler syntax error: unexpected semicolon at line %d", gateAttempts*10)
		}
		return nil
	}
	runner.MaxRepairAttempts = 3
	runner.MaxTurns = 10
	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "fix compiler bug"}}

	err := runner.RunUntilDone(context.Background())
	if err != nil {
		t.Fatalf("expected repair loop to succeed eventually, got error: %v", err)
	}
	if runner.State.Phase != agent.PhaseComplete {
		t.Fatalf("expected PhaseComplete, got %s (reason: %s)", runner.State.Phase, runner.State.StopReason)
	}
	if runner.RepairAttempts != 2 {
		t.Fatalf("expected 2 repair attempts, got %d", runner.RepairAttempts)
	}

	// Verify repair_attempt events
	repairEventCount := 0
	for _, ev := range events {
		if ev.Kind == "repair_attempt" {
			repairEventCount++
		}
	}
	if repairEventCount != 2 {
		t.Fatalf("expected 2 repair_attempt events, got %d", repairEventCount)
	}

	// Verify synthetic error was injected into conversation
	foundRepairMsg := false
	for _, m := range runner.Messages {
		if strings.HasPrefix(m.ID, "repair-") && strings.Contains(m.Content, "Verification gate failed") {
			foundRepairMsg = true
			break
		}
	}
	if !foundRepairMsg {
		t.Fatal("expected synthetic repair message in runner.Messages")
	}
}

func TestVerificationGuidedRepairLoopFailsClosedWhenExhausted(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{
		"*": {
			{Kind: "text", Text: "Attempting solution"},
			{Kind: "complete"},
		},
	})

	runner := NewRunner(Services{
		Models: fake,
		Tools:  &stubTools{},
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
	}, "R-repair-exhausted", "S1")

	// Gate always fails
	runner.QualityGate = func() error {
		return errors.New("sanitizer ASan memory leak detected")
	}
	runner.MaxRepairAttempts = 2
	runner.MaxTurns = 10
	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "fix memory leak"}}

	err := runner.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("expected failure after repair attempts exhausted")
	}
	if runner.State.Phase != agent.PhaseFailed {
		t.Fatalf("expected PhaseFailed, got %s", runner.State.Phase)
	}
	if runner.RepairAttempts != 2 {
		t.Fatalf("expected exactly 2 repair attempts, got %d", runner.RepairAttempts)
	}
}

// Enhancement 4: Atomic Pause and Safe-Yield on SIGINT / Context Cancellation
func TestAtomicPauseAndSafeYield(t *testing.T) {
	dir := t.TempDir()
	store := checkpoint.New(dir)

	var events []agent.AgentEvent
	runner := NewRunner(Services{
		Models:      completingFake(),
		Tools:       &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
		Events: func(ev agent.AgentEvent) {
			events = append(events, ev)
		},
	}, "R-pause", "S1")

	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "do work"}}

	// Test Pause method directly
	if err := runner.Pause("operator pressed Ctrl+C"); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	if runner.State.Phase != agent.PhaseYield {
		t.Fatalf("expected PhaseYield after pause, got %s", runner.State.Phase)
	}
	if runner.State.StopReason != "operator pressed Ctrl+C" {
		t.Fatalf("expected stop reason 'operator pressed Ctrl+C', got '%s'", runner.State.StopReason)
	}

	// Verify run_paused event
	foundPaused := false
	for _, ev := range events {
		if ev.Kind == "run_paused" && ev.Payload["reason"] == "operator pressed Ctrl+C" {
			foundPaused = true
		}
	}
	if !foundPaused {
		t.Fatal("expected run_paused event to be emitted")
	}

	// Verify checkpoint was saved
	cp, err := store.Latest("R-pause")
	if err != nil {
		t.Fatalf("checkpoint not saved on pause: %v", err)
	}
	if cp.State.Phase != agent.PhaseYield {
		t.Fatalf("checkpoint phase = %s, want %s", cp.State.Phase, agent.PhaseYield)
	}

	// Test RunUntilDone with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner2 := NewRunner(Services{
		Models:      completingFake(),
		Tools:       &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
	}, "R-pause-ctx", "S1")
	runner2.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "work"}}

	err = runner2.RunUntilDone(ctx)
	if err == nil {
		t.Fatal("expected context canceled error")
	}
	if runner2.State.Phase != agent.PhaseYield {
		t.Fatalf("expected runner to safe-yield on canceled context, got phase %s", runner2.State.Phase)
	}
	cp2, err := store.Latest("R-pause-ctx")
	if err != nil {
		t.Fatalf("checkpoint must be saved on context cancel: %v", err)
	}
	if cp2.State.Phase != agent.PhaseYield {
		t.Fatalf("checkpoint phase = %s, want %s", cp2.State.Phase, agent.PhaseYield)
	}
}
