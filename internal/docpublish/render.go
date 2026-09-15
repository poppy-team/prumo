package docpublish

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Artifact is one rendered projection file.
type Artifact struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Tokens  int    `json:"tokens"`
}

// Renderer projects the graph into a set of files. Renderers are independent of
// any site generator; the Starlight implementation is a reference adapter
// (W18.1, W18.2).
type Renderer interface {
	Name() string
	Render(g Graph) ([]Artifact, error)
}

// Renderers returns the reference adapters.
func Renderers() map[string]Renderer {
	list := []Renderer{markdownRenderer{}, llmsRenderer{}, searchRenderer{}, starlightRenderer{}, apiReferenceRenderer{}}
	out := make(map[string]Renderer, len(list))
	for _, r := range list {
		out[r.Name()] = r
	}
	return out
}

// markdownRenderer writes the raw Markdown projection with route metadata.
type markdownRenderer struct{}

func (markdownRenderer) Name() string { return "markdown" }

func (markdownRenderer) Render(g Graph) ([]Artifact, error) {
	artifacts := []Artifact{}
	for _, p := range g.Human() {
		path := filepath.ToSlash(filepath.Join("markdown", pageFilePath(p.Route, ".md")))
		artifacts = append(artifacts, Artifact{Path: path, Content: p.Markdown, Tokens: p.Tokens})
	}
	return artifacts, nil
}

// pageFilePath maps a route to a file path, giving the site index a real name
// instead of an empty basename.
func pageFilePath(route, ext string) string {
	trimmed := strings.Trim(strings.TrimPrefix(route, "/"), "/")
	if trimmed == "" {
		trimmed = "index"
	}
	return trimmed + ext
}

// apiReferenceRenderer publishes the pages generated from machine contracts.
// It never authors content: it renders what the contracts declare (W18.13).
type apiReferenceRenderer struct{}

func (apiReferenceRenderer) Name() string { return "api-reference" }

func (apiReferenceRenderer) Render(g Graph) ([]Artifact, error) {
	artifacts := []Artifact{}
	for _, page := range g.Reference {
		path := filepath.ToSlash(filepath.Join("reference", pageFilePath(page.Route, ".md")))
		artifacts = append(artifacts, Artifact{Path: path, Content: page.Markdown, Tokens: estimateTokens(page.Markdown)})
	}
	return artifacts, nil
}

// llmsRenderer writes the AI-retrieval index and the full corpus (W18.9).
type llmsRenderer struct{}

func (llmsRenderer) Name() string { return "llms" }

func (llmsRenderer) Render(g Graph) ([]Artifact, error) {
	var index, full strings.Builder
	index.WriteString("# Prumo documentation\n\n")
	if g.Version != "" {
		index.WriteString("Version: " + g.Version + "\n\n")
	}
	index.WriteString("## Pages\n\n")
	for _, p := range g.Current() {
		line := fmt.Sprintf("- [%s](/%s%s): %s\n", p.Title, strings.TrimPrefix(p.Route, "/"), routeSuffix(p), p.Source)
		index.WriteString(line)
	}
	full.WriteString("# Prumo documentation (full)\n\n")
	for _, p := range g.Current() {
		full.WriteString("## " + p.Title + "\n\n")
		full.WriteString("Source: " + p.Source + " · Route: " + p.Route + "\n\n")
		full.WriteString(p.Markdown)
		full.WriteString("\n\n---\n\n")
	}
	return []Artifact{
		{Path: "llms.txt", Content: index.String(), Tokens: estimateTokens(index.String())},
		{Path: "llms-full.txt", Content: full.String(), Tokens: estimateTokens(full.String())},
	}, nil
}

