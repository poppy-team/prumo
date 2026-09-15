package uimap

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Severity of a finding. `error` fails the gate; `warning` is reported without
// failing, and is only used where a rule cannot be settled by reading declared
// data.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is one violation the map must fix.
type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Node     string   `json:"node,omitempty"`
	Message  string   `json:"message"`
	Hint     string   `json:"hint,omitempty"`
}

// Report is the outcome of validating a map.
type Report struct {
	Map      string    `json:"map"`
	Surface  string    `json:"surface"`
	Elements int       `json:"elements"`
	Edges    int       `json:"edges"`
	Findings []Finding `json:"findings"`
	// Vocabulary is the state and token vocabulary the map was checked
	// against, so a report states its own basis instead of implying one.
	States     int `json:"states_available"`
	Tokens     int `json:"tokens_available"`
	Derived    int `json:"derived_elements"`
	Undeclared int `json:"undeclared_candidates"`
}

// OK reports whether the map passed.
func (r Report) OK() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return false
		}
	}
	return true
}

// nodeKinds is the closed kind vocabulary, asserted equal to the schema enum by
// the conformance gate.
func NodeKinds() []string {
	return []string{"shell", "region", "pane", "list", "input", "text",
		"button", "menu", "status", "overlay", "spinner"}
}

// edgeKinds is the closed interconnection vocabulary.
func EdgeKinds() []string {
	return []string{"focus", "data", "event", "overlay", "navigates-to", "blocks"}
}

func Regions() []string {
	return []string{"root", "header", "body", "footer", "sidebar", "overlay"}
}

func Alignments() []string { return []string{"start", "center", "end", "stretch"} }

func Stacks() []string { return []string{"row", "column", "absolute"} }

// sizeClasses is the layout vocabulary, shared with the interaction
// specification's terminal size classes.
func SizeClasses() []string { return []string{"compact", "standard", "wide"} }

// Vocabulary is what a map is validated against, read from the sources the map
// itself names.
type Vocabulary struct {
	// States are the states the surface's matrix marks applicable. Marking a
	// state not applicable is a decision the matrix already made, so a map may
	// not contradict it.
	States map[string]bool
	// Tokens are the declared design token ids.
	Tokens map[string]bool
	// MatrixSurface is the surface the state matrix describes.
	MatrixSurface string
}

// StateMatrix is the subset of the state matrix this package reads.
type StateMatrix struct {
	ID      string `json:"id"`
	Surface string `json:"surface"`
	States  []struct {
		State      string `json:"state"`
		Applicable *bool  `json:"applicable"`
	} `json:"states"`
}

// TokenSet is the subset of the design token set this package reads.
type TokenSet struct {
	Tokens []struct {
		ID string `json:"id"`
	} `json:"tokens"`
}

// LoadVocabulary reads the state matrix and token set a map references.
//
// A missing source is an error rather than an empty vocabulary: validating
// against nothing would pass every map, which is precisely the failure the
// producer exists to prevent.
func LoadVocabulary(root string, m Map) (Vocabulary, error) {
	v := Vocabulary{States: map[string]bool{}, Tokens: map[string]bool{}}
	if m.StatesSource != "" {
		path := resolve(root, m.StatesSource)
		data, err := os.ReadFile(path)
		if err != nil {
			return Vocabulary{}, fmt.Errorf("uimap: states source %s: %w", m.StatesSource, err)
		}
		var matrix StateMatrix
		if err := json.Unmarshal(data, &matrix); err != nil {
			return Vocabulary{}, fmt.Errorf("uimap: parse %s: %w", m.StatesSource, err)
		}
		v.MatrixSurface = matrix.Surface
		for _, state := range matrix.States {
			if state.Applicable != nil && !*state.Applicable {
				continue
			}
			v.States[state.State] = true
		}
	}
	if m.TokensSource != "" {
		path := resolve(root, m.TokensSource)
		data, err := os.ReadFile(path)
		if err != nil {
			return Vocabulary{}, fmt.Errorf("uimap: tokens source %s: %w", m.TokensSource, err)
		}
		var set TokenSet
		if err := json.Unmarshal(data, &set); err != nil {
			return Vocabulary{}, fmt.Errorf("uimap: parse %s: %w", m.TokensSource, err)
		}
		for _, token := range set.Tokens {
			v.Tokens[token.ID] = true
		}
	}
	return v, nil
}

