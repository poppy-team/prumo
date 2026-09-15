package agentsurface

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Surface is one rendered agent instruction projection.
type Surface struct {
	Adapter     string   `json:"adapter"`
	Scope       string   `json:"scope"`
	Path        string   `json:"path"`
	Region      string   `json:"region,omitempty"`
	Content     string   `json:"content"`
	Fingerprint string   `json:"fingerprint"`
	Tokens      int      `json:"tokens"`
	Rules       []string `json:"rules,omitempty"`
	Sources     []string `json:"source_units,omitempty"`
}

// ArtifactMode reports how a surface is written: into a managed region of a
// curated file, or as a generated standalone file.
func (s Surface) ArtifactMode() string {
	if s.Region != "" {
		return "region"
	}
	return "file"
}

// LocalPath is where a Build writes this surface. Managed regions always land
// in the curated file; standalone vendor files are redirected to the runtime
// directory so a projection never becomes a canonical repository artifact.
func (s Surface) LocalPath(outDir string) string {
	if s.Region != "" || outDir == "" {
		return s.Path
	}
	return filepath.ToSlash(filepath.Join(outDir, s.Path))
}

// Adapter projects the IR into one tool's instruction format.
type Adapter interface {
	Name() string
	Render(ir IR, scope string) (Surface, bool, error)
}

// Adapters returns the adapter registry. The core never references a vendor
// format outside these implementations (W16.13).
func Adapters() map[string]Adapter {
	list := []Adapter{
		agentsMDAdapter{},
		copilotAdapter{},
		cursorAdapter{},
		claudeAdapter{},
		skillAdapter{},
	}
	registry := make(map[string]Adapter, len(list))
	for _, a := range list {
		registry[a.Name()] = a
	}
	return registry
}

var (
	provenancePattern = regexp.MustCompile(`^<!-- prumo:generated adapter=(\S+) scope=(\S*) fingerprint=([0-9a-f]+)(?: sources=(\S*))? -->`)
	slugPattern       = regexp.MustCompile(`[^a-z0-9]+`)
)

// provenance renders the generated-artifact marker that carries adapter,
// scope, source fingerprint and source units (W16.9).
func provenance(adapter, scope, fingerprint string, sources []string) string {
	scopeToken := NormalScope(scope)
	if scopeToken == "" {
		scopeToken = "root"
	}
	line := fmt.Sprintf("<!-- prumo:generated adapter=%s scope=%s fingerprint=%s", adapter, scopeToken, fingerprint)
	if len(sources) > 0 {
		line += " sources=" + strings.Join(sources, ",")
	}
	return line + " -->"
}

// StripProvenance removes the generated marker and returns the body plus the
// recorded fingerprint.
func StripProvenance(content string) (string, string, bool) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		match := provenancePattern.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			return content, "", false
		}
		body := strings.Join(append(append([]string{}, lines[:i]...), lines[i+1:]...), "\n")
		return body, match[3], true
	}
	return content, "", false
}

func slug(scope string) string {
	scope = NormalScope(scope)
	if scope == "" {
		return "root"
	}
	return strings.Trim(slugPattern.ReplaceAllString(strings.ToLower(scope), "-"), "-")
}

func scopeLabel(scope string) string {
	if NormalScope(scope) == "" {
		return "root"
	}
	return NormalScope(scope)
}

// describeScopePath is used as a glob for path-scoped vendor adapters.
func scopeGlob(scope string) string {
	if NormalScope(scope) == "" {
		return "**"
	}
	return NormalScope(scope) + "/**"
}

