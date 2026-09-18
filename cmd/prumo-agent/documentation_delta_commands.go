package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/protocol"
)

type docsDeltaRequest struct {
	action   string
	root     string
	goal     string
	id       string
	state    string
	changed  []string
	evidence []string
}

func runDocsDelta(asJSON bool, args []string, root, goal string) int {
	request, err := parseDocsDelta(args, root, goal)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if request.action == "preview" {
		impact, err := docengine.AnalyzeImpact(request.root, []string{})
		if err != nil {
			return serviceError(asJSON, err)
		}
		return printDeltaResult(asJSON, docengine.MakeDelta(request.goal, impact))
	}
	switch request.action {
	case "propose":
		if request.goal == "" {
			fmt.Fprintln(os.Stderr, "error: docs delta propose requires --goal")
			return exitUsage
		}
		registry, err := docengine.LoadRegistry(repoRoot())
		if err != nil {
			return serviceError(asJSON, err)
		}
		bindings, err := docengine.LoadBindings(request.root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		delta := docengine.NewDelta(request.goal, docengine.AnalyzeImpacts(registry, bindings, request.changed), time.Now())
		if err := docengine.SaveDelta(request.root, delta); err != nil {
			return serviceError(asJSON, err)
		}
		return printDeltaResult(asJSON, delta)
	case "list":
		deltas, err := docengine.ListDeltas(request.root)
		if err != nil {
			return serviceError(asJSON, err)
		}
		return printDeltaResult(asJSON, deltas)
	case "show":
		delta, err := docengine.LoadDelta(request.root, request.id)
		if err != nil {
			return serviceError(asJSON, err)
		}
		return printDeltaResult(asJSON, delta)
	case "transition":
		delta, err := docengine.TransitionDelta(request.root, request.id, request.state, request.evidence, time.Now())
		if err != nil {
			return serviceError(asJSON, err)
		}
		return printDeltaResult(asJSON, delta)
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported docs delta command: %s\n", request.action)
		return exitUsage
	}
}

func parseDocsDelta(args []string, root, goal string) (docsDeltaRequest, error) {
	request := docsDeltaRequest{action: "preview", root: root, goal: goal}
	if len(args) <= 2 || strings.HasPrefix(args[2], "--") {
		return request, nil
	}
	switch args[2] {
	case "propose", "list", "show", "transition":
		request.action = args[2]
	default:
		return request, nil
	}
	positionals := []string{}
	for index := 3; index < len(args); index++ {
		switch args[index] {
		case "--path", "--goal", "--id", "--state":
			if index+1 >= len(args) {
				return docsDeltaRequest{}, fmt.Errorf("%s requires a value", args[index])
			}
			switch args[index] {
			case "--path":
				request.root = args[index+1]
			case "--goal":
				request.goal = args[index+1]
			case "--id":
				request.id = args[index+1]
			case "--state":
				request.state = args[index+1]
			}
			index++
		case "--evidence":
			if index+1 >= len(args) {
				return docsDeltaRequest{}, fmt.Errorf("--evidence requires a value")
			}
			request.evidence = append(request.evidence, args[index+1])
			index++
		default:
			if strings.HasPrefix(args[index], "--") {
				return docsDeltaRequest{}, fmt.Errorf("unsupported docs delta option: %s", args[index])
			}
			positionals = append(positionals, args[index])
		}
	}
	switch request.action {
	case "propose":
		request.changed = positionals
	case "list", "show", "transition":
		if len(positionals) > 0 {
			return docsDeltaRequest{}, fmt.Errorf("docs delta %s takes no positional arguments", request.action)
		}
	}
	if (request.action == "show" || request.action == "transition") && request.id == "" {
		return docsDeltaRequest{}, fmt.Errorf("docs delta %s requires --id", request.action)
	}
	if request.action == "transition" && request.state == "" {
		return docsDeltaRequest{}, fmt.Errorf("docs delta transition requires --state")
	}
	return request, nil
}

func printDeltaResult(asJSON bool, result any) int {
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	return exitOK
}
