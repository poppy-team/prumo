package uimap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Target is one audience a projection is produced for.
//
// The targets are separate because the audiences are: a developer reads a
// navigable reference, a site reader reads pages, and a code agent reads a
// bounded instruction surface. A project that wants the map for its agent and not
// for a public site is expressing a real product decision, not a preference for
// less output.
type Target string

const (
	// TargetDeveloper is the navigable reference under the runtime directory.
	TargetDeveloper Target = "developer"
	// TargetSite is the published documentation site.
	TargetSite Target = "site"
	// TargetAgent is the bounded agent surface plus the on-demand resource.
	TargetAgent Target = "agent"
)

// Targets is the complete target vocabulary, in a stable order.
func Targets() []Target { return []Target{TargetDeveloper, TargetSite, TargetAgent} }

// DerivationModes are the ways a project can let the compiler read its code.
func DerivationModes() []string { return []string{"off", "verify-only", "fill-gaps"} }

// Config is the effective interface map configuration for one project.
type Config struct {
	Enabled bool
	// Targets are the projections to produce, in vocabulary order.
	Targets []Target
	// Derivation is one of DerivationModes. The default is `verify-only`:
	// reporting elements the map forgot is always safe, while adding rows to a
	// reviewed artifact is a change the author should opt into.
	Derivation string
	// Path is the canonical map, relative to the project root.
	Path string
	// Reason explains why the map is on or off, so a skip is never silent.
	Reason string
}

// uiCapabilities are the project capabilities that make a project UI-bearing.
// They match the applicability capabilities of the `ui.*` contracts, so "this
// project has an interface" means the same thing to the gate and to this
// compiler.
var uiCapabilities = map[string]bool{
	"ui": true, "tui": true, "desktop-gui": true, "web-application": true,
}

// uiTypes are the `project.type` values that imply an interface, for the many
// projects that never write a capability list.
var uiTypes = map[string]bool{
	"tui": true, "gui": true, "desktop": true, "web": true, "mobile": true, "desktop-gui": true,
}

// ResolveConfig decides the effective configuration.
//
// The default is deliberately asymmetric. A project with no interface gets
// nothing — the map would be empty ceremony. A project with an interface gets the
// map on, because that is where the omission costs something: an undocumented
// CLI flag is found by `--help`, and an undocumented button is found by nobody.
func ResolveConfig(root string) (Config, error) {
	cfg := Config{
		Targets:    Targets(),
		Derivation: "verify-only",
		Path:       DefaultPath(root),
	}
	raw, err := readManifest(root)
	if err != nil {
		return Config{}, err
	}
	uiCapable, why := projectIsUICapable(raw)
	section, hasSection := raw["ui"]

	// No `ui` section: follow the capability, and say so.
	if !hasSection {
		cfg.Enabled = uiCapable
		if uiCapable {
			cfg.Reason = "project declares an interface (" + why + ")"
		} else {
			cfg.Reason = "project declares no interface (" + why + ")"
		}
		return cfg, nil
	}
	ui, ok := section.(map[string]any)
	if !ok {
		return Config{}, fmt.Errorf("uimap: prumo.json `ui` must be an object")
	}
	mapSection, _ := ui["interface_map"].(map[string]any)
	if mapSection == nil {
		cfg.Enabled = uiCapable
		cfg.Reason = "no `ui.interface_map` section; following the project's declared interface (" + why + ")"
		return cfg, nil
	}

	if enabled, ok := mapSection["enabled"].(bool); ok {
		cfg.Enabled = enabled
		if !enabled {
			cfg.Reason = "disabled by `ui.interface_map.enabled`"
			cfg.Targets = nil
			return cfg, nil
		}
		cfg.Reason = "enabled by `ui.interface_map.enabled`"
	} else {
		cfg.Enabled = uiCapable
		if uiCapable {
			cfg.Reason = "no explicit switch; project declares an interface (" + why + ")"
		} else {
			cfg.Reason = "no explicit switch; project declares no interface (" + why + ")"
		}
	}

	if rawTargets, ok := mapSection["targets"].([]any); ok {
		targets := make([]Target, 0, len(rawTargets))
		for _, item := range rawTargets {
			name, _ := item.(string)
			if !isTarget(name) {
				return Config{}, fmt.Errorf("uimap: unknown `ui.interface_map.targets` entry %q (want %v)",
					name, targetNames())
			}
			targets = append(targets, Target(name))
		}
		if len(targets) == 0 {
			// Enabled but projecting nowhere is a configuration that produces
			// nothing while claiming to produce something. Refuse it rather
			// than surprise the reader with an empty directory.
			return Config{}, fmt.Errorf("uimap: `ui.interface_map.targets` is empty; disable the map instead of asking for no output")
		}
		cfg.Targets = orderTargets(targets)
	}

	if mode, ok := mapSection["derivation"].(string); ok {
		if !contains(DerivationModes(), mode) {
			return Config{}, fmt.Errorf("uimap: `ui.interface_map.derivation` is %q (want %v)", mode, DerivationModes())
		}
		cfg.Derivation = mode
	}
	if path, ok := mapSection["path"].(string); ok && strings.TrimSpace(path) != "" {
		if filepath.IsAbs(path) {
			cfg.Path = path
		} else {
			cfg.Path = filepath.Join(root, path)
		}
	}
	return cfg, nil
}