// renderBullets renders the rules that apply to a scope for one adapter and
// reports which rule IDs and source units the surface actually carries.
func renderBullets(ir IR, adapter, scope string) (string, []string, []string) {
	rules := ir.EffectiveRules(scope)
	seen := map[string]bool{}
	var bullets, sources, ids []string
	for _, r := range rules {
		if !r.Enabled(adapter) {
			continue
		}
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		bullets = append(bullets, "- "+strings.TrimSpace(r.Text))
		sources = append(sources, r.Sources...)
		ids = append(ids, r.ID)
	}
	sort.Strings(sources)
	return strings.Join(bullets, "\n"), dedupe(sources), ids
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func buildSurface(adapter, scope, path, region, head, body string, sources, rules []string) Surface {
	content := body
	if head != "" {
		content = head + "\n" + body
	}
	content = strings.TrimRight(content, "\n") + "\n"
	fingerprint := Fingerprint(content)
	rendered := provenance(adapter, scope, fingerprint, sources) + "\n" + content
	return Surface{
		Adapter:     adapter,
		Scope:       NormalScope(scope),
		Path:        path,
		Region:      region,
		Content:     rendered,
		Fingerprint: fingerprint,
		Rules:       rules,
		Sources:     sources,
	}
}

// agentsMDAdapter projects the root scope into the AGENTS.md managed region and
// nested scopes into their own AGENTS.md file (W16.3, W16.4).
type agentsMDAdapter struct{}

func (agentsMDAdapter) Name() string { return "agents-md" }

func (a agentsMDAdapter) Render(ir IR, scope string) (Surface, bool, error) {
	body, sources, rules := renderBullets(ir, a.Name(), scope)
	if body == "" {
		return Surface{}, false, nil
	}
	if NormalScope(scope) == "" {
		return buildSurface(a.Name(), scope, "AGENTS.md", "agents-core", "", body, sources, rules), true, nil
	}
	head := "# AGENTS.md (" + scopeLabel(scope) + ")\n\nApplies to `" + scopeGlob(scope) + "`."
	return buildSurface(a.Name(), scope, NormalScope(scope)+"/AGENTS.md", "", head, body, sources, rules), true, nil
}

// copilotAdapter projects root rules into the repository-wide Copilot
// instructions file and nested scopes into path-specific instruction files
// (W16.5).
type copilotAdapter struct{}

func (copilotAdapter) Name() string { return "copilot" }

func (a copilotAdapter) Render(ir IR, scope string) (Surface, bool, error) {
	body, sources, rules := renderBullets(ir, a.Name(), scope)
	if body == "" {
		return Surface{}, false, nil
	}
	if NormalScope(scope) == "" {
		head := "# Prumo agent instructions\n\n" + strings.ReplaceAll(body, "- ", "- ")
		return buildSurface(a.Name(), scope, ".github/copilot-instructions.md", "", head, "", sources, rules), true, nil
	}
	path := ".github/instructions/" + slug(scope) + ".instructions.md"
	head := "---\napplyTo: \"" + scopeGlob(scope) + "\"\n---\n"
	return buildSurface(a.Name(), scope, path, "", head, body, sources, rules), true, nil
}

// cursorAdapter projects rules into .cursor/rules/*.mdc, always scoped by glob
// so Cursor never applies a rule outside its scope (W16.6).
type cursorAdapter struct{}

func (cursorAdapter) Name() string { return "cursor" }

func (a cursorAdapter) Render(ir IR, scope string) (Surface, bool, error) {
	body, sources, rules := renderBullets(ir, a.Name(), scope)
	if body == "" {
		return Surface{}, false, nil
	}
	path := ".cursor/rules/prumo-" + slug(scope) + ".mdc"
	head := "---\ndescription: Prumo instruction rules for " + scopeLabel(scope) +
		"\nglobs: \"" + scopeGlob(scope) + "\"\n---\n"
	return buildSurface(a.Name(), scope, path, "", head, body, sources, rules), true, nil
}

// claudeAdapter avoids duplicating AGENTS.md: the root surface imports it and
// nested scopes get their own rule files (W16.7).
type claudeAdapter struct{}

func (claudeAdapter) Name() string { return "claude" }

func (a claudeAdapter) Render(ir IR, scope string) (Surface, bool, error) {
	body, sources, rules := renderBullets(ir, a.Name(), scope)
	if body == "" {
		return Surface{}, false, nil
	}
	if NormalScope(scope) == "" {
		head := "# CLAUDE.md\n\nRepository instructions live in the canonical agent surface:\n\n@AGENTS.md\n"
		return buildSurface(a.Name(), scope, "CLAUDE.md", "", head, "", sources, rules), true, nil
	}
	path := ".claude/rules/prumo-" + slug(scope) + ".md"
	return buildSurface(a.Name(), scope, path, "", "", body, sources, rules), true, nil
}

// skillAdapter projects rules into a SKILL.md file. The frontmatter is the
// vendor's required metadata format and exists only inside generated runtime
// projections; it never becomes a canonical repository artifact (W16.8).
type skillAdapter struct{}

func (skillAdapter) Name() string { return "skill-md" }

func (a skillAdapter) Render(ir IR, scope string) (Surface, bool, error) {
	body, sources, rules := renderBullets(ir, a.Name(), scope)
	if body == "" {
		return Surface{}, false, nil
	}
	path := ".agents/skills/prumo-" + slug(scope) + "/SKILL.md"
	head := "---\nname: prumo-" + slug(scope) +
		"\ndescription: Prumo framework instructions for " + scopeLabel(scope) + ".\n---\n"
	return buildSurface(a.Name(), scope, path, "", head, body, sources, rules), true, nil
}
