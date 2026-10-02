package antigravity

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	prumo "github.com/raillen/prumo"
	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/harness/doccompile"
	"github.com/raillen/prumo/internal/install"
	"github.com/raillen/prumo/internal/protocol"
)

// agentDoc pairs the file an Antigravity workspace expects with the workforce
// agent whose real contract belongs in it. The file names are the established
// ones a workspace already refers to; only the content is corrected.
type agentDoc struct {
	file string
	id   string
}

// defaultAgents is the roster emitted when neither the caller nor the project
// manifest names one. It is the workforce default, not an ad-hoc list: an
// Antigravity workspace with no selection still gets working agents, each with
// its real contract, instead of a one-line stub or an empty subagents directory.
var defaultAgents = []agentDoc{
	{"architect.md", "architect"},
	{"executor.md", "implementer"},
	{"verifier.md", "quality-reviewer"},
	{"systems-architect.md", "systems-architect"},
	{"isolation-auditor.md", "isolation-auditor"},
	{"brand-designer.md", "brand-designer"},
	{"creative-director.md", "creative-director"},
	{"svg-artist.md", "svg-artist"},
	{"advertising-designer.md", "advertising-designer"},
	{"motion-designer.md", "motion-designer"},
	{"ui-component-engineer.md", "ui-component-engineer"},
	{"visual-identity-auditor.md", "visual-identity-auditor"},
	{"prototyper.md", "prototyper"},
}

// selectedAgentIDs resolves which workforce agents this compile should emit:
// the caller's explicit selection first, then the project manifest.
func selectedAgentIDs(opts connectors.CompileOptions, projectRoot string) []string {
	if len(opts.Agents) > 0 {
		return opts.Agents
	}
	return readProjectAgents(projectRoot)
}

