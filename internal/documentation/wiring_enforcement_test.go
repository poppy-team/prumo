package docengine

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The 2026-09-23 audit found seven gaps marked done whose implementation was a
// library with no production caller. The rule adopted there is that "done"
// requires a production caller, not a green test. This file makes that rule
// executable for the components where it was violated, so the same drift cannot
// come back unnoticed.
//
// The check is deliberately a source scan rather than a coverage measurement:
// the failure was never that the code was untested, it was that nothing outside
// the tests called it.

// repoRootFrom walks up from the test's working directory to the module root.
func repoRootFrom(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("module root not found from test working directory")
	return ""
}

// hasProductionImporter reports whether any non-test Go file in the module
// imports wantPkg. The match is the quoted import path, not a bare substring:
// package paths are named in prose and comments all over this repository, and a
// substring match reports a comment as wiring.
//
// Files under prumo-viewer and prumo-tui are skipped: the first is not Go and
// the second is a separate module with its own dependency set, so neither is
// evidence about this module's wiring.
func hasProductionImporter(t *testing.T, root, wantPkg string) bool {
	t.Helper()
	found := false
	quoted := `"` + wantPkg + `"`
	importBlock := regexp.MustCompile(`(?s)import\s*\((.*?)\)`)
	singleImport := regexp.MustCompile(`(?m)^import\s+(?:[\w.]+\s+)?"([^"]+)"`)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "prumo-viewer" || name == "prumo-tui" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		source := string(data)
		if !strings.Contains(source, quoted) {
			return nil
		}
		if block := importBlock.FindStringSubmatch(source); block != nil {
			if strings.Contains(block[1], quoted) {
				found = true
				return filepath.SkipDir
			}
		}
		if single := singleImport.FindStringSubmatch(source); single != nil && single[1] == wantPkg {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return found
}

// unwiredComponents lists the packages that were implemented and tested but had
// no production caller at the time of the audit, together with the gap that
// records the decision to wire them.
var unwiredComponents = []struct {
	pkg      string
	gap      string
	wave     string
	caller   string
	evidence string
}{
	{
		pkg:      "internal/harness/gateway",
		gap:      "GAP-102",
		wave:     "4",
		caller:   "the agent runner must select a provider through the gateway",
		evidence: "docs/harness/gap-register.md",
	},
	{
		pkg:      "internal/harness/team",
		gap:      "GAP-101",
		wave:     "5",
		caller:   "the CLI or daemon must create a nested run per delegated role",
		evidence: "docs/harness/gap-register.md",
	},
	{
		pkg:      "internal/runtime/failure",
		gap:      "GAP-116",
		wave:     "3",
		caller:   "tool execution must classify failures with this taxonomy",
		evidence: "docs/harness/gap-register.md",
	},
	{
		pkg:      "internal/runtime/journal",
		gap:      "GAP-126",
		wave:     "3",
		caller:   "the tool-call lifecycle must record side-effect intent here",
		evidence: "docs/harness/gap-register.md",
	},
	{
		pkg:      "internal/environment",
		gap:      "GAP-110",
		wave:     "1",
		caller:   "the executor must run tools through this environment port",
		evidence: "docs/harness/gap-register.md",
	},
	{
		pkg:      "internal/harness/directive",
		gap:      "GAP-127",
		wave:     "3",
		caller:   "the daemon must construct runners with a compiled directive",
		evidence: "docs/harness/gap-register.md",
	},
	{
		pkg:      "internal/decision",
		gap:      "GAP-102",
		wave:     "4",
		caller:   "routing must consult the decision chain",
		evidence: "docs/harness/gap-register.md",
	},
	{
		pkg:      "internal/automation",
		gap:      "GAP-134",
		wave:     "5",
		caller:   "a plan must execute recipes through this engine",
		evidence: "docs/harness/gap-register.md",
	},
}

// TestUnwiredComponentsAreTrackedAsPartial fails when a component the audit
// found unwired gains a production caller without the gap register being updated
// to say so. That is the safe direction to fail: someone wired it and the
// governance record is now stale.
func TestUnwiredComponentsAreTrackedAsPartial(t *testing.T) {
	root := repoRootFrom(t)
	registerPath := filepath.Join(root, "docs", "harness", "gap-register.md")
	register, err := os.ReadFile(registerPath)
	if err != nil {
		t.Fatalf("read gap register: %v", err)
	}
	text := string(register)

	for _, component := range unwiredComponents {
		component := component
		t.Run(component.pkg, func(t *testing.T) {
			wired := hasProductionImporter(t, root, component.pkg)
			mentionsGap := strings.Contains(text, component.gap)

			if !mentionsGap {
				t.Fatalf(
					"%s is not referenced by the gap register, so its wiring state is untracked. "+
						"Every package audited on 2026-09-23 must have a gap that states whether it is "+
						"in the production path. Add the row (or point this test at the right gap id).",
					component.pkg,
				)
			}

			if wired {
				// A caller now exists. The register must not still be asserting the
				// component is unwired, otherwise the record is stale.
				t.Fatalf(
					"%s now has a production caller (%s), so %s must be updated: the register still "+
						"carries the pre-wiring state. Update the row's status and evidence before "+
						"removing this component from the list.",
					component.pkg,
					component.caller,
					component.gap,
				)
			}
		})
	}
}

// TestGapRegisterAuditSectionExists keeps the audit's own record present. The
// section is what turns "a library exists" into "here is what is missing", and
// losing it would be the first step back to an untracked claim.
func TestGapRegisterAuditSectionExists(t *testing.T) {
	root := repoRootFrom(t)
	register, err := os.ReadFile(filepath.Join(root, "docs", "harness", "gap-register.md"))
	if err != nil {
		t.Fatalf("read gap register: %v", err)
	}
	text := string(register)

	required := []string{
		"## 1.1 Auditoria de implementação — 2026-09-23",
		"`✅ done` exige **caller de",
		"GAP-097",
		"GAP-170",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Errorf("gap register is missing the audit's own record: %q", needle)
		}
	}
}

// TestArchitectureDecisionsAreRecorded keeps the four decisions the audit
// forced to exist discoverable from the register, so a reader who starts at the
// gaps finds the reasoning without having to know the ADR numbers.
func TestArchitectureDecisionsAreRecorded(t *testing.T) {
	root := repoRootFrom(t)
	adrs := map[string]string{
		"017-single-level-subagent-delegation.md": "subagent delegation",
		"018-global-daemon.md":                    "global daemon",
		"019-tool-call-lifecycle.md":              "tool-call lifecycle",
		"020-quota-aware-model-routing.md":        "quota-aware routing",
	}
	for file, what := range adrs {
		path := filepath.Join(root, "docs", "adr", file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("ADR for %s is missing: %v", what, err)
			continue
		}
		content := string(data)
		if !strings.Contains(content, "Status: Accepted") {
			t.Errorf("%s does not declare an accepted status", file)
		}
		if !strings.Contains(content, "## Consequences") {
			t.Errorf("%s records a decision without its consequences", file)
		}
	}
}
