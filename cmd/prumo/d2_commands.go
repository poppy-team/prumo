package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/raillen/prumo/internal/environment"
	"github.com/raillen/prumo/internal/modelregistry"
	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/toolgateway"
	"github.com/raillen/prumo/internal/tooling"
)

func defaultToolRegistry() *toolgateway.Registry {
	r := toolgateway.NewRegistry()
	// The native coding tools come from the catalogue that implements them, so
	// the registry cannot describe a tool that does not exist. The four
	// hand-written descriptors that used to sit here had no implementation and
	// were advertised by `prumo tool list` anyway (GAP-158).
	toolgateway.RegisterACI(r)

	// The analysis tools below are different in kind: each is a verb implemented
	// by a case in runTool, so the registry describes something that exists. They
	// are listed here because `tool list` and `tool evaluate` operate on the
	// registry, and an analysis verb missing from it is one the policy surface
	// cannot reason about.
	for _, d := range analysisDescriptors() {
		r.Register(d)
	}
	return r
}

// analysisDescriptors are the repository-analysis verbs runTool implements.
func analysisDescriptors() []toolgateway.Descriptor {
	return []toolgateway.Descriptor{
		{ID: "scan-secrets", Version: 1, Description: "Scan source files for secrets and credentials",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"security", "scan"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 15000, OutputLimit: 102400},
		{ID: "scan-headers", Version: 1, Description: "Check source files for security-relevant headers",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"security", "headers"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 15000, OutputLimit: 102400},
		{ID: "stride", Version: 1, Description: "Run a STRIDE threat model over the project",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"security", "threat-model"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 20000, OutputLimit: 102400},
		{ID: "security-checklist", Version: 1, Description: "Check the project against the security checklist",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"security", "checklist"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 15000, OutputLimit: 102400},
		{ID: "check-permissions", Version: 1, Description: "Check file permissions",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"security", "permissions"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 10000, OutputLimit: 102400},
		{ID: "analyze-complexity", Version: 1, Description: "Analyse code complexity hotspots",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"code", "complexity"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 20000, OutputLimit: 102400},
		{ID: "check-bare-errors", Version: 1, Description: "Find discarded errors",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"code", "quality"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 15000, OutputLimit: 102400},
		{ID: "check-subprocesses", Version: 1, Description: "Find subprocess invocations",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"code", "process"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 10000, OutputLimit: 102400},
		{ID: "check-escape-hatches", Version: 1, Description: "Find escape hatches that bypass policy",
			Kind: toolgateway.ReadOnly, Trust: "core", Capabilities: []string{"code", "policy"},
			FilesystemScope: []string{"project-root"}, TimeoutMS: 10000, OutputLimit: 102400},
	}
}

func defaultModelRegistry() *modelregistry.Registry {
	r := modelregistry.NewRegistry()
	r.Register(modelregistry.Descriptor{
		ID:               "local-default",
		Version:          1,
		Provider:         "local",
		Privacy:          modelregistry.Local,
		ContextTokens:    8192,
		StructuredOutput: true,
		ToolUse:          true,
		LatencyClass:     "fast",
	})
	r.Register(modelregistry.Descriptor{
		ID:               "cloud-standard",
		Version:          1,
		Provider:         "trusted-cloud",
		Privacy:          modelregistry.Trusted,
		ContextTokens:    32768,
		StructuredOutput: true,
		ToolUse:          true,
		LatencyClass:     "standard",
	})
	r.Register(modelregistry.Descriptor{
		ID:               "cloud-deep",
		Version:          1,
		Provider:         "external-cloud",
		Privacy:          modelregistry.External,
		ContextTokens:    65536,
		StructuredOutput: true,
		ToolUse:          true,
		LatencyClass:     "deep",
	})
	return r
}

