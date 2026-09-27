package windsurf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/harness/doccompile"
	"github.com/raillen/prumo/internal/install"
	"github.com/raillen/prumo/internal/protocol"
)

func init() {
	connectors.Register(NewConnector())
}

// Connector implements connectors.Connector for Codeium Windsurf IDE.
type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) ID() string   { return "windsurf" }
func (c *Connector) Name() string { return "Codeium Windsurf IDE Connector" }

func (c *Connector) Contract() connectors.Contract {
	return connectors.Contract{
		ID:            "windsurf",
		Version:       protocol.CLIVersion,
		ProtocolRange: ">=0.5.0 <0.6.0",
		Capabilities: []string{
			connectors.CapAdvise,
			connectors.CapCommands,
			connectors.CapRestrictTools,
		},
		Enforcement: connectors.EnforcementStandard,
		Hooks:       []string{},
		Install: map[string]any{
			"directory": ".windsurf",
			"config":    ".windsurfrules",
		},
		Cleanup: map[string]any{
			"scope":   "project",
			"pattern": ".windsurf/**",
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

	windsurfDir := filepath.Join(projectRoot, ".windsurf")
	created := []string{}
	managedFragments := []string{}

	// 1. .windsurfrules in project root
	rulesContent := `# Windsurf Guidelines
This project uses Prumo v0.5 with Windsurf IDE.

- Follow Lean Progressive Context: smallest sufficient context, progressive expansion, pointer over payload.
- Read ENTRYPOINT.md, prumo.json, and the active Goal.
- Treat Prumo as an external CLI utility available in PATH ('prumo'). Run 'prumo <command>' or 'prumo --help' for project operations and lifecycle. Do not inspect or search for internal framework development source code.
- Do not scan or read the entire repository by default.
- Stop when verification evidence is sufficient.
`
	rulesPath := filepath.Join(projectRoot, ".windsurfrules")
	if existingData, err := os.ReadFile(rulesPath); err == nil && len(existingData) > 0 {
		cleaned, _ := doccompile.RemoveRegion(string(existingData), "connector-windsurf")
		hasUserContent := strings.TrimSpace(cleaned) != ""
		merged := doccompile.UpsertRegion(string(existingData), "connector-windsurf", rulesContent)
		if err := writeText(rulesPath, merged); err != nil {
			return nil, err
		}
		if hasUserContent {
			managedFragments = append(managedFragments, ".windsurfrules:connector-windsurf")
		} else {
			created = append(created, rulesPath)
		}
	} else {
		content := doccompile.UpsertRegion("", "connector-windsurf", rulesContent)
		if err := writeText(rulesPath, content); err != nil {
			return nil, err
		}
		created = append(created, rulesPath)
	}

	// 2. MCP configuration (.windsurf/mcp_config.json)
	mcpConfig := map[string]any{
		"mcpServers": map[string]any{
			"prumo": map[string]any{
				"command": "prumo",
				"args":    []string{"mcp", "serve"},
			},
		},
	}
	mcpPath := filepath.Join(windsurfDir, "mcp_config.json")
	if err := writeJSON(mcpPath, mcpConfig); err != nil {
		return nil, err
	}
	created = append(created, mcpPath)

	// 3. Ownership marker
	ownership := map[string]any{
		"prumo_generated": true,
		"prumo_version":   protocol.CLIVersion,
		"generator":       "windsurf-connector",
		"target":          "windsurf",
		"managed":         true,
		"created_paths":   created,
	}
	markerPath := filepath.Join(windsurfDir, ".prumo-generated.json")
	if err := writeJSON(markerPath, ownership); err != nil {
		return nil, err
	}
	created = append(created, markerPath)

	sort.Strings(created)
	return &connectors.CompileResult{
		Target:           "windsurf",
		CreatedPaths:     created,
		ManagedFragments: managedFragments,
		ManifestPath:     markerPath,
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
		Connector:        "windsurf",
		Scope:            "project",
		ProjectRoot:      projectRoot,
		CreatedPaths:     res.CreatedPaths,
		ManagedFragments: res.ManagedFragments,
	}
	if err := connectors.SaveCleanup(home, "windsurf", cleanup); err != nil {
		return nil, err
	}

	manifest, err := install.LoadManifest(home)
	if err != nil {
		return nil, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.Connectors["windsurf"] = "installed"
	manifest.CreatedPaths = append(manifest.CreatedPaths, res.CreatedPaths...)
	if err := install.SaveManifest(home, manifest); err != nil {
		return nil, err
	}

	return &connectors.InstallResult{
		Connector:        "windsurf",
		Status:           "installed",
		Scope:            "project",
		CreatedPaths:     res.CreatedPaths,
		ManagedFragments: res.ManagedFragments,
		CleanupPath:      install.CleanupPath(home, "windsurf"),
		Contract:         c.Contract(),
	}, nil
}

func (c *Connector) Uninstall(home string, projectRoot string, opts connectors.UninstallOptions) (*connectors.UninstallResult, error) {
	return connectors.ExecuteCleanup(home, "windsurf", projectRoot, ".windsurf")
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
		Connector: "windsurf",
		Valid:     true,
		Files:     []string{},
	}

	required := []string{
		".windsurfrules",
		".windsurf/mcp_config.json",
		".windsurf/.prumo-generated.json",
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
