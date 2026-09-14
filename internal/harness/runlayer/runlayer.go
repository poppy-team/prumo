// Package runlayer wires cross-cutting run concerns around the NativeAgent
// Runner: hard budget envelopes fed by model usage + tool counts, persisted
// permission resolutions, protocol evidence records with an optional strict
// quality gate, and a bridge from AgentEvent timelines to the observability
// event sink. Runners stay deterministic; this layer owns side effects.
package runlayer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/raillen/prumo/internal/budget"
	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/perm"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
	"github.com/raillen/prumo/internal/observability"
	evidence "github.com/raillen/prumo/internal/protocol/evidence"
)

// Tracker enforces one run's budget envelope (mutex-guarded; budget.Envelope
// is a value type).
type Tracker struct {
	mu       sync.Mutex
	envelope budget.Envelope
}

// NewTracker builds a hard envelope from limits (0 = untracked dimension).
func NewTracker(tokens, usd, tools float64) *Tracker {
	return &Tracker{envelope: budget.Envelope{
		Version: 1, Scope: "run",
		Limits: map[string]float64{"tokens": tokens, "cost_usd": usd, "tool_calls": tools},
		Usage:  map[string]float64{}, Mode: "hard",
	}}
}

// ConsumeUsage feeds model usage into the envelope (tokens + cost).
func (t *Tracker) ConsumeUsage(u agent.Usage) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	next, err := t.envelope.Consume(map[string]float64{
		"tokens":   float64(u.InputTokens + u.OutputTokens),
		"cost_usd": u.CostUSD,
	})
	if err != nil {
		return err
	}
	t.envelope = next
	return nil
}

// ConsumeTools records n tool calls.
func (t *Tracker) ConsumeTools(n int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	next, err := t.envelope.Consume(map[string]float64{"tool_calls": float64(n)})
	if err != nil {
		return err
	}
	t.envelope = next
	return nil
}

// Snapshot returns a copy of current usage.
func (t *Tracker) Snapshot() map[string]float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := map[string]float64{}
	for k, v := range t.envelope.Usage {
		out[k] = v
	}
	return out
}

// Save persists the envelope for audit/resume visibility.
func (t *Tracker) Save(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	data, err := json.MarshalIndent(t.envelope, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ToolReport is one executed tool call outcome.
type ToolReport struct {
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
}

// CountingTools decorates a ToolExecutor with budget counting + reports.
type CountingTools struct {
	Base    harnessruntime.ToolExecutor
	Tracker *Tracker
	mu      sync.Mutex
	Reports []ToolReport
}

func (c *CountingTools) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	if c.Tracker != nil {
		if err := c.Tracker.ConsumeTools(1); err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, err
		}
	}
	res, err := c.Base.Execute(ctx, call)
	c.mu.Lock()
	c.Reports = append(c.Reports, ToolReport{Name: call.Name, ExitCode: res.ExitCode})
	c.mu.Unlock()
	return res, err
}

func (c *CountingTools) KindOf(name string) string { return c.Base.KindOf(name) }

// ReportsCopy returns collected reports.
func (c *CountingTools) ReportsCopy() []ToolReport {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ToolReport{}, c.Reports...)
}

// StrictGate returns a Runner QualityGate requiring at least one passing
// test.run observation before completion. No test evidence → failed.
func StrictGate() func(reports []ToolReport) error {
	return func(reports []ToolReport) error {
		for _, r := range reports {
			if r.Name == "test.run" && r.ExitCode == 0 {
				return nil
			}
		}
		return fmt.Errorf("strict gate: completion requires a passing test.run")
	}
}

// DumpPermissions appends engine resolutions as JSONL (audit trail).
func DumpPermissions(path string, engine *perm.Engine) error {
	if engine == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, res := range engine.Log {
		data, err := json.Marshal(res)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// WriteEvidence validates and persists the run's protocol evidence record.
func WriteEvidence(path, runID, phase, stopReason string, usage map[string]float64, reports []ToolReport) (evidence.Record, error) {
	rec := evidence.Record{
		ID:      "ev-" + runID + "-" + phase,
		Type:    "harness_run",
		Summary: fmt.Sprintf("run %s %s (%s) usage=%v reports=%d", runID, phase, stopReason, usage, len(reports)),
	}
	if err := evidence.Validate(map[string]any{"id": rec.ID, "type": rec.Type}); err != nil {
		return evidence.Record{}, err
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return evidence.Record{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return evidence.Record{}, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return evidence.Record{}, err
	}
	return rec, os.Rename(tmp, path)
}

// BridgeToObservability projects timeline events into the observability sink.
// The file is created even for empty timelines (explicit "no events").
func BridgeToObservability(path string, events []agent.AgentEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(events) == 0 {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		return f.Close()
	}
	for _, ev := range events {
		obs := observability.NewEvent(ev.ID, ev.Kind, ev.RunID, ev.Payload)
		if err := observability.Append(path, obs); err != nil {
			return err
		}
	}
	return nil
}
