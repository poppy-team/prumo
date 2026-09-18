package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/workforcesync"
)

func runWorkforce(asJSON bool, home string, args []string) int {
	if len(args) == 0 {
		if asJSON {
			return envelopeError(protocol.CodeConfiguration, "workforce subcommand required: sync, list")
		}
		fmt.Fprintf(os.Stderr, "error: workforce requires a subcommand (sync, list)\n")
		return exitUsage
	}

	subcommand := args[0]
	rest := args[1:]

	path := "."
	offline := false
	forceRemote := false
	remoteBaseURL := ""
	version := protocol.CLIVersion
	var targetSkills []string

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--path":
			if i+1 < len(rest) {
				path = rest[i+1]
				i++
			}
		case "--offline":
			offline = true
		case "--force-remote":
			forceRemote = true
		case "--remote":
			if i+1 < len(rest) {
				remoteBaseURL = rest[i+1]
				i++
			}
		case "--version":
			if i+1 < len(rest) {
				version = rest[i+1]
				i++
			}
		case "--skills":
			if i+1 < len(rest) {
				targetSkills = strings.Split(rest[i+1], ",")
				i++
			}
		}
	}

	switch subcommand {
	case "sync":
		service := workforcesync.NewService()
		opts := workforcesync.Options{
			ProjectRoot:   path,
			HomeDir:       home,
			RepoRoot:      repoRoot(),
			RemoteBaseURL: remoteBaseURL,
			Version:       version,
			OfflineOnly:   offline,
			ForceRemote:   forceRemote,
			TargetSkills:  targetSkills,
		}

		result, err := service.Sync(opts)
		if err != nil {
			return serviceError(asJSON, err)
		}

		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}

		fmt.Printf("Workforce synchronized (Prumo v%s)\n", opts.Version)
		fmt.Printf("Total skills installed: %d\n", len(result.InstalledSkills))
		if len(result.RemoteDownloads) > 0 {
			fmt.Printf("Downloaded from remote: %s\n", strings.Join(result.RemoteDownloads, ", "))
		}
		if len(result.CacheHits) > 0 {
			fmt.Printf("Cached skills used: %d\n", len(result.CacheHits))
		}
		fmt.Println("Lockfile updated: prumo.lock")
		return exitOK

	case "list":
		skillsDir := filepath.Join(path, ".ai", "skills")
		entries, err := os.ReadDir(skillsDir)
		installed := []string{}
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					installed = append(installed, e.Name())
				}
			}
		}

		coreSkills := []string{"clean-code", "cognitive-clarity", "testing-quality"}

		payload := map[string]any{
			"installed_skills": installed,
			"core_skills":      coreSkills,
		}

		if asJSON {
			return printEnvelope(protocol.OkEnvelope(payload))
		}

		fmt.Println("Core Skills (mandatory):")
		for _, c := range coreSkills {
			fmt.Printf("  - %s\n", c)
		}
		fmt.Printf("\nInstalled Skills in %s (%d):\n", skillsDir, len(installed))
		for _, inst := range installed {
			fmt.Printf("  - %s\n", inst)
		}
		return exitOK

	default:
		if asJSON {
			return envelopeError(protocol.CodeConfiguration, fmt.Sprintf("unknown workforce subcommand: %s", subcommand))
		}
		fmt.Fprintf(os.Stderr, "error: unknown workforce subcommand: %s\n", subcommand)
		return exitUsage
	}
}
