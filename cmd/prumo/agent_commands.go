// Command: prumo agent — headless harness entrypoint (HA2..HA11).
//
// Subcommands:
//
//	run      --goal <text> --path <dir> [--provider fake|openai-compat|anthropic] [--model ...] [--max-turns N]
//	resume   --run <id> --path <dir>
//	handoff  --run <id> --from native --to <agent> [--path <dir>]
//	events   --run <id> --path <dir>
//	protocol [--client <version>]
//	serve    --path <dir> [--socket <path>]        (local daemon, blocks)
//	ps       [--socket <path>] [--path <dir>]
//	logs     --run <id> [--socket <path>] [--path <dir>]
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/raillen/prumo/internal/harness/aci"
	"github.com/raillen/prumo/internal/harness/acpserver"
	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/contextv2"
	"github.com/raillen/prumo/internal/harness/daemon"
	"github.com/raillen/prumo/internal/harness/extagent"
	"github.com/raillen/prumo/internal/harness/gateway"
	"github.com/raillen/prumo/internal/harness/handoff"
	"github.com/raillen/prumo/internal/harness/knowledge"
	"github.com/raillen/prumo/internal/harness/mcp"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
	"github.com/raillen/prumo/internal/harness/runlayer"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/toolgateway"
)

func defaultSocket(root string) string {
	return filepath.Join(root, ".prumo", "runtime", "harness", "agentd.sock")
}

func runAgent(asJSON bool, args []string) int {
	if len(args) == 0 {
		return runTui(asJSON, nil)
	}
	switch args[0] {
	case "tui":
		return runTui(asJSON, args[1:])
	case "run":
		return runAgentRun(asJSON, args[1:])
	case "resume":
		return runAgentResume(asJSON, args[1:])
	case "handoff":
		return runAgentHandoff(asJSON, args[1:])
	case "events":
		return runAgentEvents(asJSON, args[1:])
	case "protocol":
		return runAgentProtocol(asJSON, args[1:])
	case "serve":
		return runAgentServe(asJSON, args[1:])
	case "ps":
		return runAgentPs(asJSON, args[1:])
	case "logs":
		return runAgentLogs(asJSON, args[1:])
	case "steer":
		return runAgentSteer(asJSON, args[1:])
	case "approve":
		return runAgentPermission(asJSON, args[1:], true)
	case "deny":
		return runAgentPermission(asJSON, args[1:], false)
	case "stop":
		return runAgentStop(asJSON, args[1:])
	case "schedule":
		return runAgentSchedule(asJSON, args[1:])
	case "unschedule":
		return runAgentUnschedule(asJSON, args[1:])
	case "jobs":
		return runAgentJobs(asJSON, args[1:])
	case "promote":
		return runAgentPromote(asJSON, args[1:])
	case "gc":
		return runAgentGC(asJSON, args[1:])
	case "acp":
		return runAgentACP(asJSON, args[1:])
	case "providers":
		return runAgentProviders(asJSON, args[1:])
	case "models":
		return runAgentModels(asJSON, args[1:])
	case "diff":
		return runAgentDiff(asJSON, args[1:])
	default:
		if strings.HasPrefix(args[0], "-") {
			return runTui(asJSON, args)
		}
		return exitUsage
	}
}

func agentFlags(args []string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		if len(args[i]) > 2 && args[i][:2] == "--" {
			key := args[i][2:]
			if i+1 < len(args) && !(len(args[i+1]) > 2 && args[i+1][:2] == "--") {
				out[key] = args[i+1]
				i++
			} else {
				out[key] = ""
			}
		}
	}
	return out
}

