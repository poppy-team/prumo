package runtime

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// The provider advertised tool-calling and the request carried no tools, because
// nothing in the runner asked the executor what it could run. The daemon, the
// CLI and the team binder all inherited that: a real model had no way to know
// the tools existed (GAP-114).

// specRecorder is an executor that can describe itself, which is what the native
// one now does.
type specRecorder struct {
	specs []agent.ToolSpec
	seen  []agent.ModelRequest
}

func (s *specRecorder) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "ok"}, nil
}
func (s *specRecorder) KindOf(string) string      { return "read-only" }
func (s *specRecorder) OperationOf(string) string { return "" }
func (s *specRecorder) Specs() []agent.ToolSpec   { return s.specs }

// contextSpecs is the other shape an executor can have.
type contextSpecs struct {
	recorder *specRecorder
}

func (c contextSpecs) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	return c.recorder.Execute(ctx, call)
}
func (c contextSpecs) KindOf(string) string      { return "read-only" }
func (c contextSpecs) OperationOf(string) string { return "" }
func (c contextSpecs) Specs(context.Context) ([]agent.ToolSpec, error) {
	return c.recorder.specs, nil
}

// requestRecorder wraps a provider and keeps the request it was handed, so a
// test can assert on what actually went on the wire rather than on an internal
// helper.
type requestRecorder struct {
	inner    model.Provider
	requests []agent.ModelRequest
}

func (r *requestRecorder) Name() string                     { return r.inner.Name() }
func (r *requestRecorder) Capabilities() model.Capabilities { return r.inner.Capabilities() }
func (r *requestRecorder) Models(ctx context.Context) ([]string, error) {
	return r.inner.Models(ctx)
}
func (r *requestRecorder) Health(ctx context.Context) (string, error) {
	return r.inner.Health(ctx)
}
func (r *requestRecorder) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	r.requests = append(r.requests, req)
	return r.inner.Stream(ctx, req)
}

func TestTheRequestTheProviderReceivesCarriesTheToolSurface(t *testing.T) {
	// The link this closes: the executor can describe its tools, so the request
	// the provider receives has them. Everything before this ran a real model
	// with an empty catalogue.
	tools := []agent.ToolSpec{{
		Name: "fs.read", Description: "read a file as text",
		Schema: map[string]any{"type": "object", "required": []string{"path"}},
	}}
	provider := &requestRecorder{
		inner: model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}}),
	}
	r := NewRunner(Services{
		Models: provider,
		Tools:  &specRecorder{specs: tools},
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
	}, "R-specs", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read the readme"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(provider.requests) == 0 {
		t.Fatal("the provider was never called")
	}
	got := provider.requests[0].Tools
	if len(got) != 1 || got[0].Name != "fs.read" {
		t.Fatalf("the request must carry the executor's tools, got %+v", got)
	}
	if got[0].Schema == nil {
		t.Fatal("the schema must travel with the tool, not just its name")
	}
}

func TestAnExplicitToolSpecsWinsOverTheExecutor(t *testing.T) {
	executor := &specRecorder{specs: []agent.ToolSpec{{Name: "from.executor"}}}
	explicit := []agent.ToolSpec{{Name: "from.caller"}}
	r := NewRunner(Services{
		Models:    model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}}),
		Tools:     executor,
		Perms:     perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		ToolSpecs: func() []agent.ToolSpec { return explicit },
	}, "R-explicit", "S1")
	got := r.toolSpecs(context.Background())
	if len(got) != 1 || got[0].Name != "from.caller" {
		t.Fatalf("an explicit ToolSpecs must win, got %+v", got)
	}
}

func TestAnExecutorThatCannotBeListedYieldsNoToolsWithoutFailing(t *testing.T) {
	// A tool surface that cannot be enumerated is still executable; failing the
	// turn over a listing would be the wrong trade.
	r := NewRunner(Services{
		Models: model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}}),
		Tools:  &specRecorder{}, // no specs, no error
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
	}, "R-none", "S1")
	if got := r.toolSpecs(context.Background()); len(got) != 0 {
		t.Fatalf("an executor with no specs must yield none, got %+v", got)
	}
}

func TestTheContextShapedSpecsAreAlsoUnderstood(t *testing.T) {
	recorder := &specRecorder{specs: []agent.ToolSpec{{Name: "ctx.tool"}}}
	r := NewRunner(Services{
		Models: model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}}),
		Tools:  contextSpecs{recorder: recorder},
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
	}, "R-ctx", "S1")
	got := r.toolSpecs(context.Background())
	if len(got) != 1 || got[0].Name != "ctx.tool" {
		t.Fatalf("both Specs shapes must be understood, got %+v", got)
	}
}

func TestNoToolsMeansNoSpecsRatherThanAFailure(t *testing.T) {
	r := NewRunner(Services{
		Models: model.NewFake(nil),
		Perms:  perm.New(perm.Policy{}),
	}, "R-bare", "S1")
	if got := r.toolSpecs(context.Background()); got != nil {
		t.Fatalf("a run with no tools must report none, got %+v", got)
	}
}
