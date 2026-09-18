package main

import "testing"

// TestEveryCommandCategoryIsPrinted guards the defect that hid `agent` from
// `prumo help`: the renderer walks a hand-maintained order list, and a category
// absent from it is silently dropped. A command nobody can discover from the
// top-level help may as well not exist, so the registry and the renderer must
// agree.
func TestEveryCommandCategoryIsPrinted(t *testing.T) {
	printed := map[string]bool{}
	for _, category := range helpCategoryOrder {
		printed[category] = true
	}
	for name, info := range commandRegistry {
		if !printed[info.Category] {
			t.Errorf("command %q is registered under category %q, which `prumo help` never prints",
				name, info.Category)
		}
	}
}

// TestRegisteredCommandsHaveHelp asserts each command is documented well enough
// to be usable: a usage line and at least one example.
func TestRegisteredCommandsHaveHelp(t *testing.T) {
	for name, info := range commandRegistry {
		if info.Name != name {
			t.Errorf("registry key %q does not match Name %q", name, info.Name)
		}
		if info.Summary == "" || info.Usage == "" || info.Description == "" {
			t.Errorf("command %q has incomplete help", name)
		}
		if len(info.Examples) == 0 {
			t.Errorf("command %q has no example", name)
		}
	}
}
