package antigravity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/install"
	"github.com/raillen/prumo/internal/protocol"
)

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
	preserved := []string{}

	// 1. GEMINI.md entrypoint in project root
	geminiMD := `# GEMINI.md
This project uses Prumo v0.5 with Google Antigravity.

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
	// This file belongs to the user first. It is a convention an agent reads, so
	// overwriting it destroys their project instructions, and listing it as
	// created meant the uninstall removed it outright — a loss with no way back
	// (GAP-141). A pre-existing file is left exactly as it is and the connector
	// carries on without claiming it.
	if err := writeText(geminiMDPath, geminiMD); err != nil {
		if errors.Is(err, install.ErrFileBelongsToSomeoneElse) {
			preserved = append(preserved, geminiMDPath)
		} else {
			return nil, err
		}
	} else {
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

	// 4. Subagents
	subagents := map[string]string{
		"architect.md": "# Architect Subagent\nRole: Architecture, Boundaries & Schema Design\n",
		"executor.md":  "# Executor Subagent\nRole: Implementation, Refactoring & Clean Code\n",
		"verifier.md":  "# Verifier Subagent\nRole: Exhaustive Testing, Security & Quality Gates\n",
	}
	for name, content := range subagents {
		subPath := filepath.Join(agentsDir, "subagents", name)
		if err := writeText(subPath, content); err != nil {
			return nil, err
		}
		created = append(created, subPath)
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
	if err := writeRecordJSON(markerPath, ownership); err != nil {
		return nil, err
	}
	created = append(created, markerPath)

	sort.Strings(created)
	return &connectors.CompileResult{
		Target:         "antigravity",
		CreatedPaths:   created,
		PreservedPaths: preserved,
		ManifestPath:   configPath,
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
		Connector:    "antigravity",
		Scope:        "project",
		CreatedPaths: res.CreatedPaths,
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
		Connector:      "antigravity",
		Status:         "installed",
		Scope:          "project",
		CreatedPaths:   res.CreatedPaths,
		PreservedPaths: res.PreservedPaths,
		CleanupPath:    install.CleanupPath(home, "antigravity"),
		Contract:       c.Contract(),
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

// writeText writes generated content unless doing so would destroy a file this
// framework did not write.
//
// Every connector carried its own copy of this, each of which clobbered whatever
// was at the path. For a file inside the connector's own dot-directory that is a
// nuisance; for AGENTS.md, CLAUDE.md and GEMINI.md it destroyed the user's
// instructions, and the path then went into the created list, so the uninstall
// deleted the file outright (GAP-141).
//
// A refusal is reported as install.ErrFileBelongsToSomeoneElse so the caller can
// carry on without claiming ownership, rather than as a failure — the write did
// not fail, it was declined.
func writeText(path, content string) error {
	result, err := install.WriteManaged(path, install.Marked(content))
	if err != nil {
		return err
	}
	if result.Skipped != nil {
		return install.ErrFileBelongsToSomeoneElse
	}
	return nil
}

func writeJSON(path string, val any) error {
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}
	return writeText(path, string(data)+"\n")
}

// writeRecordJSON writes a file that records this framework's own ownership.
//
// It is the one write path allowed to replace its content unconditionally, because
// the ownership manifest's content changes by design — it lists what this run
// created, so the first run and the second produce different documents. Applying
// the general rule to it made the second install refuse to update the record of
// its own work (GAP-141).
func writeRecordJSON(path string, val any) error {
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}
	if _, err := install.WriteRecord(path, string(data)+"\n"); err != nil {
		return err
	}
	return nil
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
