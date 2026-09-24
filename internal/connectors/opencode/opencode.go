package opencode

import (
	"crypto/sha256"
	"encoding/hex"
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

// Connector implements connectors.Connector for OpenCode.
type Connector struct{}

// NewConnector creates a new OpenCode connector instance.
func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) ID() string   { return "opencode" }
func (c *Connector) Name() string { return "OpenCode Native Harness" }

func (c *Connector) Contract() connectors.Contract {
	return connectors.Contract{
		ID:            "opencode",
		Version:       protocol.CLIVersion,
		ProtocolRange: ">=0.5.0 <0.6.0",
		Capabilities: []string{
			connectors.CapAdvise,
			connectors.CapRestrictTools,
			connectors.CapPreToolBlock,
			connectors.CapPostToolVerify,
			connectors.CapIsolateSubagents,
			connectors.CapSessionHooks,
			connectors.CapNativePlugin,
			connectors.CapCommands,
			connectors.CapSubagents,
			connectors.CapSkills,
		},
		Enforcement: connectors.EnforcementStrict,
		Hooks: []string{
			connectors.HookSessionStart,
			connectors.HookSessionEnd,
			connectors.HookToolBefore,
			connectors.HookToolAfter,
		},
		Install: map[string]any{
			"directory": ".opencode",
			"config":    ".opencode/opencode.json",
			"plugin":    ".opencode/plugins/prumo.ts",
		},
		Cleanup: map[string]any{
			"scope":   "project",
			"pattern": ".opencode/**",
		},
	}
}