func runAgentRun(asJSON bool, args []string) int {
	f := agentFlags(args)
	goal := f["goal"]
	if goal == "" {
		goal = "headless run"
	}
	root := f["path"]
	if root == "" {
		root = "."
	}
	providerName := f["provider"]
	if providerName == "" {
		providerName = "fake"
	}
	modelName := f["model"]
	baseURL := f["base-url"]
	apiKey := f["api-key"]
	if apiKey == "" {
		apiKey = os.Getenv("PRUMO_MODEL_API_KEY")
	}
	maxTurns := 5
	if v, ok := f["max-turns"]; ok {
		fmt.Sscanf(v, "%d", &maxTurns)
	}
	runID := f["run"]
	if runID == "" {
		runID = "R-agent-1"
	}

	var provider model.Provider
	switch providerName {
	case "fake":
		provider = model.NewFake(map[string][]model.ScriptStep{
			"*": {
				{Kind: "text", Text: "goal: " + goal},
				{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", TurnID: "turn-1", Name: "git.status", IdempotencyKey: runID + ":c1"}},
				{Kind: "complete"},
			},
		})
	case "fake-tools", "opencode":
		var err error
		provider, err = model.ForName(providerName, baseURL, apiKey, modelName)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if aware, ok := provider.(interface{ SetWorkspace(string) }); ok {
			aware.SetWorkspace(root)
		}
	case "openai-compat":
		if baseURL == "" {
			baseURL = os.Getenv("PRUMO_MODEL_BASE_URL")
		}
		if baseURL == "" {
			return serviceError(asJSON, fmt.Errorf("openai-compat requires --base-url or PRUMO_MODEL_BASE_URL"))
		}
		if err := checkModelDestination(baseURL, f); err != nil {
			return serviceError(asJSON, err)
		}
		provider = model.NewOpenAICompatWithPolicy(baseURL, apiKey, modelName, destinationPolicy(f)).
			WithHeaders(model.ModelHeaders())
	case "anthropic":
		if baseURL == "" {
			baseURL = os.Getenv("PRUMO_MODEL_BASE_URL")
		}
		if baseURL != "" {
			if err := checkModelDestination(baseURL, f); err != nil {
				return serviceError(asJSON, err)
			}
		}
		provider = model.NewAnthropicWithPolicy(baseURL, apiKey, modelName, destinationPolicy(f))
	default:
		return serviceError(asJSON, fmt.Errorf("unknown provider %s (fake|fake-tools|openai-compat|anthropic|opencode)", providerName))
	}

	// The gateway is what the runner calls, so routing, retry, the circuit
	// breaker and the quota filter are in the path rather than built and
	// unreachable (GAP-102).
	provider = routeThroughGateway(provider, modelName, baseURL)

	dir := filepath.Join(root, ".prumo", "runtime", "harness")
	eventLog := filepath.Join(dir, "events-"+runID+".jsonl")
	ctxBudget := 8000
	if v, ok := f["context-budget"]; ok {
		fmt.Sscanf(v, "%d", &ctxBudget)
	}
	ctxLevel := f["context-level"]
	compileContext := func() string {
		m := contextv2.CompileWorkspace(runID, goal, root, ctxBudget, ctxLevel)
		data, err := json.MarshalIndent(m, "", "  ")
		if err == nil {
			_ = os.MkdirAll(dir, 0o755)
			_ = os.WriteFile(filepath.Join(dir, "context-"+runID+".json"), data, 0o644)
		}
		appendAgentEvent(eventLog, agent.AgentEvent{ID: runID + "-ctx", RunID: runID, Kind: "context.compiled",
			Payload: map[string]any{"included": len(m.Included), "tokens": m.EstimatedTokens, "pressure": m.Pressure, "level": m.Level}, CreatedAt: agent.Now()})
		return "ctx-" + runID
	}
	tools, cleanupTools, err := agentTools(root, f)
	if err != nil {
		return serviceError(asJSON, err)
	}
	defer cleanupTools()
	budgetTokens, budgetUSD, budgetTools := 0.0, 0.0, 0.0
	if v, ok := f["budget-tokens"]; ok {
		fmt.Sscanf(v, "%f", &budgetTokens)
	}
	if v, ok := f["budget-usd"]; ok {
		fmt.Sscanf(v, "%f", &budgetUSD)
	}
	if v, ok := f["budget-tools"]; ok {
		fmt.Sscanf(v, "%f", &budgetTools)
	}
	tracker := runlayer.NewTracker(budgetTokens, budgetUSD, budgetTools)
	counting := &runlayer.CountingTools{Base: tools, Tracker: tracker}
	policy, err := permissionPolicy(f)
	if err != nil {
		return serviceError(asJSON, err)
	}
	engine := perm.New(policy)
	checkpoints := checkpoint.New(dir)
	strict := false
	if _, ok := f["strict"]; ok {
		strict = true
	}
	var timeline []agent.AgentEvent
	appendAgentEvent(eventLog, agent.AgentEvent{ID: runID + "-started", RunID: runID, Kind: "run.started", Payload: map[string]any{"goal": goal, "provider": providerName}, CreatedAt: agent.Now()})
	runner := harnessruntime.NewRunner(harnessruntime.Services{
		Models:        provider,
		Tools:         counting,
		Perms:         engine,
		Checkpoints:   checkpoints,
		EffectJournal: checkpoints,
		Events: func(ev agent.AgentEvent) {
			if !asJSON {
				fmt.Fprintf(os.Stderr, "[%s] %s\n", ev.Kind, ev.TurnID)
			}
			appendAgentEvent(eventLog, ev)
			timeline = append(timeline, ev)
		},
		ContextManifest: func(_ context.Context, _ agent.NativeAgentState) (string, error) {
			return compileContext(), nil
		},
		ToolSpecs: func() []agent.ToolSpec {
			type specer interface {
				Specs(context.Context) ([]agent.ToolSpec, error)
			}
			if s, ok := tools.(specer); ok {
				specs, err := s.Specs(context.Background())
				if err == nil {
					return specs
				}
			}
			return nil
		},
	}, runID, "S-1")
	runner.MaxTurns = maxTurns
	if v, ok := f["compact-keep"]; ok {
		fmt.Sscanf(v, "%d", &runner.CompactKeep)
	}
	if v, ok := f["compact-budget"]; ok {
		fmt.Sscanf(v, "%d", &runner.CompactBudget)
	} else if runner.CompactKeep > 0 {
		runner.CompactBudget = ctxBudget
	}
	if strict {
		reports := counting
		runner.QualityGate = func() error { return runlayer.StrictGate()(reports.ReportsCopy()) }
	}
	if path, ok := f["gates"]; ok && path != "" {
		policies, err := runlayer.LoadGatePolicies(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		reports := counting
		usage := tracker
		runner.QualityGate = func() error {
			return runlayer.GatesQualityGate(policies, reports.ReportsCopy, usage.Snapshot)()
		}
	}
	runner.Svc.ReserveBudget = tracker.Reserve
	runner.Svc.BudgetExhausted = tracker.Exhausted
	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: goal, CreatedAt: agent.Now()}}
	kstore := knowledge.New()
	knowledge.SeedRequirement(kstore, runID, goal)
	knowledgePath := filepath.Join(dir, "knowledge-"+runID+".json")
	saveKnowledge := func() {
		knowledge.SeedEvidence(kstore, runID, string(runner.State.Phase), runner.State.StopReason, runner.State.RunID+"-latest")
		_ = kstore.Save(knowledgePath)
	}
	finishRun := func() {
		saveKnowledge()
		_ = tracker.Save(filepath.Join(dir, "budget-"+runID+".json"))
		_ = runlayer.DumpPermissions(filepath.Join(dir, "permissions-"+runID+".jsonl"), engine)
		if _, evErr := runlayer.WriteEvidence(filepath.Join(dir, "evidence-"+runID+".json"),
			runID, goal, string(runner.State.Phase), runner.State.StopReason, tracker.Snapshot(), counting.ReportsCopy()); evErr != nil {
			fmt.Fprintf(os.Stderr, "evidence: %v\n", evErr)
		}
		_ = runlayer.BridgeToObservability(filepath.Join(dir, "obs-"+runID+".jsonl"), timeline)
		_, _ = checkpoints.Prune(5)
	}
	if err := runner.RunUntilDone(context.Background()); err != nil {
		finishRun()
		appendAgentEvent(eventLog, agent.AgentEvent{ID: runID + "-failed", RunID: runID, Kind: "run.failed", Payload: map[string]any{"error": err.Error()}, CreatedAt: agent.Now()})
		return serviceError(asJSON, err)
	}
	appendAgentEvent(eventLog, agent.AgentEvent{ID: runID + "-finished", RunID: runID, Kind: "run.finished", Payload: map[string]any{"phase": string(runner.State.Phase), "stop_reason": runner.State.StopReason}, CreatedAt: agent.Now()})
	finishRun()
	result := map[string]any{"run_id": runID, "phase": string(runner.State.Phase), "stop_reason": runner.State.StopReason, "revision": runner.State.Revision, "turns": runner.TurnsDone}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("Run %s: %s (%s)\n", runID, runner.State.Phase, runner.State.StopReason)
	return exitOK
}

