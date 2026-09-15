// Package uimap compiles a complete map of a user interface: the composition
// tree, where each element sits, and how the elements interconnect.
//
// # Why this exists
//
// Four contracts already oblige a UI-bearing project to document its interface:
// `ui.component-contracts` ("component inventory, anatomy, component states,
// component interactions, implementation trace"), `ui.information-architecture`
// ("surface hierarchy"), `ui.screen-inventory` ("screen inventory, screen
// purpose, entry points") and `ui.layout` ("layout model, responsive behavior").
// `schemas/ui-component-contract.schema.json` has described a single component
// since W5, and nothing ever produced one: the obligation was satisfied by a
// waiver, so the gate was green while no inventory existed.
//
// This package is the producer. One artifact answers all four obligations
// instead of four partial documents, because they describe the same object seen
// from four angles — and four documents about one object is how they drift apart.
//
// # Declared and derived
//
// Structure, labels, intent and placement are **declared**: a compiler cannot
// infer what a button is for. Symbols, enumerations and coverage are **derived**
// from the implementation: a compiler can prove those. A declared node always
// wins; derivation fills what the declaration omits; and where both describe the
// same node with different facts, that is an error rather than a silent pick.
// Without that last rule derivation would be an override channel nobody reviews.
//
// # Projections
//
// The map is canonical. Everything a reader consumes is derived from it: a
// developer reference, site pages, and a bounded agent surface. Derived artifacts
// live under the runtime directory and never become canonical files, per the
// repository's rule that summaries and indexes are not sources.
//
// # Scope
//
// The core is provider- and framework-neutral. `GoSymbolDeriver` is the deriver
// shipped here because this repository is Go; another stack adds an adapter
// instead of teaching the core about its framework.
package uimap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Node is one element of the interface.
type Node struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Label          string   `json:"label,omitempty"`
	Role           string   `json:"role,omitempty"`
	Intent         string   `json:"intent,omitempty"`
	Slot           Slot     `json:"slot"`
	SizeClasses    []string `json:"size_classes,omitempty"`
	States         []string `json:"states,omitempty"`
	Tokens         []string `json:"tokens,omitempty"`
	Inputs         []string `json:"inputs,omitempty"`
	Children       []Node   `json:"children,omitempty"`
	Implementation *Impl    `json:"implementation,omitempty"`
	Origin         string   `json:"origin,omitempty"`
}

// Slot says where an element sits, semantically.
//
// A terminal has no stable pixel geometry — coordinates change with every resize
// — so position is a region plus an order plus an alignment. A platform that does
// have stable pixels may add geometry, which is why it is optional and carries
// its own platform tag rather than pretending to be universal.
type Slot struct {
	Region string `json:"region"`
	Order  int    `json:"order"`
	Align  string `json:"align"`
	Weight int    `json:"weight,omitempty"`
	Stack  string `json:"stack,omitempty"`
	// Condition states when this element is present at all.
	//
	// It exists because the alternative is a lie. A shell whose stages are
	// mutually exclusive has four children occupying one region, and a layout
	// model without conditions reads that as four panes stacked: the validator
	// caught exactly that on the interface it was written for. Declaring the
	// condition lets two siblings share a position *because they never coexist*,
	// which is a fact a reader needs and a renderer can rely on.
	Condition string    `json:"condition,omitempty"`
	Geometry  *Geometry `json:"geometry,omitempty"`
}

// Geometry is optional pixel placement for platforms where pixels are stable.
type Geometry struct {
	Platform string  `json:"platform"`
	X        float64 `json:"x,omitempty"`
	Y        float64 `json:"y,omitempty"`
	Width    float64 `json:"width,omitempty"`
	Height   float64 `json:"height,omitempty"`
	Unit     string  `json:"unit,omitempty"`
}

// Impl is the implementation trace of an element.
type Impl struct {
	Symbol       string `json:"symbol,omitempty"`
	Path         string `json:"path,omitempty"`
	Evidence     string `json:"evidence,omitempty"`
	Status       string `json:"status,omitempty"`
	AbsentReason string `json:"absent_reason,omitempty"`
}

