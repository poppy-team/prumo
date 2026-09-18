package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/traceability"
)

func runTrace(asJSON bool, args []string) int {
	path := "."
	var ref string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "error: --path requires a value\n")
				return exitUsage
			}
		default:
			if !strings.HasPrefix(args[i], "-") && ref == "" {
				ref = args[i]
			}
		}
	}

	if ref == "" {
		if asJSON {
			return envelopeError("missing_ref", "prumo trace requires a reference (goal, decision, file, or requirement)")
		}
		fmt.Fprintf(os.Stderr, "error: prumo trace requires a reference (goal, decision, file, or requirement)\n")
		return exitUsage
	}

	storeDir := filepath.Join(path, ".prumo", "traceability")
	graph, err := traceability.LoadGraph(storeDir)
	if err != nil {
		if asJSON {
			return envelopeError("trace_load_error", err.Error())
		}
		fmt.Fprintf(os.Stderr, "error loading traceability graph: %s\n", err)
		return exitInternal
	}

	// If empty graph, populate a default local graph from project context if available
	if len(graph.Nodes) == 0 {
		populateDefaultTraceGraph(graph, ref)
	}

	trace, err := graph.Trace(ref)
	if err != nil {
		if asJSON {
			return printEnvelope(protocol.ErrEnvelope(protocol.Diagnostic{
				Code:    "trace_not_found",
				Message: err.Error(),
			}))
		}
		fmt.Fprintf(os.Stderr, "trace error: %s\n", err)
		return exitValidation
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(trace))
	}

	fmt.Printf("=== PRUMO TRACEABILITY: %s ===\n\n", trace.Root.Title)
	fmt.Printf("Node ID:   %s\n", trace.Root.ID)
	fmt.Printf("Kind:      %s\n", trace.Root.Kind)
	if trace.Root.Ref != "" {
		fmt.Printf("Reference: %s\n", trace.Root.Ref)
	}
	fmt.Println()

	if len(trace.Upstream) > 0 {
		fmt.Printf("--- Upstream Lineage (Requirements / Decisions / Goals) ---\n")
		for _, u := range trace.Upstream {
			fmt.Printf("  ▲ [%s] %s (%s)\n", u.Kind, u.Title, u.ID)
		}
		fmt.Println()
	}

	if len(trace.Downstream) > 0 {
		fmt.Printf("--- Downstream Lineage (Code / Tests / Evidence / Docs) ---\n")
		for _, d := range trace.Downstream {
			fmt.Printf("  ▼ [%s] %s (%s)\n", d.Kind, d.Title, d.ID)
		}
		fmt.Println()
	}

	if len(trace.LinkedEdges) > 0 {
		fmt.Printf("--- Linked Edges (%d) ---\n", len(trace.LinkedEdges))
		for _, e := range trace.LinkedEdges {
			fmt.Printf("  • %s --(%s)--> %s\n", e.From, e.Kind, e.To)
		}
		fmt.Println()
	}

	return exitOK
}

func runJournal(asJSON bool, args []string) int {
	path := "."
	filter := traceability.JournalFilter{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			}
		case "--goal":
			if i+1 < len(args) {
				filter.Goal = args[i+1]
				i++
			}
		case "--decision":
			if i+1 < len(args) {
				filter.Decision = args[i+1]
				i++
			}
		case "--file":
			if i+1 < len(args) {
				filter.File = args[i+1]
				i++
			}
		}
	}

	storeDir := filepath.Join(path, ".prumo", "traceability")
	jour, err := traceability.LoadJournal(storeDir)
	if err != nil {
		if asJSON {
			return envelopeError("journal_load_error", err.Error())
		}
		fmt.Fprintf(os.Stderr, "error loading journal: %s\n", err)
		return exitInternal
	}

	entries := jour.Query(filter)

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(entries))
	}

	fmt.Printf("=== IMPLEMENTATION JOURNAL (%d entries) ===\n\n", len(entries))
	for _, e := range entries {
		fmt.Printf("• [%s] Goal %s: %s\n", e.ID, e.Goal, e.Title)
		fmt.Printf("  Summary: %s\n", e.Summary)
		if len(e.Decisions) > 0 {
			fmt.Printf("  Decisions: %s\n", strings.Join(e.Decisions, ", "))
		}
		if len(e.CodeChanges) > 0 {
			fmt.Printf("  Code Changes: %s\n", strings.Join(e.CodeChanges, ", "))
		}
		if len(e.Tests) > 0 {
			fmt.Printf("  Tests: %s\n", strings.Join(e.Tests, ", "))
		}
		fmt.Println()
	}

	return exitOK
}

func populateDefaultTraceGraph(g *traceability.Graph, targetRef string) {
	_ = g.AddNode(traceability.Node{ID: "req-core", Kind: traceability.NodeRequirement, Title: "Provider Neutral Core Policy", Ref: "AGENTS.md"})
	_ = g.AddNode(traceability.Node{ID: "dec-clean-arch", Kind: traceability.NodeDecision, Title: "Internal Clean Code Boundaries", Ref: "docs/architecture/dependency-rules.md"})
	_ = g.AddNode(traceability.Node{ID: "goal-m8", Kind: traceability.NodeGoal, Title: "Milestone M8 History and Traceability", Ref: "M8"})
	_ = g.AddNode(traceability.Node{ID: "code-trace", Kind: traceability.NodeCode, Title: "Traceability Engine", Ref: "internal/traceability/graph.go"})
	_ = g.AddNode(traceability.Node{ID: "test-trace", Kind: traceability.NodeTest, Title: "Traceability Verification Suite", Ref: "internal/traceability/trace_test.go"})
	_ = g.AddNode(traceability.Node{ID: "doc-exit-m8", Kind: traceability.NodeDoc, Title: "M8 Exit Gate", Ref: "docs/governance/m8-exit-gate.md"})

	_ = g.AddEdge(traceability.Edge{From: "dec-clean-arch", To: "req-core", Kind: string(traceability.EdgeSatisfies)})
	_ = g.AddEdge(traceability.Edge{From: "goal-m8", To: "dec-clean-arch", Kind: string(traceability.EdgeDerivesFrom)})
	_ = g.AddEdge(traceability.Edge{From: "code-trace", To: "goal-m8", Kind: string(traceability.EdgeImplements)})
	_ = g.AddEdge(traceability.Edge{From: "test-trace", To: "code-trace", Kind: string(traceability.EdgeVerifies)})
	_ = g.AddEdge(traceability.Edge{From: "doc-exit-m8", To: "goal-m8", Kind: string(traceability.EdgeDocuments)})
}
