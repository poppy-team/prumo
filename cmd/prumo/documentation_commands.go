package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/agentsurface"
	"github.com/raillen/prumo/internal/docintel"
	"github.com/raillen/prumo/internal/doclifecycle"
	"github.com/raillen/prumo/internal/docpublish"
	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/gauntlet"
	"github.com/raillen/prumo/internal/protocol"
)

func runDocumentation(asJSON bool, args []string) int {
	if len(args) < 2 || args[0] != "docs" {
		fmt.Fprintln(os.Stderr, "error: expected docs <contracts|profiles|audit|readiness|impact|delta|contradictions|authority|agents>")
		return exitUsage
	}
	root := "."
	goal := ""
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --path requires a value")
				return exitUsage
			}
			root = args[i+1]
			i++
		case "--goal":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --goal requires a value")
				return exitUsage
			}
			goal = args[i+1]
			i++
		}
	}
	registry, err := docengine.LoadRegistry(repoRoot())
	if err != nil {
		return serviceError(asJSON, err)
	}
	var result any
	exitCode := exitOK
	switch args[1] {
	case "contracts":
		if len(args) > 2 && args[2] == "show" {
			if len(args) < 4 {
				fmt.Fprintln(os.Stderr, "error: docs contracts show requires id")
				return exitUsage
			}
			contract, ok := registry.Contracts[args[3]]
			if !ok {
				return serviceError(asJSON, fmt.Errorf("unknown documentation contract: %s", args[3]))
			}
			result = contract
		} else {
			ids := []string{}
			for id := range registry.Contracts {
				ids = append(ids, id)
			}
			sortStrings(ids)
			result = ids
		}
	case "profiles":
		if len(args) > 2 && args[2] == "show" {
			if len(args) < 4 {
				fmt.Fprintln(os.Stderr, "error: docs profiles show requires id")
				return exitUsage
			}
			profile, ok := registry.Profiles[args[3]]
			if !ok {
				return serviceError(asJSON, fmt.Errorf("unknown documentation profile: %s", args[3]))
			}
			result = profile
		} else {
			ids := []string{}
			for id := range registry.Profiles {
				ids = append(ids, id)
			}
			sortStrings(ids)
			result = ids
		}
	case "audit":
		report, err := docengine.Audit(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result = report
	case "readiness":
		report, err := docengine.Readiness(root, goal)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result = report
	case "impact":
		changed := []string{}
		if len(args) > 2 && args[2] != "--path" {
			changed = append(changed, args[2:]...)
		}
		impact, err := docengine.AnalyzeImpact(root, changed)
		if err != nil {
			return serviceError(asJSON, err)
		}
		impact = appendAgentSurfaceImpacts(root, changed, impact)
		result = impact
	case "metrics":
		metrics, err := docintel.Collect(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result = metrics
	case "adopt":
		sub := "inspect"
		if len(args) > 2 && args[2] != "--path" {
			sub = args[2]
		}
		target := root
		for i := 3; i < len(args); i++ {
			if args[i] == "--target" && i+1 < len(args) {
				target = args[i+1]
				i++
			}
		}
		if sub == "propose" {
			plan, err := docintel.Propose(target)
			if err != nil {
				return serviceError(asJSON, err)
			}
			result = plan
		} else {
			report, err := docintel.Inspect(target)
			if err != nil {
				return serviceError(asJSON, err)
			}
			result = report
		}
	case "translate", "media", "release":
		lifecycleResult, code := runDocsLifecycle(asJSON, args, root, args[1])
		result = lifecycleResult
		exitCode = code
	case "verify":
		// --strict also requires semantic readiness, so the CI gate cannot pass
		// on lexical coverage alone (W19.9).
		strict := false
		for _, arg := range args[2:] {
			if arg == "--strict" {
				strict = true
			}
		}
		verify := docengine.VerifyDocs
		if strict {
			verify = docengine.VerifyDocsStrict
		}
		report, err := verify(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		// The lifecycle plane (terminology, version policy) is merged here
		// because it classifies documents through the authority gate; keeping
		// the merge at the CLI boundary avoids an import cycle in the engine.
		lifecycleFindings, err := lifecycleVerifyFindings(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		report.Checks = append(report.Checks, "terminology", "version-policy")
		report.Findings = append(report.Findings, lifecycleFindings...)
		sortVerifyFindings(report.Findings)
		report.OK = len(report.Findings) == 0
		result = report
		if !report.OK {
			exitCode = exitValidation
		}
	case "gauntlet":
		runRecord, code := runDocsGauntlet(asJSON, args, root, goal)
		result = runRecord
		exitCode = code
	case "build", "manifest", "query", "doctor":
		published, code := runDocsPublishing(asJSON, args, root, args[1])
		result = published
		exitCode = code
	case "site":
		// `prumo docs site build|verify` names the same projections as
		// build/doctor; the grouped form keeps the documented surface honest.
		action := "build"
		rest := args[2:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
			action = rest[0]
			rest = rest[1:]
		}
		if action == "verify" {
			action = "doctor"
		}
		published, code := runDocsPublishing(asJSON, append([]string{args[0], "site", ""}, rest...), root, action)
		result = published
		exitCode = code
	case "plan":
		planResult, code := runDocsPlan(asJSON, args, root, goal)
		result = planResult
		exitCode = code
	case "explain":
		explainResult, code := runDocsExplain(asJSON, args, root, goal)
		result = explainResult
		exitCode = code
	case "delta":
		return runDocsDelta(asJSON, args, root, goal)
	case "contradictions":
		findings, err := docengine.DetectContradictions(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result = findings
	case "authority":
		report, err := docengine.CheckAuthority(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result = report
		if !report.OK {
			exitCode = exitValidation
		}
	case "agents":
		agentsResult, code := runDocsAgents(asJSON, args, root)
		result = agentsResult
		exitCode = code
	case "resources":
		resources, code := runDocsResources(asJSON, args, root)
		result = resources
		exitCode = code
	case "mutations":
		mutations, code := runDocsMutations(asJSON, args, root)
		result = mutations
		exitCode = code
	case "glossary":
		status, err := doclifecycle.CheckGlossary(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result = status
		if !status.OK {
			exitCode = exitValidation
		}
	case "version":
		report, err := doclifecycle.CheckVersionPolicy(root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result = report
		if !report.OK {
			exitCode = exitValidation
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported docs command: %s\n", args[1])
		return exitUsage
	}
	if asJSON {
		if code := printEnvelope(protocol.OkEnvelope(result)); code != exitOK {
			return code
		}
		return exitCode
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	return exitCode
}

// runDocsLifecycle reports and verifies the post-generation lifecycle:
// translation currency, media currency and release readiness (W20).
func runDocsLifecycle(asJSON bool, args []string, root, action string) (any, int) {
	sub := "status"
	if len(args) > 2 {
		sub = args[2]
	}
	switch action {
	case "translate":
		statuses, err := doclifecycle.TranslationStatuses(root)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		if sub == "verify" {
			findings, err := doclifecycle.VerifyPseudoLoc(root)
			if err != nil {
				return nil, serviceError(asJSON, err)
			}
			payload := map[string]any{"units": statuses, "findings": findings, "ok": len(findings) == 0}
			if len(findings) > 0 {
				return payload, exitValidation
			}
			return payload, exitOK
		}
		return map[string]any{"units": statuses, "stale": staleUnits(statuses)}, exitOK
	case "media":
		statuses, err := doclifecycle.MediaStatuses(root)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		if sub == "verify" {
			stale := staleUnits(statuses)
			payload := map[string]any{"units": statuses, "stale": stale, "ok": len(stale) == 0}
			if len(stale) > 0 {
				return payload, exitValidation
			}
			return payload, exitOK
		}
		return map[string]any{"units": statuses, "stale": staleUnits(statuses)}, exitOK
	case "release":
		version := protocol.CLIVersion
		for i := 3; i < len(args); i++ {
			if args[i] == "--version" && i+1 < len(args) {
				version = args[i+1]
				i++
			}
		}
		report, err := doclifecycle.CheckRelease(root, version)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		if !report.Ready {
			return report, exitValidation
		}
		return report, exitOK
	}
	return nil, exitUsage
}

// lifecycleVerifyFindings projects the terminology and version-policy checks
// into the verification report, so `docs verify` is the one gate an operator
// runs (W20.2, W20.6).
func lifecycleVerifyFindings(root string) ([]docengine.VerifyFinding, error) {
	findings := []docengine.VerifyFinding{}
	documents, err := docengine.ScanDocuments(root)
	if err != nil {
		return nil, err
	}
	glossary, err := doclifecycle.LoadGlossary(root)
	if err != nil {
		return nil, err
	}
	for _, finding := range doclifecycle.GlossaryFindings(root, glossary, documents) {
		findings = append(findings, docengine.VerifyFinding{Kind: "glossary:" + finding.Kind,
			Path: finding.Target, Detail: finding.Detail})
	}
	policy, err := doclifecycle.CheckVersionPolicy(root)
	if err != nil {
		return nil, err
	}
	for _, finding := range policy.Findings {
		findings = append(findings, docengine.VerifyFinding{Kind: "version-policy:" + finding.Kind,
			Path: finding.Target, Detail: finding.Detail})
	}
	return findings, nil
}

func sortVerifyFindings(findings []docengine.VerifyFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Line < findings[j].Line
	})
}

func staleUnits(statuses []doclifecycle.UnitStatus) []string {
	stale := []string{}
	for _, status := range statuses {
		if status.Stale {
			stale = append(stale, status.ID)
		}
	}
	return stale
}

// runDocsGauntlet runs the deterministic documentation gauntlet and records the
// run. The model critic stays off unless a policy turns it on; a deterministic
// failure always stops the run (W19).
func runDocsGauntlet(asJSON bool, args []string, root, goal string) (any, int) {
	write := false
	for i := 3; i < len(args); i++ {
		if args[i] == "--write" {
			write = true
		}
	}
	report, err := docengine.VerifyDocs(root)
	if err != nil {
		return nil, serviceError(asJSON, err)
	}
	policy := gauntlet.DefaultPolicy()
	problems := make([]gauntlet.Finding, 0, len(report.Findings))
	for i, finding := range report.Findings {
		problems = append(problems, gauntlet.Finding{
			ID:        fmt.Sprintf("%s-%d", finding.Kind, i+1),
			Dimension: gauntlet.DimensionFor(finding.Kind),
			Severity:  gauntlet.SeverityFor(finding.Kind),
			Status:    "open",
			Critic:    "deterministic",
			Detail:    fmt.Sprintf("%s:%d %s", finding.Path, finding.Line, finding.Detail),
			Evidence:  []string{finding.Path},
		})
	}
	runID := fmt.Sprintf("docs-%s", time.Now().UTC().Format("20060102T150405"))
	record := gauntlet.BuildRun(policy, goal, runID, report.Checks, problems)
	payload := map[string]any{"report": report, "run": record}
	if write {
		rel, err := gauntlet.Save(root, record)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		payload["written"] = rel
	}
	if !record.Passed() {
		return payload, exitValidation
	}
	return payload, exitOK
}

// runDocsPublishing projects the documentation graph into publishing and
// AI-retrieval surfaces (W18). build/manifest/query/doctor share one graph.
func runDocsPublishing(asJSON bool, args []string, root, action string) (any, int) {
	out := ""
	locale := "en"
	versioned := false
	limit := 10
	selected := []string{}
	rest := []string{}
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --out requires a value")
				return nil, exitUsage
			}
			out = args[i+1]
			i++
		case "--renderers":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --renderers requires a comma-separated list")
				return nil, exitUsage
			}
			for _, name := range strings.Split(args[i+1], ",") {
				if trimmed := strings.TrimSpace(name); trimmed != "" {
					selected = append(selected, trimmed)
				}
			}
			i++
		case "--locale":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --locale requires a value")
				return nil, exitUsage
			}
			locale = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --limit requires a value")
				return nil, exitUsage
			}
			if n, err := strconv.Atoi(args[i+1]); err == nil {
				limit = n
			}
			i++
		case "--versioned":
			versioned = true
		default:
			rest = append(rest, args[i])
		}
	}
	opts := docpublish.Options{Version: protocol.CLIVersion, Locale: locale, VersionedRoutes: versioned}
	switch action {
	case "build":
		result, err := docpublish.BuildGraph(root, out, opts, selected)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		return result, exitOK
	case "manifest":
		graph, err := docpublish.LoadWithOptions(root, opts)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		return map[string]any{"version": graph.Version, "locale": graph.Locale,
			"routes": graph.Routes(), "pages": len(graph.Pages)}, exitOK
	case "query":
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "error: docs query requires a search term")
			return nil, exitUsage
		}
		results, err := docpublish.Query(root, strings.Join(rest, " "), limit)
		if err != nil {
			return nil, serviceError(asJSON, fmt.Errorf("search index unavailable; run `prumo docs build` first: %w", err))
		}
		return results, exitOK
	case "doctor":
		graph, err := docpublish.LoadWithOptions(root, opts)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		findings := graph.Doctor()
		report := map[string]any{"pages": len(graph.Pages), "findings": findings, "ok": len(findings) == 0}
		if len(findings) > 0 {
			return report, exitValidation
		}
		return report, exitOK
	}
	return nil, exitUsage
}

