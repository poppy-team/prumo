package main

import (
	"strings"
	"testing"
)

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

// TestHelpAdvertisesOnlyRealSubcommands is the regression test for the finding
// that `prumo help plan` advertised init/show/update and `prumo help repo`
// advertised branch-check/commit-check/pr-check — none of which the parser
// accepts. A user who follows the help text gets an "unknown subcommand" error,
// so the advertised vocabulary must be derived from the same list the parsers
// switch on rather than maintained by hand.
func TestHelpAdvertisesOnlyRealSubcommands(t *testing.T) {
	realPlan := []string{"status", "questions", "resume", "answer", "decisions", "delta", "blueprint"}
	plan, ok := commandRegistry["plan"]
	if !ok {
		t.Fatal("plan has no help entry")
	}
	advertised := map[string]bool{}
	for _, line := range plan.Subcommands {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		advertised[fields[0]] = true
	}
	for _, name := range realPlan {
		if !advertised[name] {
			t.Errorf("plan help does not advertise the real subcommand %q", name)
		}
		delete(advertised, name)
	}
	for name := range advertised {
		t.Errorf("plan help advertises %q but the parser does not implement it", name)
	}

	// repo has exactly one verb, and it requires a subcommand.
	repo, ok := commandRegistry["repo"]
	if !ok {
		t.Fatal("repo has no help entry")
	}
	for _, verb := range []string{"branch-check", "commit-check", "pr-check"} {
		for _, line := range repo.Subcommands {
			if strings.HasPrefix(line, verb) {
				t.Errorf("repo help advertises %q but `prumo repo %s` is not implemented", verb, verb)
			}
		}
	}
}
