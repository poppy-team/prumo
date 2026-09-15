package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/uimap"
)

// runUI is `prumo ui`: the interface map command family.
//
//	ui map    [--path <dir>] [--write] [--target a,b] [--file <map>]
//	ui verify [--path <dir>] [--file <map>]
//	ui impact <node> [--path <dir>] [--file <map>]
//	ui config [--path <dir>]
func runUI(asJSON bool, args []string) int {
	if len(args) == 0 {
		return exitUsage
	}
	switch args[0] {
	case "map":
		return runUIMap(asJSON, args[1:])
	case "verify":
		return runUIVerify(asJSON, args[1:])
	case "impact":
		return runUIImpact(asJSON, args[1:])
	case "config":
		return runUIConfig(asJSON, args[1:])
	default:
		return exitUsage
	}
}

// uiOptions reads the flags the family shares.
func uiOptions(args []string) (root string, file string, targets []uimap.Target, write bool, rest []string) {
	root = "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 < len(args) {
				root = args[i+1]
				i++
			}
		case "--file":
			if i+1 < len(args) {
				file = args[i+1]
				i++
			}
		case "--target":
			if i+1 < len(args) {
				for _, name := range strings.Split(args[i+1], ",") {
					targets = append(targets, uimap.Target(strings.TrimSpace(name)))
				}
				i++
			}
		case "--write":
			write = true
		default:
			rest = append(rest, args[i])
		}
	}
	return root, file, targets, write, rest
}

func runUIMap(asJSON bool, args []string) int {
	root, file, targets, write, _ := uiOptions(args)
	result, err := uimap.Compile(root, uimap.Options{Write: write, Path: file, Targets: targets})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"skipped":     result.Skipped,
			"reason":      result.Reason,
			"config":      result.Config.ConfigSummary(),
			"surface":     result.Map.Surface,
			"digest":      result.Map.SourceDigest,
			"elements":    result.Report.Elements,
			"edges":       result.Report.Edges,
			"findings":    result.Report.Findings,
			"ok":          result.Report.OK(),
			"derivation":  result.Config.Derivation,
			"targets":     result.Config.Targets,
			"projections": projectionPaths(result),
		}))
	}
	fmt.Print(uimap.ReportText(result))
	if write {
		fmt.Printf("wrote %d projection(s) under %s\n", len(result.Projections), uimap.ProjectionRoot(root))
	}
	if !result.Report.OK() {
		return exitValidation
	}
	return exitOK
}

func runUIVerify(asJSON bool, args []string) int {
	root, file, _, _, _ := uiOptions(args)
	// Verification also checks that the projections on disk match the map:
	// a reference describing the previous interface misleads the reader who
	// opens it, and that is a gate failure rather than a warning.
	result, err := uimap.Compile(root, uimap.Options{Path: file})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if result.Skipped {
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{
				"skipped": true, "reason": result.Reason, "ok": true}))
		}
		fmt.Println(result.Reason)
		return exitOK
	}
	stale := uimap.CheckProjections(root, result.Projections)
	findings := append(append([]uimap.Finding{}, result.Report.Findings...), stale...)
	ok := result.Report.OK() && len(stale) == 0
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"skipped":  false,
			"surface":  result.Map.Surface,
			"digest":   result.Map.SourceDigest,
			"elements": result.Report.Elements,
			"edges":    result.Report.Edges,
			"findings": findings,
			"stale":    len(stale),
			"ok":       ok,
			"config":   result.Config.ConfigSummary(),
		}))
	}
	result.Report.Findings = findings
	fmt.Print(uimap.ReportText(result))
	if !ok {
		return exitValidation
	}
	return exitOK
}

func runUIImpact(asJSON bool, args []string) int {
	root, file, _, _, rest := uiOptions(args)
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "error: ui impact requires an element id, e.g. node:palette")
		return exitUsage
	}
	node := rest[0]
	if !strings.HasPrefix(node, "node:") {
		node = "node:" + node
	}
	result, err := uimap.Compile(root, uimap.Options{Path: file})
	if err != nil {
		return serviceError(asJSON, err)
	}
	if result.Skipped {
		fmt.Println(result.Reason)
		return exitOK
	}
	impact, err := result.Map.ImpactOf(node)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"origin":  impact.Origin,
			"label":   impact.OriginLabel,
			"reaches": impact.Transit,
		}))
	}
	fmt.Printf("%s (%s) reaches %d element(s):\n", impact.Origin, impact.OriginLabel, len(impact.Transit))
	for _, t := range impact.Transit {
		fmt.Printf("  %d hop(s) %-4s %-34s %s\n", t.Hop, t.Direction, t.NodeID, t.Via)
	}
	return exitOK
}

func runUIConfig(asJSON bool, args []string) int {
	root, _, _, _, _ := uiOptions(args)
	cfg, err := uimap.ResolveConfig(root)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"enabled":    cfg.Enabled,
			"targets":    cfg.Targets,
			"derivation": cfg.Derivation,
			"path":       cfg.Path,
			"reason":     cfg.Reason,
			"summary":    cfg.ConfigSummary(),
		}))
	}
	fmt.Println(cfg.ConfigSummary())
	fmt.Printf("map: %s\n", cfg.Path)
	fmt.Printf("targets: %v · derivation: %s\n", cfg.Targets, cfg.Derivation)
	return exitOK
}

func projectionPaths(result uimap.Result) []map[string]any {
	out := make([]map[string]any, 0, len(result.Projections))
	for _, p := range result.Projections {
		out = append(out, map[string]any{
			"target": string(p.Target), "path": p.Path, "tokens": p.Tokens,
		})
	}
	return out
}
