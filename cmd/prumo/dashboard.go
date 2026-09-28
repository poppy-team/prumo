package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/project"
	"github.com/raillen/prumo/internal/protocol"
)

func runDashboard(asJSON bool, explicitHome string) int {
	home, err := installationHome(explicitHome)
	if err != nil {
		home = "~/.prumo"
	}

	// Try to find an active project root in current directory
	projectRoot, projectErr := project.FindRoot(".")

	if projectErr == nil {
		// Inside an existing Prumo workspace
		manifestPath := filepath.Join(projectRoot, "prumo.json")
		manifestBytes, err := os.ReadFile(manifestPath)
		var manifest map[string]any
		if err == nil {
			_ = json.Unmarshal(manifestBytes, &manifest)
		}

		projectName := filepath.Base(projectRoot)
		if projMap, ok := manifest["project"].(map[string]any); ok {
			if n, ok := projMap["name"].(string); ok && n != "" {
				projectName = n
			}
		}

		phase := "P00"
		goal := "none"
		if goalsMap, ok := manifest["goals"].(map[string]any); ok {
			if p, ok := goalsMap["active_phase"].(string); ok && p != "" {
				phase = p
			}
			if g, ok := goalsMap["active_goal"].(string); ok && g != "" {
				goal = g
			}
		}

		agentsCount := countManifestEntries(filepath.Join(projectRoot, ".ai", "agents", "manifest.json"), "agents")
		skillsCount := countManifestEntries(filepath.Join(projectRoot, ".ai", "skills", "manifest.json"), "skills")
		recipesCount := countManifestEntries(filepath.Join(projectRoot, ".ai", "recipes", "manifest.json"), "recipes")

		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{
				"type":         "workspace",
				"version":      protocol.CLIVersion,
				"project_name": projectName,
				"project_root": projectRoot,
				"phase":        phase,
				"active_goal":  goal,
				"workforce": map[string]int{
					"agents":  agentsCount,
					"skills":  skillsCount,
					"recipes": recipesCount,
				},
			}))
		}

		fmt.Printf("Prumo CLI v%s — Project Workspace\n\n", protocol.CLIVersion)
		fmt.Printf("  Project:       %s (Protocol v3)\n", projectName)
		fmt.Printf("  Root:          %s\n", projectRoot)
		fmt.Printf("  Active Phase:  %s\n", phase)
		if goal == "none" {
			fmt.Printf("  Active Goal:   (none selected)\n")
		} else {
			fmt.Printf("  Active Goal:   %s\n", goal)
		}
		if agentsCount > 0 || skillsCount > 0 || recipesCount > 0 {
			fmt.Printf("  Workforce:     %d agents | %d skills | %d recipes\n", agentsCount, skillsCount, recipesCount)
		}
		fmt.Println()
		fmt.Println("Quick Actions:")
		fmt.Println("  prumo doctor             # Check workspace conformance and health")
		fmt.Println("  prumo compile --all      # Compile agent adapters (AGENTS.md, CLAUDE.md, etc.)")
		fmt.Println("  prumo agent              # Launch interactive coding agent TUI")
		fmt.Println("  prumo run                # Execute active goal loop")
		fmt.Println("  prumo --help             # View all commands and options")
		return exitOK
	}

	// Outside a project: display global environment & onboarding dashboard
	availableConnectors := connectors.List()
	connectorNames := make([]string, 0, len(availableConnectors))
	for _, c := range availableConnectors {
		connectorNames = append(connectorNames, c.Name())
	}

	harnesses := []string{}
	for _, candidate := range []string{"opencode", "codex", "code", "gemini", "copilot"} {
		if onPath(candidate) {
			harnesses = append(harnesses, candidate)
		}
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"type":               "global",
			"version":            protocol.CLIVersion,
			"home":               home,
			"connectors":         connectorNames,
			"detected_harnesses": harnesses,
		}))
	}

	fmt.Printf("Prumo CLI v%s — Agentic Engineering Harness & Workforce System\n\n", protocol.CLIVersion)
	fmt.Println("Global Environment:")
	fmt.Printf("  Home:        %s\n", home)
	fmt.Printf("  Connectors:  %s\n", strings.Join(connectorNames, ", "))
	if len(harnesses) > 0 {
		fmt.Printf("  Harnesses:   %s (detected on PATH)\n", strings.Join(harnesses, ", "))
	} else {
		fmt.Printf("  Harnesses:   none detected on PATH (install opencode, code, or claude)\n")
	}
	fmt.Println()
	fmt.Println("Getting Started:")
	fmt.Println("  prumo init [path]        # Initialize a new Prumo workspace (zero-config)")
	fmt.Println("  prumo adopt [path]       # Scan and adopt an existing codebase (brownfield)")
	fmt.Println("  prumo agent              # Launch interactive coding agent TUI")
	fmt.Println("  prumo setup              # Initialize global configuration (~/.prumo)")
	fmt.Println("  prumo upgrade            # Check for and install CLI updates")
	fmt.Println()
	fmt.Println("Explore:")
	fmt.Println("  prumo --help             # View all available commands and flags")
	fmt.Println("  Documentation:           https://prumo-framework.vercel.app")
	return exitOK
}

func countManifestEntries(path, key string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0
	}
	if list, ok := raw[key].([]any); ok {
		return len(list)
	}
	return 0
}