// runDocsPlan is the Goal documentation preflight/postflight (W17.5, W17.7,
// W17.8): without --changed it stores the preflight plan; with --changed it
// reconciles the stored plan against the impact actually observed.
func runDocsPlan(asJSON bool, args []string, root, goal string) (any, int) {
	if strings.TrimSpace(goal) == "" {
		fmt.Fprintln(os.Stderr, "error: docs plan requires --goal <id>")
		return nil, exitUsage
	}
	changed := []string{}
	reconcile := false
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--changed":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --changed requires a comma-separated list")
				return nil, exitUsage
			}
			for _, path := range strings.Split(args[i+1], ",") {
				if trimmed := strings.TrimSpace(path); trimmed != "" {
					changed = append(changed, trimmed)
				}
			}
			i++
		case "--reconcile":
			reconcile = true
		}
	}
	if reconcile {
		plan, err := docengine.LoadPlan(root, goal)
		if err != nil {
			return nil, serviceError(asJSON, fmt.Errorf("no stored plan for goal %s: %w", goal, err))
		}
		registry, err := docengine.LoadRegistry(repoRoot())
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		bindings, err := docengine.LoadBindings(root)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		actual := docengine.AnalyzeImpactsIn(root, registry, bindings, changed, nil)
		return docengine.Reconcile(plan, actual), exitOK
	}
	plan, err := docengine.BuildPlan(root, goal, changed, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return nil, serviceError(asJSON, err)
	}
	if err := docengine.SavePlan(root, plan); err != nil {
		return nil, serviceError(asJSON, err)
	}
	code := exitOK
	if len(plan.Findings) > 0 {
		code = exitValidation
	}
	return plan, code
}