func runTool(asJSON bool, root string, args []string) int {
	if len(args) == 0 {
		return exitUsage
	}
	reg := defaultToolRegistry()
	switch args[0] {
	case "list":
		tools := reg.List()
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(tools))
		}
		fmt.Printf("%-15s %-15s %-8s %s\n", "ID", "KIND", "TRUST", "DESCRIPTION")
		fmt.Println("---------------------------------------------------------------")
		for _, t := range tools {
			fmt.Printf("%-15s %-15s %-8s %s\n", t.ID, t.Kind, t.Trust, t.Description)
		}
		return exitOK
	case "inspect":
		if len(args) < 2 {
			fmt.Println("error: tool inspect requires tool ID")
			return exitUsage
		}
		tool, ok := reg.Get(args[1])
		if !ok {
			fmt.Printf("error: tool not found: %s\n", args[1])
			return exitValidation
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(tool))
		}
		fmt.Printf("Tool: %s (version: %d)\nKind: %s\nTrust: %s\nCapabilities: %v\n",
			tool.ID, tool.Version, tool.Kind, tool.Trust, tool.Capabilities)
		return exitOK
	case "evaluate":
		if len(args) < 2 {
			fmt.Println("error: tool evaluate requires tool ID")
			return exitUsage
		}
		tool, ok := reg.Get(args[1])
		if !ok {
			fmt.Printf("error: tool not found: %s\n", args[1])
			return exitValidation
		}
		target := ""
		if len(args) > 2 {
			target = args[2]
		}
		safe := hasFlag(args, "--safe")
		dec := toolgateway.Evaluate(tool, root, target, safe)
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(dec))
		}
		if dec.Allowed {
			fmt.Printf("Allowed: %s\n", dec.Reason)
			return exitOK
		}
		fmt.Printf("Denied: %s\n", dec.Reason)
		return exitValidation
	case "scan-secrets":
		path := root
		if len(args) > 1 {
			path = args[1]
		}
		findings, err := tooling.ScanSecrets(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(findings))
		}
		if len(findings) == 0 {
			fmt.Println("No secrets detected. Codebase clean.")
			return exitOK
		}
		fmt.Printf("Detected %d potential secret(s):\n", len(findings))
		for _, f := range findings {
			fmt.Printf("  %s:%d [%s] %s\n", f.File, f.Line, f.Rule, f.Preview)
		}
		return exitValidation
	case "scan-headers":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: scan-headers requires target URL")
			return exitUsage
		}
		targetURL := args[1]
		findings, err := tooling.ScanHeaders(targetURL)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(findings))
		}
		for _, f := range findings {
			fmt.Printf("[%-7s] %-28s : %s\n", f.Status, f.Header, f.Detail)
		}
		return exitOK
	case "stride":
		component := "SystemComponent"
		if len(args) > 1 {
			component = args[1]
		}
		data := tooling.GenerateStrideTemplate(component)
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(data))
		}
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(out))
		return exitOK
	case "security-checklist":
		pr := "1"
		if len(args) > 1 {
			pr = args[1]
		}
		checklist := tooling.GenerateSecurityChecklist(pr)
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]string{"checklist": checklist}))
		}
		fmt.Println(checklist)
		return exitOK
	case "check-permissions":
		path := root
		if len(args) > 1 {
			path = args[1]
		}
		findings, err := tooling.CheckPermissions(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(findings))
		}
		if len(findings) == 0 {
			fmt.Println("Permissions clean: no world-writable files found.")
			return exitOK
		}
		fmt.Printf("Found %d permission warning(s):\n", len(findings))
		for _, f := range findings {
			fmt.Printf("  %s (%s): %s\n", f.Path, f.Mode, f.Reason)
		}
		return exitValidation
	case "analyze-complexity":
		path := root
		if len(args) > 1 {
			path = args[1]
		}
		findings, err := tooling.AnalyzeComplexity(path, 60, 600)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(findings))
		}
		if len(findings) == 0 {
			fmt.Println("Complexity clean: all functions and files within thresholds.")
			return exitOK
		}
		fmt.Printf("Found %d complexity warning(s):\n", len(findings))
		for _, f := range findings {
			fmt.Printf("  %s:%d [%s] %s\n", f.File, f.Line, f.Metric, f.Message)
		}
		return exitValidation
	case "check-bare-errors":
		path := root
		if len(args) > 1 {
			path = args[1]
		}
		findings, err := tooling.CheckBareErrors(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(findings))
		}
		if len(findings) == 0 {
			fmt.Println("Clean: no bare errors or swallowed exceptions found.")
			return exitOK
		}
		fmt.Printf("Found %d bare error/suppression instance(s):\n", len(findings))
		for _, f := range findings {
			fmt.Printf("  %s:%d [%s] %s\n", f.File, f.Line, f.Pattern, f.Preview)
		}
		return exitValidation
	case "check-subprocesses":
		path := root
		if len(args) > 1 {
			path = args[1]
		}
		findings, err := tooling.CheckSubprocesses(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(findings))
		}
		if len(findings) == 0 {
			fmt.Println("Clean: no hazardous subprocess patterns found.")
			return exitOK
		}
		fmt.Printf("Found %d subprocess warning(s):\n", len(findings))
		for _, f := range findings {
			fmt.Printf("  %s:%d [%s] %s\n", f.File, f.Line, f.Pattern, f.Preview)
		}
		return exitValidation
	case "check-escape-hatches":
		path := root
		if len(args) > 1 {
			path = args[1]
		}
		report, err := tooling.ScanEscapeHatches(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(report))
		}
		if report.Clean {
			fmt.Printf("Clean: %d file(s) scanned, %d registered escape hatch(es), 0 unregistered violations.\n",
				report.TotalScannedFiles, report.RegisteredFindings)
			return exitOK
		}
		fmt.Printf("Found %d unregistered escape hatch violation(s) across %d file(s):\n",
			report.UnregisteredFindings, report.TotalScannedFiles)
		for _, f := range report.Findings {
			if !f.Registered {
				fmt.Printf("  %s:%d [%s] %s -> %s\n", f.File, f.Line, f.HatchType, f.Snippet, f.Message)
			}
		}
		return exitValidation
	default:
		return exitUsage
	}
}