func routeSuffix(p Page) string {
	parts := []string{}
	if p.Locale != "" {
		parts = append(parts, p.Locale)
	}
	if p.Version != "" {
		parts = append(parts, "v"+p.Version)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// SearchEntry is one retrievable documentation chunk (W18.4).
type SearchEntry struct {
	Route    string `json:"route"`
	Source   string `json:"source"`
	Title    string `json:"title"`
	Section  string `json:"section"`
	Locale   string `json:"locale"`
	Audience string `json:"audience"`
	Heading  string `json:"heading,omitempty"`
	Text     string `json:"text"`
}

type searchRenderer struct{}

func (searchRenderer) Name() string { return "search" }

func (searchRenderer) Render(g Graph) ([]Artifact, error) {
	entries := []SearchEntry{}
	for _, p := range g.Pages {
		for _, chunk := range chunkPage(p) {
			entries = append(entries, chunk)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Route != entries[j].Route {
			return entries[i].Route < entries[j].Route
		}
		return entries[i].Heading < entries[j].Heading
	})
	data, err := json.MarshalIndent(map[string]any{
		"version": g.Version,
		"locale":  g.Locale,
		"entries": entries,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return []Artifact{{Path: "search-index.json", Content: string(append(data, '\n')), Tokens: len(entries)}}, nil
}

// chunkPage splits a page by level-2 headings so retrieval works on semantic
// units instead of whole documents.
func chunkPage(p Page) []SearchEntry {
	sections := map[string][]string{}
	order := []string{}
	current := ""
	for _, line := range strings.Split(p.Markdown, "\n") {
		if strings.HasPrefix(line, "## ") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if _, ok := sections[current]; !ok {
				order = append(order, current)
			}
			continue
		}
		sections[current] = append(sections[current], line)
	}
	entries := []SearchEntry{}
	for _, heading := range order {
		body := strings.TrimSpace(strings.Join(sections[heading], "\n"))
		if body == "" {
			continue
		}
		entries = append(entries, SearchEntry{
			Route: p.Route, Source: p.Source, Title: p.Title, Section: p.Section,
			Locale: p.Locale, Audience: string(p.Audience), Heading: heading, Text: body,
		})
	}
	if len(entries) == 0 {
		entries = append(entries, SearchEntry{
			Route: p.Route, Source: p.Source, Title: p.Title, Section: p.Section,
			Locale: p.Locale, Audience: string(p.Audience), Text: strings.TrimSpace(p.Markdown),
		})
	}
	return entries
}

// starlightRenderer is the reference site adapter: MDX pages with frontmatter,
// a sidebar derived from the graph, and locale/version routes (W18.2–W18.6).
type starlightRenderer struct{}

func (starlightRenderer) Name() string { return "starlight" }

func (starlightRenderer) Render(g Graph) ([]Artifact, error) {
	artifacts := []Artifact{}
	sidebar := map[string][]map[string]string{}
	for _, p := range g.Human() {
		route := localizedRoute(p, g.VersionedRoutes)
		frontmatter := []string{"---", "title: " + quote(p.Title), "slug: " + quote(route)}
		if p.Locale != "" {
			frontmatter = append(frontmatter, "locale: "+quote(p.Locale))
		}
		if p.Version != "" {
			frontmatter = append(frontmatter, "version: "+quote(p.Version))
		}
		frontmatter = append(frontmatter, "source: "+quote(p.Source), "---", "")
		content := strings.Join(frontmatter, "\n") + p.Markdown
		path := filepath.ToSlash(filepath.Join("starlight", "content", "docs", pageFilePath(route, ".mdx")))
		artifacts = append(artifacts, Artifact{Path: path, Content: content, Tokens: p.Tokens})
		sidebar[p.Section] = append(sidebar[p.Section], map[string]string{"label": p.Title, "link": p.Route, "localized_link": route})
	}
	sections := make([]string, 0, len(sidebar))
	for section := range sidebar {
		sections = append(sections, section)
	}
	sort.Strings(sections)
	items := []map[string]any{}
	for _, section := range sections {
		sort.Slice(sidebar[section], func(i, j int) bool { return sidebar[section][i]["link"] < sidebar[section][j]["link"] })
		items = append(items, map[string]any{"label": section, "items": sidebar[section]})
	}
	data, err := json.MarshalIndent(map[string]any{"sidebar": items, "locales": localesOf(g), "versions": versionsOf(g)}, "", "  ")
	if err != nil {
		return nil, err
	}
	artifacts = append(artifacts, Artifact{Path: "starlight/site-manifest.json", Content: string(append(data, '\n'))})
	return artifacts, nil
}

// localizedRoute builds a locale-aware route, and a version-aware one when the
// graph declares versioned routes (W18.5, W18.6).
func localizedRoute(p Page, versioned bool) string {
	route := p.Route
	if versioned && p.Version != "" {
		route = "/v" + p.Version + route
	}
	if p.Locale != "" && p.Locale != "en" {
		route = "/" + p.Locale + route
	}
	return route
}

func localesOf(g Graph) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range g.Pages {
		if p.Locale != "" && !seen[p.Locale] {
			seen[p.Locale] = true
			out = append(out, p.Locale)
		}
	}
	sort.Strings(out)
	return out
}

func versionsOf(g Graph) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range g.Pages {
		if p.Version != "" && !seen[p.Version] {
			seen[p.Version] = true
			out = append(out, p.Version)
		}
	}
	sort.Strings(out)
	return out
}

func quote(value string) string {
	return strconv.Quote(value)
}
