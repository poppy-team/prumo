// Package eval is the Harness eval suite: one deterministic scenario per
// DoD capability, all runnable without external tokens.
package eval

import (
	"context"
	"strings"
	"testing"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/harness/aci"
	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/contextv2"
	"github.com/raillen/prumo/internal/harness/doccompile"
	"github.com/raillen/prumo/internal/harness/extagent"
	"github.com/raillen/prumo/internal/harness/gateway"
	"github.com/raillen/prumo/internal/harness/handoff"
	"github.com/raillen/prumo/internal/harness/humandocs"
	"github.com/raillen/prumo/internal/harness/knowledge"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
	"github.com/raillen/prumo/internal/harness/team"
)

type tools struct{ m map[string]agent.ToolResult }

func (t tools) Execute(_ context.Context, c agent.ToolCall) (agent.ToolResult, error) {
	if r, ok := t.m[c.Name]; ok {
		r.ToolCallID = c.ID
		return r, nil
	}
	return agent.ToolResult{ToolCallID: c.ID, Output: "ok"}, nil
}
func (t tools) KindOf(n string) string {
	if n == "edit.delete" {
		return "destructive"
	}
	return "read-only"
}

func baseRunner(t *testing.T, p model.Provider, run string) *harnessruntime.Runner {
	t.Helper()
	return harnessruntime.NewRunner(harnessruntime.Services{
		Models: p, Tools: tools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
		ContextManifest: func(_ context.Context, _ agent.NativeAgentState) (string, error) {
			return "ctx-" + run, nil
		},
	}, run, "S")
}

func TestEvalSimpleCompletion(t *testing.T) {
	r := baseRunner(t, model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "text", Text: "done"}, {Kind: "complete"}}}), "R-eval-1")
	r.Messages = []agent.Message{{ID: "m", Role: agent.RoleUser, Content: "hi"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestEvalToolTrajectory(t *testing.T) {
	r := baseRunner(t, model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c", Name: "fs.read", IdempotencyKey: "k"}}, {Kind: "complete"}}}), "R-eval-2")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m", Role: agent.RoleUser, Content: "read"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.Obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(r.Obs))
	}
}

func TestEvalPermissionDenialAndResume(t *testing.T) {
	p := model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c", Name: "x", IdempotencyKey: "k"}}, {Kind: "complete"}}})
	r := baseRunner(t, p, "R-eval-3")
	r.Svc.Perms = perm.New(perm.Policy{DefaultAction: agent.PermissionDeny})
	r.Messages = []agent.Message{{ID: "m", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err == nil {
		t.Fatal("expected denial")
	}
	r2 := baseRunner(t, p, "R-eval-3b")
	r2.Svc.Perms = perm.New(perm.Policy{DefaultAction: agent.PermissionAsk})
	r2.Messages = []agent.Message{{ID: "m", Role: agent.RoleUser, Content: "x"}}
	if err := r2.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r2.State.Phase != agent.PhaseYield {
		t.Fatalf("expected permission yield, got %s", r2.State.Phase)
	}
}

func TestEvalProviderRetryFallback(t *testing.T) {
	g := gateway.New()
	bad := model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "error", Error: "500", Retryable: true}}})
	good := model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}})
	g.Register(namedProvider{"bad", bad})
	g.Register(namedProvider{"good", good})
	route := g.Select([]gateway.RouteTarget{{Provider: "bad"}, {Provider: "good"}})
	if _, used, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, false); err != nil || used != "good" {
		t.Fatalf("fallback failed: %v %s", err, used)
	}
	if _, _, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, true); err == nil {
		t.Fatal("post-side-effect fallback must require Handoff")
	}
}

type namedProvider struct {
	name string
	p    *model.FakeProvider
}

func (n namedProvider) Name() string                     { return n.name }
func (n namedProvider) Capabilities() model.Capabilities { return n.p.Capabilities() }
func (n namedProvider) Models(ctx context.Context) ([]string, error) {
	return n.p.Models(ctx)
}
func (n namedProvider) Health(ctx context.Context) (string, error) { return n.p.Health(ctx) }
func (n namedProvider) Stream(ctx context.Context, r agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	return n.p.Stream(ctx, r)
}

