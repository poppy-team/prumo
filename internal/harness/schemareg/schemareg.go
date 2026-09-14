// Schema registry (GAP-029): versioned inventory of machine contracts with
// a migration framework. Today every contract is v1 and migration is the
// identity — the framework exists so v2 can land with tests, not surprises.
package schemareg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Entry pins one contract's version.
type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version int    `json:"version"`
}

// Known lists the harness-owned contracts and their versions.
func Known() []Entry {
	return []Entry{
		{Name: "harness-checkpoint", Path: "schemas/harness-checkpoint.schema.json", Version: 1},
		{Name: "harness-handoff", Path: "schemas/harness-handoff.schema.json", Version: 1},
		{Name: "agent-event", Path: "schemas/agent-event.schema.json", Version: 1},
		{Name: "protocol-manifest", Path: "schemas/protocol-manifest.json", Version: 1},
	}
}

// ValidateAll parses every known schema under root.
func ValidateAll(root string) error {
	entries := Known()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(root, e.Path))
		if err != nil {
			return fmt.Errorf("schema %s: %w", e.Name, err)
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("schema %s invalid JSON: %w", e.Name, err)
		}
	}
	return nil
}

// Migrate upgrades a versioned document payload toward target. v1 is
// current for all known contracts, so migration is the identity; unknown
// versions fail loudly instead of silently passing through.
func Migrate(name string, fromVersion int, doc map[string]any) (map[string]any, error) {
	known := -1
	for _, e := range Known() {
		if e.Name == name {
			known = e.Version
		}
	}
	if known < 0 {
		return nil, fmt.Errorf("unknown contract %q", name)
	}
	if fromVersion == known {
		return doc, nil
	}
	if fromVersion > known {
		return nil, fmt.Errorf("contract %s v%d newer than registry v%d", name, fromVersion, known)
	}
	return nil, fmt.Errorf("no migration path for %s v%d → v%d", name, fromVersion, known)
}
