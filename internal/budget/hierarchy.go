package budget

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrBlockedBudget      = errors.New("budget: hard limit exhausted, triggering safe stop blocked_budget")
	ErrReviewReserveProtected = errors.New("budget: cannot consume from review reserve during implementation phase")
)

// Scope hierarchy constants per Rule 28.
const (
	ScopeMonthly   = "monthly"
	ScopeWorkspace = "workspace"
	ScopeProject   = "project"
	ScopeGoal      = "goal"
	ScopeRun       = "run"
	ScopeAgent     = "agent"
	ScopeStep      = "step"
)

// PricingRates holds cost per unit in cents (USD or BRL).
type PricingRates struct {
	InputTokenCentsPer1k  float64 `json:"input_token_cents_per_1k"`
	OutputTokenCentsPer1k float64 `json:"output_token_cents_per_1k"`
	ToolCallCents         float64 `json:"tool_call_cents"`
	Currency              string  `json:"currency"`
}

// DefaultPricingRates provides standard baseline pricing.
func DefaultPricingRates() PricingRates {
	return PricingRates{
		InputTokenCentsPer1k:  0.30, // 0.3 cents per 1k tokens
		OutputTokenCentsPer1k: 1.20,
		ToolCallCents:         0.05,
		Currency:              "BRL",
	}
}

// HierarchicalEnvelope tracks detailed budget metrics across scopes.
type HierarchicalEnvelope struct {
	Scope              string             `json:"scope"`
	ParentScope        string             `json:"parent_scope,omitempty"`
	HardLimitCents     int                `json:"hard_limit_cents"`
	SoftLimitCents     int                `json:"soft_limit_cents"`
	ReviewReserveCents int                `json:"review_reserve_cents"`
	ObservedCostCents  float64            `json:"observed_cost_cents"`
	MetricsUsage       map[string]float64 `json:"metrics_usage"`
	Currency           string             `json:"currency"`
	IsBlocked          bool               `json:"is_blocked"`
}

// BudgetCheckpoint represents the saved state when hard limit is hit.
type BudgetCheckpoint struct {
	CheckpointID   string    `json:"checkpoint_id"`
	RunID          string    `json:"run_id"`
	TaskID         string    `json:"task_id"`
	Scope          string    `json:"scope"`
	ExhaustedLimit int       `json:"exhausted_limit_cents"`
	CurrentCost    float64   `json:"current_cost_cents"`
	Timestamp      time.Time `json:"timestamp"`
	Status         string    `json:"status"` // blocked_budget
}

// Manager orchestrates hierarchical envelopes and checks.
type Manager struct {
	mu        sync.RWMutex
	envelopes map[string]*HierarchicalEnvelope // scopeID -> Envelope
	pricing   PricingRates
}

// NewManager initializes a hierarchical Budget Manager.
func NewManager(pricing PricingRates) *Manager {
	if pricing.Currency == "" {
		pricing = DefaultPricingRates()
	}
	return &Manager{
		envelopes: make(map[string]*HierarchicalEnvelope),
		pricing:   pricing,
	}
}

// SetEnvelope configures a hierarchical envelope with hard limit and review reserve.
func (m *Manager) SetEnvelope(scopeID, scopeType, parentScope string, hardLimitCents, softLimitCents, reviewReserveCents int) *HierarchicalEnvelope {
	m.mu.Lock()
	defer m.mu.Unlock()

	env := &HierarchicalEnvelope{
		Scope:              scopeType,
		ParentScope:        parentScope,
		HardLimitCents:     hardLimitCents,
		SoftLimitCents:     softLimitCents,
		ReviewReserveCents: reviewReserveCents,
		MetricsUsage:       make(map[string]float64),
		Currency:           m.pricing.Currency,
	}
	m.envelopes[scopeID] = env
	return env
}

// RecordConsumption records token/tool usage and calculates observed cost.
func (m *Manager) RecordConsumption(scopeID string, isReviewPhase bool, inputTokens, outputTokens, toolCalls int) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	env, ok := m.envelopes[scopeID]
	if !ok {
		return 0, fmt.Errorf("envelope %s not found", scopeID)
	}

	cost := (float64(inputTokens)/1000.0)*m.pricing.InputTokenCentsPer1k +
		(float64(outputTokens)/1000.0)*m.pricing.OutputTokenCentsPer1k +
		float64(toolCalls)*m.pricing.ToolCallCents

	newCost := env.ObservedCostCents + cost

	// Check review reserve protection: if not in review phase, cannot dip into review reserve
	if !isReviewPhase && env.ReviewReserveCents > 0 {
		effectiveLimit := float64(env.HardLimitCents - env.ReviewReserveCents)
		if effectiveLimit > 0 && newCost > effectiveLimit {
			env.IsBlocked = true
			return cost, fmt.Errorf("%w (attempted into review reserve)", ErrReviewReserveProtected)
		}
	}

	// Check hard limit
	if env.HardLimitCents > 0 && newCost > float64(env.HardLimitCents) {
		env.IsBlocked = true
		return cost, fmt.Errorf("%w for scope %s (cost %.2f > hard limit %d)", ErrBlockedBudget, env.Scope, newCost, env.HardLimitCents)
	}

	env.ObservedCostCents = newCost
	env.MetricsUsage["input_tokens"] += float64(inputTokens)
	env.MetricsUsage["output_tokens"] += float64(outputTokens)
	env.MetricsUsage["tool_calls"] += float64(toolCalls)

	return cost, nil
}

// CreateBlockedCheckpoint saves a durable checkpoint when blocked_budget occurs.
func (m *Manager) CreateBlockedCheckpoint(checkpointDir, runID, taskID, scopeID string) (*BudgetCheckpoint, error) {
	m.mu.RLock()
	env := m.envelopes[scopeID]
	m.mu.RUnlock()

	if env == nil {
		return nil, fmt.Errorf("envelope %s not found", scopeID)
	}

	chk := &BudgetCheckpoint{
		CheckpointID:   fmt.Sprintf("chk-budget-%d", time.Now().UnixNano()),
		RunID:          runID,
		TaskID:         taskID,
		Scope:          env.Scope,
		ExhaustedLimit: env.HardLimitCents,
		CurrentCost:    env.ObservedCostCents,
		Timestamp:      time.Now().UTC(),
		Status:         "blocked_budget",
	}

	if checkpointDir != "" {
		if err := os.MkdirAll(checkpointDir, 0755); err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(chk, "", "  ")
		if err != nil {
			return nil, err
		}
		filePath := filepath.Join(checkpointDir, fmt.Sprintf("%s.json", chk.CheckpointID))
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return nil, err
		}
	}

	return chk, nil
}
