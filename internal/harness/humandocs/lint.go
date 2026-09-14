// Lint (GAP-028): structural checks over a generated human-docs tree.
// Lint never judges prose quality — only presence, wiring and policy
// hygiene (no TODO(CURATED) inside GENERATED files, index rows matching
// units, reference tables well-formed).
package humandocs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LintIssue is one structural finding.
type LintIssue struct {
	Path   string `json:"path"`
	Rule   string `json:"rule"`
	Detail string `json:"detail"`
}

// Lint checks a tree produced by GenerateTree against its plan.
func Lint(root string, plan Plan) []LintIssue {
	var issues []LintIssue
	join := func(p string) string {
		if root == "" || root == "." {
			return p
		}
		return root + "/" + p
	}
	readme, err := os.ReadFile(join("README.md"))
	if err != nil || len(readme) == 0 {
		issues = append(issues, LintIssue{Path: "README.md", Rule: "readme-present", Detail: "missing or empty"})
	}
	ref, err := os.ReadFile(join("docs/reference.md"))
	if err != nil || !strings.Contains(string(ref), "|") {
		issues = append(issues, LintIssue{Path: "docs/reference.md", Rule: "reference-table", Detail: "missing or tableless"})
	}
	index, err := os.ReadFile(join("docs/INDEX.md"))
	if err != nil {
		issues = append(issues, LintIssue{Path: "docs/INDEX.md", Rule: "index-present", Detail: "missing"})
	} else {
		for _, u := range plan.Units {
			if !strings.Contains(string(index), u.ID) {
				issues = append(issues, LintIssue{Path: "docs/INDEX.md", Rule: "index-complete", Detail: "unit not indexed: " + u.ID})
			}
		}
	}
	for _, generated := range []string{join("README.md"), join("docs/reference.md")} {
		data, err := os.ReadFile(generated)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "TODO(CURATED)") {
			issues = append(issues, LintIssue{Path: filepath.Base(generated), Rule: "generated-pure", Detail: "curated marker inside GENERATED file"})
		}
	}
	return issues
}

// LintError formats issues as an error (nil when clean).
func LintError(root string, plan Plan) error {
	issues := Lint(root, plan)
	if len(issues) == 0 {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d lint issues:", len(issues))
	for _, is := range issues {
		fmt.Fprintf(&b, "\n- %s [%s] %s", is.Path, is.Rule, is.Detail)
	}
	return fmt.Errorf("%s", b.String())
}
