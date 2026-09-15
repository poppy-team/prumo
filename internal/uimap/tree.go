package uimap

import (
	"fmt"
	"sort"
	"strings"
)

// TreeLine is one rendered row of the interdependence tree.
type TreeLine struct {
	Depth  int
	NodeID string
	Kind   string
	Label  string
	// Slot is the rendered position, e.g. "body/2 start".
	Slot string
	// Traces are the implementation symbols and evidence attached to the node.
	Traces []string
	// Origin is `declared` or `derived`.
	Origin string
}

// TreeLines renders the composition tree as flattened rows.
//
// The function is separate from the projecting functions so every projection can
// share the indentation, the position rendering and the trace format. Two
// projections that format the same tree differently is how a tree stops being an
// overview and becomes two documents.
func (m Map) TreeLines() []TreeLine {
	return treeLines(m.Tree, 0)
}

// treeLines renders any node slice as flattened rows, with `base` as the depth of
// the slice's own roots. A region page renders a subtree, and it needs the same
// formatting as the whole-tree view — two renderers for one tree is how a
// reference and its region page start disagreeing.
func treeLines(nodes []Node, base int) []TreeLine {
	var lines []TreeLine
	var walk func(nodes []Node, depth int)
	walk = func(nodes []Node, depth int) {
		for _, node := range nodes {
			line := TreeLine{
				Depth:  depth - base,
				NodeID: node.ID,
				Kind:   node.Kind,
				Label:  node.Label,
				Slot:   FormatSlot(node.Slot),
				Origin: fallback(node.Origin, "declared"),
			}
			if node.Implementation != nil {
				if node.Implementation.Symbol != "" {
					line.Traces = append(line.Traces, "`"+node.Implementation.Symbol+"`")
				}
				if node.Implementation.Status != "" {
					line.Traces = append(line.Traces, node.Implementation.Status)
				}
				if node.Implementation.Evidence != "" {
					line.Traces = append(line.Traces, node.Implementation.Evidence)
				}
			}
			lines = append(lines, line)
			walk(node.Children, depth+1)
		}
	}
	walk(nodes, base)
	return lines
}

// FormatSlot renders a position compactly and unambiguously: region, order and
// alignment are the whole position on a surface without pixels.
func FormatSlot(s Slot) string {
	parts := []string{fmt.Sprintf("%s/%d", fallback(s.Region, "?"), s.Order), fallback(s.Align, "?")}
	if s.Weight > 0 {
		parts = append(parts, fmt.Sprintf("w%d", s.Weight))
	}
	if s.Stack != "" {
		parts = append(parts, s.Stack)
	}
	if s.Geometry != nil {
		parts = append(parts, fmt.Sprintf("%s@%g,%g %gx%g%s",
			s.Geometry.Platform, s.Geometry.X, s.Geometry.Y, s.Geometry.Width, s.Geometry.Height, s.Geometry.Unit))
	}
	return strings.Join(parts, " ")
}

// MarkdownTree renders the composition tree as a Markdown bullet list with the
// position and implementation trace on each row.
func (m Map) MarkdownTree() string {
	return markdownTree(m.TreeLines())
}

// MarkdownSubtree renders the composition tree rooted at the given nodes, which
// is what a per-region page needs.
func MarkdownSubtree(nodes []Node) string {
	return markdownTree(treeLines(nodes, 0))
}