func resolve(root, rel string) string {
	if strings.HasPrefix(rel, "/") {
		return rel
	}
	return root + "/" + rel
}

// Resolver decides whether an implementation symbol exists.
type Resolver interface {
	Resolves(symbol, path string) bool
}

// Validate checks a map against its vocabulary and its implementation.
//
// Every rule here fails closed. The stance is deliberate: the obligation this
// artifact satisfies spent its whole life waived because nothing checked it, and
// an artifact that can describe a button which does not exist, in a position
// nobody defined, is not an inventory — it is prose with fields.
func Validate(root string, m Map, vocab Vocabulary, resolver Resolver) Report {
	report := Report{Map: m.SourcePath, Surface: m.Surface, States: len(vocab.States), Tokens: len(vocab.Tokens)}
	add := func(f Finding) { report.Findings = append(report.Findings, f) }

	if !strings.HasPrefix(m.ID, "ui:") {
		add(Finding{Code: "map-id", Severity: SeverityError,
			Message: fmt.Sprintf("map id %q must be namespaced as ui:<surface>", m.ID)})
	}
	if len(m.Platforms) == 0 {
		add(Finding{Code: "map-platforms", Severity: SeverityError,
			Message: "map declares no platform, so its placement model is unknown",
			Hint:    "declare at least one of tui, desktop, web, mobile"})
	}
	if m.StatesSource != "" && vocab.MatrixSurface != "" && m.Surface != vocab.MatrixSurface {
		add(Finding{Code: "surface-mismatch", Severity: SeverityError,
			Message: fmt.Sprintf("map surface %q does not match the state matrix surface %q", m.Surface, vocab.MatrixSurface),
			Hint:    "the map and the matrix must describe the same surface, or neither can reference the other"})
	}

	platforms := map[string]bool{}
	for _, p := range m.Platforms {
		platforms[p] = true
	}

	if len(m.Tree) == 0 {
		add(Finding{Code: "empty-tree", Severity: SeverityError, Message: "map has no elements"})
		return report
	}
	if len(m.Tree) != 1 {
		add(Finding{Code: "multiple-roots", Severity: SeverityError,
			Message: fmt.Sprintf("map has %d roots; a map describes one surface, so it has one root", len(m.Tree)),
			Hint:    "split a second surface into its own map with its own state matrix"})
	}
	if len(m.Tree) == 1 && m.Tree[0].Kind != "shell" {
		add(Finding{Code: "root-kind", Severity: SeverityError, Node: m.Tree[0].ID,
			Message: fmt.Sprintf("root element is kind %q; a surface root is a shell", m.Tree[0].Kind)})
	}

	walks := m.Flatten()
	report.Elements = len(walks)
	report.Edges = len(m.Edges)

	seen := map[string]int{}
	for _, walk := range walks {
		node := walk.Node
		seen[node.ID]++
	}

	// Unique ids first: every later message names a node, and a duplicated id
	// makes those messages ambiguous.
	duplicates := make([]string, 0)
	for id, count := range seen {
		if count > 1 {
			duplicates = append(duplicates, id)
		}
	}
	sort.Strings(duplicates)
	for _, id := range duplicates {
		add(Finding{Code: "duplicate-id", Severity: SeverityError, Node: id,
			Message: fmt.Sprintf("element id %q is used %d times", id, seen[id]),
			Hint:    "ids are how edges and impact reports name elements, so they must be unique"})
	}

	index := m.Index()
	for _, walk := range walks {
		report.Findings = append(report.Findings, validateNode(nodeContext{walk: walk, vocab: vocab, platforms: platforms, resolver: resolver})...)
	}

	// Sibling order must be total among elements that coexist, otherwise two
	// renderers can disagree about the layout while both claiming to follow the
	// map. Elements that declare a condition are alternatives, not neighbours, so
	// they may share a position — and at least one of any sharing pair must say
	// so, which is what stops "they never coexist" from being an unstated
	// assumption the reader has to supply.
	var checkSiblings func(nodes []Node)
	checkSiblings = func(nodes []Node) {
		type key struct{ region string }
		orders := map[key]map[int][]Node{}
		for _, node := range nodes {
			k := key{region: node.Slot.Region}
			if orders[k] == nil {
				orders[k] = map[int][]Node{}
			}
			orders[k][node.Slot.Order] = append(orders[k][node.Slot.Order], node)
		}
		for k, byOrder := range orders {
			for order, group := range byOrder {
				if len(group) < 2 {
					continue
				}
				// Unconditional siblings cannot share a position: nothing tells a
				// renderer which one to draw.
				var simultaneous []string
				for _, node := range group {
					if node.Slot.Condition == "" {
						simultaneous = append(simultaneous, node.ID)
					}
				}
				if len(simultaneous) > 1 {
					sort.Strings(simultaneous)
					add(Finding{Code: "ambiguous-order", Severity: SeverityError, Node: simultaneous[0],
						Message: fmt.Sprintf("elements %s share region %q order %d without a condition",
							strings.Join(simultaneous, ", "), k.region, order),
						Hint: "distinct order values make the layout order total; alternatives may share one only by declaring `condition`"})
				}
			}
		}
		for _, node := range nodes {
			checkSiblings(node.Children)
		}
	}
	checkSiblings(m.Tree)

	// Edges.
	edgeSeen := map[string]bool{}
	for _, edge := range m.Edges {
		if !contains(EdgeKinds(), edge.Kind) {
			add(Finding{Code: "edge-kind", Severity: SeverityError,
				Message: fmt.Sprintf("edge kind %q is outside the vocabulary %v", edge.Kind, EdgeKinds())})
			continue
		}
		if _, ok := index[edge.From]; !ok {
			add(Finding{Code: "edge-endpoint", Severity: SeverityError,
				Message: fmt.Sprintf("edge %s starts at unknown element %q", edge.Kind, edge.From)})
		}
		if _, ok := index[edge.To]; !ok {
			add(Finding{Code: "edge-endpoint", Severity: SeverityError,
				Message: fmt.Sprintf("edge %s ends at unknown element %q", edge.Kind, edge.To)})
		}
		if edge.From == edge.To {
			add(Finding{Code: "edge-self", Severity: SeverityError, Node: edge.From,
				Message: fmt.Sprintf("edge %s points at its own source", edge.Kind),
				Hint:    "a self-edge carries no information; state the condition on the element instead"})
		}
		key := edge.Kind + "|" + edge.From + "|" + edge.To
		if edgeSeen[key] {
			add(Finding{Code: "edge-duplicate", Severity: SeverityError, Node: edge.From,
				Message: fmt.Sprintf("interconnection %s → %s declared twice", edge.From, edge.To)})
		}
		edgeSeen[key] = true
	}
	return report
}

