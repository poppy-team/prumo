// Semantic impact matchers (W17.1–W17.4): the typed trigger vocabulary is
// extended beyond paths and extensions so a change can be described by what it
// *is* (a symbol, a schema field, a design token, a UI component, an API, an
// event, a permission, a locale, media, a Goal/Wave/ADR/release, evidence).
//
// Every matcher is deterministic and evidence-based:
//
//	symbol:<ident>      the identifier occurs in a changed source file
//	schema:<name>       a changed file is the named JSON Schema (with/without extension)
//	token:<id>          a changed design-token file declares the token id
//	ui:<id>             a changed path is bound to the contract that owns the ui.* id
//	api|event|permission|locale|media|goal|wave|adr|release|evidence:<id>
//	                    the id occurs in a changed file or in the contract's bound documents
//
// Unresolvable triggers never match: a matcher that cannot see its subject must
// not claim impact.
package docengine

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/traceability"
)

// SemanticTriggerKinds lists the identifier-based namespaces.
var SemanticTriggerKinds = []string{
	"api", "event", "permission", "locale", "media", "goal", "wave", "adr", "release", "evidence",
}

var sourceExtensions = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".py": true, ".rs": true,
}

// MatchTriggerIn resolves a trigger with repository access. It is the semantic
// entry point; MatchTrigger stays for callers without a root (legacy matchers
// only).
func MatchTriggerIn(root, trigger string, changed, sources []string, g *traceability.Graph) (bool, string) {
	if name, arg, ok := strings.Cut(trigger, ":"); ok {
		switch name {
		case "symbol":
			if symbolTouched(root, arg, changed) {
				return true, "trigger:symbol:" + arg
			}
			return false, ""
		case "schema":
			if schemaTouched(arg, changed) {
				return true, "trigger:schema:" + arg
			}
			return false, ""
		case "token":
			if tokenTouched(root, arg, changed) {
				return true, "trigger:token:" + arg
			}
			return false, ""
		case "ui":
			if uiTouched(root, arg, changed) {
				return true, "trigger:ui:" + arg
			}
			return false, ""
		default:
			if containsString(SemanticTriggerKinds, name) {
				if identifierTouched(root, arg, changed, sources) {
					return true, "trigger:" + name + ":" + arg
				}
				return false, ""
			}
		}
	}
	return MatchTrigger(trigger, changed, sources, g)
}

func symbolTouched(root, ident string, changed []string) bool {
	ident = strings.TrimSpace(ident)
	ident = strings.TrimSuffix(ident, "()")
	if ident == "" {
		return false
	}
	word := regexp.MustCompile(`\b` + regexp.QuoteMeta(ident) + `\b`)
	for _, path := range changed {
		if !sourceExtensions[strings.ToLower(filepath.Ext(path))] {
			continue
		}
		if word.MatchString(fileContent(root, path)) {
			return true
		}
	}
	return false
}

func schemaTouched(name string, changed []string) bool {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".schema.json")
	name = strings.TrimSuffix(name, ".json")
	if name == "" {
		return false
	}
	for _, path := range changed {
		base := filepath.Base(filepath.ToSlash(path))
		if base == name+".schema.json" || base == name+".json" {
			return true
		}
	}
	return false
}

func tokenTouched(root, id string, changed []string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, path := range changed {
		if !strings.Contains(strings.ToLower(path), "token") {
			continue
		}
		if strings.Contains(fileContent(root, path), id) {
			return true
		}
	}
	return false
}

func uiTouched(root, id string, changed []string) bool {
	id = strings.TrimSpace(strings.TrimPrefix(id, "ui."))
	if id == "" {
		return false
	}
	registry, err := LoadRegistry(root)
	if err != nil {
		return false
	}
	bindings, err := LoadBindings(root)
	if err != nil {
		return false
	}
	contracts := map[string][]string{}
	for _, b := range bindings {
		contracts[b.ContractID] = b.Sources
	}
	for _, contract := range registry.Contracts {
		if !strings.HasPrefix(contract.ID, "ui.") && contract.ID != "ui.documentation" {
			continue
		}
		if contract.ID != "ui."+id && contract.ID != id {
			continue
		}
		for _, source := range contracts[contract.ID] {
			for _, path := range changed {
				if path == source || strings.HasPrefix(path, strings.TrimSuffix(source, "/")+"/") {
					return true
				}
			}
		}
	}
	return false
}

// identifierTouched reports whether a named identifier is present in a changed
// file or in the documents bound to the contract. Both are real occurrences of
// the name, never a fuzzy guess.
func identifierTouched(root, id string, changed, sources []string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`)
	for _, path := range changed {
		if pattern.MatchString(fileContent(root, path)) {
			return true
		}
	}
	for _, source := range sources {
		if pattern.MatchString(fileContent(root, source)) {
			return true
		}
	}
	return false
}

func fileContent(root, rel string) string {
	if root == "" || rel == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return ""
	}
	return string(data)
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// AnalyzeImpactsIn is AnalyzeImpactsWithGraph over the semantic matchers.
func AnalyzeImpactsIn(root string, registry Registry, bindings []Binding, changed []string, g *traceability.Graph) []Impact {
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
			if ok, reason := MatchTriggerIn(root, trigger, changed, byContract[contract.ID], g); ok {
				docs := append([]string{}, byContract[contract.ID]...)
				sort.Strings(docs)
				impacts = append(impacts, Impact{ContractID: contract.ID, Documents: docs, Reason: reason, Severity: impactSeverity(trigger)})
				break
			}
		}
	}
	return impacts
}

// impactSeverity maps a trigger namespace to a severity. Semantic namespaces
// (symbol/schema/api/event/permission/token/ui) are contract-bearing changes;
// path/ext are structural.
func impactSeverity(trigger string) string {
	name, _, _ := strings.Cut(trigger, ":")
	switch name {
	case "symbol", "schema", "api", "event", "permission", "token", "ui", "media", "locale":
		return "high"
	default:
		return "medium"
	}
}
