package uimap

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Derived is what the implementation can prove about the interface.
//
// Everything here comes from reading code, so none of it is opinion: exported
// types and functions that could implement an element, and enumerations that
// look like a shell's stages or a palette's commands.
type Derived struct {
	// Symbols are every exported symbol name the sources declare, qualified as
	// `Package.Symbol` the way a map's implementation trace writes them.
	Symbols []string
	// Elements are the subset that could *be* an interface element: types,
	// methods and exported fields.
	//
	// The distinction earns its place. A constant named
	// `DefaultPaletteCapacity` and a field named `Evidence.Status` both match an
	// element-ish word, and reporting them as elements the map forgot turned a
	// useful signal into thirty lines of noise — a warning list nobody reads is
	// worse than no warning list. An element is a type, a method or a field; a
	// capacity is not an interface.
	Elements []string
	// Constants are exported constant identifiers, which is where a shell's
	// stages and a palette's action ids live.
	Constants []string
	// Sources are the files read, so a report can state its own scope.
	Sources []string
}

// SymbolSet is a lookup over Derived.Symbols.
func (d Derived) SymbolSet() map[string]bool {
	set := make(map[string]bool, len(d.Symbols))
	for _, s := range d.Symbols {
		set[s] = true
	}
	return set
}

// SymbolDeriver reads an implementation and reports what it declares.
//
// The core defines the seam instead of a second rule set: a stack whose
// components live in another language or another framework adds a deriver
// adapter. That is what keeps the interface map provider-neutral while still
// letting verification be real.
type SymbolDeriver interface {
	// Derive reads the sources named by the patterns, relative to root.
	Derive(root string, patterns []string) (Derived, error)
}

// GoSymbolDeriver derives symbols from Go source.
//
// It is deliberately shallow. It reads declarations, not semantics: it can prove
// that `tui.Model` exists and cannot know that a node labelled "Run goal" is the
// button wired to it. Guessing that link would make the derivation an authority
// on intent, which is the one thing only the declaration can supply.
type GoSymbolDeriver struct{}

// Derive walks the given glob patterns and collects exported declarations.
func (GoSymbolDeriver) Derive(root string, patterns []string) (Derived, error) {
	out := Derived{}
	seen := map[string]bool{}
	for _, pattern := range patterns {
		matches, err := ExpandPattern(root, pattern)
		if err != nil {
			return Derived{}, err
		}
		sort.Strings(matches)
		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			// Tests are not the interface. Reading them would let a map point at
			// a symbol that only exists under `go test`, and it offered test
			// functions as elements the moment the scope included them.
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			decls, err := readGoFile(path, filepath.Dir(rel))
			if err != nil {
				return Derived{}, err
			}
			if !seen[rel] {
				seen[rel] = true
				out.Sources = append(out.Sources, rel)
			}
			out.Symbols = append(out.Symbols, decls.Symbols...)
			out.Elements = append(out.Elements, decls.Elements...)
			out.Constants = append(out.Constants, decls.Constants...)
		}
	}
	sort.Strings(out.Symbols)
	sort.Strings(out.Elements)
	sort.Strings(out.Constants)
	return out, nil
}

// fileDecls is what one source file declares, split by what each name can be
// used for.
type fileDecls struct {
	Symbols   []string
	Elements  []string
	Constants []string
}

// SymbolResolver answers "does this symbol exist" from a derived set.
type SymbolResolver struct {
	known map[string]bool
}

// NewSymbolResolver builds a resolver over a derived set.
func NewSymbolResolver(derived Derived) *SymbolResolver {
	r := &SymbolResolver{known: map[string]bool{}}
	for _, s := range derived.Symbols {
		r.known[s] = true
	}
	return r
}

// Resolves reports whether a symbol is known.
//
// Both the qualified form (`tui.Model`) and the bare method form
// (`Model.Render`) resolve, so the map does not have to encode the deriver's
// naming convention to be checkable.
func (r *SymbolResolver) Resolves(symbol, path string) bool {
	if symbol == "" {
		return false
	}
	if r.known[symbol] {
		return true
	}
	// Accept the fallback form as well: a lot of interfaces are named by
	// receiver and method without the package.
	if strings.Count(symbol, ".") == 1 {
		for candidate := range r.known {
			if strings.HasSuffix(candidate, "."+symbol) {
				return true
			}
		}
	}
	return false
}

// MergeResult is the outcome of combining a declaration with derivation.
type MergeResult struct {
	// Map is the merged map, with derived nodes labelled as such.
	Map Map
	// Findings are the divergences. A divergence is an error: when the
	// declaration and the implementation disagree about the same element,
	// silently preferring one makes the other a lie.
	Findings []Finding
	// Undeclared lists derived candidates that no declaration mentions. In
	// `fill-gaps` they were added; in `verify-only` they are reported only.
	Undeclared []string
}

