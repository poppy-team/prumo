package claudecode

import (
	"encoding/json"
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

// Connector implements connectors.Connector for Claude Code.
type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) ID() string   { return "claude-code" }
func (c *Connector) Name() string { return "Anthropic Claude Code Connector" }

func (c *Connector) Contract() connectors.Contract {
	return connectors.Contract{
		ID:            "claude-code",
		Version:       protocol.CLIVersion,
		ProtocolRange: ">=0.5.0 <0.6.0",
		Capabilities: []string{
			connectors.CapAdvise,
			connectors.CapCommands,
			connectors.CapSessionHooks,
			connectors.CapRestrictTools,
			connectors.CapSubagents,
		},
		Enforcement: connectors.EnforcementStandard,
		Hooks: []string{
			connectors.HookSessionStart,
			connectors.HookSessionEnd,
		},
		Install: map[string]any{
			"directory": ".claude",
			"config":    "CLAUDE.md",
		},
		Cleanup: map[string]any{
			"scope":   "project",
			"pattern": ".claude/**",
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

	claudeDir := filepath.Join(projectRoot, ".claude")
	created := []string{}

	// 1. CLAUDE.md entrypoint in project root
	claudeMD := `# CLAUDE.md
This project uses Prumo v0.5.

- Follow Lean Progressive Context: smallest sufficient context, progressive expansion, pointer over payload.
- Read ENTRYPOINT.md, prumo.json, and the active Goal.
- Treat Prumo as an external CLI utility available in PATH ('prumo'). Run 'prumo <command>' or 'prumo --help' for project operations and lifecycle. Do not inspect or search for internal framework development source code.
- Do not scan or read the entire repository by default.
- Stop when verification evidence is sufficient.
`
	claudeMDPath := filepath.Join(projectRoot, "CLAUDE.md")
	if err := writeText(claudeMDPath, claudeMD); err != nil {
		return nil, err
	}
	created = append(created, claudeMDPath)

	// 2. Settings
	settings := map[string]any{
		"version":       protocol.CLIVersion,
		"harness":       "claude-code",
		"instructions":  "CLAUDE.md",
		"subagents_dir": ".claude/agents",
		"skills_dir":    ".claude/skills",
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := writeJSON(settingsPath, settings); err != nil {
		return nil, err
	}
	created = append(created, settingsPath)

	// 3. Subagents
	subagents := map[string]string{
		"architect.md": "# Architect Subagent\nRole: Architecture and design\n",
		"executor.md":  "# Executor Subagent\nRole: Implementation and refactoring\n",
		"verifier.md":  "# Verifier Subagent\nRole: Tests and quality gates\n",
	}
	for name, content := range subagents {
		subPath := filepath.Join(claudeDir, "agents", name)
		if err := writeText(subPath, content); err != nil {
			return nil, err
		}
		created = append(created, subPath)
	}

	// 4. Skills
	skills := map[string]string{
		"code-review/SKILL.md": "# Code Review Skill\nPurpose: Clean code verification\n",
	}
	for name, content := range skills {
		skillPath := filepath.Join(claudeDir, "skills", name)
		if err := writeText(skillPath, content); err != nil {
			return nil, err
		}
		created = append(created, skillPath)
	}

	// 5. Ownership marker
	ownership := map[string]any{
		"prumo_generated": true,
		"prumo_version":   protocol.CLIVersion,
		"generator":       "claude-code-connector",
		"target":          "claude-code",
		"managed":         true,
		"created_paths":   created,
	}
	markerPath := filepath.Join(claudeDir, ".prumo-generated.json")
	if err := writeJSON(markerPath, ownership); err != nil {
		return nil, err
	}
	created = append(created, markerPath)

	sort.Strings(created)
	return &connectors.CompileResult{
		Target:       "claude-code",
		CreatedPaths: created,
		ManifestPath: settingsPath,
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
		Connector:    "claude-code",
		Scope:        "project",
		CreatedPaths: res.CreatedPaths,
	}
	if err := connectors.SaveCleanup(home, "claude-code", cleanup); err != nil {
		return nil, err
	}

	manifest, err := install.LoadManifest(home)
	if err != nil {
		return nil, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.Connectors["claude-code"] = "installed"
	manifest.CreatedPaths = append(manifest.CreatedPaths, res.CreatedPaths...)
	if err := install.SaveManifest(home, manifest); err != nil {
		return nil, err
	}

	return &connectors.InstallResult{
		Connector:    "claude-code",
		Status:       "installed",
		Scope:        "project",
		CreatedPaths: res.CreatedPaths,
		CleanupPath:  install.CleanupPath(home, "claude-code"),
		Contract:     c.Contract(),
	}, nil
}

func (c *Connector) Uninstall(home string, projectRoot string, opts connectors.UninstallOptions) (*connectors.UninstallResult, error) {
	return connectors.ExecuteCleanup(home, "claude-code", projectRoot, ".claude")
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
		Connector: "claude-code",
		Valid:     true,
		Files:     []string{},
	}

	required := []string{
		"CLAUDE.md",
		".claude/settings.json",
		".claude/.prumo-generated.json",
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
