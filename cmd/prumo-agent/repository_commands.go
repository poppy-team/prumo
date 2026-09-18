package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/repositorypolicy"
)

func runRepositoryPolicy(asJSON bool, args []string) int {
	if len(args) < 2 || args[0] != "repo" || args[1] != "policy" {
		fmt.Fprintf(os.Stderr, "error: expected repo policy <check|explain|plan|apply>\n")
		return exitUsage
	}
	command := "check"
	if len(args) > 2 {
		command = args[2]
	}
	root := "."
	dryRun := false
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --path requires a value")
				return exitUsage
			}
			root = args[i+1]
			i++
		case "--dry-run":
			dryRun = true
		}
	}
	service := repositorypolicy.NewService(root)
	policy, err := service.Load()
	if err != nil {
		return serviceError(asJSON, err)
	}
	if command == "apply" {
		token := firstEnvironment("GITHUB_TOKEN", "GH_TOKEN")
		if token == "" {
			fmt.Fprintln(os.Stderr, "error: repository policy apply requires GITHUB_TOKEN or GH_TOKEN")
			return exitConfig
		}
		result, err := service.Apply(policy, token, dryRun)
		if err != nil {
			return serviceError(asJSON, err)
		}
		result["dry_run"] = dryRun
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("Applied: %v Skipped: %v Dry run: %v\n", result["applied"], result["skipped"], dryRun)
		return exitOK
	}
	local, err := repositorypolicy.InspectLocal(root)
	if err != nil {
		return serviceError(asJSON, err)
	}
	checks := repositorypolicy.EvaluateLocal(policy, local)
	result := map[string]any{"policy": policy, "local": local, "checks": checks, "actions": repositorypolicy.Plan(policy, checks, false)}
	if token := firstEnvironment("GITHUB_TOKEN", "GH_TOKEN"); token != "" {
		if remote, remoteErr := service.CheckRemote(policy, token); remoteErr != nil {
			result["remote"] = map[string]any{"status": repositorypolicy.Unavailable, "error": remoteErr.Error()}
		} else {
			result["remote"] = remote
			if remoteChecks, ok := remote["checks"].([]repositorypolicy.Check); ok {
				result["checks"] = append(checks, remoteChecks...)
			}
		}
	}
	if command == "explain" {
		result["explanation"] = explainPolicy(checks)
	}
	if command == "apply" {
		result["dry_run"] = true
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	for _, check := range checks {
		fmt.Printf("%-35s %-14s %s\n", check.ID, check.Status, check.Explanation)
	}
	if command == "plan" || command == "apply" {
		fmt.Printf("Actions: %d\n", len(result["actions"].([]repositorypolicy.Action)))
	}
	return exitOK
}

func firstEnvironment(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func explainPolicy(checks []repositorypolicy.Check) string {
	parts := make([]string, 0, len(checks))
	for _, check := range checks {
		parts = append(parts, fmt.Sprintf("%s: expected=%v actual=%v (%s)", check.ID, check.Expected, check.Actual, check.Explanation))
	}
	return strings.Join(parts, "\n")
}
