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

	harnessmodel "github.com/raillen/prumo/internal/harness/model"
)

// Projection is one rendered artifact for one audience.
type Projection struct {
	Target Target
	// Path is relative to the projection root.
	Path string
	// Content is the rendered artifact.
	Content string
	// Tokens is the estimated size, reported for the agent surface because a
	// context mechanism has to state its cost rather than assert it is small.
	Tokens int
}

// GeneratorVersion is the projection format version. Bumping it makes every
// existing projection stale on purpose, so a format change cannot be mistaken
// for a fresh artifact.
const GeneratorVersion = 1

// agentSurfaceBudget bounds how many tree lines the agent surface carries.
//
// The agent surface is instruction, not inventory: an agent reads it to know the
// interface exists and where to look, not to reconstruct the whole tree. The
// full inventory stays available on demand, which is what keeps this file inside
// the context budget instead of being the reason a context overflows.
const agentSurfaceBudget = 80

// freshnessHeader is the first line of every projection.
//
// A projection without a revision is a projection nobody can tell is old. The
// digest makes "regenerate me" a fact rather than a judgement call.
func freshnessHeader(m Map, cfg Config) string {
	return fmt.Sprintf("<!-- prumo:interface-map generator=%d map=%s config=%s source=%s -->",
		GeneratorVersion, m.SourceDigest, configDigest(cfg), filepath.Base(m.SourcePath))
}

func configDigest(cfg Config) string {
	names := make([]string, 0, len(cfg.Targets))
	for _, t := range cfg.Targets {
		names = append(names, string(t))
	}
	sort.Strings(names)
	sum := sha256.Sum256([]byte(strings.Join(names, ",") + "|" + cfg.Derivation))
	return "sha256:" + hex.EncodeToString(sum[:])[:12]
}

// Project renders every projection the configuration asks for.
func Project(m Map, cfg Config, report Report, derived Derived) []Projection {
	var out []Projection
	if cfg.Wants(TargetDeveloper) {
		out = append(out, developerProjection(m, cfg, report), mermaidProjection(m, cfg))
	}
	if cfg.Wants(TargetSite) {
		out = append(out, siteProjections(m, cfg, report)...)
	}
	if cfg.Wants(TargetAgent) {
		out = append(out, agentProjection(m, cfg, derived), agentIndexProjection(m, cfg))
	}
	for i := range out {
		out[i].Tokens = harnessmodel.EstimateTokens(out[i].Content, "")
	}
	return out
}

// ProjectionRoot is where derived artifacts live.
//
// They never land in `docs/`: the repository's rule is that derived summaries do
// not become canonical files, and an index that can be regenerated must not be
// mistaken for a source someone has to maintain.
func ProjectionRoot(root string) string {
	return filepath.Join(root, ".prumo", "runtime", "interface-map")
}

// WriteProjections writes the artifacts, creating directories as needed.
func WriteProjections(root string, projections []Projection) error {
	base := ProjectionRoot(root)
	for _, projection := range projections {
		path := filepath.Join(base, filepath.FromSlash(projection.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("uimap: %w", err)
		}
		if err := os.WriteFile(path, []byte(projection.Content), 0o644); err != nil {
			return fmt.Errorf("uimap: write %s: %w", path, err)
		}
	}
	return nil
}

// CheckProjections reports artifacts that are missing or stale.
//
// A stale projection is a finding, not a warning: a reader who opens a reference
// describing the previous interface has been misled, and the fix is one command.
func CheckProjections(root string, projections []Projection) []Finding {
	var findings []Finding
	base := ProjectionRoot(root)
	for _, projection := range projections {
		path := filepath.Join(base, filepath.FromSlash(projection.Path))
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Code: "projection-missing", Severity: SeverityError,
				Message: fmt.Sprintf("projection %s for the %s target is missing", projection.Path, projection.Target),
				Hint:    "run `prumo ui map --write`"})
			continue
		}
		want := strings.SplitN(projection.Content, "\n", 2)[0]
		got := strings.SplitN(string(data), "\n", 2)[0]
		if want != got {
			findings = append(findings, Finding{Code: "projection-stale", Severity: SeverityError,
				Message: fmt.Sprintf("projection %s does not match the current map", projection.Path),
				Hint:    "run `prumo ui map --write`"})
		}
	}
	return findings
}

// --- developer ---------------------------------------------------------------

