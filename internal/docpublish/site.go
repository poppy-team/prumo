// Package docpublish projects the canonical documentation graph into publishing
// and AI-retrieval surfaces (W18).
//
// One graph, several projections: the human site, raw Markdown, llms.txt,
// llms-full.txt, a search index and an MCP-friendly query surface are all
// derived from the same pages. No site generator is a dependency: the Starlight
// adapter is a reference renderer, not a requirement.
//
// Everything this package writes is derived runtime state.
package docpublish

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	docengine "github.com/raillen/prumo/internal/documentation"
)

// DefaultOutDir receives every projection.
const DefaultOutDir = ".prumo/runtime/docs-site"

// Audience distinguishes agent-facing instruction surfaces from human pages.
type Audience string

const (
	AudienceHuman Audience = "human"
	AudienceAgent Audience = "agent"
)

// Page is one documentation page in the graph.
type Page struct {
	Route    string   `json:"route"`
	Source   string   `json:"source"`
	Title    string   `json:"title"`
	Section  string   `json:"section"`
	Audience Audience `json:"audience"`
	Locale   string   `json:"locale"`
	Version  string   `json:"version,omitempty"`
	Markdown string   `json:"markdown"`
	Tokens   int      `json:"tokens"`
	Order    int      `json:"order"`
	Headings []string `json:"headings,omitempty"`
	IsIndex  bool     `json:"is_index,omitempty"`
	// Historical pages are archived records (ADRs, migration notes, progress
	// logs). They stay published for humans but are excluded from AI retrieval
	// so obsolete statements never contaminate current context (W20.11).
	Historical bool `json:"historical,omitempty"`
}

// Graph is the complete documentation graph before rendering.
type Graph struct {
	Version         string `json:"version"`
	Locale          string `json:"locale"`
	VersionedRoutes bool   `json:"versioned_routes"`
	Pages           []Page `json:"pages"`
	// Reference holds pages generated from authoritative machine contracts
	// rather than from Markdown sources (W18.13). They are kept separate so a
	// generated page can never be mistaken for an authored one.
	Reference []ReferencePage `json:"reference,omitempty"`
}

// Options controls how the graph is projected. Version prefixing is opt-in: a
// repository with a single in-development version should not bury its pages
// under a version segment, while a site publishing several versions needs it
// (W18.5).
type Options struct {
	Version         string
	Locale          string
	VersionedRoutes bool
}

// Load builds the graph with default options (no version prefix).
func Load(root, version, locale string) (Graph, error) {
	return LoadWithOptions(root, Options{Version: version, Locale: locale})
}

// agentOnly lists instruction surfaces that must never be published as human
// documentation: they exist for agents and are projected by the agent surface
// compiler instead.
var agentOnly = map[string]bool{
	"AGENTS.md": true, "ENTRYPOINT.md": true, "FRAMEWORK.md": true,
}

type sourceFile struct {
	path    string
	content string
}

// LoadWithOptions reads the documentation sources that participate in
// publishing.
func LoadWithOptions(root string, opts Options) (Graph, error) {
	files, err := collect(root)
	if err != nil {
		return Graph{}, err
	}
	graph := Graph{Version: opts.Version, Locale: opts.Locale, VersionedRoutes: opts.VersionedRoutes, Pages: []Page{}}
	for i, file := range files {
		page := buildPage(root, file, opts.Version, opts.Locale)
		page.Order = i
		graph.Pages = append(graph.Pages, page)
	}
	sort.SliceStable(graph.Pages, func(i, j int) bool { return graph.Pages[i].Route < graph.Pages[j].Route })
	graph.Reference = buildReferencePages(root)
	return graph, nil
}

// rootDocs are the tracked top-level documents that participate in publishing;
// AGENTS.md and ENTRYPOINT.md are agent-facing pages, excluded from the human
// site but indexed for AI retrieval.
var rootDocs = []string{"README.md", "CONTRIBUTING.md", "SECURITY.md", "CHANGELOG.md", "AGENTS.md", "ENTRYPOINT.md", "FRAMEWORK.md"}

func collect(root string) ([]sourceFile, error) {
	var files []sourceFile
	for _, name := range rootDocs {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		files = append(files, sourceFile{path: name, content: string(data)})
	}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == "agents" && strings.HasSuffix(filepath.ToSlash(path), "/docs/agents") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		files = append(files, sourceFile{path: filepath.ToSlash(rel), content: string(data)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, nil
}

func buildPage(root string, file sourceFile, version, locale string) Page {
	page := Page{
		Source:   file.path,
		Route:    routeOf(file.path),
		Title:    titleOf(file.content, file.path),
		Section:  sectionOf(file.path),
		Audience: AudienceHuman,
		Locale:   locale,
		Markdown: file.content,
		Tokens:   estimateTokens(file.content),
		Headings: headingsOf(file.content),
		IsIndex:  strings.EqualFold(filepath.Base(file.path), "README.md"),
	}
	if agentOnly[file.path] || strings.HasPrefix(file.path, "docs/agents/") {
		page.Audience = AudienceAgent
	}
	if role, ok := docengine.RoleOf(root, file.path); ok && role == docengine.RoleHistorical {
		page.Historical = true
	}
	if version != "" {
		page.Version = version
	}
	return page
}

// routeOf maps a source path to a site route: docs/ is stripped, README files
// become their directory index, and repository-level documents live under the
// /root/ namespace so they can never collide with a docs/ route.
func routeOf(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimSuffix(path, ".md")
	if !strings.Contains(path, "/") {
		if path == "README" {
			return "/"
		}
		return "/root/" + path
	}
	path = strings.TrimPrefix(path, "docs/")
	if strings.HasSuffix(path, "/README") {
		path = strings.TrimSuffix(path, "README")
	}
	path = strings.Trim(path, "/")
	return "/" + path
}

func titleOf(content, path string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	return base
}

func sectionOf(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) == 1 {
		return "root"
	}
	if parts[0] == "docs" && len(parts) > 1 {
		return parts[1]
	}
	return parts[0]
}

func headingsOf(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "#") {
			out = append(out, strings.TrimSpace(strings.TrimLeft(line, "#")))
		}
	}
	return out
}