// Wants reports whether a target is enabled.
func (c Config) Wants(target Target) bool {
	for _, t := range c.Targets {
		if t == target {
			return true
		}
	}
	return false
}

func readManifest(root string) (map[string]any, error) {
	data, err := os.ReadFile(filepath.Join(root, "prumo.json"))
	if err != nil {
		// A missing manifest is not this package's problem to report: the
		// caller already knows whether it has a project.
		return map[string]any{}, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("uimap: parse prumo.json: %w", err)
	}
	return raw, nil
}

// projectIsUICapable reports whether the manifest declares an interface, and the
// evidence for it. The evidence is returned so a skip can say why instead of
// leaving the reader to grep the manifest.
func projectIsUICapable(raw map[string]any) (bool, string) {
	if features, ok := raw["features"].([]any); ok {
		for _, item := range features {
			name, _ := item.(string)
			if uiCapabilities[name] {
				return true, "features: " + name
			}
		}
	}
	if project, ok := raw["project"].(map[string]any); ok {
		if types, ok := project["type"].([]any); ok {
			for _, item := range types {
				name, _ := item.(string)
				if uiTypes[name] {
					return true, "project.type: " + name
				}
			}
		}
	}
	return false, "no ui capability in features or project.type"
}

func isTarget(name string) bool {
	for _, t := range Targets() {
		if string(t) == name {
			return true
		}
	}
	return false
}

func targetNames() []string {
	out := make([]string, 0, len(Targets()))
	for _, t := range Targets() {
		out = append(out, string(t))
	}
	return out
}

// orderTargets returns the requested targets in vocabulary order and without
// duplicates, so two configurations describing the same set produce byte-equal
// output.
func orderTargets(requested []Target) []Target {
	want := map[Target]bool{}
	for _, t := range requested {
		want[t] = true
	}
	out := make([]Target, 0, len(want))
	for _, t := range Targets() {
		if want[t] {
			out = append(out, t)
		}
	}
	return out
}

// ConfigSummary is the one-line description of an effective configuration, for
// reports: a skip must print its reason, and an on-map must print its scope.
func (c Config) ConfigSummary() string {
	if !c.Enabled {
		return "interface map: off (" + c.Reason + ")"
	}
	names := make([]string, 0, len(c.Targets))
	for _, t := range c.Targets {
		names = append(names, string(t))
	}
	sort.Strings(names)
	return fmt.Sprintf("interface map: on · targets %s · derivation %s · %s",
		strings.Join(names, "+"), c.Derivation, c.Reason)
}