func runAgentResume(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	runID := f["run"]
	if runID == "" {
		return serviceError(asJSON, fmt.Errorf("resume requires --run <id>"))
	}
	dir := filepath.Join(root, ".prumo", "runtime", "harness")
	store := checkpoint.New(dir)
	cp, err := store.Latest(runID)
	if err != nil {
		return serviceError(asJSON, err)
	}
	// A run stopped for approval has to be answered, not stepped past. Saying
	// so is the honest outcome: answering a gate requires the live permission
	// engine, which only the running daemon holds.
	if len(cp.State.PendingPerms) > 0 {
		result := map[string]any{
			"run_id": runID, "resumed_from": cp.ID, "phase": string(cp.State.Phase),
			"pending_permissions": cp.State.PendingPerms, "resumed": false,
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("Run %s is waiting for approval: %s\n", runID, strings.Join(cp.State.PendingPerms, ", "))
		fmt.Printf("Answer it against a live daemon: prumo agent approve --run %s --request <id>\n", runID)
		return exitOK
	}

	// A checkpoint with no conversation cannot be continued. This used to report
	// the run as resumed and set its phase to complete, which claimed work that
	// was never done (GAP-123).
	if !cp.Resumable() {
		result := map[string]any{
			"run_id": runID, "resumed_from": cp.ID, "phase": string(cp.State.Phase),
			"resumed": false,
			"reason":  "checkpoint carries no conversation, so the turn cannot be continued",
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("Cannot resume %s from %s: the checkpoint carries no conversation.\n", runID, cp.ID)
		fmt.Printf("The run reached %s; continuing it needs the conversation that was not recorded.\n", cp.State.Phase)
		return exitOK
	}

	// A continuation calls a model, so the provider is the caller's to name.
	// Defaulting to a fake provider here would produce a completed run whose
	// output no model ever wrote.
	providerName := f["provider"]
	if providerName == "" {
		providerName = "fake"
	}
	baseURL := f["base-url"]
	apiKey := f["api-key"]
	modelName := f["model"]
	var provider model.Provider
	switch providerName {
	case "fake":
		provider = model.NewFake(map[string][]model.ScriptStep{
			"*": {{Kind: "text", Text: "continued from checkpoint"}, {Kind: "complete"}},
		})
	case "openai-compat":
		if baseURL == "" {
			baseURL = os.Getenv("PRUMO_MODEL_BASE_URL")
		}
		if baseURL == "" {
			return serviceError(asJSON, fmt.Errorf("openai-compat requires --base-url or PRUMO_MODEL_BASE_URL"))
		}
		if err := checkModelDestination(baseURL, f); err != nil {
			return serviceError(asJSON, err)
		}
		provider = model.NewOpenAICompatWithPolicy(baseURL, apiKey, modelName, destinationPolicy(f)).
			WithHeaders(model.ModelHeaders())
	case "anthropic":
		if baseURL == "" {
			baseURL = os.Getenv("PRUMO_MODEL_BASE_URL")
		}
		if baseURL != "" {
			if err := checkModelDestination(baseURL, f); err != nil {
				return serviceError(asJSON, err)
			}
		}
		provider = model.NewAnthropicWithPolicy(baseURL, apiKey, modelName, destinationPolicy(f))
	default:
		var factoryErr error
		provider, factoryErr = model.ForName(providerName, baseURL, apiKey, modelName)
		if factoryErr != nil {
			return serviceError(asJSON, factoryErr)
		}
	}
	if aware, ok := provider.(interface{ SetWorkspace(string) }); ok {
		aware.SetWorkspace(root)
	}
	// Routed for the same reason as the first call site: the gateway is the
	// runner's provider, and a raw adapter means none of its routing runs
	// (GAP-102).
	provider = routeThroughGateway(provider, modelName, baseURL)

	runner := harnessruntime.NewRunner(harnessruntime.Services{
		Models: provider, Tools: aci.New(root),
		Perms:         perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   store,
		EffectJournal: store,
	}, runID, cp.State.SessionID)
	runner.RestoreFrom(cp)

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	stepsBefore := runner.TurnsDone
	// RunUntilDone carries the runaway-loop guard; a hand-rolled loop here
	// stepped forever against a provider that keeps asking for tools.
	if runErr := runner.RunUntilDone(runCtx); runErr != nil {
		return serviceError(asJSON, runErr)
	}
	continued := runner.TurnsDone > stepsBefore

	result := map[string]any{
		"run_id": runID, "resumed_from": cp.ID, "phase": string(runner.State.Phase),
		"stop_reason": runner.State.StopReason, "resumed": continued,
		"messages_restored": len(cp.Messages),
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("Resumed %s from %s: %s (%d message(s) restored, %d step(s) taken)\n",
		runID, cp.ID, runner.State.Phase, len(cp.Messages), runner.TurnsDone)
	return exitOK
}

func runAgentHandoff(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	runID := f["run"]
	if runID == "" {
		return serviceError(asJSON, fmt.Errorf("handoff requires --run <id>"))
	}
	from := f["from"]
	if from == "" {
		from = "native"
	}
	to := f["to"]
	if to == "" {
		to = "external"
	}
	dir := filepath.Join(root, ".prumo", "runtime", "harness")
	store := checkpoint.New(dir)
	cp, err := store.Latest(runID)
	if err != nil {
		// Allow handoff from live state when no checkpoint exists yet.
		cp = agent.Checkpoint{ID: runID + "-live", RunID: runID, State: agent.NativeAgentState{RunID: runID, ContextManifestID: "ctx-" + runID}}
	}
	b, err := handoff.Build(from, to, cp.State, "rev-live", "handoff for "+runID, nil)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if err := b.Validate(); err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(b))
	}
	fmt.Printf("Handoff %s: %s -> %s (checkpoint %s)\n", b.Handoff.ID, from, to, b.Refs["checkpoint"])
	return exitOK
}

// appendAgentEvent persists one timeline event as JSONL (best-effort;
// event loss never fails the run — checkpoints carry resume state).
func appendAgentEvent(path string, ev agent.AgentEvent) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
	_, _ = daemon.RotateLog(path, 2000)
}

