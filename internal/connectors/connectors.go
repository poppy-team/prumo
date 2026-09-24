package connectors

import (
	"fmt"
	"sort"
	"sync"
)

// Standard capability identifiers per trust-model and connector specification.
const (
	CapAdvise           = "advise"
	CapRestrictTools    = "restrict_tools"
	CapPreToolBlock     = "pre_tool_block"
	CapPostToolVerify   = "post_tool_verify"
	CapIsolateSubagents = "isolate_subagents"
	CapSessionHooks     = "session_hooks"
	CapNativePlugin     = "native_plugin"
	CapCommands         = "commands"
	CapSubagents        = "subagents"
)

// Standard enforcement levels.
const (
	EnforcementAdvisory = "advisory"
	EnforcementStandard = "standard"
	EnforcementStrict   = "strict"
)

// Standard lifecycle hooks.
const (
	HookSessionStart = "session.start"
	HookSessionEnd   = "session.end"
	HookToolBefore   = "tool.before_execute"
	HookToolAfter    = "tool.after_execute"
)

// Contract defines machine-readable capabilities and guarantees of a connector harness.
type Contract struct {
	ID            string         `json:"id"`
	Version       string         `json:"version"`
	ProtocolRange string         `json:"protocol_range"`
	Capabilities  []string       `json:"capabilities"`
	Enforcement   string         `json:"enforcement"`
	Hooks         []string       `json:"hooks,omitempty"`
	Install       map[string]any `json:"install,omitempty"`
	Cleanup       map[string]any `json:"cleanup,omitempty"`
}

// Supports checks if the contract supports a given capability.
func (c Contract) Supports(capability string) bool {
	for _, v := range c.Capabilities {
		if v == capability {
			return true
		}
	}
	return false
}

// CompileOptions configures connector compilation.
type CompileOptions struct {
	RepoRoot     string   `json:"repo_root,omitempty"`
	Target       string   `json:"target,omitempty"`
	Force        bool     `json:"force,omitempty"`
	Agents       []string `json:"agents,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	ToolGuards   bool     `json:"tool_guards,omitempty"`
	SessionHooks bool     `json:"session_hooks,omitempty"`
}

// CompileResult represents the output of compiling a connector harness.
type CompileResult struct {
	Target       string   `json:"target"`
	CreatedPaths []string `json:"created_paths"`
	// PreservedPaths are files that already existed, are not ours, and were left
	// untouched. Reported rather than silently dropped, for the same reason as on
	// InstallResult: an install that quietly skipped your AGENTS.md looks
	// identical to one that wrote it (GAP-141).
	PreservedPaths []string       `json:"preserved_paths,omitempty"`
	ManifestPath   string         `json:"manifest_path"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// InstallOptions configures installation.
type InstallOptions struct {
	Scope    string `json:"scope,omitempty"` // "project" or "global"
	DryRun   bool   `json:"dry_run,omitempty"`
	Force    bool   `json:"force,omitempty"`
	RepoRoot string `json:"repo_root,omitempty"`
}

// InstallResult reports results of connector installation.
type InstallResult struct {
	Connector    string   `json:"connector"`
	Status       string   `json:"status"`
	Scope        string   `json:"scope"`
	CreatedPaths []string `json:"created_paths"`
	// PreservedPaths are files that already existed, are not ours, and were left
	// untouched. They are reported rather than silently dropped: an install that
	// quietly skipped your AGENTS.md looks identical to one that wrote it, and the
	// only way to know your instructions are still there is to say so (GAP-141).
	PreservedPaths []string `json:"preserved_paths,omitempty"`
	CleanupPath    string   `json:"cleanup_path"`
	Contract       Contract `json:"contract"`
}

// UninstallOptions configures uninstallation.
type UninstallOptions struct {
	Scope string `json:"scope,omitempty"`
	Force bool   `json:"force,omitempty"`
}

// UninstallResult reports results of connector uninstallation.
type UninstallResult struct {
	Connector string   `json:"connector"`
	Removed   []string `json:"removed"`
	Leftovers []string `json:"leftovers"`
	Clean     bool     `json:"clean"`
}

// ValidationResult reports validation findings for a connector installation.
type ValidationResult struct {
	Connector string   `json:"connector"`
	Valid     bool     `json:"valid"`
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	Files     []string `json:"files,omitempty"`
}

// Connector interface defines the lifecycle for an Prumo connector harness.
type Connector interface {
	ID() string
	Name() string
	Contract() Contract
	Compile(projectRoot string, opts CompileOptions) (*CompileResult, error)
	Install(home string, projectRoot string, opts InstallOptions) (*InstallResult, error)
	Uninstall(home string, projectRoot string, opts UninstallOptions) (*UninstallResult, error)
	Validate(projectRoot string) (*ValidationResult, error)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Connector{}
)

// Register registers a connector implementation.
func Register(c Connector) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[c.ID()] = c
}

// Get retrieves a connector by ID.
func Get(id string) (Connector, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	c, ok := registry[id]
	if !ok {
		return nil, fmt.Errorf("unknown connector: %q", id)
	}
	return c, nil
}

// List returns all registered connectors sorted by ID.
func List() []Connector {
	registryMu.RLock()
	defer registryMu.RUnlock()
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Connector, 0, len(ids))
	for _, id := range ids {
		out = append(out, registry[id])
	}
	return out
}

// ResetRegistry clears the registry (primarily for tests).
func ResetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]Connector{}
}
