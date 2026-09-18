package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/docpublish"
	"github.com/raillen/prumo/internal/protocol"
)

// docsValueFlags are the flags that consume the next argument. They are skipped
// when a docs subcommand's positional arguments are extracted, so a flag value
// can never be mistaken for a subcommand.
var docsValueFlags = map[string]bool{
	"--path": true, "--goal": true, "--out": true, "--changed": true,
	"--level": true, "--budget": true, "--id": true, "--required": true,
	"--locale": true, "--renderers": true, "--limit": true,
}

// positionalDocsArgs returns the arguments of `docs <command>` with flags and
// their values removed.
func positionalDocsArgs(args []string) []string {
	out := []string{}
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			if docsValueFlags[arg] && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, arg)
	}
	return out
}

// runDocsResources exposes the read-only documentation resource surface. It is
// the headless view of the same resources an MCP client would list and read
// (W18.11).
func runDocsResources(asJSON bool, args []string, root string) (any, int) {
	positional := positionalDocsArgs(args)
	sub := "list"
	if len(positional) > 0 {
		sub = positional[0]
	}
	server, err := docpublish.NewServer(root, docpublish.Options{Version: protocol.CLIVersion})
	if err != nil {
		return nil, serviceError(asJSON, err)
	}
	switch sub {
	case "list":
		resources := server.List()
		return map[string]any{"resources": resources, "count": len(resources)}, exitOK
	case "read":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "error: docs resources read requires a uri, e.g. prumo://docs/product/vision")
			return nil, exitUsage
		}
		content, err := server.Read(positional[1])
		if err != nil {
			return nil, serviceError(asJSON, err)
		}
		if !asJSON {
			fmt.Print(content.Text)
			return content, exitOK
		}
		return content, exitOK
	case "tools":
		tools := server.Tools()
		return map[string]any{"tools": tools, "count": len(tools)}, exitOK
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported docs resources subcommand: %s (list|read|tools)\n", sub)
		return nil, exitUsage
	}
}

// loadMutationPolicy reads the mutation gate from the documentation lifecycle
// registry. An absent or unreadable policy denies everything: not configuring
// the writer must never mean enabling it.
func loadMutationPolicy(root string) (docpublish.MutationPolicy, error) {
	policy := docpublish.MutationPolicy{}
	data, err := os.ReadFile(filepath.Join(root, "docs", "lifecycle.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return policy, nil
		}
		return policy, err
	}
	var registry struct {
		Mutations struct {
			Allowed         []string `json:"allowed"`
			RequireApproval bool     `json:"require_approval"`
		} `json:"mutations"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return policy, err
	}
	policy.Allowed = registry.Mutations.Allowed
	policy.RequireApproval = registry.Mutations.RequireApproval
	return policy, nil
}

// runDocsMutations reports the mutation gate. It never mutates: enabling the
// writer is a configuration decision, and an operator must be able to see the
// state before and after making it (W18.12).
func runDocsMutations(asJSON bool, args []string, root string) (any, int) {
	positional := positionalDocsArgs(args)
	sub := "describe"
	if len(positional) > 0 {
		sub = positional[0]
	}
	if sub != "describe" {
		fmt.Fprintf(os.Stderr, "error: unsupported docs mutations subcommand: %s (describe)\n", sub)
		return nil, exitUsage
	}
	policy, err := loadMutationPolicy(root)
	if err != nil {
		return nil, serviceError(asJSON, err)
	}
	server := docpublish.MutationServer{Policy: policy}
	report := server.Describe()
	report["note"] = "the read-only surface cannot mutate; enabling a verb here is the only way documentation changes through this surface"
	return report, exitOK
}