func runAgentEvents(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	runID := f["run"]
	if runID == "" {
		return serviceError(asJSON, fmt.Errorf("events requires --run <id>"))
	}
	path := filepath.Join(root, ".prumo", "runtime", "harness", "events-"+runID+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		return serviceError(asJSON, fmt.Errorf("no event log for run %s: %w", runID, err))
	}
	lines := []string{}
	for _, ln := range splitLines(string(data)) {
		if ln != "" {
			lines = append(lines, ln)
		}
	}
	if asJSON {
		evs := make([]map[string]any, 0, len(lines))
		for _, ln := range lines {
			var m map[string]any
			if err := json.Unmarshal([]byte(ln), &m); err == nil {
				evs = append(evs, m)
			}
		}
		return printEnvelope(protocol.OkEnvelope(map[string]any{"run_id": runID, "events": evs}))
	}
	for _, ln := range lines {
		fmt.Println(ln)
	}
	return exitOK
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func runAgentProtocol(asJSON bool, args []string) int {
	f := agentFlags(args)
	if _, ok := f["manifest"]; ok {
		manifest := harnessprotocol.Manifest()
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(manifest))
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		fmt.Println(string(data))
		return exitOK
	}
	result := map[string]any{
		"version":        harnessprotocol.Version,
		"min_compatible": harnessprotocol.MinCompatible,
		"schemas":        harnessprotocol.Schemas,
		"ops":            harnessprotocol.Ops,
	}
	if v, ok := f["client"]; ok && v != "" {
		server, compatible, err := harnessprotocol.Negotiate(v)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result["server"] = server
		result["compatible"] = compatible
		result["client"] = v
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("protocol %s (min %s)\n", harnessprotocol.Version, harnessprotocol.MinCompatible)
	return exitOK
}

func runAgentServe(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	sock := f["socket"]
	if sock == "" {
		sock = defaultSocket(root)
	}
	store := filepath.Join(root, ".prumo", "runtime", "harness")
	tools, cleanupTools, err := agentTools(root, f)
	if err != nil {
		return serviceError(asJSON, err)
	}
	defer cleanupTools()
	release, err := daemon.AcquireLock(store)
	if err != nil {
		return serviceError(asJSON, err)
	}
	defer release()
	policy, err := permissionPolicy(f)
	if err != nil {
		return serviceError(asJSON, err)
	}
	srv := daemon.New(sock, store, daemon.Deps{Tools: tools, Workspace: root, PermPolicy: policy})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if !asJSON {
		fmt.Printf("serving harness daemon on %s\n", sock)
	}
	if listen, ok := f["listen"]; ok && listen != "" {
		token := f["token"]
		if token == "" {
			if file, ok := f["token-file"]; ok && file != "" {
				data, err := os.ReadFile(file)
				if err != nil {
					return serviceError(asJSON, fmt.Errorf("token file: %w", err))
				}
				token = strings.TrimSpace(string(data))
			}
		}
		if token == "" {
			token = os.Getenv("PRUMO_DAEMON_TOKEN")
		}
		cfg := daemon.RemoteConfig{ListenAddr: listen, CertFile: f["tls-cert"], KeyFile: f["tls-key"], Token: token}
		if !asJSON {
			fmt.Printf("remote endpoint on %s (token-authenticated)\n", listen)
		}
		if err := srv.ServeRemote(ctx, cfg); err != nil {
			return serviceError(asJSON, err)
		}
		return exitOK
	}
	if err := srv.Serve(ctx); err != nil {
		return serviceError(asJSON, err)
	}
	return exitOK
}

// agentTools selects the tool executor: local workspace (default) or
// container-isolated command execution with host-side file tools.
func agentTools(root string, f map[string]string) (harnessruntime.ToolExecutor, func(), error) {
	noop := func() {}
	sandbox := f["sandbox"]
	exec := aci.New(root)
	if _, ok := f["egress-deny"]; ok {
		var allow []string
		if v, ok := f["egress-allow"]; ok && v != "" {
			allow = strings.Split(v, ",")
		}
		exec.Egress = &aci.EgressPolicy{DefaultDeny: true, AllowHosts: allow}
	}
	var base harnessruntime.ToolExecutor = exec
	if sandbox == "container" {
		image := f["sandbox-image"]
		if image == "" {
			return nil, noop, fmt.Errorf("container sandbox requires --sandbox-image <image> (no implicit pulls)")
		}
		rt := aci.DetectContainerRuntime()
		if rt == "" {
			return nil, noop, fmt.Errorf("container sandbox requested but no docker/podman runtime detected")
		}
		runner := aci.CLIRunner{Runtime: rt}
		if v, ok := f["sandbox-runtime"]; ok && v != "" {
			runner.ExtraArgs = []string{"--runtime=" + v}
		}
		if !runner.Available() {
			return nil, noop, fmt.Errorf("container runtime %q unreachable: refusing to run unisolated", rt)
		}
		base = aci.NewContainer(root, image, runner)
	} else if sandbox != "" && sandbox != "local" {
		return nil, noop, fmt.Errorf("unknown sandbox %q (local|container)", sandbox)
	}
	mcpCmd, ok := f["mcp"]
	if !ok || mcpCmd == "" {
		return base, noop, nil
	}
	parts := strings.Fields(mcpCmd)
	tr, err := mcp.StartStdio(context.Background(), parts[0], parts[1:]...)
	if err != nil {
		return nil, noop, fmt.Errorf("mcp server start: %w", err)
	}
	adapter := mcp.Adapter{
		Client: &mcp.Client{Transport: tr},
		Server: toolgateway.MCPServerDescriptor{ID: "cli-mcp", Transport: "stdio", Command: parts[0], Trust: "untrusted"},
	}
	return mcp.Fanout{Base: base, MCP: adapter}, func() { _ = tr.Close() }, nil
}

func daemonClient(f map[string]string) daemon.Client {
	sock := f["socket"]
	if sock == "" {
		root := f["path"]
		if root == "" {
			root = "."
		}
		sock = defaultSocket(root)
	}
	c := daemon.Client{SocketPath: sock}
	if addr, ok := f["remote"]; ok && addr != "" {
		c.RemoteAddr = addr
		c.Token = f["token"]
		if c.Token == "" {
			if file, ok := f["token-file"]; ok && file != "" {
				if data, err := os.ReadFile(file); err == nil {
					c.Token = strings.TrimSpace(string(data))
				}
			}
		}
		if c.Token == "" {
			c.Token = os.Getenv("PRUMO_DAEMON_TOKEN")
		}
		c.CACertFile = f["remote-tls-cert"]
	}
	return c
}

func runAgentPs(asJSON bool, args []string) int {
	res, err := daemonClient(agentFlags(args)).List()
	if err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	runs, _ := res["runs"].([]any)
	if len(runs) == 0 {
		fmt.Println("no runs")
		return exitOK
	}
	for _, r := range runs {
		m, _ := r.(map[string]any)
		line := fmt.Sprintf("%s %s %s", m["run_id"], m["status"], m["phase"])
		if ids, ok := m["pending_permissions"].([]any); ok && len(ids) > 0 {
			// The fingerprint travels with the id because an answer has to
			// quote it: the id names the request, the fingerprint names the
			// call (GAP-107). Without it shown here, `agent approve` asks for
			// something the operator cannot obtain.
			prints, _ := m["permission_fingerprints"].(map[string]any)
			parts := make([]string, 0, len(ids))
			for _, id := range ids {
				name := fmt.Sprint(id)
				if prints != nil {
					if fp, ok := prints[name].(string); ok && fp != "" {
						name += ":" + fp
					}
				}
				parts = append(parts, name)
			}
			line += " waiting-for=" + strings.Join(parts, ",")
		}
		fmt.Println(line)
	}
	return exitOK
}

func runAgentLogs(asJSON bool, args []string) int {
	f := agentFlags(args)
	runID := f["run"]
	if runID == "" {
		return serviceError(asJSON, fmt.Errorf("logs requires --run <id>"))
	}
	res, err := daemonClient(f).Events(runID)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if ok, _ := res["ok"].(bool); !ok {
		return serviceError(asJSON, fmt.Errorf("%v", res["error"]))
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	for _, e := range res["events"].([]any) {
		data, _ := json.Marshal(e)
		fmt.Println(string(data))
	}
	return exitOK
}

func runAgentSteer(asJSON bool, args []string) int {
	f := agentFlags(args)
	runID := f["run"]
	if runID == "" {
		return serviceError(asJSON, fmt.Errorf("steer requires --run <id>"))
	}
	message := f["message"]
	if message == "" {
		return serviceError(asJSON, fmt.Errorf("steer requires --message <text>"))
	}
	res, err := daemonClient(f).Call(map[string]any{"op": "steer", "run_id": runID, "message": message})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if ok, _ := res["ok"].(bool); !ok {
		return serviceError(asJSON, fmt.Errorf("%v", res["error"]))
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	fmt.Printf("Steered %s\n", runID)
	return exitOK
}

// permissionPolicy reads --permission (allow|ask|deny) and --ask-kind (a comma
// separated list of tool kinds). With no flags it is the daemon default, so an
// unconfigured run behaves exactly as it did before the flags existed.
func permissionPolicy(f map[string]string) (perm.Policy, error) {
	policy := daemon.DefaultPermPolicy()
	switch f["permission"] {
	case "":
	case "allow":
		policy.DefaultAction = agent.PermissionAllow
	case "ask":
		policy.DefaultAction = agent.PermissionAsk
	case "deny":
		policy.DefaultAction = agent.PermissionDeny
	default:
		return policy, fmt.Errorf("unknown --permission %q (allow|ask|deny)", f["permission"])
	}
	if v, ok := f["ask-kind"]; ok && v != "" {
		policy.AskKinds = strings.Split(v, ",")
	}
	return policy, nil
}

// destinationPolicy decides which network a provider may be reached on.
//
// The default refuses loopback, private ranges and the cloud metadata endpoint,
// because --base-url is a command line value and the harness sends the
// conversation and the API key wherever it names. A local gateway is a real
// need, so --allow-local-model opts into it explicitly rather than the default
// quietly allowing it.
func destinationPolicy(f map[string]string) model.DestinationPolicy {
	if _, ok := f["allow-local-model"]; ok {
		return model.LocalDevelopmentDestinationPolicy()
	}
	return model.DefaultDestinationPolicy()
}

func checkModelDestination(baseURL string, f map[string]string) error {
	return model.ValidateDestinationURL(baseURL, destinationPolicy(f))
}

// runAgentPermission answers a permission request on a live daemon. Approving
// and denying share one path: the daemon sees a different decision, not a
// different operation.
func runAgentPermission(asJSON bool, args []string, allow bool) int {
	f := agentFlags(args)
	verb := "deny"
	if allow {
		verb = "approve"
	}
	runID := f["run"]
	if runID == "" {
		return serviceError(asJSON, fmt.Errorf("%s requires --run <id>", verb))
	}
	requestID := f["request"]
	if requestID == "" {
		return serviceError(asJSON, fmt.Errorf("%s requires --request <id> (see `prumo agent ps`)", verb))
	}
	// The approval is bound to the content, so the approver quotes the
	// fingerprint `agent ps` showed them. Without it the daemon refuses, because
	// a request id names a request rather than a specific call (GAP-107).
	fingerprint := f["fingerprint"]
	if fingerprint == "" {
		return serviceError(asJSON, fmt.Errorf("%s requires --fingerprint <hash> (see `prumo agent ps`)", verb))
	}
	op := "deny"
	if allow {
		op = "approve"
	}
	res, err := daemonClient(f).Call(map[string]any{
		"op": op, "run_id": runID, "request_id": requestID,
		"fingerprint": fingerprint, "reason": f["reason"],
	})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if ok, _ := res["ok"].(bool); !ok {
		return serviceError(asJSON, fmt.Errorf("%v", res["error"]))
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	decision := "Denied"
	if allow {
		decision = "Approved"
	}
	fmt.Printf("%s %s (%s)\n", decision, requestID, runID)
	return exitOK
}

func runAgentStop(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	store := filepath.Join(root, ".prumo", "runtime", "harness")
	if err := daemon.Stop(store); err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{"stopped": true}))
	}
	fmt.Println("daemon stopped")
	return exitOK
}

func runAgentSchedule(asJSON bool, args []string) int {
	f := agentFlags(args)
	goal := f["goal"]
	if goal == "" {
		return serviceError(asJSON, fmt.Errorf("schedule requires --goal <text>"))
	}
	var every float64
	if v, ok := f["every"]; ok {
		fmt.Sscanf(v, "%f", &every)
	}
	res, err := daemonClient(f).Call(map[string]any{
		"op": "schedule", "goal": goal, "provider": f["provider"],
		"every_secs": every, "job_id": f["job"],
	})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if ok, _ := res["ok"].(bool); !ok {
		return serviceError(asJSON, fmt.Errorf("%v", res["error"]))
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	fmt.Printf("Scheduled %s\n", res["job_id"])
	return exitOK
}

func runAgentUnschedule(asJSON bool, args []string) int {
	f := agentFlags(args)
	if f["job"] == "" {
		return serviceError(asJSON, fmt.Errorf("unschedule requires --job <id>"))
	}
	res, err := daemonClient(f).Call(map[string]any{"op": "unschedule", "job_id": f["job"]})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if ok, _ := res["ok"].(bool); !ok {
		return serviceError(asJSON, fmt.Errorf("%v", res["error"]))
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	fmt.Printf("Unscheduled %s\n", f["job"])
	return exitOK
}

func runAgentJobs(asJSON bool, args []string) int {
	res, err := daemonClient(agentFlags(args)).Call(map[string]any{"op": "jobs"})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	jobs, _ := res["jobs"].([]any)
	if len(jobs) == 0 {
		fmt.Println("no jobs")
		return exitOK
	}
	for _, j := range jobs {
		m, _ := j.(map[string]any)
		fmt.Printf("%s every=%vs goal=%s\n", m["job_id"], m["every_secs"], m["goal"])
	}
	return exitOK
}

// runAgentPromote turns a persisted planning session into a build run.
// Without --start it prints the promotion (goal + refs); with --start it
// launches the run immediately.
func runAgentPromote(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	var data []byte
	var err error
	if file := f["session-file"]; file != "" {
		data, err = os.ReadFile(file)
	} else if id := f["session"]; id != "" {
		data, err = os.ReadFile(filepath.Join(root, ".ai", "plan", "sessions", id+".json"))
	} else {
		return serviceError(asJSON, fmt.Errorf("promote requires --session <id> or --session-file <path>"))
	}
	if err != nil {
		return serviceError(asJSON, err)
	}
	runID := f["run"]
	if runID == "" {
		runID = "R-promote-1"
	}
	promo, err := handoff.PromoteBytes(data, runID)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if _, ok := f["start"]; !ok {
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{
				"session_id": promo.SessionID, "run_id": promo.RunID, "goal": promo.GoalText(),
				"decisions": promo.DecisionIDs, "open": promo.OpenIDs, "sources": promo.Sources,
			}))
		}
		fmt.Printf("Promotion %s → %s\n%s\n", promo.SessionID, promo.RunID, promo.Summary)
		return exitOK
	}
	newArgs := []string{"run", "--goal", promo.GoalText(), "--path", root, "--run", runID}
	for _, k := range []string{"provider", "model", "base-url", "max-turns", "sandbox", "sandbox-image", "strict", "gates", "context-budget", "budget-tokens", "budget-usd", "budget-tools"} {
		if v, ok := f[k]; ok {
			if v == "" {
				newArgs = append(newArgs, "--"+k)
			} else {
				newArgs = append(newArgs, "--"+k, v)
			}
		}
	}
	return runAgentRun(asJSON, newArgs)
}

