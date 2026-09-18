package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/experience"
	"github.com/raillen/prumo/internal/protocol"
)

func runExperience(asJSON bool, args []string) int {
	if len(args) == 0 {
		return experienceUsage()
	}

	path := "."
	cleaned := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--path" {
			if i+1 < len(args) {
				path = args[i+1]
				i++
				continue
			}
		}
		cleaned = append(cleaned, args[i])
	}
	args = cleaned
	if len(args) == 0 {
		return experienceUsage()
	}

	storeDir := filepath.Join(path, ".prumo", "experience")
	prov, err := experience.NewFileProvider(storeDir)
	if err != nil {
		if asJSON {
			return envelopeError("experience_init_error", err.Error())
		}
		fmt.Fprintf(os.Stderr, "error initializing experience provider: %s\n", err)
		return exitInternal
	}

	switch args[0] {
	case "status":
		props, _ := prov.GetProposals()
		statusData := map[string]any{
			"storage_dir": storeDir,
			"proposals":   len(props),
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(statusData))
		}
		fmt.Println("=== PRUMO EXPERIENCE LAYER STATUS ===")
		fmt.Printf("Storage Directory: %s\n", storeDir)
		fmt.Printf("Active Proposals:  %d\n", len(props))
		return exitOK

	case "handoff":
		return runExperienceHandoff(prov, asJSON, args[1:])

	case "events":
		sessionID := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--session" && i+1 < len(args) {
				sessionID = args[i+1]
				i++
			}
		}
		if sessionID == "" {
			fmt.Fprintf(os.Stderr, "error: experience events requires --session <id>\n")
			return exitUsage
		}
		events, err := prov.GetEvents(sessionID)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(events))
		}
		fmt.Printf("=== SESSION EVENTS (%s: %d events) ===\n\n", sessionID, len(events))
		for _, ev := range events {
			fmt.Printf("[%s] %s: %s\n", ev.Timestamp, ev.Type, ev.Summary)
		}
		return exitOK

	default:
		fmt.Fprintf(os.Stderr, "error: unknown experience subcommand: %s\n", args[0])
		return exitUsage
	}
}

func runExperienceHandoff(prov *experience.FileProvider, asJSON bool, args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: prumo experience handoff [create|show|ack] [options]\n")
		return exitUsage
	}

	switch args[0] {
	case "create":
		id := ""
		from := ""
		to := ""
		goal := ""
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--id":
				if i+1 < len(args) {
					id = args[i+1]
					i++
				}
			case "--from":
				if i+1 < len(args) {
					from = args[i+1]
					i++
				}
			case "--to":
				if i+1 < len(args) {
					to = args[i+1]
					i++
				}
			case "--goal":
				if i+1 < len(args) {
					goal = args[i+1]
					i++
				}
			}
		}

		if id == "" || from == "" || to == "" {
			fmt.Fprintf(os.Stderr, "error: handoff create requires --id, --from, and --to\n")
			return exitUsage
		}

		summary := experience.Summary{
			SessionID: id,
			Completed: []string{"Completed planning phase"},
			NextSteps: []string{"Implement core components"},
		}

		ho, err := experience.CreateHandoff(id, from, to, "run-auto", goal, summary, []string{"dec-001"}, nil, nil)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if err := prov.CreateHandoff(ho); err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(ho))
		}
		fmt.Printf("Created handoff %s (%s -> %s)\n", ho.ID, ho.From, ho.To)
		return exitOK

	case "show":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "error: handoff show requires an ID\n")
			return exitUsage
		}
		ho, err := prov.GetHandoff(args[1])
		if err != nil {
			if asJSON {
				return envelopeError("handoff_not_found", err.Error())
			}
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitValidation
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(ho))
		}
		fmt.Printf("=== HANDOFF: %s (%s) ===\n", ho.ID, ho.Status)
		fmt.Printf("From: %s  ->  To: %s\n", ho.From, ho.To)
		if ho.GoalID != "" {
			fmt.Printf("Goal: %s\n", ho.GoalID)
		}
		fmt.Printf("Next Steps: %s\n", strings.Join(ho.Summary.NextSteps, ", "))
		return exitOK

	case "ack":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "error: handoff ack requires an ID\n")
			return exitUsage
		}
		actor := "agent-recipient"
		for i := 2; i < len(args); i++ {
			if args[i] == "--actor" && i+1 < len(args) {
				actor = args[i+1]
				i++
			}
		}
		ho, err := prov.GetHandoff(args[1])
		if err != nil {
			return serviceError(asJSON, err)
		}
		if err := ho.Acknowledge(actor); err != nil {
			return serviceError(asJSON, err)
		}
		_ = prov.CreateHandoff(ho)
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(ho))
		}
		fmt.Printf("Acknowledged handoff %s by %s\n", ho.ID, actor)
		return exitOK

	default:
		fmt.Fprintf(os.Stderr, "error: unknown handoff subcommand: %s\n", args[0])
		return exitUsage
	}
}

func experienceUsage() int {
	fmt.Fprintf(os.Stderr, "usage: prumo experience <status|handoff|events> [args]\n")
	return exitUsage
}