func markdownTree(lines []TreeLine) string {
	var b strings.Builder
	for _, line := range lines {
		indent := strings.Repeat("  ", line.Depth)
		title := line.Label
		if title == "" {
			title = line.NodeID
		}
		fmt.Fprintf(&b, "%s- **%s** · `%s` · %s", indent, title, line.Kind, line.Slot)
		if line.Origin == "derived" {
			b.WriteString(" · _derived_")
		}
		if len(line.Traces) > 0 {
			b.WriteString(" · " + strings.Join(line.Traces, " "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Mermaid renders the composition tree as a Mermaid graph with the
// interconnection edges layered on.
//
// A tree diagram cannot show a focus cycle or a data flow between distant
// branches, and the two things a reader most needs to check are exactly those.
// Mermaid is chosen because it is text: it diffs, it reviews, and it renders
// without a build step — the same reasons the repository's other artifacts are
// Markdown or JSON.
func (m Map) Mermaid() string {
	var b strings.Builder
	b.WriteString("graph TD\n")
	for _, line := range m.TreeLines() {
		label := line.Label
		if label == "" {
			label = strings.TrimPrefix(line.NodeID, "node:")
		}
		if line.Kind != "" {
			label = line.Kind + ": " + label
		}
		fmt.Fprintf(&b, "  %s[\"%s\"]\n", mermaidID(line.NodeID), escapeMermaid(label))
	}
	for _, walk := range m.Flatten() {
		if walk.Parent == "" {
			continue
		}
		fmt.Fprintf(&b, "  %s --> %s\n", mermaidID(walk.Parent), mermaidID(walk.Node.ID))
	}
	for _, edge := range m.Edges {
		fmt.Fprintf(&b, "  %s -. \"%s\" .-> %s\n",
			mermaidID(edge.From), escapeMermaid(edge.Kind), mermaidID(edge.To))
	}
	return b.String()
}

// mermaidID turns a node id into a Mermaid-safe identifier.
func mermaidID(id string) string {
	return slug(id)
}

func escapeMermaid(value string) string {
	return strings.ReplaceAll(value, "\"", "'")
}

// CrossLink is a rendered interconnection.
type CrossLink struct {
	Kind      string
	From      string
	FromLabel string
	To        string
	ToLabel   string
	Label     string
	Condition string
}

// CrossLinks renders every edge with both endpoint labels resolved.
func (m Map) CrossLinks() []CrossLink {
	index := m.Index()
	label := func(id string) string {
		node, ok := index[id]
		if !ok {
			return id
		}
		if node.Label != "" {
			return node.Label
		}
		return id
	}
	out := make([]CrossLink, 0, len(m.Edges))
	for _, edge := range m.Edges {
		out = append(out, CrossLink{
			Kind: edge.Kind, From: edge.From, FromLabel: label(edge.From),
			To: edge.To, ToLabel: label(edge.To), Label: edge.Label, Condition: edge.Condition,
		})
	}
	return out
}

// Transit is one element reached from a starting point, and why.
type Transit struct {
	NodeID    string
	Label     string
	Via       string
	Hop       int
	Direction string
}

// Impact is the closure of elements affected by changing one element.
type Impact struct {
	Origin      string
	OriginLabel string
	// Transit lists every affected element with the edge that reached it, so a
	// reviewer can argue with the reasoning instead of trusting a list.
	Transit []Transit
}

// ImpactOf walks the interdependence graph outward from a node.
//
// Both directions are walked, because "what breaks if I change this" and "what
// does this depend on" are the same traversal read differently, and a report that
// answers only one of them invites a change that satisfies the diff and breaks
// the surface. Composition counts as a link in both directions: a parent depends
// on its children rendering and a child depends on its parent placing it.
func (m Map) ImpactOf(nodeID string) (Impact, error) {
	index := m.Index()
	origin, ok := index[nodeID]
	if !ok {
		return Impact{}, fmt.Errorf("uimap: no element %q in the map", nodeID)
	}
	impact := Impact{Origin: nodeID, OriginLabel: fallback(origin.Label, nodeID)}

	type link struct {
		to  string
		via string
		dir string
	}
	adjacency := map[string][]link{}
	addLink := func(from, to, via, dir string) {
		adjacency[from] = append(adjacency[from], link{to: to, via: via, dir: dir})
	}
	for _, walk := range m.Flatten() {
		if walk.Parent != "" {
			addLink(walk.Parent, walk.Node.ID, "composition: contains", "down")
			addLink(walk.Node.ID, walk.Parent, "composition: sits in", "up")
		}
	}
	for _, edge := range m.Edges {
		addLink(edge.From, edge.To, edge.Kind+": "+fallback(edge.Label, "no label"), "down")
		// `blocks` and `navigates-to` are directional in meaning as well as in
		// drawing, so their reverse is reported as its own relationship rather
		// than as the same edge read backwards.
		addLink(edge.To, edge.From, "inverse "+edge.Kind+": "+fallback(edge.Label, "no label"), "up")
	}

	visited := map[string]int{nodeID: 0}
	queue := []string{nodeID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		hop := visited[current]
		for _, next := range adjacency[current] {
			if _, seen := visited[next.to]; seen {
				continue
			}
			visited[next.to] = hop + 1
			node := index[next.to]
			impact.Transit = append(impact.Transit, Transit{
				NodeID: next.to, Label: fallback(node.Label, next.to),
				Via: next.via, Hop: hop + 1, Direction: next.dir,
			})
			queue = append(queue, next.to)
		}
	}
	sort.Slice(impact.Transit, func(i, j int) bool {
		if impact.Transit[i].Hop != impact.Transit[j].Hop {
			return impact.Transit[i].Hop < impact.Transit[j].Hop
		}
		return impact.Transit[i].NodeID < impact.Transit[j].NodeID
	})
	return impact, nil
}