// runAgentACP exposes the harness as an ACP v1 agent over stdio, backed
// by the local daemon (point --socket/--path at a running daemon).
func runAgentACP(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	sock := f["socket"]
	if sock == "" {
		sock = defaultSocket(root)
	}
	srv := acpserver.NewServer(acpserver.DaemonBackend{Daemon: daemon.Client{SocketPath: sock}})
	if !asJSON {
		fmt.Fprintln(os.Stderr, "prumo acp agent on stdio (ACP v1 subset)")
	}
	if err := srv.Serve(context.Background()); err != nil {
		return serviceError(asJSON, err)
	}
	return exitOK
}

// runAgentGC enforces retention: prune checkpoints, collect aged orphans.
// Provenance never dangles: runs owning checkpoints keep everything.
func runAgentGC(asJSON bool, args []string) int {
	f := agentFlags(args)
	root := f["path"]
	if root == "" {
		root = "."
	}
	policy := checkpoint.DefaultRetention()
	if v, ok := f["keep"]; ok {
		fmt.Sscanf(v, "%d", &policy.KeepCheckpoints)
	}
	if v, ok := f["max-age-days"]; ok {
		fmt.Sscanf(v, "%d", &policy.MaxAgeDays)
	}
	rep, err := checkpoint.New(filepath.Join(root, ".prumo", "runtime", "harness")).GC(policy)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"checkpoints_pruned": rep.CheckpointsPruned, "artifacts_removed": rep.ArtifactsRemoved,
		}))
	}
	fmt.Printf("pruned %d checkpoints, removed %d orphan artifacts\n", rep.CheckpointsPruned, len(rep.ArtifactsRemoved))
	return exitOK
}