// readProjectAgents reads the selected agent ids from the project's manifest.
func readProjectAgents(projectRoot string) []string {
	data, err := os.ReadFile(filepath.Join(projectRoot, ".ai", "agents", "manifest.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		Agents []string `json:"agents"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	return doc.Agents
}

// readAgentContract returns the full AGENT.md contract for one agent, preferring
// the project-local workforce copy and falling back to the embedded one.
func readAgentContract(repoRoot, id string) string {
	local := filepath.Join(repoRoot, "src", "prumo", "resources", "workforce", "agents", id, "AGENT.md")
	if data, err := os.ReadFile(local); err == nil {
		return string(data)
	}
	data, err := fs.ReadFile(prumo.EmbeddedWorkforce(), path.Join("agents", id, "AGENT.md"))
	if err == nil {
		return string(data)
	}
	return ""
}

func init() {
	connectors.Register(NewConnector())
}

// Connector implements connectors.Connector for Google Antigravity IDE and CLI.
type Connector struct{}

// NewConnector creates a new Antigravity connector instance.
func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) ID() string   { return "antigravity" }
func (c *Connector) Name() string { return "Google Antigravity Connector" }

func (c *Connector) Contract() connectors.Contract {
	return connectors.Contract{
		ID:            "antigravity",
		Version:       protocol.CLIVersion,
		ProtocolRange: ">=0.5.0 <0.6.0",
		Capabilities: []string{
			connectors.CapAdvise,
			connectors.CapCommands,
			connectors.CapSessionHooks,
			connectors.CapIsolateSubagents,
			connectors.CapSubagents,
			connectors.CapRestrictTools,
		},
		Enforcement: connectors.EnforcementStandard,
		Hooks: []string{
			connectors.HookSessionStart,
			connectors.HookSessionEnd,
			connectors.HookToolBefore,
			connectors.HookToolAfter,
		},
		Install: map[string]any{
			"directory": ".agents",
			"config":    "GEMINI.md",
		},
		Cleanup: map[string]any{
			"scope":   "project",
			"pattern": ".agents/**",
		},
	}
}

func (c *Connector) Compile(projectRoot string, opts connectors.CompileOptions) (*connectors.CompileResult, error) {
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		projectRoot = cwd
	}

	agentsDir := filepath.Join(projectRoot, ".agents")
	created := []string{}
	managedFragments := []string{}

	// 1. GEMINI.md entrypoint in project root using managed region
	geminiMD := `# GEMINI.md
This project uses Prumo v0.6 with Google Antigravity.

- Follow Lean Progressive Context: smallest sufficient context, progressive expansion, pointer over payload.
- Read ENTRYPOINT.md, prumo.json, and the active Goal before taking any actions.
- Treat Prumo as an external CLI utility available in PATH ('prumo'). Run 'prumo <command>' or 'prumo --help' for project operations and lifecycle. Do not inspect or search for internal framework development source code.
- Test-Driven Development: Every implementation requires exhaustive automated tests (unit, integration, conformance).
- Security First: Zero hardcoded secrets, follow least privilege, audit dependencies and sanitize inputs.
- Clean Architecture: High cohesion, low coupling, modularity, explicit domain boundaries.
- Continuous Documentation: Keep documentation and CHANGELOG.md synchronized with implementation.
- Directory Documentation: Ensure each folder contains a structured README.md.
`
	geminiMDPath := filepath.Join(projectRoot, "GEMINI.md")
	if existingData, err := os.ReadFile(geminiMDPath); err == nil && len(existingData) > 0 {
		cleaned, _ := doccompile.RemoveRegion(string(existingData), "connector-antigravity")
		hasUserContent := strings.TrimSpace(cleaned) != ""
		merged := doccompile.UpsertRegion(string(existingData), "connector-antigravity", geminiMD)
		if err := writeText(geminiMDPath, merged); err != nil {
			return nil, err
		}
		if hasUserContent {
			managedFragments = append(managedFragments, "GEMINI.md:connector-antigravity")
		} else {
			created = append(created, geminiMDPath)
		}
	} else {
		content := doccompile.UpsertRegion("", "connector-antigravity", geminiMD)
		if err := writeText(geminiMDPath, content); err != nil {
			return nil, err
		}
		created = append(created, geminiMDPath)
	}

	// 2. Harness Config
	config := map[string]any{
		"version":      protocol.CLIVersion,
		"harness":      "antigravity",
		"instructions": "GEMINI.md",
		"skills_dir":   ".agents/skills",
		"rules_dir":    ".agents/rules",
		"hooks_file":   ".agents/hooks.json",
	}
	configPath := filepath.Join(agentsDir, "config.json")
	if err := writeJSON(configPath, config); err != nil {
		return nil, err
	}
	created = append(created, configPath)

	// 3. Hierarchical Rules in .agents/rules/
	rules := map[string]string{
		"coding-standards.md": `# Coding Standards
- Clean Code pragmático: explicit responsibilities, small functions, domain naming.
- Zero abstraction without concrete necessity.
- Return explicit, typed errors; never swallow exceptions or fail silently.
- Every folder must contain an explanatory README.md.
`,
		"testing-quality.md": `# Testing & Quality Gate
- Test pyramid: unit, integration, conformance, security SAST, performance, UI/e2e.
- Pre-commit gates: gofmt/format clean, linter clean, race detector passing.
- Continuous verification before completing any task.
`,
		"security.md": `# Security Contract
- No secrets or credentials in source code or commits.
- Strict input validation and sanitization.
- Least-privilege access for subprocesses, file operations, and network.
`,
	}
	for name, content := range rules {
		rulePath := filepath.Join(agentsDir, "rules", name)
		if err := writeText(rulePath, content); err != nil {
			return nil, err
		}
		created = append(created, rulePath)
	}

	// 4. Subagents carry the full runtime contract, not a one-line stub.
	docs := make([]agentDoc, 0, len(defaultAgents))
	if ids := selectedAgentIDs(opts, projectRoot); len(ids) > 0 {
		for _, id := range ids {
			docs = append(docs, agentDoc{file: id + ".md", id: id})
		}
	} else {
		docs = defaultAgents
	}
	emitted := map[string]bool{}
	for _, doc := range docs {
		if emitted[doc.file] {
			continue
		}
		emitted[doc.file] = true
		if content := readAgentContract(opts.RepoRoot, doc.id); content != "" {
			subPath := filepath.Join(agentsDir, "subagents", doc.file)
			if err := writeText(subPath, content); err != nil {
				return nil, err
			}
			created = append(created, subPath)
		}
	}

	// 5. Hooks configuration (.agents/hooks.json)
	hooks := map[string]any{
		"version": "1.0",
		"hooks": map[string]any{
			connectors.HookSessionStart: []map[string]string{
				{"type": "command", "exec": "prumo status --json"},
			},
			connectors.HookSessionEnd: []map[string]string{
				{"type": "command", "exec": "prumo doctor --json"},
			},
			connectors.HookToolBefore: []map[string]string{
				{"type": "audit", "check": "security_boundary"},
			},
			connectors.HookToolAfter: []map[string]string{
				{"type": "audit", "check": "evidence_collection"},
			},
		},
	}
	hooksPath := filepath.Join(agentsDir, "hooks.json")
	if err := writeJSON(hooksPath, hooks); err != nil {
		return nil, err
	}
	created = append(created, hooksPath)

	// 6. Skills in .agents/skills/
	skills := opts.Skills
	if len(skills) == 0 {
		skillsManifestPath := filepath.Join(projectRoot, ".ai", "skills", "manifest.json")
		if data, err := os.ReadFile(skillsManifestPath); err == nil {
			var parsed struct {
				Skills []string `json:"skills"`
			}
			if json.Unmarshal(data, &parsed) == nil {
				skills = parsed.Skills
			}
		}
	}
	if len(skills) == 0 {
		skills = []string{"clean-code", "testing-quality", "secure-coding"}
	}

	for _, skillID := range skills {
		skillDir := filepath.Join(agentsDir, "skills", skillID)
		sourceSkillDir := ""
		if opts.RepoRoot != "" {
			cand := filepath.Join(opts.RepoRoot, "src", "prumo", "resources", "workforce", "skills", skillID)
			if info, err := os.Stat(cand); err == nil && info.IsDir() {
				sourceSkillDir = cand
			}
		}

		if sourceSkillDir != "" {
			copied := copyDirectory(sourceSkillDir, skillDir)
			created = append(created, copied...)
		} else {
			skillMD := fmt.Sprintf(`---
name: %s
description: Prumo skill for %s with automated quality checks.
---

# Skill: %s

## Instructions
Execute skill procedures following Lean Progressive Context.
Verify outputs against quality checklists before declaring completion.
`, skillID, skillID, skillID)
			skillPath := filepath.Join(skillDir, "SKILL.md")
			if err := writeText(skillPath, skillMD); err != nil {
				return nil, err
			}
			created = append(created, skillPath)
		}
	}

	// 7. Ownership marker
	ownership := map[string]any{
		"prumo_generated": true,
		"prumo_version":   protocol.CLIVersion,
		"generator":       "connector-antigravity",
		"target":          "antigravity",
		"managed":         true,
		"created_paths":   created,
	}
	markerPath := filepath.Join(agentsDir, ".prumo-generated.json")
	if err := writeJSON(markerPath, ownership); err != nil {
		return nil, err
	}
	created = append(created, markerPath)

	sort.Strings(created)
	return &connectors.CompileResult{
		Target:           "antigravity",
		CreatedPaths:     created,
		ManagedFragments: managedFragments,
		ManifestPath:     configPath,
		Metadata: map[string]any{
			"rules_count":  len(rules),
			"skills_count": len(skills),
		},
	}, nil
}

