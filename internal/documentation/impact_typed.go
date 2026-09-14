// Typed impact triggers (GAP-045): migrate update-trigger matching from
// bare-substring guessing to explicit matchers. Legacy bare tokens keep
// working but are labeled legacy-substring in reasons for migration
// visibility. Supported:
//
//	path:<prefix>        changed path under prefix
//	ext:<.ext>           changed file extension
//	contract:<id>        changed path covered by the contract's bindings
//	rel:<kind>:<ref>     changed path node linked by typed edge to ref
//	<bare token>         legacy substring (deprecated)
package docengine

import (
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/traceability"
)

// MatchTrigger resolves one trigger against changed paths. Sources are the
// binding-declared source refs of the candidate contract; g may be nil
// (rel: matchers then miss, everything else still resolves).
func MatchTrigger(trigger string, changed, sources []string, g *traceability.Graph) (bool, string) {
	if name, arg, ok := strings.Cut(trigger, ":"); ok {
		switch name {
		case "path":
			prefix := strings.TrimSuffix(arg, "/")
			for _, path := range changed {
				if path == arg || strings.HasPrefix(path, prefix+"/") {
					return true, "trigger:path:" + arg
				}
			}
			return false, ""
		case "ext":
			ext := strings.ToLower(arg)
			for _, path := range changed {
				if strings.HasSuffix(strings.ToLower(path), ext) {
					return true, "trigger:ext:" + arg
				}
			}
			return false, ""
		case "contract":
			for _, path := range changed {
				for _, src := range sources {
					if path == src || strings.HasPrefix(path, strings.TrimSuffix(src, "/")+"/") {
						return true, "trigger:contract:" + arg
					}
				}
			}
			return false, ""
		case "rel":
			kind, ref, ok := strings.Cut(arg, ":")
			if !ok || g == nil {
				return false, ""
			}
			for _, path := range changed {
				if linkedBy(path, traceability.EdgeKind(kind), ref, g) {
					return true, "trigger:rel:" + arg
				}
			}
			return false, ""
		}
	}
	if triggerMatches(trigger, changed) {
		return true, "legacy-substring:" + trigger
	}
	return false, ""
}

func linkedBy(path string, kind traceability.EdgeKind, ref string, g *traceability.Graph) bool {
	if !kind.Valid() || g == nil {
		return false
	}
	matches := func(nodeRef string) bool {
		return nodeRef != "" && (nodeRef == path || strings.HasSuffix(nodeRef, "/"+strings.TrimPrefix(path, "/")))
	}
	ids := map[string]bool{}
	for id, n := range g.Nodes {
		if matches(n.Ref) {
			ids[id] = true
		}
	}
	for _, e := range g.Edges {
		if traceability.EdgeKind(e.Kind) != kind {
			continue
		}
		if ids[e.From] && nodeRefIs(g, e.To, ref) {
			return true
		}
		if ids[e.To] && nodeRefIs(g, e.From, ref) {
			return true
		}
	}
	return false
}

func nodeRefIs(g *traceability.Graph, id, ref string) bool {
	n, ok := g.Nodes[id]
	if !ok {
		return false
	}
	return n.Ref == ref || n.ID == ref
}

// AnalyzeImpactsWithGraph is AnalyzeImpacts over typed triggers.
func AnalyzeImpactsWithGraph(registry Registry, bindings []Binding, changed []string, g *traceability.Graph) []Impact {
	byContract := map[string][]string{}
	for _, b := range bindings {
		byContract[b.ContractID] = append(byContract[b.ContractID], b.Sources...)
	}
	contracts := make([]Contract, 0, len(registry.Contracts))
	for _, c := range registry.Contracts {
		contracts = append(contracts, c)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].ID < contracts[j].ID })
	impacts := []Impact{}
	for _, contract := range contracts {
		for _, trigger := range contract.UpdateTriggers {
			if ok, reason := MatchTrigger(trigger, changed, byContract[contract.ID], g); ok {
				docs := append([]string{}, byContract[contract.ID]...)
				sort.Strings(docs)
				impacts = append(impacts, Impact{ContractID: contract.ID, Documents: docs, Reason: reason, Severity: "medium"})
				break
			}
		}
	}
	return impacts
}