// runDocsExplain explains one planned obligation or documented contract.
func runDocsExplain(asJSON bool, args []string, root, goal string) (any, int) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "error: docs explain requires an obligation or contract id")
		return nil, exitUsage
	}
	target := args[2]
	if strings.TrimSpace(goal) != "" {
		if plan, err := docengine.LoadPlan(root, goal); err == nil {
			if explanation, err := plan.Explain(target); err == nil {
				return map[string]any{"goal": goal, "target": target, "explanation": explanation}, exitOK
			}
		}
	}
	sources, contract, err := docengine.ExplainBinding(root, target)
	if err != nil {
		return nil, serviceError(asJSON, err)
	}
	return map[string]any{"target": target, "contract": contract, "sources": sources}, exitOK
}

// appendAgentSurfaceImpacts folds compiled agent surfaces into a documentation
// impact analysis: changing rule sources, the IR or a surface's scope makes the
// affected projections stale (W16.15). The documentation engine stays unaware
// of the agent surface package; the CLI composes them.
func appendAgentSurfaceImpacts(root string, changed []string, impacts []docengine.Impact) []docengine.Impact {
	ir, err := agentsurface.LoadIR(root)
	if err != nil || len(changed) == 0 {
		return impacts
	}
	for _, surface := range agentsurface.AffectedSurfaces(ir, changed) {
		impacts = append(impacts, docengine.Impact{
			ContractID: "agent-surface:" + surface.Adapter,
			Documents:  []string{surface.Path},
			Reason:     "compiled agent surface projects changed knowledge",
			Severity:   "medium",
		})
	}
	sort.Slice(impacts, func(i, j int) bool { return impacts[i].ContractID < impacts[j].ContractID })
	return impacts
}