// nodeContext carries what validating one node needs, so validateNode stays a
// function of its inputs rather than reaching for the whole map.
type nodeContext struct {
	walk      Walk
	vocab     Vocabulary
	platforms map[string]bool
	resolver  Resolver
}

func validateNode(c nodeContext) []Finding {
	var findings []Finding
	node := c.walk.Node
	add := func(code string, msg string, hint string) {
		findings = append(findings, Finding{Code: code, Severity: SeverityError, Node: node.ID, Message: msg, Hint: hint})
	}

	if !strings.HasPrefix(node.ID, "node:") {
		add("node-id", fmt.Sprintf("element id %q must be namespaced as node:<name>", node.ID), "")
	}
	if !contains(NodeKinds(), node.Kind) {
		add("node-kind", fmt.Sprintf("kind %q is outside the vocabulary %v", node.Kind, NodeKinds()), "")
	}
	// Position is mandatory: a component with no position is an omission, and
	// defaulting it would hide exactly the fact the reader came for.
	if node.Slot.Region == "" {
		add("missing-position", "element declares no region, so its position is unknown", "every element sits in a region")
	} else if !contains(Regions(), node.Slot.Region) {
		add("slot-region", fmt.Sprintf("region %q is outside the vocabulary %v", node.Slot.Region, Regions()), "")
	}
	if node.Slot.Align == "" {
		add("missing-align", "element declares no alignment", "declare start, center, end or stretch")
	} else if !contains(Alignments(), node.Slot.Align) {
		add("slot-align", fmt.Sprintf("alignment %q is outside the vocabulary %v", node.Slot.Align, Alignments()), "")
	}
	if node.Slot.Order < 0 {
		add("slot-order", fmt.Sprintf("order %d is negative", node.Slot.Order), "")
	}
	if node.Slot.Stack != "" && !contains(Stacks(), node.Slot.Stack) {
		add("slot-stack", fmt.Sprintf("stack %q is outside the vocabulary %v", node.Slot.Stack, Stacks()), "")
	}
	if node.Slot.Geometry != nil {
		g := node.Slot.Geometry
		if g.Platform == "" {
			add("geometry-platform", "geometry declares no platform", "pixels are only stable on a named platform")
		} else if !c.platforms[g.Platform] {
			add("geometry-platform", fmt.Sprintf("geometry targets platform %q, which the map does not declare", g.Platform), "")
		}
		if g.Unit == "" {
			add("geometry-unit", "geometry declares no unit", "declare px, rem or dp so a number means something")
		}
		if g.Width < 0 || g.Height < 0 {
			add("geometry-size", "geometry has a negative dimension", "")
		}
	}
	if NodeKindNeedsLabel(node.Kind) && strings.TrimSpace(node.Label) == "" {
		add("missing-label", fmt.Sprintf("kind %q renders text but declares no label", node.Kind),
			"a nameless element is an omission; say what the user reads")
	}
	for _, class := range node.SizeClasses {
		if !contains(SizeClasses(), class) {
			add("size-class", fmt.Sprintf("size class %q is outside the vocabulary %v", class, SizeClasses()), "")
		}
	}
	for _, state := range node.States {
		if len(c.vocab.States) == 0 {
			break
		}
		if !c.vocab.States[state] {
			add("unknown-state", fmt.Sprintf("state %q is not applicable on this surface per the state matrix", state),
				"the matrix already decided applicability; a map may not contradict it")
		}
	}
	for _, token := range node.Tokens {
		if len(c.vocab.Tokens) == 0 {
			break
		}
		if !c.vocab.Tokens[token] {
			add("unknown-token", fmt.Sprintf("token %q is not declared in the design token set", token),
				"a token nobody declared cannot be painted")
		}
	}
	findings = append(findings, validateImplementation(node, c.resolver)...)
	return findings
}