func runModel(asJSON bool, root string, args []string) int {
	if len(args) == 0 {
		return exitUsage
	}
	reg := defaultModelRegistry()
	switch args[0] {
	case "list":
		models := reg.List()
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(models))
		}
		fmt.Printf("%-16s %-12s %-10s %-8s %s\n", "ID", "PROVIDER", "PRIVACY", "CONTEXT", "LATENCY")
		fmt.Println("-----------------------------------------------------------------")
		for _, m := range models {
			fmt.Printf("%-16s %-12s %-10s %-8d %s\n", m.ID, m.Provider, m.Privacy, m.ContextTokens, m.LatencyClass)
		}
		return exitOK
	case "route":
		req := modelregistry.RouteRequest{
			MinimumContext: 1000,
		}
		for i := 1; i < len(args); i++ {
			if args[i] == "--data-class" && i+1 < len(args) {
				req.DataClass = args[i+1]
				i++
			}
			if args[i] == "--tokens" && i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					req.MinimumContext = n
				}
				i++
			}
			if args[i] == "--tools" {
				req.NeedTools = true
			}
			if args[i] == "--latency" && i+1 < len(args) {
				req.LatencyPreference = args[i+1]
				i++
			}
		}
		resp, err := reg.Route(req)
		if err != nil {
			if asJSON {
				return envelopeError(protocol.CodeValidationFailed, err.Error())
			}
			fmt.Printf("error: %v\n", err)
			return exitValidation
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(resp))
		}
		fmt.Printf("Selected Route: %s (provider: %s)\n", resp.Primary.ID, resp.Primary.Provider)
		fmt.Printf("Explanation: %s\n", resp.Explanation)
		if len(resp.Fallbacks) > 0 {
			fmt.Printf("Fallbacks (%d):\n", len(resp.Fallbacks))
			for _, fb := range resp.Fallbacks {
				fmt.Printf("  - %s (provider: %s)\n", fb.ID, fb.Provider)
			}
		}
		return exitOK
	default:
		return exitUsage
	}
}

func runEnv(asJSON bool, root string, args []string) int {
	if len(args) == 0 {
		return exitUsage
	}
	switch args[0] {
	case "list":
		local := environment.NewLocalEnvironment(root, true)
		envs := []environment.Descriptor{
			local.Descriptor(),
			{
				ID:              "env-worktree-template",
				Version:         1,
				Kind:            "worktree",
				Isolation:       environment.IsolationWorktree,
				Network:         "deny",
				FilesystemRoots: []string{root},
			},
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(envs))
		}
		fmt.Printf("%-24s %-10s %-15s %s\n", "ID", "KIND", "ISOLATION", "NETWORK")
		fmt.Println("-----------------------------------------------------------------")
		for _, e := range envs {
			fmt.Printf("%-24s %-10s %-15s %s\n", e.ID, e.Kind, e.Isolation, e.Network)
		}
		return exitOK
	default:
		return exitUsage
	}
}
