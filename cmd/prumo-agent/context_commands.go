package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/raillen/prumo/internal/harness/contextv2"
	"github.com/raillen/prumo/internal/protocol"
)

// runContextCompile emits a replayable ContextManifest for a goal at an explicit
// budget and disclosure level (W4.7). It is the headless entry point: the same
// compilation the runtime performs, with the policy named in the output.
func runContextCompile(asJSON bool, args []string) int {
	goal := ""
	budget := 8000
	level := ""
	id := ""
	required := []string{}
	path := pathFlag(args)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--goal", "--budget", "--level", "--id", "--required", "--path":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: %s requires a value\n", args[i])
				return exitUsage
			}
			value := args[i+1]
			i++
			switch args[i-1] {
			case "--goal":
				goal = value
			case "--budget":
				n, err := strconv.Atoi(value)
				if err != nil || n <= 0 {
					fmt.Fprintf(os.Stderr, "error: --budget must be a positive integer\n")
					return exitUsage
				}
				budget = n
			case "--level":
				level = value
			case "--id":
				id = value
			case "--required":
				for _, ref := range strings.Split(value, ",") {
					if trimmed := strings.TrimSpace(ref); trimmed != "" {
						required = append(required, trimmed)
					}
				}
			}
		}
	}
	if strings.TrimSpace(goal) == "" {
		fmt.Fprintln(os.Stderr, "error: context compile requires --goal <text>")
		return exitUsage
	}
	runID := id
	if runID == "" {
		runID = contextIDFromGoal(goal)
	}
	manifest := contextv2.CompileWorkspace(runID, goal, path, budget, level)
	saved, err := contextv2.SaveManifest(path, manifest)
	if err != nil {
		return serviceError(asJSON, err)
	}
	explanation := contextv2.Explain(manifest, budget, required)
	result := map[string]any{"manifest": manifest, "explanation": explanation, "path": saved}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("%s  %d tokens / %d budget  level %s  %s\n",
		manifest.ID, manifest.EstimatedTokens, budget, manifest.Level, manifest.Pressure)
	for _, entry := range explanation.Included {
		marker := " "
		if entry.Truncated {
			marker = "~"
		}
		fmt.Printf("  %s %-40s %5d  %s\n", marker, entry.Ref, entry.Tokens, entry.Reason)
	}
	for _, note := range explanation.Notes {
		fmt.Printf("note: %s\n", note)
	}
	fmt.Printf("written: %s\n", saved)
	if !explanation.Sufficiency.Sufficient {
		return exitValidation
	}
	return exitOK
}

// runContextExplain answers what a compilation contained, why, and what it left
// out (W4.8).
func runContextExplain(asJSON bool, args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "error: context explain requires a manifest id, e.g. CTX-run-1")
		return exitUsage
	}
	id := args[0]
	path := pathFlag(args)
	budget := 0
	required := []string{}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--budget", "--required", "--path":
			if i+1 >= len(args) {
				return exitUsage
			}
			value := args[i+1]
			i++
			switch args[i-1] {
			case "--budget":
				if n, err := strconv.Atoi(value); err == nil {
					budget = n
				}
			case "--required":
				for _, ref := range strings.Split(value, ",") {
					if trimmed := strings.TrimSpace(ref); trimmed != "" {
						required = append(required, trimmed)
					}
				}
			}
		}
	}
	manifest, err := contextv2.LoadManifest(path, id)
	if err != nil {
		return serviceError(asJSON, err)
	}
	explanation := contextv2.Explain(manifest, budget, required)
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(explanation))
	}
	fmt.Printf("%s  level %s  %d tokens  %s\n", explanation.ID, explanation.Level,
		explanation.EstimatedTokens, explanation.Pressure)
	fmt.Printf("included %d:\n", explanation.IncludedCount)
	for _, entry := range explanation.Included {
		marker := " "
		if entry.Truncated {
			marker = "~"
		}
		fmt.Printf("  %s %-40s %5d  %s\n", marker, entry.Ref, entry.Tokens, entry.Reason)
	}
	fmt.Printf("excluded %d:\n", explanation.ExcludedCount)
	for _, ex := range explanation.Excluded {
		if ex.Detail != "" {
			fmt.Printf("  %-40s %-16s %s\n", ex.Ref, ex.Reason, ex.Detail)
			continue
		}
		fmt.Printf("  %-40s %s\n", ex.Ref, ex.Reason)
	}
	for _, note := range explanation.Notes {
		fmt.Printf("note: %s\n", note)
	}
	if !explanation.Sufficiency.Sufficient {
		return exitValidation
	}
	return exitOK
}

// contextIDFromGoal derives a readable, stable run id from a goal so a default
// compilation can still be looked up afterwards.
func contextIDFromGoal(goal string) string {
	slug := strings.Trim(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '-'
		}
	}, goal), "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	if len(slug) > 48 {
		slug = strings.Trim(slug[:48], "-")
	}
	if slug == "" {
		return "context"
	}
	return slug
}