// runDocsAgents compiles and verifies the agent instruction surfaces. build
// regenerates managed regions in place and vendor projections under runtime;
// verify is the context-rot CI gate (W16.14).
func runDocsAgents(asJSON bool, args []string, root string) (any, int) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "error: expected docs agents <build|verify|explain>")
		return nil, exitUsage
	}
	switch args[2] {
	case "build":
		out := ""
		for i := 3; i < len(args); i++ {
			if args[i] == "--out" {
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "error: --out requires a value")
					return nil, exitUsage
				}
				out = args[i+1]
				i++
			}
		}
		result, err := agentsurface.Build(root, out)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		return result, exitOK
	case "verify":
		report, err := agentsurface.Verify(root)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		if !report.OK {
			return report, exitValidation
		}
		return report, exitOK
	case "explain":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "error: docs agents explain requires an adapter")
			return nil, exitUsage
		}
		scope := ""
		if len(args) > 4 {
			scope = args[4]
		}
		ir, err := agentsurface.LoadIR(root)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		surface, err := agentsurface.Explain(ir, args[3], scope)
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		return surface, exitOK
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported docs agents command: %s\n", args[2])
		return nil, exitUsage
	}
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func documentationRoot(path string) string {
	if path == "" {
		return "."
	}
	return filepath.Clean(path)
}