// Compile compiles native OpenCode artifacts into .opencode/ under projectRoot.
func (c *Connector) Compile(projectRoot string, opts connectors.CompileOptions) (*connectors.CompileResult, error) {
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		projectRoot = cwd
	}

	opencodeDir := filepath.Join(projectRoot, ".opencode")
	created := []string{}

	// 1. Tool Guards: .opencode/guards/tool-policy.json
	guardPolicy := map[string]any{
		"version":     protocol.CLIVersion,
		"enforcement": "strict",
		"primitives": []string{
			connectors.CapPreToolBlock,
			connectors.CapPostToolVerify,
			connectors.CapRestrictTools,
		},
		"blocked_commands": []string{
			"rm -rf /",
			"rm -rf /*",
			"mkfs",
			":(){ :|:& };:",
			"dd if=/dev/zero",
		},
		"protected_paths": []string{
			".git/",
			".prumo/credentials",
			".env",
		},
		"require_confirmation": []string{
			"git push --force",
			"git reset --hard",
			"git clean -fdx",
		},
	}
	guardPath := filepath.Join(opencodeDir, "guards", "tool-policy.json")
	if err := writeJSON(guardPath, guardPolicy); err != nil {
		return nil, fmt.Errorf("failed to write tool policy: %w", err)
	}
	created = append(created, guardPath)

	// 2. TypeScript Plugin: .opencode/plugins/prumo.ts
	pluginContent := `/**
 * Prumo - OpenCode Native Plugin
 * Version: 0.5.0
 * 
 * Provides native harness integration between OpenCode and Prumo:
 * - Pre-tool validation (tool guards)
 * - Session lifecycle hooks (start, end, tool events)
 * - Lean Progressive Context injection
 * - Command routing to Prumo CLI
 */

import * as fs from 'fs';
import * as path from 'path';

export interface PrumoPluginConfig {
  prumoHome?: string;
  projectRoot?: string;
  enforceToolGuards?: boolean;
}

export interface ToolGuardResult {
  allowed: boolean;
  reason?: string;
}

// Tool Guard: validates tool execution before execution (pre-tool block)
export function validateToolExecution(toolName: string, params: Record<string, any>, policyPath?: string): ToolGuardResult {
  const dangerousCommands = [
    'rm -rf /',
    'rm -rf /*',
    'mkfs',
    ':(){ :|:& };:',
    'dd if=/dev/zero'
  ];

  if (toolName === 'bash' || toolName === 'execute_command' || toolName === 'run_command') {
    const cmd = String(params.command || params.cmd || params.CommandLine || '');
    for (const dangerous of dangerousCommands) {
      if (cmd.includes(dangerous)) {
        return {
          allowed: false,
          reason: 'Prumo Tool Guard blocked execution of dangerous command: ' + dangerous
        };
      }
    }
  }

  if (toolName === 'write_file' || toolName === 'write_to_file') {
    const target = String(params.path || params.TargetFile || '');
    if (target.includes('.git/') || target.includes('.prumo/credentials')) {
      return {
        allowed: false,
        reason: 'Prumo Tool Guard protected sensitive path: ' + target
      };
    }
  }

  return { allowed: true };
}

// Session Lifecycle Hook: Start
export function onSessionStart(sessionID: string, projectRoot: string): { contextPrompt: string } {
  const entrypointPath = path.join(projectRoot, 'ENTRYPOINT.md');
  let entrypointText = '';
  if (fs.existsSync(entrypointPath)) {
    entrypointText = fs.readFileSync(entrypointPath, 'utf8');
  }

  return {
    contextPrompt: [
      '[Prumo v0.5] Native OpenCode Harness Active',
      'Follow Lean Progressive Context: smallest sufficient context, pointer over payload.',
      'Treat Prumo as an external CLI utility available in PATH (\'prumo\'). Use \'prumo <command>\' for project operations. Do not inspect internal framework development source code.',
      'Active Entrypoint:',
      entrypointText
    ].join('\n\n')
  };
}

// Session Lifecycle Hook: End
export function onSessionEnd(sessionID: string, projectRoot: string, summary?: string): void {
  const expDir = path.join(projectRoot, '.prumo', 'experience');
  if (fs.existsSync(expDir)) {
    const eventFile = path.join(expDir, 'session-' + sessionID + '.json');
    const payload = {
      session_id: sessionID,
      event_type: 'session_completed',
      summary: summary || 'OpenCode session completed',
      timestamp: new Date().toISOString()
    };
    fs.writeFileSync(eventFile, JSON.stringify(payload, null, 2));
  }
}
`
	pluginPath := filepath.Join(opencodeDir, "plugins", "prumo.ts")
	if err := writeText(pluginPath, pluginContent); err != nil {
		return nil, fmt.Errorf("failed to write TypeScript plugin: %w", err)
	}
	created = append(created, pluginPath)

	// 3. Primary Agent: .opencode/agents/prumo.md
	prumoAgent := `---
name: Prumo
role: Primary Orchestrator
description: Canonical Prumo primary agent in OpenCode
version: 0.5.0
---

# Prumo Primary Agent

You are the primary orchestrator of Prumo within OpenCode.

## Core Directives
1. **Lean Progressive Context (LPC/PCA)**: Smallest sufficient context, progressive expansion, pointer over payload. Never preload the whole repository.
2. **Prumo CLI as Black Box**: Treat Prumo as an external CLI utility available in PATH ('prumo'). Run 'prumo <command>' or 'prumo --help' for project operations and lifecycle. Do not search for or inspect framework development source code.
3. **Authority Hierarchy**:
   1. Canonical repository specs and schemas
   2. Accepted local engineering documentation
   3. Notion Living Book
   4. Agent inference
4. **Goal Discipline**: Work strictly inside locked Goals. Verification criteria and evidence determine completion.
5. **Delegation**: Delegate to specialized subagents:
   - architect: architecture, schemas, and ADRs
   - executor: pragmatic implementation and code changes
   - verifier: test suites, linters, and quality gates
6. **Zero-Transcript Experience**: Record structured evidence and session handoffs without conversational bloat.
`
	prumoAgentPath := filepath.Join(opencodeDir, "agents", "prumo.md")
	if err := writeText(prumoAgentPath, prumoAgent); err != nil {
		return nil, fmt.Errorf("failed to write primary agent: %w", err)
	}
	created = append(created, prumoAgentPath)

	// 4. Subagents: .opencode/agents/architect.md, executor.md, verifier.md
	subagents := map[string]string{
		"architect.md": `# Architect Agent
Role: Architecture, Schemas, and ADRs
Focus: Dependency rules, draft 2020-12 schemas, boundary integrity.
`,
		"executor.md": `# Executor Agent
Role: Implementation and Refactoring
Focus: Pragmatic clean code, explicit errors, deterministic behavior.
`,
		"verifier.md": `# Verifier Agent
Role: Verification and Quality Gates
Focus: Deterministic tests, race detection, linting, and evidence recording.
`,
	}
	for name, content := range subagents {
		subagentPath := filepath.Join(opencodeDir, "agents", name)
		if err := writeText(subagentPath, content); err != nil {
			return nil, fmt.Errorf("failed to write subagent %s: %w", name, err)
		}
		created = append(created, subagentPath)
	}

	// 5. Skills: .opencode/skills/
	skills := map[string]string{
		"code-review/SKILL.md": `# Code Review Skill
Purpose: Automated pragmatic clean code and security boundary review.
Risk Level: low
`,
		"goal-management/SKILL.md": `# Goal Management Skill
Purpose: Manage and transition lifecycle states of Prumo goals.
Risk Level: low
`,
		"evidence-collection/SKILL.md": `# Evidence Collection Skill
Purpose: Capture deterministic test runs and verification results as evidence.
Risk Level: low
`,
	}
	for relPath, content := range skills {
		skillPath := filepath.Join(opencodeDir, "skills", relPath)
		if err := writeText(skillPath, content); err != nil {
			return nil, fmt.Errorf("failed to write skill %s: %w", relPath, err)
		}
		created = append(created, skillPath)
	}

	// 6. Commands: .opencode/commands/prumo.json
	commands := map[string]any{
		"commands": []map[string]any{
			{"name": "goal", "description": "Manage Prumo goals", "action": "prumo goal"},
			{"name": "plan", "description": "Inspect Living Plan", "action": "prumo plan"},
			{"name": "trace", "description": "Trace requirements to code and evidence", "action": "prumo trace"},
			{"name": "experience", "description": "View experience events and handoff", "action": "prumo experience"},
			{"name": "adopt", "description": "Scan and adopt repository", "action": "prumo adopt"},
			{"name": "status", "description": "Show Prumo project status", "action": "prumo status"},
		},
	}
	commandsPath := filepath.Join(opencodeDir, "commands", "prumo.json")
	if err := writeJSON(commandsPath, commands); err != nil {
		return nil, fmt.Errorf("failed to write commands: %w", err)
	}
	created = append(created, commandsPath)

	// 7. OpenCode Config: .opencode/opencode.json
	opencodeConfig := map[string]any{
		"$schema":       "https://opencode.ai/schema/v1.json",
		"name":          "Prumo OpenCode Integration",
		"version":       protocol.CLIVersion,
		"plugin":        []string{"plugins/prumo.ts"},
		"primary_agent": "agents/prumo.md",
		"agents_dir":    "agents",
		"skills_dir":    "skills",
		"commands_file": "commands/prumo.json",
		"guards_file":   "guards/tool-policy.json",
		"hooks": map[string]bool{
			"on_session_start": true,
			"on_session_end":   true,
			"on_tool_before":   true,
			"on_tool_after":    true,
		},
	}
	configPath := filepath.Join(opencodeDir, "opencode.json")
	if err := writeJSON(configPath, opencodeConfig); err != nil {
		return nil, fmt.Errorf("failed to write opencode.json: %w", err)
	}
	created = append(created, configPath)

	// 8. Ownership Marker: .opencode/.prumo-generated.json
	ownership := map[string]any{
		"prumo_generated": true,
		"prumo_version":   protocol.CLIVersion,
		"generator":       "opencode-compiler",
		"target":          "opencode",
		"managed":         true,
		"created_paths":   created,
	}
	ownershipPath := filepath.Join(opencodeDir, ".prumo-generated.json")
	if err := writeRecordJSON(ownershipPath, ownership); err != nil {
		return nil, fmt.Errorf("failed to write ownership marker: %w", err)
	}
	created = append(created, ownershipPath)

	sort.Strings(created)

	return &connectors.CompileResult{
		Target:       "opencode",
		CreatedPaths: created,
		ManifestPath: configPath,
		// Derived from what was just written, not copied from the declaration.
		// The config carries the hook map, the plugin exists, the guard policy
		// exists, subagents and skills were written, and the commands file was
		// written — so those are the capabilities this compile actually
		// delivered (GAP-142).
		Implemented:      connectors.DeriveImplemented(opencodeConfig, created),
		ImplementedHooks: connectors.DeriveImplementedHooksFromConfig(opencodeConfig),
		Metadata: map[string]any{
			"primary_agent": prumoAgentPath,
			"plugin":        pluginPath,
			"guard_policy":  guardPath,
		},
	}, nil
}