// runAgentModels asks a live daemon what a provider can serve. The catalogue
// belongs to the harness: a client that carried its own would be asserting what
// it cannot verify.
func runAgentModels(asJSON bool, args []string) int {
	f := agentFlags(args)
	res, err := daemonClient(f).Models(f["provider"], f["base-url"], f["model"])
	if err != nil {
		return serviceError(asJSON, err)
	}
	if ok, _ := res["ok"].(bool); !ok {
		return serviceError(asJSON, fmt.Errorf("%v", res["error"]))
	}
	models, _ := res["models"].([]any)
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	if len(models) == 0 {
		fmt.Println("no models reported")
		return exitOK
	}
	for _, m := range models {
		fmt.Println(m)
	}
	return exitOK
}

func runAgentProviders(asJSON bool, args []string) int {
	f := agentFlags(args)
	prober := extagent.Prober{OpenCodeURL: f["opencode-url"]}
	rows := prober.Matrix(context.Background())
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{"providers": rows}))
	}
	for _, r := range rows {
		state := "unavailable"
		if r.Available {
			state = "available"
		}
		ver := r.Version
		if ver == "" {
			ver = "-"
		}
		fmt.Printf("%-16s %-6s %-11s %-12s %s\n", r.Name, r.Kind, state, ver, r.Detail)
	}
	return exitOK
}