func TestEvalCrashResumeAndNoDuplicateEffects(t *testing.T) {
	dir := t.TempDir()
	store := checkpoint.New(dir)
	fx := agent.PendingEffect{ID: "fx1", Kind: "edit", IdempotencyKey: "idem-1", Target: "a.go"}
	if ok, err := store.RecordIntent(fx); err != nil || !ok {
		t.Fatal("intent must proceed")
	}
	if err := store.RecordOutcome("fx1", agent.EffectApplied); err != nil {
		t.Fatal(err)
	}
	if ok, _ := store.RecordIntent(agent.PendingEffect{ID: "fx2", Kind: "edit", IdempotencyKey: "idem-1"}); ok {
		t.Fatal("replay must skip duplicate side effect")
	}
	cp := agent.Checkpoint{ID: "cp1", RunID: "R-eval-5", State: agent.NativeAgentState{RunID: "R-eval-5", Phase: agent.PhaseCheckpoint}}
	if err := store.Save(cp); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Latest("R-eval-5"); err != nil {
		t.Fatal(err)
	}
}

func TestEvalCrossProviderHandoff(t *testing.T) {
	ctx := context.Background()
	// Native side: checkpoint state becomes a typed pointer-first bundle.
	b, err := handoff.Build("native", "fake-agent", agent.NativeAgentState{RunID: "R-eval-6", ContextManifestID: "ctx"}, "rev", "move", nil)
	if err != nil || b.Validate() != nil {
		t.Fatal("handoff must validate")
	}
	// External side: session opens, receives the continuation summary
	// (never a transcript), runs to its own completion, then closes.
	f := &extagent.FakeAgent{}
	s, err := f.CreateSession(ctx, b.Handoff.ID)
	if err != nil || s.ID == "" {
		t.Fatal("external session must open")
	}
	summary := "handoff " + b.Handoff.ID + " checkpoint=" + b.Refs["checkpoint"] + " ctx=" + b.Refs["context_manifest"]
	if err := f.Send(ctx, s.ID, summary); err != nil {
		t.Fatalf("transfer failed: %v", err)
	}
	ch, err := f.Events(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	completed := false
	for ev := range ch {
		if ev.Kind == "external.session.completed" {
			completed = true
		}
	}
	if !completed {
		t.Fatal("external session must complete the transferred handoff")
	}
	if err := f.Close(ctx, s.ID); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestEvalContextPressure(t *testing.T) {
	m := contextv2.Compile("R-eval-7", []contextv2.Item{
		{Ref: "a", Authority: "canonical", Score: 1, TokenCost: 100},
		{Ref: "b", Authority: "canonical", Score: 0.5, TokenCost: 10000},
	}, 200, "L1")
	if m.EstimatedTokens > 200 || len(m.Included) == 0 {
		t.Fatalf("packing failed: %+v", m)
	}
}

func TestEvalBudgetExhaustion(t *testing.T) {
	r := baseRunner(t, model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "usage", Usage: &agent.Usage{InputTokens: 999999}}, {Kind: "complete"}}}), "R-eval-8")
	r.Svc.ConsumeBudget = func(agent.Usage) error { return errBudget }
	r.Messages = []agent.Message{{ID: "m", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err == nil {
		t.Fatal("expected budget exhaustion")
	}
}

var errBudget = errString("hard budget exhausted")

type errString string

func (e errString) Error() string { return string(e) }

func TestEvalSandboxDenial(t *testing.T) {
	e := aci.New(t.TempDir())
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "fs.read", Arguments: map[string]any{"path": "../../secret"}})
	if res.ExitCode == 0 {
		t.Fatal("sandbox must deny escape")
	}
	ce := aci.NewContainer(t.TempDir(), "img", aci.UnavailableRunner{Runtime: "docker"})
	res, _ = ce.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "process.exec", Arguments: map[string]any{"command": "echo hi"}})
	if res.ExitCode == 0 {
		t.Fatal("container sandbox must refuse without runtime (no silent host fallback)")
	}
}

func TestEvalExternalAgent(t *testing.T) {
	f := &extagent.FakeAgent{}
	ch, err := f.Events(context.Background(), "s1")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for range ch {
		n++
	}
	if n == 0 {
		t.Fatal("expected normalized external events")
	}
}