// Install compiles native artifacts and registers OpenCode with PRUMO_HOME.
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

	compileRes, err := c.Compile(projectRoot, connectors.CompileOptions{RepoRoot: opts.RepoRoot})
	if err != nil {
		return nil, err
	}

	// Record cleanup manifest
	cleanup := install.CleanupManifest{
		Connector:        "opencode",
		Scope:            "project",
		CreatedPaths:     compileRes.CreatedPaths,
		ManagedFragments: []string{},
		Backups:          []string{},
	}
	cleanupPath := install.CleanupPath(home, "opencode", projectRoot)
	if err := os.MkdirAll(filepath.Dir(cleanupPath), 0755); err != nil {
		return nil, err
	}
	cleanupData, err := json.MarshalIndent(cleanup, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(cleanupPath, append(cleanupData, '\n'), 0644); err != nil {
		return nil, err
	}

	// Update installation manifest
	manifest, err := install.LoadManifest(home)
	if err != nil {
		return nil, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.Connectors["opencode"] = "installed"
	manifest.CreatedPaths = append(manifest.CreatedPaths, compileRes.CreatedPaths...)
	if err := install.SaveManifest(home, manifest); err != nil {
		return nil, err
	}

	return &connectors.InstallResult{
		Connector:    "opencode",
		Status:       "installed",
		Scope:        "project",
		CreatedPaths: compileRes.CreatedPaths,
		CleanupPath:  cleanupPath,
		Contract:     c.Contract(),
	}, nil
}

// Uninstall cleanly removes managed OpenCode artifacts.
func (c *Connector) Uninstall(home string, projectRoot string, opts connectors.UninstallOptions) (*connectors.UninstallResult, error) {
	if home == "" {
		h, err := install.HomeDir("")
		if err != nil {
			return nil, err
		}
		home = h
	}

	cleanupPath := install.CleanupPath(home, "opencode", projectRoot)
	data, err := os.ReadFile(cleanupPath)
	if err != nil {
		return nil, fmt.Errorf("no cleanup manifest found for opencode: %w", err)
	}

	var cleanup install.CleanupManifest
	if err := json.Unmarshal(data, &cleanup); err != nil {
		return nil, fmt.Errorf("corrupt cleanup manifest: %w", err)
	}

	removed, leftovers := install.RemoveManagedPaths(home, cleanup.CreatedPaths)

	// Clean up .opencode dir if empty
	if projectRoot != "" {
		opencodeDir := filepath.Join(projectRoot, ".opencode")
		_ = os.Remove(opencodeDir)
	}

	// Remove cleanup manifest
	_ = os.Remove(cleanupPath)

	// Update installation manifest
	manifest, err := install.LoadManifest(home)
	if err == nil {
		delete(manifest.Connectors, "opencode")
		_ = install.SaveManifest(home, manifest)
	}

	return &connectors.UninstallResult{
		Connector: "opencode",
		Removed:   removed,
		Leftovers: leftovers,
		Clean:     len(leftovers) == 0,
	}, nil
}

// Validate checks that an OpenCode installation conforms to contract requirements.
func (c *Connector) Validate(projectRoot string) (*connectors.ValidationResult, error) {
	if projectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		projectRoot = cwd
	}

	opencodeDir := filepath.Join(projectRoot, ".opencode")
	result := &connectors.ValidationResult{
		Connector: "opencode",
		Valid:     true,
		Files:     []string{},
	}

	requiredFiles := []string{
		"opencode.json",
		"plugins/prumo.ts",
		"agents/prumo.md",
		"guards/tool-policy.json",
		"commands/prumo.json",
		".prumo-generated.json",
	}

	for _, rel := range requiredFiles {
		target := filepath.Join(opencodeDir, rel)
		info, err := os.Stat(target)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("missing required artifact: .opencode/%s", rel))
			continue
		}
		if info.IsDir() {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("expected file, got directory: .opencode/%s", rel))
			continue
		}
		result.Files = append(result.Files, target)
	}

	// Verify ownership marker
	ownershipPath := filepath.Join(opencodeDir, ".prumo-generated.json")
	if data, err := os.ReadFile(ownershipPath); err == nil {
		var marker map[string]any
		if err := json.Unmarshal(data, &marker); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, "invalid JSON in .prumo-generated.json")
		} else {
			if marker["managed"] != true {
				result.Valid = false
				result.Errors = append(result.Errors, ".prumo-generated.json missing managed: true")
			}
			if marker["target"] != "opencode" {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("target mismatch in marker: got %v", marker["target"]))
			}
		}
	}

	return result, nil
}

// Helper write utilities
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
func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
