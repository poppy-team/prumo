package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/raillen/prumo/internal/contextcompiler"
	"github.com/raillen/prumo/internal/observability"
	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/runtime"
)

func runRuntime(asJSON bool, args []string) int {
	if len(args) == 0 {
		return exitUsage
	}
	root := "."
	id := "R-bootstrap"
	for i := 1; i < len(args); i++ {
		if args[i] == "--path" && i+1 < len(args) {
			root = args[i+1]
			i++
		}
		if args[i] == "--run" && i+1 < len(args) {
			id = args[i+1]
			i++
		}
	}
	path := filepath.Join(root, ".prumo", "runtime", "continuation.json")
	switch args[0] {
	case "runtime":
		entries, _ := os.ReadDir(filepath.Join(root, ".prumo", "runtime"))
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"runtimes": names}))
		}
		fmt.Println(names)
		return exitOK
	case "tool":
		return runTool(asJSON, root, args[1:])
	case "model":
		return runModel(asJSON, root, args[1:])
	case "env":
		return runEnv(asJSON, root, args[1:])
	case "debug":
		if len(args) < 2 || args[1] != "bundle" {
			return exitUsage
		}
		bundle := map[string]any{"run_id": id, "sanitized": true, "repository": runtime.InspectRepository(root), "note": "prompts, secrets, transcripts, and raw sensitive payloads excluded"}
		if err := runtime.SaveJSON(filepath.Join(root, ".prumo", "runtime", "debug-"+id+".json"), bundle); err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(bundle))
		}
		fmt.Printf("Debug bundle written for %s\n", id)
		return exitOK
	case "context":
		sources := []contextcompiler.Source{{Ref: "ENTRYPOINT.md", Authority: "canonical", Freshness: "current", TokenCost: 100}, {Ref: "docs/PRUMO.md", Authority: "canonical", Freshness: "current", TokenCost: 100}, {Ref: "README.md", Authority: "reference", Freshness: "current", TokenCost: 100}}
		manifest := contextcompiler.Compile(id, sources, 200)
		if err := runtime.SaveJSON(filepath.Join(root, ".prumo", "runtime", "context", id+".manifest.json"), manifest); err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(manifest))
		}
		fmt.Printf("Context manifest: %d tokens (%s)\n", manifest.EstimatedTokens, manifest.Pressure)
		return exitOK
	case "budget":
		if len(args) > 1 && args[1] == "explain" {
			data, err := runtime.LoadJSON[map[string]any](filepath.Join(root, ".prumo", "runtime", "budget.json"))
			if err != nil {
				return serviceError(asJSON, err)
			}
			if asJSON {
				return printEnvelope(protocol.OkEnvelope(data))
			}
			fmt.Printf("Budget: %v\n", data)
			return exitOK
		}
		data, err := runtime.LoadJSON[map[string]any](filepath.Join(root, ".prumo", "runtime", "budget.json"))
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(data))
		}
		fmt.Printf("%v\n", data)
		return exitOK
	case "run":
		if len(args) > 1 && args[1] == "context" {
			sources := []contextcompiler.Source{{Ref: "ENTRYPOINT.md", Authority: "canonical", Freshness: "current", TokenCost: 100}, {Ref: "docs/PRUMO.md", Authority: "canonical", Freshness: "current", TokenCost: 100}, {Ref: "README.md", Authority: "reference", Freshness: "current", TokenCost: 100}}
			manifest := contextcompiler.Compile(id, sources, 200)
			if err := runtime.SaveJSON(filepath.Join(root, ".prumo", "runtime", "context", id+".manifest.json"), manifest); err != nil {
				return serviceError(asJSON, err)
			}
			if asJSON {
				return printEnvelope(protocol.OkEnvelope(manifest))
			}
			fmt.Printf("Context manifest: %d tokens (%s)\n", manifest.EstimatedTokens, manifest.Pressure)
			return exitOK
		}
		if len(args) > 1 && (args[1] == "cancel" || args[1] == "resume") {
			run, err := runtime.LoadJSON[runtime.Run](filepath.Join(root, ".prumo", "runtime", "runs", id+".json"))
			if err != nil {
				return serviceError(asJSON, err)
			}
			if args[1] == "cancel" {
				run, err = run.Transition(runtime.RunCancelled)
			} else {
				run, err = run.Transition(runtime.RunRunning)
			}
			if err != nil {
				return serviceError(asJSON, err)
			}
			if err := runtime.SaveJSON(filepath.Join(root, ".prumo", "runtime", "runs", id+".json"), run); err != nil {
				return serviceError(asJSON, err)
			}
			if asJSON {
				return printEnvelope(protocol.OkEnvelope(run))
			}
			fmt.Printf("Run %s: %s\n", run.ID, run.Status)
			return exitOK
		}
		if len(args) > 1 && args[1] == "show" {
			record, err := runtime.LoadJSON[runtime.Run](filepath.Join(root, ".prumo", "runtime", "runs", id+".json"))
			if err != nil {
				return serviceError(asJSON, err)
			}
			if asJSON {
				return printEnvelope(protocol.OkEnvelope(record))
			}
			fmt.Printf("Run %s: %s\n", record.ID, record.Status)
			return exitOK
		}
		run := runtime.NewRun(id, "", "")
		if err := runtime.SaveJSON(filepath.Join(root, ".prumo", "runtime", "runs", id+".json"), run); err != nil {
			return serviceError(asJSON, err)
		}
		record := runtime.ContinuationFromRun(run, runtime.InspectRepository(root), nil, []string{"start work"}, nil, []string{"inspect Goal and repository state"})
		if err := runtime.SaveJSON(path, record); err != nil {
			return serviceError(asJSON, err)
		}
		if err := runtime.SaveJSON(filepath.Join(root, ".prumo", "runtime", "budget.json"), map[string]any{"version": 1, "scope": "run", "limits": map[string]any{"input_tokens": 8000, "output_tokens": 3000, "tool_calls": 20}, "usage": map[string]any{}, "reservations": []any{}, "mode": "soft"}); err != nil {
			return serviceError(asJSON, err)
		}
		if err := observability.Append(filepath.Join(root, ".prumo", "runtime", "events.jsonl"), observability.NewEvent("run.created", "run.created", id, nil)); err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(run))
		}
		fmt.Printf("Created Run %s\n", id)
		return exitOK
	case "continue":
		record, err := runtime.LoadJSON[runtime.ContinuationRecord](path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if hasFlag(args, "--prompt") {
			fmt.Print(runtime.RenderPrompt(record))
			return exitOK
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(record))
		}
		fmt.Print(runtime.RenderPrompt(record))
		return exitOK
	default:
		return exitUsage
	}
}