func developerProjection(m Map, cfg Config, report Report) Projection {
	var b strings.Builder
	b.WriteString(freshnessHeader(m, cfg) + "\n")
	fmt.Fprintf(&b, "# Interface map — %s\n\n", m.Surface)
	fmt.Fprintf(&b, "> Canonical source: `%s` · digest `%s` · platforms %s\n>\n",
		relPath(m.SourcePath), m.SourceDigest, strings.Join(m.Platforms, ", "))
	b.WriteString("> Derived artifact. Edit the map, not this file: this is regenerated.\n\n")

	b.WriteString("## Composition\n\n")
	b.WriteString("Every element, in reading order, with where it sits and what implements it.\n\n")
	b.WriteString(m.MarkdownTree())
	b.WriteString("\n")

	if len(m.Edges) > 0 {
		b.WriteString("## Interconnections\n\n")
		b.WriteString("The tree shows what contains what. These are the links a tree cannot carry: focus\n")
		b.WriteString("order, data flow, events, overlays and navigation.\n\n")
		b.WriteString("| Kind | From | To | Carries | Condition |\n|---|---|---|---|---|\n")
		for _, link := range m.CrossLinks() {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n", link.Kind,
				link.FromLabel, link.ToLabel, fallback(link.Label, "—"), fallback(link.Condition, "—"))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Diagram\n\n")
	b.WriteString("Solid arrows are composition; dotted arrows are interconnections. Render it anywhere\n")
	b.WriteString("Mermaid is supported.\n\n```mermaid\n")
	b.WriteString(m.Mermaid())
	b.WriteString("```\n\n")

	b.WriteString("## Inventory by kind\n\n")
	b.WriteString(inventoryTable(m))
	b.WriteString("\n## Verification\n\n")
	fmt.Fprintf(&b, "- Elements: **%d** · interconnections: **%d**\n", report.Elements, report.Edges)
	fmt.Fprintf(&b, "- States available on this surface: **%d** · tokens available: **%d**\n", report.States, report.Tokens)
	fmt.Fprintf(&b, "- Findings: **%d**\n", len(report.Findings))
	if report.Undeclared > 0 {
		fmt.Fprintf(&b, "- Elements the implementation declares but the map does not describe: **%d**\n", report.Undeclared)
	}
	return Projection{Target: TargetDeveloper, Path: "developer/reference.md", Content: b.String()}
}

func mermaidProjection(m Map, cfg Config) Projection {
	return Projection{
		Target:  TargetDeveloper,
		Path:    "developer/diagram.mmd",
		Content: freshnessHeader(m, cfg) + "\n" + m.Mermaid(),
	}
}

func inventoryTable(m Map) string {
	lines := m.TreeLines()
	kinds := map[string][]TreeLine{}
	for _, line := range lines {
		kinds[line.Kind] = append(kinds[line.Kind], line)
	}
	names := make([]string, 0, len(kinds))
	for kind := range kinds {
		names = append(names, kind)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, kind := range names {
		fmt.Fprintf(&b, "### `%s` (%d)\n\n", kind, len(kinds[kind]))
		b.WriteString("| Element | Label | Position | Implementation |\n|---|---|---|---|\n")
		for _, line := range kinds[kind] {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", line.NodeID,
				fallback(line.Label, "—"), line.Slot, fallback(strings.Join(line.Traces, " "), "—"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// --- site --------------------------------------------------------------------

func siteProjections(m Map, cfg Config, report Report) []Projection {
	out := []Projection{{
		Target:  TargetSite,
		Path:    "site/interface-map.md",
		Content: siteOverviewPage(m, cfg, report),
	}}
	// One page per top-level region: a reader arrives with a question about a
	// part of the interface, not about the interface as a whole.
	for _, walk := range m.Flatten() {
		if walk.Depth != 1 {
			continue
		}
		out = append(out, Projection{
			Target:  TargetSite,
			Path:    "site/interface-map-" + slug(walk.Node.ID) + ".md",
			Content: siteRegionPage(m, cfg, walk),
		})
	}
	return out
}

func siteFrontMatter(title, description string) string {
	return fmt.Sprintf("---\ntitle: %s\ndescription: %s\n---\n\n", title, description)
}

func siteOverviewPage(m Map, cfg Config, report Report) string {
	var b strings.Builder
	b.WriteString(freshnessHeader(m, cfg) + "\n")
	b.WriteString(siteFrontMatter("Interface map — "+m.Surface,
		"Every element of the interface, where it sits and how it connects."))
	fmt.Fprintf(&b, "# Interface map\n\n**%s** on %s.\n\n", m.Surface, strings.Join(m.Platforms, ", "))
	fmt.Fprintf(&b, "%d elements and %d interconnections.\n\n", report.Elements, report.Edges)
	b.WriteString("## Composition\n\n")
	b.WriteString(m.MarkdownTree())
	if len(m.Edges) > 0 {
		b.WriteString("\n## Interconnections\n\n| Kind | From | To | Carries |\n|---|---|---|---|\n")
		for _, link := range m.CrossLinks() {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", link.Kind, link.FromLabel, link.ToLabel, fallback(link.Label, "—"))
		}
	}
	b.WriteString("\n## Diagram\n\n```mermaid\n")
	b.WriteString(m.Mermaid())
	b.WriteString("```\n")
	return b.String()
}

func siteRegionPage(m Map, cfg Config, walk Walk) string {
	title := fallback(walk.Node.Label, fallback(walk.Node.Role, walk.Node.ID))
	var b strings.Builder
	b.WriteString(freshnessHeader(m, cfg) + "\n")
	b.WriteString(siteFrontMatter(title, fallback(walk.Node.Role, "Region of the interface.")))
	fmt.Fprintf(&b, "# %s\n\n", title)
	if walk.Node.Role != "" {
		fmt.Fprintf(&b, "%s\n\n", walk.Node.Role)
	}
	fmt.Fprintf(&b, "Position: `%s`\n\n", FormatSlot(walk.Node.Slot))
	b.WriteString("## Elements\n\n")
	b.WriteString(MarkdownSubtree([]Node{walk.Node}))
	touching := m.edgesTouching(walk.Node.ID)
	if len(touching) > 0 {
		b.WriteString("\n## Interconnections\n\n| Kind | From | To | Carries |\n|---|---|---|---|\n")
		for _, link := range touching {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", link.Kind, link.FromLabel, link.ToLabel, fallback(link.Label, "—"))
		}
	}
	return b.String()
}

// edgesTouching returns the interconnections that touch a node or any of its
// descendants, so a region page answers "what links into this part".
func (m Map) edgesTouching(nodeID string) []CrossLink {
	inRegion := map[string]bool{}
	var collect func(nodes []Node, inside bool)
	collect = func(nodes []Node, inside bool) {
		for _, node := range nodes {
			here := inside || node.ID == nodeID
			if here {
				inRegion[node.ID] = true
			}
			collect(node.Children, here)
		}
	}
	collect(m.Tree, false)

	var out []CrossLink
	for _, link := range m.CrossLinks() {
		if inRegion[link.From] || inRegion[link.To] {
			out = append(out, link)
		}
	}
	return out
}

// --- agent -------------------------------------------------------------------

func agentProjection(m Map, cfg Config, derived Derived) Projection {
	var b strings.Builder
	b.WriteString(freshnessHeader(m, cfg) + "\n")
	fmt.Fprintf(&b, "# Interface map — %s\n\n", m.Surface)
	b.WriteString("This project has an interface, and this is its complete map: the composition tree,\n")
	b.WriteString("where each element sits, and how the elements interconnect. Read it before\n")
	b.WriteString("changing anything a user sees.\n\n")

	b.WriteString("## Non-negotiables\n\n")
	b.WriteString("- Every element a user sees or operates has a label and a position. An element\n")
	b.WriteString("  without a position is an omission, not a default.\n")
	b.WriteString("- Composition is a tree. A focus cycle, an overlay or a data flow is an\n")
	b.WriteString("  interconnection edge, never a second parent.\n")
	b.WriteString("- Every element points at a real symbol, or declares `not-implemented` with a\n")
	b.WriteString("  reason. A map that describes an unwritten interface is worse than no map.\n")
	b.WriteString("- Changing an element means changing the map in the same commit. The gate fails\n")
	b.WriteString("  otherwise, which is the point.\n\n")

	b.WriteString("## Composition\n\n")
	lines := m.TreeLines()
	if len(lines) > agentSurfaceBudget {
		for _, line := range lines[:agentSurfaceBudget] {
			b.WriteString(renderAgentLine(line))
		}
		fmt.Fprintf(&b, "\n_%d further elements omitted from this surface; the full inventory is at\n`%s` or via `prumo ui map`._\n",
			len(lines)-agentSurfaceBudget, relPath(ProjectionRoot(""))+"developer/reference.md")
	} else {
		for _, line := range lines {
			b.WriteString(renderAgentLine(line))
		}
	}

	if len(m.Edges) > 0 {
		b.WriteString("\n## Interconnections by kind\n\n")
		counts := map[string][]string{}
		for _, link := range m.CrossLinks() {
			counts[link.Kind] = append(counts[link.Kind], fmt.Sprintf("%s → %s", link.FromLabel, link.ToLabel))
		}
		names := make([]string, 0, len(counts))
		for kind := range counts {
			names = append(names, kind)
		}
		sort.Strings(names)
		for _, kind := range names {
			fmt.Fprintf(&b, "- `%s` (%d): %s\n", kind, len(counts[kind]), strings.Join(counts[kind], "; "))
		}
	}

	if len(derived.Sources) > 0 {
		fmt.Fprintf(&b, "\n## Derived from\n\n%s\n", strings.Join(derived.Sources, ", "))
	}
	b.WriteString("\n## Commands\n\n```\nprumo ui map            # the compiled map and its projections\n")
	b.WriteString("prumo ui verify         # gate: schema, vocabulary, positions, symbols, freshness\n")
	b.WriteString("prumo ui impact <node>  # what a change to one element reaches\n```\n")
	return Projection{Target: TargetAgent, Path: "agent/interface-map.md", Content: b.String()}
}

// renderAgentLine is the compact tree row the agent surface uses: indentation
// carries composition, and the position is on every line because "where is this"
// is the question the surface exists to answer.
func renderAgentLine(line TreeLine) string {
	indent := strings.Repeat("  ", line.Depth)
	label := line.Label
	if label == "" {
		label = strings.TrimPrefix(line.NodeID, "node:")
	}
	trace := ""
	if len(line.Traces) > 0 {
		trace = " — " + strings.Join(line.Traces, " ")
	}
	return fmt.Sprintf("%s- `%s` %s [%s]%s\n", indent, line.Kind, label, line.Slot, trace)
}

// agentIndexProjection is the machine-readable index an agent retrieves from,
// rather than a second narrative a reader has to reconcile.
func agentIndexProjection(m Map, cfg Config) Projection {
	type indexNode struct {
		ID       string   `json:"id"`
		Kind     string   `json:"kind"`
		Label    string   `json:"label,omitempty"`
		Role     string   `json:"role,omitempty"`
		Position string   `json:"position"`
		States   []string `json:"states,omitempty"`
		Tokens   []string `json:"tokens,omitempty"`
		Inputs   []string `json:"inputs,omitempty"`
		Symbol   string   `json:"symbol,omitempty"`
		Status   string   `json:"status,omitempty"`
		Origin   string   `json:"origin,omitempty"`
		Parent   string   `json:"parent,omitempty"`
	}
	type index struct {
		Surface   string      `json:"surface"`
		MapDigest string      `json:"map_digest"`
		Platforms []string    `json:"platforms"`
		Elements  []indexNode `json:"elements"`
		Links     []Edge      `json:"interconnections,omitempty"`
	}
	built := index{Surface: m.Surface, MapDigest: m.SourceDigest, Platforms: m.Platforms, Links: m.Edges}
	for _, walk := range m.Flatten() {
		node := indexNode{
			ID: walk.Node.ID, Kind: walk.Node.Kind, Label: walk.Node.Label, Role: walk.Node.Role,
			Position: FormatSlot(walk.Node.Slot), States: walk.Node.States, Tokens: walk.Node.Tokens,
			Inputs: walk.Node.Inputs, Origin: walk.Node.Origin, Parent: walk.Parent,
		}
		if walk.Node.Implementation != nil {
			node.Symbol = walk.Node.Implementation.Symbol
			node.Status = walk.Node.Implementation.Status
		}
		built.Elements = append(built.Elements, node)
	}
	data, err := json.MarshalIndent(built, "", "  ")
	if err != nil {
		// Marshalling this structure cannot fail; if it ever does, an empty
		// index is still better than a panic inside a report command.
		data = []byte("{}")
	}
	return Projection{
		Target:  TargetAgent,
		Path:    "agent/index.json",
		Content: freshnessHeader(m, cfg) + "\n" + string(data) + "\n",
	}
}

// relPath makes a path relative to the working tree for display.
func relPath(path string) string {
	if idx := strings.Index(path, "/docs/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
