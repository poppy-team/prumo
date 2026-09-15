// Package gauntlet defines bounded quality-iteration policy. The default mode
// is off: no model critic runs unless a policy explicitly activates it, and a
// model critic never overrides a deterministic failure.
package gauntlet

import (
	"fmt"
	"sort"
)

// Mode controls activation. Global default is off (W7.6).
type Mode string

const (
	ModeOff   Mode = "off"
	ModeAuto  Mode = "auto"
	ModeForce Mode = "force"
)

// Dimensions are the quality axes a documentation gauntlet may evaluate.
func Dimensions() []string {
	return []string{
		"accessibility", "agent_retrieval_quality", "cross_document_consistency",
		"duplication", "example_verification", "factual_grounding",
		"freshness", "information_architecture", "localization",
		"reference_integrity", "semantic_coverage", "task_findability",
		"token_efficiency",
	}
}

// DropReason explains why an improvement attempt was discarded.
type StopReason string

const (
	StopGatesPassed   StopReason = "gates-passed"
	StopNoFindings    StopReason = "no-findings"
	StopMaxRounds     StopReason = "max-rounds"
	StopBudget        StopReason = "budget-exhausted"
	StopNoImprovement StopReason = "no-improvement"
	StopHuman         StopReason = "human-decision"
)

// Stopping is the deterministic stop contract.
type Stopping struct {
	GatesPassed         bool `json:"gates_passed"`
	NoCriticalFindings  bool `json:"no_critical_findings"`
	MaxRounds           int  `json:"max_rounds,omitempty"`
	BudgetTokens        int  `json:"budget_tokens,omitempty"`
	NoImprovementRounds int  `json:"no_improvement_rounds,omitempty"`
}

// Policy is the Gauntlet configuration.
type Policy struct {
	Mode                Mode     `json:"mode"`
	MaxRounds           int      `json:"max_rounds"`
	CriticIsolation     bool     `json:"critic_isolation"`
	Dimensions          []string `json:"dimensions,omitempty"`
	Stopping            Stopping `json:"stopping"`
	ScoreInflationGuard bool     `json:"score_inflation_guard"`
}

// DefaultPolicy returns the safe default: off, isolated critics, bounded rounds.
func DefaultPolicy() Policy {
	return Policy{
		Mode:                ModeOff,
		MaxRounds:           3,
		CriticIsolation:     true,
		ScoreInflationGuard: true,
		Stopping: Stopping{
			GatesPassed:        true,
			NoCriticalFindings: true,
		},
	}
}

// Validate enforces the policy invariants.
func (p Policy) Validate() error {
	switch p.Mode {
	case ModeOff, ModeAuto, ModeForce:
	default:
		return fmt.Errorf("gauntlet: unknown mode %q (want off|auto|force)", p.Mode)
	}
	if p.MaxRounds < 1 || p.MaxRounds > 10 {
		return fmt.Errorf("gauntlet: max_rounds %d out of range 1..10", p.MaxRounds)
	}
	if !p.CriticIsolation {
		return fmt.Errorf("gauntlet: critic_isolation must be true; critics never share the implementer context")
	}
	for _, d := range p.Dimensions {
		if !validDimension(d) {
			return fmt.Errorf("gauntlet: unknown dimension %q", d)
		}
	}
	// A policy may carry dimensions while off; it must not run. Activation is
	// controlled solely by Mode (see Active).
	return nil
}

func validDimension(d string) bool {
	for _, known := range Dimensions() {
		if known == d {
			return true
		}
	}
	return false
}

// ShouldStop applies the stopping conditions after a round. It reports the
// first satisfied condition; callers own budget accounting and human escalation.
func (p Policy) ShouldStop(round int, gatesPassed bool, criticalFindings int, improved bool) (StopReason, bool) {
	if gatesPassed && criticalFindings == 0 {
		return StopGatesPassed, true
	}
	if p.MaxRounds > 0 && round >= p.MaxRounds {
		return StopMaxRounds, true
	}
	if p.Stopping.NoImprovementRounds > 0 && !improved && round >= p.Stopping.NoImprovementRounds {
		return StopNoImprovement, true
	}
	return "", false
}

// Active reports whether the policy permits running critic rounds.
func (p Policy) Active() bool { return p.Mode == ModeAuto || p.Mode == ModeForce }

// SortedDimensions returns the configured dimensions in a stable order.
func (p Policy) SortedDimensions() []string {
	out := append([]string{}, p.Dimensions...)
	sort.Strings(out)
	return out
}
