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
	"sort"
	"strings"
	"sync"
	"time"

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
	// reserved holds allowances for calls that have been made but whose usage
	// has not been reported yet. Without it a run could start N concurrent
	// calls, each individually within the limit, and land N times over it.
	reserved map[string]float64
}

// NewTracker builds a hard envelope from limits (0 = untracked dimension).
func NewTracker(tokens, usd, tools float64) *Tracker {
	return &Tracker{envelope: budget.Envelope{
		Version: 1, Scope: "run",
		Limits: map[string]float64{"tokens": tokens, "cost_usd": usd, "tool_calls": tools},
		Usage:  map[string]float64{}, Mode: "hard",
	}, reserved: map[string]float64{}}
}

// Reserve holds back an allowance before a call is made, and is released once
// the real usage is known.
//
// Checking the envelope after the usage arrives bounds nothing: the call has
// already been paid for. A turn can overshoot by its whole cost, so the last
// turn of a run is the one that can cross the ceiling. Reserving before the
// request is what makes the limit a limit rather than a post-mortem.
//
// The release takes both dimensions because a call is bounded by what it sent
// and by what it cost, and settling only one of them would leave the other
// reserved forever or charged twice.
func (t *Tracker) Reserve(tokens float64) (release func(actualTokens, actualCost float64), err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.checkHeadroom(map[string]float64{"tokens": tokens}); err != nil {
		return nil, err
	}
	t.reserved["tokens"] += tokens
	released := false
	return func(actualTokens, actualCost float64) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if released {
			return
		}
		released = true
		t.reserved["tokens"] -= tokens
		settle := map[string]float64{}
		if actualTokens > 0 {
			settle["tokens"] = actualTokens
		}
		if actualCost > 0 {
			settle["cost_usd"] = actualCost
		}
		if len(settle) == 0 {
			return
		}
		// The cost is already incurred, so it is recorded even when it passes the
		// ceiling. The envelope's Consume refuses to cross a hard limit, which is
		// right for a request and wrong for a bill: dropping the last call's
		// usage would leave the tracker reporting less than the run actually
		// spent, and the next preflight would see headroom that does not exist.
		for key, value := range settle {
			t.envelope.Usage[key] += value
		}
	}, nil
}

// Exhausted reports whether a dimension has no headroom left, counting what is
// reserved but not yet spent. It is the preflight: a run about to make another
// call should stop when it cannot afford one.
//
// The comparison is >=, so a run sitting exactly on a limit reports exhausted.
// There is no free turn at the boundary, and a preflight that says "not
// exhausted" there invites the call that crosses it.
func (t *Tracker) Exhausted() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key, limit := range t.envelope.Limits {
		if limit <= 0 {
			continue
		}
		committed := t.envelope.Usage[key] + t.reserved[key]
		if committed >= limit {
			return fmt.Errorf("hard budget exhausted for %s: %g of %g already committed", key, committed, limit)
		}
	}
	return nil
}

// checkHeadroom accounts for outstanding reservations. Caller holds t.mu.
func (t *Tracker) checkHeadroom(extra map[string]float64) error {
	for key, limit := range t.envelope.Limits {
		if limit <= 0 {
			// Zero is unlimited. That is the documented meaning, and it is why
			// a tracker built with NewTracker(0, 0, 0) has no ceiling at all.
			continue
		}
		projected := t.envelope.Usage[key] + t.reserved[key] + extra[key]
		if projected > limit {
			return fmt.Errorf("hard budget exhausted for %s: %g of %g already committed", key, projected, limit)
		}
	}
	return nil
}

// LimitsCopy reports the configured limits, for a status response.
func (t *Tracker) LimitsCopy() map[string]float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := map[string]float64{}
	for key, value := range t.envelope.Limits {
		out[key] = value
	}
	return out
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

func (c *CountingTools) OperationOf(name string) string { return c.Base.OperationOf(name) }

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