// Edge is a non-hierarchical interconnection between two elements.
//
// The tree carries composition. Edges carry what a tree cannot hold without
// lying about it: focus order (which can cycle), data flow between distant
// branches, events, overlays, navigation and blocking relationships.
type Edge struct {
	Kind      string `json:"kind"`
	From      string `json:"from"`
	To        string `json:"to"`
	Label     string `json:"label,omitempty"`
	Condition string `json:"condition,omitempty"`
}

// Derivation configures how much the compiler reads from the implementation.
type Derivation struct {
	Sources []string `json:"sources,omitempty"`
	Mode    string   `json:"mode,omitempty"`
}

// Map is the compiled interface map.
type Map struct {
	ID           string     `json:"id"`
	Version      int        `json:"version"`
	Surface      string     `json:"surface"`
	Note         string     `json:"note,omitempty"`
	Platforms    []string   `json:"platforms"`
	StatesSource string     `json:"states_source,omitempty"`
	TokensSource string     `json:"tokens_source,omitempty"`
	Derivation   Derivation `json:"derivation,omitempty"`
	Tree         []Node     `json:"tree"`
	Edges        []Edge     `json:"edges,omitempty"`

	// SourceDigest is the digest of the file this map was loaded from, so a
	// projection can state which revision of the map it was built from.
	SourceDigest string `json:"-"`
	// SourcePath is where it was loaded from.
	SourcePath string `json:"-"`
}

// NodeKindNeedsLabel reports whether a kind renders text a user reads.
//
// A shell or a region is structure, so it may be nameless. Everything else is
// something a person sees or operates, and a nameless button is an omission
// rather than a design decision — which is why the validator refuses it instead
// of substituting the id.
func NodeKindNeedsLabel(kind string) bool {
	switch kind {
	case "shell", "region", "overlay":
		return false
	default:
		return true
	}
}

// NodeKindNeedsImplementation reports whether a kind must have a code symbol.
//
// Structure kinds may exist only as a layout decision in a renderer, so demanding
// a symbol for them would push authors into inventing one. Anything a user
// operates, sees, or that carries a state must point at real code.
func NodeKindNeedsImplementation(kind string) bool {
	switch kind {
	case "shell", "region":
		return false
	default:
		return true
	}
}

// Load reads and parses an interface map.
func Load(path string) (Map, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Map{}, fmt.Errorf("uimap: read %s: %w", path, err)
	}
	var m Map
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return Map{}, fmt.Errorf("uimap: parse %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	m.SourceDigest = "sha256:" + hex.EncodeToString(sum[:])[:16]
	m.SourcePath = path
	return m, nil
}

// DefaultPath is where the canonical map lives when a project does not say
// otherwise.
func DefaultPath(root string) string {
	return filepath.Join(root, "docs", "ui-ux", "interface-map.json")
}

// Flatten returns every node in depth-first order, each with its parent id and
// its depth. Parents come before children, which is what makes the tree
// renderable without recursion in the projections.
func (m Map) Flatten() []Walk {
	var out []Walk
	var walk func(nodes []Node, parent string, depth int)
	walk = func(nodes []Node, parent string, depth int) {
		for _, node := range nodes {
			out = append(out, Walk{Node: node, Parent: parent, Depth: depth})
			walk(node.Children, node.ID, depth+1)
		}
	}
	walk(m.Tree, "", 0)
	return out
}

// Walk is one flattened node together with its position in the tree.
type Walk struct {
	Node   Node
	Parent string
	Depth  int
}

// Index returns every node keyed by id.
func (m Map) Index() map[string]Node {
	index := map[string]Node{}
	for _, walk := range m.Flatten() {
		index[walk.Node.ID] = walk.Node
	}
	return index
}

// KindCounts reports how many elements of each kind the map declares.
func (m Map) KindCounts() map[string]int {
	counts := map[string]int{}
	for _, walk := range m.Flatten() {
		counts[walk.Node.Kind]++
	}
	return counts
}

// Summary is the one-line description of the map used in reports and statuslines.
func (m Map) Summary() string {
	kinds := m.KindCounts()
	names := make([]string, 0, len(kinds))
	for kind := range kinds {
		names = append(names, kind)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, kind := range names {
		parts = append(parts, fmt.Sprintf("%d %s", kinds[kind], kind))
	}
	return fmt.Sprintf("%s — %d elements (%s), %d interconnections, digest %s",
		m.Surface, len(m.Flatten()), strings.Join(parts, ", "), len(m.Edges), m.SourceDigest)
}
