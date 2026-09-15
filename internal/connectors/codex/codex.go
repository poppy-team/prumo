package codex

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

// Connector implements connectors.Connector for OpenAI Codex CLI.
type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) ID() string   { return "codex" }
func (c *Connector) Name() string { return "OpenAI Codex CLI Connector" }

func (c *Connector) Contract() connectors.Contract {
	return connectors.Contract{
		ID:            "codex",
		Version:       protocol.CLIVersion,
		ProtocolRange: ">=0.5.0 <0.6.0",
		Capabilities: []string{
			connectors.CapAdvise,
			connectors.CapCommands,
			connectors.CapRestrictTools,
			connectors.CapSubagents,
		},
		Enforcement: connectors.EnforcementStandard,
		Hooks:       []string{},
		Install: map[string]any{
			"directory": ".codex",
			"config":    "AGENTS.md",
		},
		Cleanup: map[string]any{
			"scope":   "project",
			"pattern": ".codex/**",
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

	codexDir := filepath.Join(projectRoot, ".codex")
	created := []string{}

	// 1. AGENTS.md entrypoint in project root
	agentsMD := `# AGENTS.md
This project uses Prumo v0.5.

- Follow Lean Progressive Context: smallest sufficient context, progressive expansion, pointer over payload.
- Read ENTRYPOINT.md, prumo.json, and the active Goal.
- Treat Prumo as an external CLI utility available in PATH ('prumo'). Run 'prumo <command>' or 'prumo --help' for project operations and lifecycle. Do not inspect or search for internal framework development source code.
- Do not scan or read the entire repository by default.
- Stop when verification evidence is sufficient.
`
	agentsMDPath := filepath.Join(projectRoot, "AGENTS.md")
	if err := writeText(agentsMDPath, agentsMD); err != nil {
		return nil, err
	}
	created = append(created, agentsMDPath)

	// 2. Configuration
	config := map[string]any{
		"version":       protocol.CLIVersion,
		"harness":       "codex",
		"instructions":  "AGENTS.md",
		"subagents_dir": ".codex/agents",
	}
	configPath := filepath.Join(codexDir, "config.json")
	if err := writeJSON(configPath, config); err != nil {
		return nil, err
	}
	created = append(created, configPath)

	// 3. Subagents
	subagents := map[string]string{
		"architect.md": "# Architect Subagent\nRole: Architecture and design\n",
		"executor.md":  "# Executor Subagent\nRole: Implementation and refactoring\n",
		"verifier.md":  "# Verifier Subagent\nRole: Tests and quality gates\n",
	}
	for name, content := range subagents {
		subPath := filepath.Join(codexDir, "agents", name)
		if err := writeText(subPath, content); err != nil {
			return nil, err
		}
		created = append(created, subPath)
	}

	// 4. Ownership marker
	ownership := map[string]any{
		"prumo_generated": true,
		"prumo_version":   protocol.CLIVersion,
		"generator":       "codex-connector",
		"target":          "codex",
		"managed":         true,
		"created_paths":   created,
	}
	markerPath := filepath.Join(codexDir, ".prumo-generated.json")
	if err := writeJSON(markerPath, ownership); err != nil {
		return nil, err
	}
	created = append(created, markerPath)

	sort.Strings(created)
	return &connectors.CompileResult{
		Target:       "codex",
		CreatedPaths: created,
		ManifestPath: configPath,
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
		Connector:    "codex",
		Scope:        "project",
		CreatedPaths: res.CreatedPaths,
	}
	if err := connectors.SaveCleanup(home, "codex", cleanup); err != nil {
		return nil, err
	}

	manifest, err := install.LoadManifest(home)
	if err != nil {
		return nil, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.Connectors["codex"] = "installed"
	manifest.CreatedPaths = append(manifest.CreatedPaths, res.CreatedPaths...)
	if err := install.SaveManifest(home, manifest); err != nil {
		return nil, err
	}

	return &connectors.InstallResult{
		Connector:    "codex",
		Status:       "installed",
		Scope:        "project",
		CreatedPaths: res.CreatedPaths,
		CleanupPath:  install.CleanupPath(home, "codex"),
		Contract:     c.Contract(),
	}, nil
}

func (c *Connector) Uninstall(home string, projectRoot string, opts connectors.UninstallOptions) (*connectors.UninstallResult, error) {
	return connectors.ExecuteCleanup(home, "codex", projectRoot, ".codex")
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
		Connector: "codex",
		Valid:     true,
		Files:     []string{},
	}

	required := []string{
		"AGENTS.md",
		".codex/config.json",
		".codex/.prumo-generated.json",
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
