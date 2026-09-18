package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/raillen/prumo/internal/automation"
	"github.com/raillen/prumo/internal/packages"
	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/runtime"
)

func runPlatform(asJSON bool, args []string) int {
	if len(args) < 2 {
		return exitUsage
	}
	switch args[0] {
	case "package":
		if len(args) > 1 && args[1] == "sync" {
			return runWorkforce(asJSON, "", append([]string{"sync"}, args[2:]...))
		}
		lock, err := runtime.LoadJSON[packages.Lock](filepath.Join(".prumo", "lock.json"))
		if err != nil {
			// Fallback to prumo.lock if .prumo/lock.json does not exist
			lock, err = runtime.LoadJSON[packages.Lock]("prumo.lock")
		}
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(lock))
		}
		for _, p := range lock.Packages {
			fmt.Printf("%s %s %s\n", p.ID, p.Version, p.Type)
		}
		return exitOK
	case "runtime":
		entries, _ := os.ReadDir(filepath.Join(".prumo", "runtime"))
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"runtimes": names}))
		}
		fmt.Println(names)
		return exitOK
	case "automation":
		if len(args) > 1 && args[1] == "failures" {
			return printEnvelope(protocol.OkEnvelope([]automation.DLQEntry{}))
		}
		return printEnvelope(protocol.OkEnvelope([]any{}))
	default:
		return exitUsage
	}
}