func TestEvalMultiAgentWorktree(t *testing.T) {
	tm := team.Team{ID: "t", Delegation: team.DelegationManual, Roles: []team.Role{{Name: "a", Workspace: "w1"}, {Name: "b", Workspace: "w1"}}}
	if err := tm.Validate(); err == nil {
		t.Fatal("expected ownership collision")
	}
	ok, cancel := context.WithCancel(context.Background())
	defer cancel()
	sum, err := team.Runner{
		Team: team.Team{ID: "t-eval", Delegation: team.DelegationManual, Roles: []team.Role{{Name: "dev", Workspace: t.TempDir()}, {Name: "rev"}}},
		Work: func(context.Context, team.Role) ([]string, map[string]float64, error) {
			return []string{"evidence"}, map[string]float64{"tool_calls": 1}, nil
		},
		ReviewerRole: "rev",
		Review: func(context.Context, team.ReviewInput) (team.ReviewVerdict, error) {
			return team.ReviewVerdict{Approve: true}, nil
		},
	}.Run(ok)
	if err != nil || sum.Status != "complete" {
		t.Fatalf("team review failed: %+v %v", sum, err)
	}
}

func TestEvalSeededRunIsCovered(t *testing.T) {
	s := knowledge.New()
	knowledge.SeedRequirement(s, "R-eval-k", "seeded goal")
	knowledge.SeedEvidence(s, "R-eval-k", "complete", "completed", "cp1")
	covered, uncovered := s.Coverage()
	if len(covered) != 1 || len(uncovered) != 0 {
		t.Fatalf("seeded run must be covered: %v %v", covered, uncovered)
	}
	dir := t.TempDir()
	path := dir + "/knowledge-R-eval-k.json"
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := knowledge.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if ready, blockers := loaded.Readiness(); !ready {
		t.Fatalf("reloaded run must be ready: %v", blockers)
	}
}

func TestEvalHumanDocsFlow(t *testing.T) {
	reg := docengine.Registry{
		Contracts: map[string]docengine.Contract{
			"c-op": {ID: "c-op", Role: "operator", Description: "operate the harness",
				RequiredKnowledge: []string{"run lifecycle"}, BlockingQuestions: []string{"who approves?"},
				EvidenceRequirements: []string{"green suite"}},
		},
		Profiles: map[string]docengine.Profile{
			"p-ops": {ID: "p-ops", Contracts: []string{"c-op"}},
		},
	}
	spec := humandocs.Spec{ID: "hd-eval", Title: "HD eval", Audience: "ops",
		Profiles: []string{"p-ops"}, Sources: map[string][]string{"c-op": {"runbook.md"}}}
	plan, err := humandocs.Planner(spec, reg)
	if err != nil {
		t.Fatal(err)
	}
	if ready, blockers := humandocs.Readiness(plan); !ready {
		t.Fatalf("seeded plan must be ready: %v", blockers)
	}
	arts, err := humandocs.GenerateTree(spec, plan, reg, t.TempDir())
	if err != nil || len(arts) != 3 {
		t.Fatalf("tree failed: %+v %v", arts, err)
	}
	brief := humandocs.Brief(plan.Units[2], reg.Contracts["c-op"])
	if !strings.Contains(brief, "TODO(CURATED)") {
		t.Fatal("brief must mark curated work")
	}
}

func TestEvalKnowledgeAndDocs(t *testing.T) {
	s := knowledge.New()
	_ = s.Put(knowledge.Record{ID: "req-1", Kind: knowledge.KindRequirement, Title: "harness", Authority: "canonical", Trust: "high", Status: "active"})
	if err := s.Commit(knowledge.Delta{ID: "d", Author: "eval", Upserts: []knowledge.Record{{ID: "ev-1", Kind: knowledge.KindEvidence, Title: "green", Authority: "reference", Trust: "high"}}, Links: []knowledge.Relation{{From: "ev-1", Type: "evidences", To: "req-1"}}}); err != nil {
		t.Fatal(err)
	}
	if ready, blockers := s.Readiness(); !ready {
		t.Fatalf("expected ready: %v", blockers)
	}
	c := doccompile.New()
	c.Add(doccompile.Node{ID: "a", Content: "# A"})
	if _, _, err := c.Render(); err != nil {
		t.Fatal(err)
	}
}