// SavePermissions rewrites a run's permission audit trail.
//
// DumpPermissions appends, which is right for a one-shot CLI run (one call per
// process). A daemon persists a run's artifacts every time it stops, and a run
// that stops for approval stops twice — appending there would duplicate the
// whole log. This variant is idempotent by construction.
func SavePermissions(path string, engine *perm.Engine) error {
	if engine == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var out []byte
	for _, res := range engine.Log {
		data, err := json.Marshal(res)
		if err != nil {
			return err
		}
		out = append(out, append(data, '\n')...)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadPermissions reads a decision trail back into an engine, so an approval
// survives the process that was waiting for it.
//
// The write side has existed since the trail was introduced; the read side did
// not. A daemon that stopped for approval, was restarted, and resumed the run
// asked the same human the same question again — because the decision was only
// ever in memory (GAP-106). A missing file is not an error: a run that never hit
// a gate has no trail to load.
//
// A corrupt line is skipped rather than fatal. A decision trail is an audit
// record; one unreadable line must not make every other approval unusable, and
// silently ignoring a line is what an append-only JSONL format is for.
func LoadPermissions(path string, engine *perm.Engine) error {
	if engine == nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var res agent.PermissionResolution
		if err := json.Unmarshal([]byte(line), &res); err != nil {
			continue
		}
		if res.RequestID == "" {
			continue
		}
		// A trail written before fingerprints existed cannot be matched to
		// content. Loading it would apply an approval to whatever carries that
		// id now, which is the hole the fingerprint closed. Such a line is kept
		// in the log for the audit and skipped for reuse.
		if res.Fingerprint == "" {
			continue
		}
		engine.Restore(res)
	}
	return nil
}

// EvidenceProducer identifies the writer of run evidence. It is recorded in
// every record so a reader can tell harness-generated evidence from evidence a
// human or an external tool produced.
const EvidenceProducer = "prumo-harness"

// WriteEvidence validates and persists the run's protocol evidence record.
//
// The record carries every field schemas/evidence.schema.json requires. Before
// the 2026-09-23 audit it wrote only id, type and summary, used a type
// (`harness_run`) absent from the schema enum, and validated through a helper
// that checked two fields — so a record that could not describe what produced
// it, when it happened, or which goal it belonged to was still accepted as
// evidence.
//
// Status is derived from the run, not asserted by the caller: a run that
// reached a failed phase, or that executed a tool reporting a non-zero exit
// code, is `failed` regardless of what the caller believes. Confidence is
// `unknown` because a run summary is not verification of the goal it ran for.
func WriteEvidence(path, runID, goalID, phase, stopReason string, usage map[string]float64, reports []ToolReport) (evidence.Record, error) {
	goal := strings.TrimSpace(goalID)
	if goal == "" {
		goal = "unassigned"
	}

	recordType := "harness_run"
	status := runStatusFrom(phase, stopReason, reports)

	rec := evidence.Record{
		ID:        "ev-" + runID + "-" + phase,
		Type:      recordType,
		Summary:   fmt.Sprintf("run %s %s (%s) usage=%v reports=%d", runID, phase, stopReason, usage, len(reports)),
		Producer:  EvidenceProducer,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    status,
		GoalID:    goal,
		RunID:     runID,
		// A run summary reports what happened; it does not verify the goal.
		// Marking it low rather than unknown would overstate it, and leaving it
		// blank would let a reader assume the strongest available claim.
		Confidence: "low",
		Metadata: map[string]any{
			"phase":        phase,
			"stop_reason":  stopReason,
			"usage":        usage,
			"report_count": len(reports),
			"failed_tools": failedToolNames(reports),
		},
	}
	if err := evidence.ValidateRecord(rec); err != nil {
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

// runStatusFrom derives a run's status from what actually happened.
func runStatusFrom(phase, stopReason string, reports []ToolReport) string {
	if strings.EqualFold(stopReason, "cancelled") || strings.EqualFold(stopReason, "denied") {
		return "cancelled"
	}
	if strings.EqualFold(stopReason, "failed") || strings.EqualFold(phase, "failed") {
		return "failed"
	}
	for _, report := range reports {
		if report.ExitCode != 0 {
			return "failed"
		}
	}
	if strings.EqualFold(phase, "complete") {
		return "passed"
	}
	return "incomplete"
}

// failedToolNames lists the tools that reported a non-zero exit code, so the
// evidence record names what failed rather than only that something did.
func failedToolNames(reports []ToolReport) []string {
	failed := []string{}
	for _, report := range reports {
		if report.ExitCode != 0 {
			failed = append(failed, fmt.Sprintf("%s(exit=%d)", report.Name, report.ExitCode))
		}
	}
	sort.Strings(failed)
	return failed
} // BridgeToObservability projects timeline events into the observability sink.
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
