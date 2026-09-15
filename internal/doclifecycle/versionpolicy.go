package doclifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/protocol"
)

// VersionPolicy declares which product versions documentation may reference
// (W20.6). Buckets are exclusive: a version is current, supported, deprecated
// or removed, never two of them.
//
// The point of declaring it is that "a current document must not reference a
// removed version" becomes checkable instead of a review habit.
type VersionPolicy struct {
	Current    string   `json:"current"`
	Supported  []string `json:"supported,omitempty"`
	Deprecated []string `json:"deprecated,omitempty"`
	Removed    []string `json:"removed,omitempty"`
	// HistoricalExempt documents are archived records: they may reference
	// versions that are no longer supported because they are a record of what
	// was true at the time.
	HistoricalExempt bool `json:"historical_exempt,omitempty"`
}

// versionLine matches a version reference in prose, e.g. v0.4 or v0.6.0. A
// leading v is required so that unrelated decimals and identifiers (W16.9,
// §3.1) are never read as releases. Go's regexp has no lookahead, so the
// optional patch component is captured and discarded instead.
var versionLine = regexp.MustCompile(`\bv(\d+)\.(\d+)(\.\d+)?`)

// releaseLines returns the product release lines a piece of prose references.
// A reference with a patch component (`v0.1.0`) is a component version — here,
// the negotiation kernel — not a product release, so it is not returned. The
// authority gate applies the same major.minor convention to version drift.
func releaseLines(line string) []string {
	lines := []string{}
	for _, match := range versionLine.FindAllStringSubmatch(line, -1) {
		if match[3] != "" {
			continue
		}
		lines = append(lines, match[1]+"."+match[2])
	}
	return lines
}

// historicalAnnotations mark a line as narrating the past rather than claiming
// current availability: a migration note that says Python v0.3 was retired is
// the record of a removal, not a document offering v0.3. The vocabulary
// deliberately matches the authority gate's drift exemption, so a line cannot
// be historical in one check and a violation in the other. An annotation must
// be on the same line as the reference; a paragraph-level exemption would hide
// the claims the check exists to find.
var historicalAnnotations = regexp.MustCompile(`(?i)historical|legacy|migrat|retired|superseded|deprecated|no longer|oracle|compatib|frozen|freeze|retir`)

// versionNumber parses a declared version value, where the leading v is
// optional (the policy writes "0.4"; prose writes "v0.4").
var versionNumber = regexp.MustCompile(`^\s*v?(\d+)\.(\d+)`)

// minorOf normalizes a version to its major.minor pair, which is the granularity
// documentation is written at. It returns "" for anything that is not a
// version, so a malformed policy entry is reported rather than silently bucketed.
func minorOf(version string) string {
	match := versionNumber.FindStringSubmatch(version)
	if match == nil {
		return ""
	}
	return match[1] + "." + match[2]
}

// ValidateVersionPolicy checks the policy for internal consistency.
func ValidateVersionPolicy(policy VersionPolicy) []Finding {
	findings := []Finding{}
	if strings.TrimSpace(policy.Current) == "" {
		findings = append(findings, Finding{Kind: "version-policy-incomplete", Target: LifecyclePath,
			Detail: "the policy must declare a current version"})
		return findings
	}
	buckets := []struct {
		name     string
		versions []string
	}{
		{"supported", policy.Supported},
		{"deprecated", policy.Deprecated},
		{"removed", policy.Removed},
	}
	owner := map[string]string{}
	current := minorOf(policy.Current)
	if current == "" {
		findings = append(findings, Finding{Kind: "version-policy-invalid", Target: policy.Current,
			Detail: "the current version is not a version number"})
	}
	owner[current] = "current"
	for _, bucket := range buckets {
		for _, version := range bucket.versions {
			minor := minorOf(version)
			if minor == "" {
				findings = append(findings, Finding{Kind: "version-policy-invalid", Target: version,
					Detail: "not a version number"})
				continue
			}
			if other, exists := owner[minor]; exists {
				findings = append(findings, Finding{Kind: "version-policy-overlap", Target: version,
					Detail: fmt.Sprintf("version %s is declared in both %q and %q", minor, other, bucket.name)})
				continue
			}
			owner[minor] = bucket.name
		}
	}
	sortFindings(findings)
	return findings
}

// CheckVersionPolicy validates the policy and checks the released version and
// the repository documents against it.
func CheckVersionPolicy(root string) (VersionPolicyReport, error) {
	lifecycle, err := LoadLifecycle(root)
	if err != nil {
		return VersionPolicyReport{}, err
	}
	policy := lifecycle.VersionPolicy
	report := VersionPolicyReport{Policy: policy, Findings: ValidateVersionPolicy(policy)}
	if len(report.Findings) > 0 {
		sortFindings(report.Findings)
		report.OK = false
		return report, nil
	}

	// The policy and the build must agree: a policy that names an older version
	// than the CLI announces is drift, not documentation.
	if minorOf(policy.Current) != minorOf(protocol.CLIVersion) {
		report.Findings = append(report.Findings, Finding{Kind: "version-policy-drift", Target: LifecyclePath,
			Detail: fmt.Sprintf("policy says the current version is %s but the release is %s", policy.Current, protocol.CLIVersion)})
	}

	removed := map[string]bool{}
	for _, version := range policy.Removed {
		removed[minorOf(version)] = true
	}
	documents, err := repositoryDocuments(root)
	if err != nil {
		return VersionPolicyReport{}, err
	}
	for _, rel := range documents {
		if policy.HistoricalExempt && isHistorical(root, rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, line := range strings.Split(string(data), "\n") {
			if historicalAnnotations.MatchString(line) {
				continue
			}
			for _, minor := range releaseLines(line) {
				if !removed[minor] || seen[minor] {
					continue
				}
				seen[minor] = true
				report.Findings = append(report.Findings, Finding{Kind: "removed-version-reference", Target: rel,
					Detail: "current document references removed version " + minor + ": " + strings.TrimSpace(line)})
			}
		}
	}
	sortFindings(report.Findings)
	report.OK = len(report.Findings) == 0
	return report, nil
}

// VersionPolicyReport is the operator-facing result.
type VersionPolicyReport struct {
	Policy   VersionPolicy `json:"policy"`
	Findings []Finding     `json:"findings"`
	OK       bool          `json:"ok"`
}

// repositoryDocuments lists the Markdown documents a version policy applies to:
// the tracked root documents plus everything under docs/.
func repositoryDocuments(root string) ([]string, error) {
	files := []string{}
	for _, name := range []string{"README.md", "AGENTS.md", "ENTRYPOINT.md", "FRAMEWORK.md", "CONTRIBUTING.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			files = append(files, name)
		}
	}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// isHistorical reports whether the authority map classifies a document as an
// archived record. Classification is delegated to the authority gate so the
// lifecycle checks agree with it; a missing map classifies nothing, so nothing
// is exempt.
func isHistorical(root, rel string) bool {
	role, ok := docengine.RoleOf(root, rel)
	return ok && role == docengine.RoleHistorical
}
