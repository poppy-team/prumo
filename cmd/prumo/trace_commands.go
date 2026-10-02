package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	docengine "github.com/raillen/prumo/internal/documentation"
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

	// A project with no persisted graph is derived from its own canonical state.
	// Nothing is invented here: an empty project yields an empty graph, and the
	// query below then reports a real "not found" instead of fabricated lineage.
	if len(graph.Nodes) == 0 {
		buildTraceGraphFromProject(graph, path)
	}

	trace, err := graph.Trace(ref)
	if err != nil {
		if len(graph.Nodes) == 0 {
			err = fmt.Errorf("no traceability graph for %s: no Goals, planning decisions or documentation bindings were found under %s", ref, path)
		} else if _, found := graph.FindNode(ref); !found {
			known := make([]string, 0, len(graph.Nodes))
			for id := range graph.Nodes {
				known = append(known, id)
			}
			sort.Strings(known)
			err = fmt.Errorf("trace node %q not found in project; known nodes: %s", ref, strings.Join(known, ", "))
		}
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

// buildTraceGraphFromProject derives a traceability graph from the project's own
// canonical state: Goals under .ai/goals, planning decisions under .ai/plan,
// harness evidence under .prumo/runtime, and documentation bindings.
//
// It deliberately invents nothing. Every node it adds corresponds to a record
// that exists on disk, because a projection must never contribute project facts
// of its own. A project with no recorded state yields an empty graph, and the
// caller reports that honestly instead of answering with invented lineage.
func buildTraceGraphFromProject(g *traceability.Graph, root string) {
	goalIDs := map[string]bool{}

	root = filepath.Clean(root)
	_ = filepath.Walk(filepath.Join(root, ".ai", "goals"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.Contains(filepath.Base(path), ".goal.") {
			return nil
		}
		record, readErr := readJSONObject(path)
		if readErr != nil {
			return nil
		}
		id := stringField(record, "id")
		if id == "" {
			return nil
		}
		title := stringField(record, "title")
		if title == "" {
			title = id
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		goalIDs[id] = true
		_ = g.AddNode(traceability.Node{
			ID: id, Kind: traceability.NodeGoal, Title: title, Ref: filepath.ToSlash(rel),
			Metadata: map[string]string{
				"state": stringField(record, "state"),
				"phase": stringField(record, "phase"),
			},
		})
		return nil
	})

	// Goal dependencies are real declared edges, so they become real graph edges.
	_ = filepath.Walk(filepath.Join(root, ".ai", "goals"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.Contains(filepath.Base(path), ".goal.") {
			return nil
		}
		record, readErr := readJSONObject(path)
		if readErr != nil {
			return nil
		}
		id := stringField(record, "id")
		if id == "" {
			return nil
		}
		for _, dep := range stringSliceField(record, "dependencies") {
			_ = g.AddEdge(traceability.Edge{From: id, To: dep, Kind: string(traceability.EdgeDerivesFrom)})
		}
		for _, ev := range objectSliceField(record, "evidence") {
			evID := stringField(ev, "id")
			if evID == "" {
				continue
			}
			_ = g.AddNode(traceability.Node{
				ID: evID, Kind: traceability.NodeEvidence, Title: evID, Ref: stringField(ev, "artifact"),
				Metadata: map[string]string{"type": stringField(ev, "type")},
			})
			_ = g.AddEdge(traceability.Edge{From: id, To: evID, Kind: string(traceability.EdgeEvidencedBy)})
		}
		return nil
	})

	// Planning decisions belong to the Goal their session is scoped to.
	_ = filepath.Walk(filepath.Join(root, ".ai", "plan", "sessions"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		record, readErr := readJSONObject(path)
		if readErr != nil {
			return nil
		}
		goal := stringField(record, "goal")
		for _, decision := range objectSliceField(record, "decisions") {
			dID := stringField(decision, "id")
			if dID == "" {
				continue
			}
			statement := stringField(decision, "statement")
			_ = g.AddNode(traceability.Node{
				ID: dID, Kind: traceability.NodeDecision, Title: statement, Ref: filepath.ToSlash(path),
				Metadata: map[string]string{
					"authority": stringField(decision, "authority"),
					"status":    stringField(decision, "status"),
				},
			})
			if goal != "" && goalIDs[goal] {
				_ = g.AddEdge(traceability.Edge{From: goal, To: dID, Kind: string(traceability.EdgeDerivesFrom)})
			}
		}
		return nil
	})

	// Documentation contracts are the requirement side of the graph; a binding
	// ties a contract to the canonical documents that satisfy it.
	bindings, bindErr := docengine.LoadBindings(root)
	if bindErr != nil {
		return
	}
	for _, binding := range bindings {
		reqID := "contract:" + binding.ContractID
		_ = g.AddNode(traceability.Node{
			ID: reqID, Kind: traceability.NodeRequirement, Title: binding.ContractID,
			Ref:      binding.ContractID,
			Metadata: map[string]string{"authority": binding.Authority, "ownership": binding.Ownership},
		})
		for _, source := range binding.Sources {
			docID := "doc:" + source
			_ = g.AddNode(traceability.Node{ID: docID, Kind: traceability.NodeDoc, Title: filepath.Base(source), Ref: source})
			_ = g.AddEdge(traceability.Edge{From: reqID, To: docID, Kind: string(traceability.EdgeSatisfies)})
		}
	}
}

func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func stringField(record map[string]any, key string) string {
	if record == nil {
		return ""
	}
	value, _ := record[key].(string)
	return strings.TrimSpace(value)
}

func stringSliceField(record map[string]any, key string) []string {
	if record == nil {
		return nil
	}
	raw, ok := record[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if value := strings.TrimSpace(fmt.Sprint(item)); value != "" && value != "<nil>" {
			out = append(out, value)
		}
	}
	return out
}

func objectSliceField(record map[string]any, key string) []map[string]any {
	if record == nil {
		return nil
	}
	raw, ok := record[key].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if object, ok := item.(map[string]any); ok {
			out = append(out, object)
		}
	}
	return out
}
