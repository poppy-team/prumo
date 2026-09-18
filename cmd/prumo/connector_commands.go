package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/connectors"
	_ "github.com/raillen/prumo/internal/connectors/antigravity"
	_ "github.com/raillen/prumo/internal/connectors/claudecode"
	_ "github.com/raillen/prumo/internal/connectors/codex"
	_ "github.com/raillen/prumo/internal/connectors/gemini"
	_ "github.com/raillen/prumo/internal/connectors/opencode"
	"github.com/raillen/prumo/internal/protocol"
)

func runConnector(asJSON bool, home string, args []string) int {
	if len(args) == 0 {
		return runConnectorList(asJSON)
	}

	sub := args[0]
	switch sub {
	case "list":
		return runConnectorList(asJSON)
	case "install":
		if len(args) < 2 {
			err := fmt.Errorf("usage: prumo connector install <name>")
			return serviceError(asJSON, err)
		}
		return runConnectorInstall(asJSON, home, args[1], args[2:])
	case "uninstall":
		if len(args) < 2 {
			err := fmt.Errorf("usage: prumo connector uninstall <name>")
			return serviceError(asJSON, err)
		}
		return runConnectorUninstall(asJSON, home, args[1], args[2:])
	case "validate":
		if len(args) < 2 {
			err := fmt.Errorf("usage: prumo connector validate <name>")
			return serviceError(asJSON, err)
		}
		return runConnectorValidate(asJSON, args[1], args[2:])
	case "negotiate":
		if len(args) < 2 {
			err := fmt.Errorf("usage: prumo connector negotiate <name> [--strict] [--caps <list>]")
			return serviceError(asJSON, err)
		}
		return runConnectorNegotiate(asJSON, args[1], args[2:])
	default:
		err := fmt.Errorf("unknown connector subcommand: %s", sub)
		return serviceError(asJSON, err)
	}
}

func runConnectorList(asJSON bool) int {
	list := connectors.List()
	records := make([]map[string]any, 0, len(list))
	for _, c := range list {
		contract := c.Contract()
		records = append(records, map[string]any{
			"id":           c.ID(),
			"name":         c.Name(),
			"version":      contract.Version,
			"enforcement":  contract.Enforcement,
			"capabilities": contract.Capabilities,
			"hooks":        contract.Hooks,
		})
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"connectors": records,
		}))
	}

	fmt.Println("Registered Prumo Connectors:")
	for _, r := range records {
		fmt.Printf("  • %s (%s) - v%s [%s]\n", r["name"], r["id"], r["version"], r["enforcement"])
		if caps, ok := r["capabilities"].([]string); ok && len(caps) > 0 {
			fmt.Printf("    Capabilities: %s\n", strings.Join(caps, ", "))
		}
		if hooks, ok := r["hooks"].([]string); ok && len(hooks) > 0 {
			fmt.Printf("    Hooks: %s\n", strings.Join(hooks, ", "))
		}
	}
	return exitOK
}

func runConnectorInstall(asJSON bool, home, id string, args []string) int {
	c, err := connectors.Get(id)
	if err != nil {
		return serviceError(asJSON, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return serviceError(asJSON, err)
	}

	targetDir := cwd
	if p, _, ok := flag(args, "--path"); ok && p != "" {
		targetDir = p
	}

	res, err := c.Install(home, targetDir, connectors.InstallOptions{})
	if err != nil {
		return serviceError(asJSON, err)
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}

	fmt.Printf("Connector %s (%s) installed successfully.\n", c.Name(), c.ID())
	fmt.Printf("Created %d artifacts in project.\n", len(res.CreatedPaths))
	fmt.Printf("Cleanup manifest: %s\n", res.CleanupPath)
	return exitOK
}

func runConnectorUninstall(asJSON bool, home, id string, args []string) int {
	c, err := connectors.Get(id)
	if err != nil {
		return serviceError(asJSON, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return serviceError(asJSON, err)
	}

	targetDir := cwd
	if p, _, ok := flag(args, "--path"); ok && p != "" {
		targetDir = p
	}

	res, err := c.Uninstall(home, targetDir, connectors.UninstallOptions{})
	if err != nil {
		return serviceError(asJSON, err)
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}

	fmt.Printf("Connector %s uninstalled.\n", id)
	fmt.Printf("Removed %d artifacts.\n", len(res.Removed))
	if len(res.Leftovers) > 0 {
		fmt.Printf("Leftovers retained: %v\n", res.Leftovers)
	}
	return exitOK
}

func runConnectorValidate(asJSON bool, id string, args []string) int {
	c, err := connectors.Get(id)
	if err != nil {
		return serviceError(asJSON, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return serviceError(asJSON, err)
	}

	targetDir := cwd
	if p, _, ok := flag(args, "--path"); ok && p != "" {
		targetDir = p
	}

	res, err := c.Validate(targetDir)
	if err != nil {
		return serviceError(asJSON, err)
	}

	if asJSON {
		if !res.Valid {
			return envelopeError("CONNECTOR_INVALID", fmt.Sprintf("connector validation failed: %v", res.Errors))
		}
		return printEnvelope(protocol.OkEnvelope(res))
	}

	if !res.Valid {
		fmt.Printf("Validation failed for connector %s:\n", id)
		for _, err := range res.Errors {
			fmt.Printf("  [ERROR] %s\n", err)
		}
		return exitValidation
	}

	fmt.Printf("Connector %s validated successfully (%d files verified).\n", id, len(res.Files))
	return exitOK
}

func runConnectorNegotiate(asJSON bool, id string, args []string) int {
	c, err := connectors.Get(id)
	if err != nil {
		return serviceError(asJSON, err)
	}
	strict := hasFlag(args, "--strict")
	caps := []string{}
	if val, _, found := flag(args, "--caps"); found && val != "" {
		for _, part := range strings.Split(val, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				caps = append(caps, trimmed)
			}
		}
	} else {
		// Default to standard capabilities
		caps = []string{connectors.CapAdvise, connectors.CapPreToolBlock, connectors.CapSessionHooks, connectors.CapIsolateSubagents}
	}

	res, err := connectors.Negotiate(c.Contract(), connectors.NegotiationRequest{
		TargetHarness:        id,
		RequiredCapabilities: caps,
		Strict:               strict,
	})
	if err != nil {
		return serviceError(asJSON, err)
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(res))
	}

	fmt.Printf("Capability Negotiation for %s (%s):\n", c.Name(), id)
	fmt.Printf("  Compatible: %v\n", res.Compatible)
	if len(res.Supported) > 0 {
		fmt.Printf("  Supported:  %s\n", strings.Join(res.Supported, ", "))
	}
	if len(res.Degraded) > 0 {
		fmt.Println("  Degraded:")
		for cap, reason := range res.Degraded {
			fmt.Printf("    • %s: %s\n", cap, reason)
		}
	}
	if len(res.Unsupported) > 0 {
		fmt.Printf("  Unsupported: %s\n", strings.Join(res.Unsupported, ", "))
	}
	return exitOK
}