func runAgentDiff(asJSON bool, args []string) int {
	f := agentFlags(args)
	runID := f["run"]
	if runID == "" {
		runID = f["run-id"]
	}
	if runID == "" && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		runID = args[0]
	}
	path := f["path"]
	if path == "" && len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		path = args[1]
	}
	res, err := daemonClient(f).Diff(runID, path)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if ok, _ := res["ok"].(bool); !ok {
		return serviceError(asJSON, fmt.Errorf("%v", res["error"]))
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}
	content, _ := res["content"].(string)
	if content != "" {
		fmt.Println(content)
	} else {
		fmt.Printf("[%s] %s\n", res["kind"], res["path"])
	}
	return exitOK
}

// routeThroughGateway puts the model gateway in front of a provider.
//
// This is the caller the gateway never had. It was built with selection,
// fallback, retry, a circuit breaker and a quota filter, and the runner was
// handed the raw adapter, so a run reached exactly one provider and none of that
// machinery ever executed (GAP-102). Wrapping here rather than inside the runtime
// keeps routing a property of how the process is wired: a caller that wants a
// bare adapter can still pass one, and a caller that wants routing asks for it.
//
// The target's privacy class is derived from where the data would actually go,
// not from the provider's name: the same adapter reaches a loopback endpoint in
// development and a vendor's datacenter in production, and a firewall that keyed
// on the adapter would call one of those safe and the other safe too.
func routeThroughGateway(provider model.Provider, modelName, baseURL string) model.Provider {
	g := gateway.New()
	g.Register(provider)
	g.DeclareTarget(gateway.RouteTarget{
		Provider: provider.Name(),
		Model:    modelName,
		Tools:    true,
		Privacy:  privacyClassFor(baseURL),
	})
	return g
}

// privacyClassFor classifies a destination for the gateway's LocalOnly filter.
//
// A loopback or private address is local. Anything else is external, because
// "we could not classify it" has to resolve to the answer that does not send
// restricted data somewhere nobody approved.
func privacyClassFor(baseURL string) string {
	if baseURL == "" {
		return "external"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "external"
	}
	host := parsed.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return "local"
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return "local"
	}
	return "external"
}
