package gemini

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

// Connector implements connectors.Connector for Google Gemini CLI.
type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) ID() string   { return "gemini" }
func (c *Connector) Name() string { return "Google Gemini CLI Connector" }

func (c *Connector) Contract() connectors.Contract {
	return connectors.Contract{
		ID:            "gemini",
		Version:       protocol.CLIVersion,
		ProtocolRange: ">=0.5.0 <0.6.0",
		Capabilities: []string{
			connectors.CapAdvise,
			connectors.CapCommands,
			connectors.CapSessionHooks,
			connectors.CapIsolateSubagents,
		},
		Enforcement: connectors.EnforcementStandard,
		Hooks: []string{
			connectors.HookSessionStart,
			connectors.HookSessionEnd,
		},
		Install: map[string]any{
			"directory": ".gemini",
			"config":    ".gemini/config.json",
		},
		Cleanup: map[string]any{
			"scope":   "project",
			"pattern": ".gemini/**",
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

	geminiDir := filepath.Join(projectRoot, ".gemini")
	created := []string{}

	// 1. Config
	config := map[string]any{
		"version":           protocol.CLIVersion,
		"harness":           "gemini-cli",
		"instructions_file": "prompts/prumo.md",
		"commands_file":     "commands.json",
		"subagents_dir":     "subagents",
	}
	configPath := filepath.Join(geminiDir, "config.json")
	if err := writeJSON(configPath, config); err != nil {
		return nil, err
	}
	created = append(created, configPath)

	// 2. Primary instructions
	instructions := `# Prumo for Gemini CLI

Follow Lean Progressive Context (LPC/PCA): smallest sufficient context, progressive expansion, pointer over payload.
Start at ENTRYPOINT.md, prumo.json, and the active Goal.
Treat Prumo as an external CLI utility available in PATH ('prumo'). Run 'prumo <command>' or 'prumo --help' for project operations and lifecycle. Do not inspect or search for internal framework development source code.
Do not scan the whole codebase unless explicitly requested.
`
	instPath := filepath.Join(geminiDir, "prompts", "prumo.md")
	if err := writeText(instPath, instructions); err != nil {
		return nil, err
	}
	created = append(created, instPath)

	// 3. Subagents
	subagents := map[string]string{
		"architect.md": "# Architect Subagent\nRole: Architecture & Schemas\n",
		"executor.md":  "# Executor Subagent\nRole: Implementation & Refactoring\n",
		"verifier.md":  "# Verifier Subagent\nRole: Tests & Quality Gates\n",
	}
	for name, content := range subagents {
		subPath := filepath.Join(geminiDir, "subagents", name)
		if err := writeText(subPath, content); err != nil {
			return nil, err
		}
		created = append(created, subPath)
	}

	// 4. Commands
	commands := map[string]any{
		"commands": []map[string]any{
			{"name": "goal", "description": "Manage Prumo goals", "command": "prumo goal"},
			{"name": "plan", "description": "Inspect Living Plan", "command": "prumo plan"},
			{"name": "trace", "description": "Trace requirements to code", "command": "prumo trace"},
			{"name": "status", "description": "Show project status", "command": "prumo status"},
		},
	}
	cmdPath := filepath.Join(geminiDir, "commands.json")
	if err := writeJSON(cmdPath, commands); err != nil {
		return nil, err
	}
	created = append(created, cmdPath)

	// 5. Ownership marker
	ownership := map[string]any{
		"prumo_generated": true,
		"prumo_version":   protocol.CLIVersion,
		"generator":       "gemini-connector",
		"target":          "gemini",
		"managed":         true,
		"created_paths":   created,
	}
	markerPath := filepath.Join(geminiDir, ".prumo-generated.json")
	if err := writeRecordJSON(markerPath, ownership); err != nil {
		return nil, err
	}
	created = append(created, markerPath)

	sort.Strings(created)
	return &connectors.CompileResult{
		Target:       "gemini",
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
		Connector:    "gemini",
		Scope:        "project",
		CreatedPaths: res.CreatedPaths,
	}
	if err := connectors.SaveCleanup(home, "gemini", cleanup); err != nil {
		return nil, err
	}

	manifest, err := install.LoadManifest(home)
	if err != nil {
		return nil, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.Connectors["gemini"] = "installed"
	manifest.CreatedPaths = append(manifest.CreatedPaths, res.CreatedPaths...)
	if err := install.SaveManifest(home, manifest); err != nil {
		return nil, err
	}

	return &connectors.InstallResult{
		Connector:      "gemini",
		Status:         "installed",
		Scope:          "project",
		CreatedPaths:   res.CreatedPaths,
		PreservedPaths: res.PreservedPaths,
		CleanupPath:    install.CleanupPath(home, "gemini"),
		Contract:       c.Contract(),
	}, nil
}

func (c *Connector) Uninstall(home string, projectRoot string, opts connectors.UninstallOptions) (*connectors.UninstallResult, error) {
	return connectors.ExecuteCleanup(home, "gemini", projectRoot, ".gemini")
}

func (c *Connector) Validate(projectRoot string) (*connectors.ValidationResult, error) {
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		projectRoot = cwd
	}

	geminiDir := filepath.Join(projectRoot, ".gemini")
	res := &connectors.ValidationResult{
		Connector: "gemini",
		Valid:     true,
		Files:     []string{},
	}

	required := []string{
		"config.json",
		"prompts/prumo.md",
		"commands.json",
		".prumo-generated.json",
	}
	for _, rel := range required {
		p := filepath.Join(geminiDir, rel)
		if _, err := os.Stat(p); err != nil {
			res.Valid = false
			res.Errors = append(res.Errors, fmt.Sprintf("missing artifact: .gemini/%s", rel))
		} else {
			res.Files = append(res.Files, p)
		}
	}

	markerPath := filepath.Join(geminiDir, ".prumo-generated.json")
	if data, err := os.ReadFile(markerPath); err == nil {
		var marker map[string]any
		if json.Unmarshal(data, &marker) == nil {
			if marker["target"] != "gemini" || marker["managed"] != true {
				res.Valid = false
				res.Errors = append(res.Errors, "invalid ownership marker for gemini")
			}
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