func (c *Connector) Install(home string, projectRoot string, opts connectors.InstallOptions) (*connectors.InstallResult, error) {
	if home == "" {
		h, err := install.HomeDir("")
		if err != nil {
			return nil, err
		}
		home = h
	}
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		projectRoot = cwd
	}

	res, err := c.Compile(projectRoot, connectors.CompileOptions{RepoRoot: opts.RepoRoot})
	if err != nil {
		return nil, err
	}

	cleanup := install.CleanupManifest{
		Connector:        "antigravity",
		Scope:            "project",
		ProjectRoot:      projectRoot,
		CreatedPaths:     res.CreatedPaths,
		ManagedFragments: res.ManagedFragments,
	}
	if err := connectors.SaveCleanup(home, "antigravity", cleanup); err != nil {
		return nil, err
	}

	manifest, err := install.LoadManifest(home)
	if err != nil {
		return nil, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.Connectors["antigravity"] = "installed"
	manifest.CreatedPaths = append(manifest.CreatedPaths, res.CreatedPaths...)
	if err := install.SaveManifest(home, manifest); err != nil {
		return nil, err
	}

	return &connectors.InstallResult{
		Connector:        "antigravity",
		Status:           "installed",
		Scope:            "project",
		CreatedPaths:     res.CreatedPaths,
		ManagedFragments: res.ManagedFragments,
		CleanupPath:      install.CleanupPath(home, "antigravity"),
		Contract:         c.Contract(),
	}, nil
}

func (c *Connector) Uninstall(home string, projectRoot string, opts connectors.UninstallOptions) (*connectors.UninstallResult, error) {
	return connectors.ExecuteCleanup(home, "antigravity", projectRoot, ".agents")
}

func (c *Connector) Validate(projectRoot string) (*connectors.ValidationResult, error) {
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		projectRoot = cwd
	}

	res := &connectors.ValidationResult{
		Connector: "antigravity",
		Valid:     true,
		Files:     []string{},
	}

	required := []string{
		"GEMINI.md",
		".agents/config.json",
		".agents/hooks.json",
		".agents/.prumo-generated.json",
	}
	for _, rel := range required {
		p := filepath.Join(projectRoot, rel)
		if _, err := os.Stat(p); err != nil {
			res.Valid = false
			res.Errors = append(res.Errors, fmt.Sprintf("missing artifact: %s", rel))
		} else {
			res.Files = append(res.Files, p)
		}
	}
	return res, nil
}

func writeText(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func writeJSON(path string, val any) error {
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}
	return writeText(path, string(data)+"\n")
}

func copyDirectory(source, target string) []string {
	created := []string{}
	_ = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return nil
		}
		dest := filepath.Join(target, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if err := writeText(dest, string(data)); err != nil {
			return nil
		}
		created = append(created, dest)
		return nil
	})
	sort.Strings(created)
	return created
}
