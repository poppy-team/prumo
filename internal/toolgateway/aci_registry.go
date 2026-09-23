package toolgateway

import "github.com/raillen/prumo/internal/harness/aci"

// RegisterACI adds the native coding tools to a registry, deriving each
// descriptor from the catalogue that implements it.
//
// The registry used to carry a hand-written list beside the executor, and the
// two drifted: four descriptors — read_file, write_file, exec_command,
// git_commit — were registered with kinds, scopes, timeouts and output limits
// and had no implementation behind them at all. `prumo tool list` therefore
// advertised capabilities that could not be invoked (GAP-158).
//
// Deriving the descriptors removes the possibility rather than closing the four
// instances: a tool that is not in the catalogue cannot be registered, so the
// registry cannot describe something that does not exist. The mapping is by
// name and a mismatch is a programming error, so it panics at startup rather
// than shipping a silent omission.
func RegisterACI(r *Registry) {
	trusted := map[string]bool{
		"process.exec": true,
		"test.run":     true,
	}
	capabilities := map[string][]string{
		"fs.read":          {"fs", "read"},
		"fs.list":          {"fs", "read"},
		"fs.search":        {"fs", "read"},
		"code.symbols":     {"code", "symbols"},
		"code.diagnostics": {"code", "diagnostics"},
		"edit.patch":       {"fs", "write"},
		"edit.create":      {"fs", "write"},
		"edit.delete":      {"fs", "write"},
		"edit.move":        {"fs", "write"},
		"process.exec":     {"exec", "shell"},
		"test.run":         {"test"},
		"git.status":       {"scm", "git"},
		"git.diff":         {"scm", "git"},
	}
	for _, tool := range aci.Catalog() {
		caps, ok := capabilities[tool.Name]
		if !ok {
			// A catalogue entry with no registry mapping would be a tool the
			// registry does not describe — the exact drift this function exists to
			// prevent, so it is loud rather than silent.
			panic("toolgateway: native tool " + tool.Name + " has no registry mapping")
		}
		trust := "core"
		if trusted[tool.Name] {
			trust = "trusted"
		}
		r.Register(Descriptor{
			ID:              tool.Name,
			Version:         1,
			Description:     tool.Description,
			Kind:            Kind(tool.Kind),
			Trust:           trust,
			Capabilities:    caps,
			FilesystemScope: []string{"project-root"},
			TimeoutMS:       timeoutFor(tool),
			OutputLimit:     outputLimitFor(tool),
		})
	}
}

func timeoutFor(tool aci.Tool) int {
	switch tool.Kind {
	case "destructive":
		return 10000
	case "side-effecting":
		return 60000
	default:
		return 15000
	}
}

func outputLimitFor(tool aci.Tool) int {
	switch tool.Name {
	case "process.exec", "test.run":
		return 204800
	case "fs.read", "fs.search", "code.symbols":
		return 102400
	default:
		return 51200
	}
}
