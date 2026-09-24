package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/install"
	"github.com/raillen/prumo/internal/protocol"
)

func installationHome(explicit string) (string, error) {
	return install.HomeDir(explicit)
}

func runSetup(asJSON bool, explicitHome string) int {
	home, err := installationHome(explicitHome)
	if err != nil {
		return serviceError(asJSON, err)
	}
	manifest, err := install.LoadManifest(home)
	if err != nil {
		return serviceError(asJSON, err)
	}
	// The manifest describes the installation actually in effect: re-running setup
	// after an upgrade records the binary doing the work, not the version that first
	// created the manifest.
	manifest.PrumoVersion = protocol.CLIVersion
	if manifest.BinaryPath == "" {
		if executable, err := os.Executable(); err == nil {
			manifest.BinaryPath = executable
		}
	}
	if err := install.SaveManifest(home, manifest); err != nil {
		return serviceError(asJSON, err)
	}
	harnesses := map[string]bool{}
	for _, candidate := range []string{"opencode", "codex", "code", "gemini", "copilot", "kiro"} {
		harnesses[candidate] = onPath(candidate)
	}
	result := map[string]any{"home": home, "manifest": manifest, "harnesses": harnesses, "path_valid": pathContains(home), "idempotent": true}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("Prumo setup complete: %s\n", home)
	fmt.Printf("Detected harnesses:")
	detected := false
	for name, present := range harnesses {
		if present {
			fmt.Printf(" %s", name)
			detected = true
		}
	}
	if !detected {
		fmt.Printf(" none")
	}
	fmt.Printf("\n")
	return exitOK
}

func onPath(name string) bool {
	for _, dir := range pathDirs() {
		if _, err := os.Stat(dir + string(os.PathSeparator) + name); err == nil {
			return true
		}
	}
	return false
}

func pathDirs() []string {
	separator := ":"
	if os.PathSeparator == '\\' {
		separator = ";"
	}
	out := []string{}
	for _, dir := range splitEnv(os.Getenv("PATH"), separator) {
		if dir != "" {
			out = append(out, dir)
		}
	}
	return out
}

func splitEnv(value, separator string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Split(value, separator)
}

func pathContains(home string) bool {
	bin := home + string(os.PathSeparator) + "bin"
	for _, dir := range pathDirs() {
		if dir == bin {
			return true
		}
	}
	return false
}

func runInstall(asJSON bool, explicitHome string, args []string) int {
	home, err := installationHome(explicitHome)
	if err != nil {
		return serviceError(asJSON, err)
	}
	manifest, err := install.LoadManifest(home)
	if err != nil {
		return serviceError(asJSON, err)
	}
	manifest.PrumoVersion = protocol.CLIVersion
	if executable, err := os.Executable(); err == nil && manifest.BinaryPath == "" {
		manifest.BinaryPath = executable
	}
	if len(args) >= 2 && args[0] == "connector" {
		return runConnectorInstall(asJSON, home, args[1], args[2:])
	}
	if len(args) >= 1 && !strings.HasPrefix(args[0], "-") {
		if _, err := connectors.Get(args[0]); err == nil {
			return runConnectorInstall(asJSON, home, args[0], args[1:])
		}
	}
	if err := install.SaveManifest(home, manifest); err != nil {
		return serviceError(asJSON, err)
	}
	result := map[string]any{"home": home, "manifest": manifest}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("Prumo installation state updated: %s\n", home)
	return exitOK
}

func runUninstall(asJSON bool, explicitHome string, args []string) int {
	home, err := installationHome(explicitHome)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if len(args) >= 1 && !strings.HasPrefix(args[0], "-") {
		if _, err := connectors.Get(args[0]); err == nil {
			return runConnectorUninstall(asJSON, home, args[0], args[1:])
		}
	}
	manifest, err := install.LoadManifest(home)
	if err != nil {
		return serviceError(asJSON, err)
	}
	removed := []string{}
	leftovers := []string{}
	if hasFlag(args, "--connectors") {
		// The index, not the manifest.
		//
		// There was one cleanup file per connector, so a connector installed in
		// three projects had one record — the last install — and uninstalling all
		// connectors cleaned up one project and left two projects' files behind
		// with nothing pointing at them (GAP-137).
		index, indexErr := install.LoadCleanupIndex(home)
		if indexErr != nil {
			return serviceError(asJSON, indexErr)
		}
		for _, entry := range index.Entries {
			cleanupData, readErr := os.ReadFile(entry.Path)
			if readErr != nil {
				// A record that cannot be read is reported, not skipped: a
				// connector whose files we cannot account for is a connector
				// whose files are still there after an uninstall that said it
				// removed them.
				leftovers = append(leftovers, entry.Connector+": "+readErr.Error())
				continue
			}
			var cleanup install.CleanupManifest
			if json.Unmarshal(cleanupData, &cleanup) != nil {
				leftovers = append(leftovers, entry.Connector+": cleanup record is unreadable")
				continue
			}
			gone, remaining := install.RemoveManagedPaths(home, cleanup.CreatedPaths)
			removed = append(removed, gone...)
			leftovers = append(leftovers, remaining...)
			_ = os.Remove(entry.Path)
		}
		manifest.Connectors = map[string]string{}
	}
	if hasFlag(args, "--purge-cache") {
		if err := install.PurgeDirectory(home + "/cache"); err != nil {
			return serviceError(asJSON, err)
		}
		removed = append(removed, home+"/cache")
	}
	if hasFlag(args, "--purge-global-config") {
		if err := install.PurgeDirectory(home + "/config"); err != nil {
			return serviceError(asJSON, err)
		}
		removed = append(removed, home+"/config")
	}
	if !hasFlag(args, "--purge-global-config") {
		if err := install.SaveManifest(home, manifest); err != nil {
			return serviceError(asJSON, err)
		}
	}
	result := map[string]any{"home": home, "removed": removed, "leftovers": leftovers, "project_data_preserved": true}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("Prumo uninstall completed; project data preserved.\n")
	return exitOK
}
