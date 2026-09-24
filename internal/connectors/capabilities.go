package connectors

import (
	"path/filepath"
	"sort"
	"strings"
)

// DeriveImplemented reports the capabilities a compilation can prove from the
// files it wrote and the config it built.
//
// The capability list in a Contract is what a caller plans against. Maintaining
// it by hand next to the generator is how a connector comes to say it can verify
// after a tool ran, or isolate subagents, while writing nothing that does either
// (GAP-142). Deriving it makes the declaration checkable, and
// TestEveryConnectorDeclaresWhatItGenerates makes it checked.
//
// Evidence, by capability:
//
//   - a subagents directory or a subagent file    -> subagents, isolate_subagents
//   - a commands file                             -> commands
//   - a skills directory or a skill file          -> skills
//   - a tool policy or guard file                 -> advise, restrict_tools
//   - on_tool_before / on_tool_after in the config -> pre_tool_block, post_tool_verify
//   - session hooks in the config                 -> session_hooks
//   - a plugin file                               -> native_plugin
//
// isolate_subagents is derived from the same subagent artifact as subagents,
// because that is what a connector writes when it separates a subagent from the
// main agent: a subagents directory is the separation. A connector that writes
// subagents without separating them should stop writing the directory.
func DeriveImplemented(config map[string]any, created []string) []string {
	found := map[string]bool{}
	plugins := 0
	for _, path := range created {
		normalised := filepath.ToSlash(path)
		switch {
		case strings.Contains(normalised, "/agents/") || strings.Contains(normalised, "/subagents/"):
			found[CapSubagents] = true
			found[CapIsolateSubagents] = true
		case strings.Contains(normalised, "/commands/") || strings.Contains(normalised, "commands.json"):
			found[CapCommands] = true
		case strings.Contains(normalised, "/skills/"):
			found[CapSkills] = true
		case strings.Contains(normalised, "/guards/") || strings.Contains(normalised, "tool-policy"):
			found[CapAdvise] = true
			found[CapRestrictTools] = true
		}
		plugins += countPluginFiles(path)
	}
	if plugins > 0 {
		found[CapNativePlugin] = true
	}

	// The config is the second source: a hook capability is delivered by the hook
	// being enabled in what was written, not by a name appearing anywhere.
	switch hooks := config["hooks"].(type) {
	case map[string]bool:
		if hooks["on_tool_before"] {
			found[CapPreToolBlock] = true
		}
		if hooks["on_tool_after"] {
			found[CapPostToolVerify] = true
		}
		if hooks["on_session_start"] || hooks["on_session_end"] {
			found[CapSessionHooks] = true
		}
	case map[string]any:
		if truthy(hooks["on_tool_before"]) {
			found[CapPreToolBlock] = true
		}
		if truthy(hooks["on_tool_after"]) {
			found[CapPostToolVerify] = true
		}
		if truthy(hooks["on_session_start"]) || truthy(hooks["on_session_end"]) {
			found[CapSessionHooks] = true
		}
	}

	out := make([]string, 0, len(found))
	for capability := range found {
		out = append(out, capability)
	}
	sort.Strings(out)
	return out
}

// countPluginFiles reports whether a written path is a target-native plugin.
func countPluginFiles(path string) int {
	base := filepath.Base(path)
	switch base {
	case "prumo.ts", "prumo.js":
		return 1
	default:
		return 0
	}
}

// truthy reads a value a generated config may have written as a bool, a string
// or a map, and reports whether it is enabled.
func truthy(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false" && v != "off"
	case map[string]any:
		enabled, _ := v["enabled"].(bool)
		return enabled
	default:
		return false
	}
}

// DeriveImplementedHooks reports which hooks a generated hooks file wires up.
//
// A connector that declares HookToolAfter and writes no entry for it tells the
// target that a post-tool check will run, and it will not. The hooks file is
// the only place a hook can actually be (GAP-142).
func DeriveImplementedHooks(hooksFile map[string]any) []string {
	out := []string{}
	inner, _ := hooksFile["hooks"].(map[string]any)
	if inner == nil {
		// A file that is itself the hook map, with no wrapper.
		inner = hooksFile
	}
	for name, value := range inner {
		if name == "version" {
			continue
		}
		if hookHasWork(value) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// hookHasWork reports whether a hook entry actually does something. An entry
// with an empty list is a declared hook with no behaviour behind it.
func hookHasWork(value any) bool {
	switch v := value.(type) {
	case []any:
		return len(v) > 0
	case []map[string]string:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	case nil:
		return false
	default:
		return true
	}
}

// DeriveImplementedHooksFromConfig reports the hooks a target config enables.
//
// A connector that puts its hooks in the main config rather than a separate
// hooks file has declared them in exactly the same way, and checking only for a
// hooks file would report it as wiring none (GAP-142).
//
// The config uses short names (on_session_start) while the contract uses the
// canonical names (session.start), so the two vocabularies are mapped rather
// than compared: a target that spells its hooks differently still has them.
func DeriveImplementedHooksFromConfig(config map[string]any) []string {
	out := []string{}
	enabled := func(name string) bool {
		switch hooks := config["hooks"].(type) {
		case map[string]bool:
			return hooks[name]
		case map[string]any:
			return truthy(hooks[name])
		default:
			return false
		}
	}
	if enabled("on_session_start") {
		out = append(out, HookSessionStart)
	}
	if enabled("on_session_end") {
		out = append(out, HookSessionEnd)
	}
	if enabled("on_tool_before") {
		out = append(out, HookToolBefore)
	}
	if enabled("on_tool_after") {
		out = append(out, HookToolAfter)
	}
	sort.Strings(out)
	return out
}
