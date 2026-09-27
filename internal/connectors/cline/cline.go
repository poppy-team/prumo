package cline

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

// Connector implements connectors.Connector for Cline and Roo Code.
type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) ID() string   { return "cline" }
func (c *Connector) Name() string { return "Cline / Roo Code Connector" }

func (c *Connector) Contract() connectors.Contract {
	return connectors.Contract{
		ID:            "cline",
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
			"directory": ".cline",
			"config":    ".clinerules",
		},
		Cleanup: map[string]any{
			"scope":   "project",
			"pattern": ".cline/**",
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

	clineDir := filepath.Join(projectRoot, ".cline")
	created := []string{}
	managedFragments := []string{}

	// 1. .clinerules in project root
	rulesContent := `# Cline Guidelines
This project uses Prumo v0.5 with Cline / Roo Code.

- Follow Lean Progressive Context: smallest sufficient context, progressive expansion, pointer over payload.
- Read ENTRYPOINT.md, prumo.json, and the active Goal.
- Treat Prumo as an external CLI utility available in PATH ('prumo'). Run 'prumo <command>' or 'prumo --help' for project operations and lifecycle. Do not inspect or search for internal framework development source code.
- Do not scan or read the entire repository by default.
- Stop when verification evidence is sufficient.
`
	rulesPath := filepath.Join(projectRoot, ".clinerules")
	if existingData, err := os.ReadFile(rulesPath); err == nil && len(existingData) > 0 {
		cleaned, _ := doccompile.RemoveRegion(string(existingData), "connector-cline")
		hasUserContent := strings.TrimSpace(cleaned) != ""
		merged := doccompile.UpsertRegion(string(existingData), "connector-cline", rulesContent)
		if err := writeText(rulesPath, merged); err != nil {
			return nil, err
		}
		if hasUserContent {
			managedFragments = append(managedFragments, ".clinerules:connector-cline")
		} else {
			created = append(created, rulesPath)
		}
	} else {
		content := doccompile.UpsertRegion("", "connector-cline", rulesContent)
		if err := writeText(rulesPath, content); err != nil {
			return nil, err
		}
		created = append(created, rulesPath)
	}

	// 2. Custom modes for Roo Code (.roomodes)
	rooModes := map[string]any{
		"customModes": []map[string]any{
			{
				"slug":           "prumo-architect",
				"name":           "Prumo Architect",
				"roleDefinition": "Software architect responsible for system design, ADRs, and boundary planning.",
				"groups":         []string{"read"},
			},
			{
				"slug":           "prumo-executor",
				"name":           "Prumo Executor",
				"roleDefinition": "Software engineer responsible for implementation, refactoring, and deterministic tests.",
				"groups":         []string{"read", "edit", "command"},
			},
		},
	}
	rooModesPath := filepath.Join(projectRoot, ".roomodes")
	if err := writeJSON(rooModesPath, rooModes); err != nil {
		return nil, err
	}
	created = append(created, rooModesPath)

	// 3. Ownership marker
	ownership := map[string]any{
		"prumo_generated": true,
		"prumo_version":   protocol.CLIVersion,
		"generator":       "cline-connector",
		"target":          "cline",
		"managed":         true,
		"created_paths":   created,
	}
	markerPath := filepath.Join(clineDir, ".prumo-generated.json")
	if err := writeJSON(markerPath, ownership); err != nil {
		return nil, err
	}
	created = append(created, markerPath)

	sort.Strings(created)
	return &connectors.CompileResult{
		Target:           "cline",
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
		Connector:        "cline",
		Scope:            "project",
		ProjectRoot:      projectRoot,
		CreatedPaths:     res.CreatedPaths,
		ManagedFragments: res.ManagedFragments,
	}
	if err := connectors.SaveCleanup(home, "cline", cleanup); err != nil {
		return nil, err
	}

	manifest, err := install.LoadManifest(home)
	if err != nil {
		return nil, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.Connectors["cline"] = "installed"
	manifest.CreatedPaths = append(manifest.CreatedPaths, res.CreatedPaths...)
	if err := install.SaveManifest(home, manifest); err != nil {
		return nil, err
	}

	return &connectors.InstallResult{
		Connector:        "cline",
		Status:           "installed",
		Scope:            "project",
		CreatedPaths:     res.CreatedPaths,
		ManagedFragments: res.ManagedFragments,
		CleanupPath:      install.CleanupPath(home, "cline"),
		Contract:         c.Contract(),
	}, nil
}

func (c *Connector) Uninstall(home string, projectRoot string, opts connectors.UninstallOptions) (*connectors.UninstallResult, error) {
	return connectors.ExecuteCleanup(home, "cline", projectRoot, ".cline")
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
		Connector: "cline",
		Valid:     true,
		Files:     []string{},
	}

	required := []string{
		".clinerules",
		".roomodes",
		".cline/.prumo-generated.json",
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