// validateImplementation enforces the traceability rule: the map either points
// at code that exists, or says out loud that the element is not built.
//
// Without the `not-implemented` escape hatch authors would invent symbols to get
// a green gate. Without the requirement to resolve, the map could describe an
// interface that was never written — which is the state this artifact spent its
// life in.
func validateImplementation(node Node, resolver Resolver) []Finding {
	if !NodeKindNeedsImplementation(node.Kind) {
		return nil
	}
	if node.Implementation == nil {
		return []Finding{{Code: "missing-implementation", Severity: SeverityError, Node: node.ID,
			Message: fmt.Sprintf("kind %q carries no implementation trace", node.Kind),
			Hint:    "name a symbol, or declare status not-implemented with a reason"}}
	}
	impl := node.Implementation
	switch impl.Status {
	case "not-implemented":
		if strings.TrimSpace(impl.AbsentReason) == "" {
			return []Finding{{Code: "absent-without-reason", Severity: SeverityError, Node: node.ID,
				Message: "element is not implemented but gives no reason",
				Hint:    "an unexplained absence is indistinguishable from a forgotten element"}}
		}
		return nil
	case "implemented", "partial", "":
		if strings.TrimSpace(impl.Symbol) == "" {
			return []Finding{{Code: "missing-symbol", Severity: SeverityError, Node: node.ID,
				Message: fmt.Sprintf("element claims status %q but names no symbol", fallback(impl.Status, "implemented"))}}
		}
		if resolver != nil && !resolver.Resolves(impl.Symbol, impl.Path) {
			return []Finding{{Code: "unresolved-symbol", Severity: SeverityError, Node: node.ID,
				Message: fmt.Sprintf("symbol %q does not resolve in this repository", impl.Symbol),
				Hint:    "either the symbol moved, or the element is not implemented and should say so"}}
		}
		return nil
	default:
		return []Finding{{Code: "implementation-status", Severity: SeverityError, Node: node.ID,
			Message: fmt.Sprintf("status %q is not one of implemented, partial, not-implemented", impl.Status)}}
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func fallback(value, when string) string {
	if value == "" {
		return when
	}
	return value
}