func estimateTokens(text string) int {
	n := len(text) / 4
	if n < 1 {
		n = 1
	}
	return n
}

// Human returns the pages a human site publishes.
func (g Graph) Human() []Page {
	var out []Page
	for _, p := range g.Pages {
		if p.Audience == AudienceHuman {
			out = append(out, p)
		}
	}
	return out
}

// Current returns the pages that describe the current state of the project:
// historical archives are excluded from AI retrieval so obsolete statements
// never contaminate current context (W20.11).
func (g Graph) Current() []Page {
	var out []Page
	for _, p := range g.Pages {
		if !p.Historical {
			out = append(out, p)
		}
	}
	return out
}

// DoctorFinding is a structural problem in the publishing graph.
type DoctorFinding struct {
	Kind   string `json:"kind"`
	Route  string `json:"route"`
	Source string `json:"source"`
	Detail string `json:"detail"`
}

// Doctor checks the graph for publish-blocking problems: missing titles,
// duplicate routes and dead internal links between pages.
func (g Graph) Doctor() []DoctorFinding {
	findings := []DoctorFinding{}
	routes := map[string]string{}
	for _, p := range g.Pages {
		if strings.TrimSpace(p.Title) == "" {
			findings = append(findings, DoctorFinding{Kind: "page-without-title", Route: p.Route, Source: p.Source,
				Detail: "no level-1 heading and no filename fallback"})
		}
		if previous, ok := routes[p.Route]; ok {
			findings = append(findings, DoctorFinding{Kind: "duplicate-route", Route: p.Route, Source: p.Source,
				Detail: "route already served by " + previous})
		}
		routes[p.Route] = p.Source
	}
	for _, p := range g.Pages {
		for _, link := range internalLinks(p.Markdown) {
			target := resolveLink(p, link)
			if target == "" {
				continue
			}
			if _, ok := routes[target]; !ok {
				findings = append(findings, DoctorFinding{Kind: "dead-internal-link", Route: p.Route, Source: p.Source,
					Detail: fmt.Sprintf("link %q resolves to %s which is not a page route", link, target)})
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Source < findings[j].Source
	})
	return findings
}

func internalLinks(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		for i := 0; i < len(line); i++ {
			if line[i] != ']' || i+1 >= len(line) || line[i+1] != '(' {
				continue
			}
			end := strings.IndexByte(line[i+2:], ')')
			if end < 0 {
				continue
			}
			link := strings.TrimSpace(line[i+2 : i+2+end])
			if link == "" || strings.Contains(link, "://") || strings.HasPrefix(link, "#") || strings.HasPrefix(link, "mailto:") {
				continue
			}
			out = append(out, link)
		}
	}
	return out
}

// resolveLink maps a relative Markdown link to a graph route, or "" when the
// link is not an in-graph documentation link. Targets outside the publishing
// graph (examples, SDKs, generated output) are out of scope here: the site
// doctor validates routes, not every file in the repository.
func resolveLink(page Page, link string) string {
	target := strings.SplitN(link, "#", 2)[0]
	target = strings.SplitN(target, "?", 2)[0]
	if target == "" || !strings.HasSuffix(target, ".md") {
		return ""
	}
	base := filepath.Dir(page.Source)
	joined := filepath.ToSlash(filepath.Clean(filepath.Join(base, target)))
	if strings.HasPrefix(joined, "..") || !strings.HasSuffix(joined, ".md") {
		return ""
	}
	if !strings.HasPrefix(joined, "docs/") && strings.Contains(joined, "/") {
		return ""
	}
	return routeOf(joined)
}

// Routes returns the deterministic route manifest.
// ReferenceRoutes lists the generated reference routes.
func (g Graph) ReferenceRoutes() []string {
	out := make([]string, 0, len(g.Reference))
	for _, page := range g.Reference {
		out = append(out, page.Route)
	}
	sort.Strings(out)
	return out
}

func (g Graph) Routes() []string {
	routes := []string{}
	for _, p := range g.Pages {
		routes = append(routes, p.Route)
	}
	sort.Strings(routes)
	return routes
}