// Merge combines declared and derived facts.
//
// The rules, in order:
//
//  1. The declaration wins for every element it describes. It is the only source
//     that can state intent, and replacing authored intent with inferred names
//     would make the map useless to the reader it exists for.
//  2. Derivation fills what the declaration omits — but only with things
//     derivation can actually establish, so merged nodes are marked `derived`
//     and carry no invented label, position or intent.
//  3. Where both describe the same element with different facts, that is a
//     finding, not a tie-break. This is the rule that stops derivation from
//     becoming an unreviewed override channel.
func Merge(declared Map, derived Derived, mode string) MergeResult {
	result := MergeResult{Map: declared}
	declaredSymbols := map[string]bool{}
	for _, walk := range declared.Flatten() {
		if walk.Node.Implementation != nil && walk.Node.Implementation.Symbol != "" {
			declaredSymbols[walk.Node.Implementation.Symbol] = true
		}
	}

	// `off` means the compiler does not look beyond proving symbols, so it must
	// not report candidates either: a mode that says "do not look" and then
	// describes what it saw is a contradiction the reader has to resolve.
	if mode == "off" || len(derived.Sources) == 0 {
		return result
	}

	// Rule 3, first half: a declared symbol the implementation does not have is
	// already caught by validation. The second half is a declared element whose
	// kind contradicts the implementation's shape — which this core cannot
	// settle, because it does not know the framework. Instead, report symbols
	// that look like interface elements but appear nowhere in the map, so a
	// forgotten element is visible instead of absent.
	for _, symbol := range derived.Elements {
		if declaredSymbols[symbol] {
			continue
		}
		if !looksLikeElement(symbol) {
			continue
		}
		result.Undeclared = append(result.Undeclared, symbol)
	}

	sort.Strings(result.Undeclared)
	// Only `fill-gaps` touches the tree. `verify-only` still reports what it
	// found — that is the caller's signal — but a mode that says "do not add"
	// must not add.
	if mode != "fill-gaps" {
		return result
	} // Rule 2: `fill-gaps` adds a single derived region holding what the
	// declaration did not mention. Keeping derived elements in their own region
	// means the tree still reads as the authored layout, and a reviewer can see
	// at a glance what the compiler contributed.
	//
	// The region hangs under the declared root rather than beside it. A map
	// describes one surface, and a second root would make the merged map fail its
	// own single-root rule — a mode that produces invalid output is a mode nobody
	// can turn on.
	if len(result.Undeclared) == 0 || len(result.Map.Tree) != 1 {
		return result
	}
	gap := Node{
		ID:     "node:derived-unmapped",
		Kind:   "region",
		Role:   "Elements the implementation declares but the map does not describe",
		Slot:   Slot{Region: "footer", Order: 9000, Align: "start", Stack: "column"},
		Origin: "derived",
	}
	for i, symbol := range result.Undeclared {
		gap.Children = append(gap.Children, Node{
			ID:             "node:derived-" + slug(symbol),
			Kind:           "text",
			Label:          symbol,
			Role:           "Derived from the implementation; the declaration has not described it yet",
			Slot:           Slot{Region: "body", Order: i, Align: "start"},
			Origin:         "derived",
			Implementation: &Impl{Symbol: symbol, Status: "implemented"},
			States:         []string{"default"},
		})
	}
	result.Map.Tree[0].Children = append(result.Map.Tree[0].Children, gap)
	return result
}

// elementMarkers are the words that make a name read as part of an interface.
//
// A heuristic, and labelled as one: it decides what to *show* the author, never
// what is true. A candidate the author rejects costs a line of review; a silently
// dropped candidate costs an undocumented element. That asymmetry is why the
// filter errs toward showing too much — but not so far that the list becomes
// scenery, which is why the caller only offers types, methods and fields.
var elementMarkers = []string{"view", "render", "palette", "panel", "menu", "button",
	"input", "prompt", "statusline", "list", "pane", "shell", "spinner", "overlay"}

func looksLikeElement(symbol string) bool {
	name := symbol
	if idx := strings.LastIndex(symbol, "."); idx >= 0 {
		name = symbol[idx+1:]
	}
	lower := strings.ToLower(name)
	for _, marker := range elementMarkers {
		// The marker must be the tail of the name, not a fragment inside a
		// longer compound: `Palette` is an element, `DefaultPaletteCapacity`
		// is a budget.
		if strings.HasSuffix(lower, marker) {
			return true
		}
	}
	return false
}

func slug(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteByte('-')
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// Derive runs the configured derivation for a map.
func Derive(root string, m Map, deriver SymbolDeriver) (Derived, error) {
	if len(m.Derivation.Sources) == 0 || deriver == nil {
		return Derived{}, nil
	}
	return deriver.Derive(root, m.Derivation.Sources)
}

// FormatUndeclared renders the undeclared candidates for a report, bounded so a
// large repository cannot flood the output it is meant to be readable in.
func FormatUndeclared(symbols []string, limit int) string {
	if len(symbols) == 0 {
		return ""
	}
	shown := symbols
	suffix := ""
	if limit > 0 && len(symbols) > limit {
		shown = symbols[:limit]
		suffix = fmt.Sprintf(" (+%d more)", len(symbols)-limit)
	}
	return strings.Join(shown, ", ") + suffix
}
