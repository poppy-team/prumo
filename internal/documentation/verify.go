// Continuous documentation verification (W19). Only deterministic checks are
// implemented here: they can fail a build, and a model critic may never
// override them.
package docengine

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// VerifyFinding is one deterministic verification result.
type VerifyFinding struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Detail string `json:"detail"`
	Line   int    `json:"line,omitempty"`
}

// VerifyReport is the deterministic verification result.
type VerifyReport struct {
	Files    int             `json:"files"`
	Checks   []string        `json:"checks"`
	Findings []VerifyFinding `json:"findings"`
	OK       bool            `json:"ok"`
}

// VerifyChecks lists the deterministic checks by name, so a report can state
// exactly what ran.
var VerifyChecks = []string{"authority", "managed-regions", "documentation-links", "claim-drift"}

var (
	mdLinkPattern     = regexp.MustCompile(`\]\(([^)\s]+\.md)(?:#[^)]*)?\)`)
	futureClaims      = regexp.MustCompile(`(?i)\b(?:future work|future increment|not yet implemented|remains planned|is planned|will be implemented|deferred)\b`)
	inlineCodePattern = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_.]{3,})`")
)

// VerifyDocs runs the deterministic documentation checks.
func VerifyDocs(root string) (VerifyReport, error) {
	report := VerifyReport{Checks: append([]string{}, VerifyChecks...), Findings: []VerifyFinding{}}
	authority, err := CheckAuthority(root)
	if err != nil {
		return VerifyReport{}, err
	}
	for _, f := range authority.Findings {
		report.Findings = append(report.Findings, VerifyFinding{Kind: "authority:" + f.Kind, Path: f.Path, Detail: f.Detail})
	}
	files, err := verificationFiles(root)
	if err != nil {
		return VerifyReport{}, err
	}
	report.Files = len(files)
	implemented := implementationWords(root)
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		content := string(data)
		report.Findings = append(report.Findings, checkLinks(root, rel, content)...)
		report.Findings = append(report.Findings, checkRegions(rel, content)...)
		report.Findings = append(report.Findings, checkClaimDrift(rel, content, implemented)...)
	}
	sortFindings(report.Findings)
	report.OK = len(report.Findings) == 0
	return report, nil
}

// VerifyDocsStrict is `prumo docs verify --strict` (W19.9). Beyond the
// deterministic checks it requires every applicable contract to be semantically
// verified, so a completion gate cannot pass on lexical coverage alone — the
// false-green path W15 removed from readiness. Deterministic findings are still
// reported first and can never be overridden by the semantic layer.
func VerifyDocsStrict(root string) (VerifyReport, error) {
	report, err := VerifyDocs(root)
	if err != nil {
		return VerifyReport{}, err
	}
	report.Checks = append(report.Checks, "semantic-readiness")
	readiness, err := Readiness(root, "")
	if err != nil {
		return VerifyReport{}, err
	}
	unverified := map[string]bool{}
	for _, contract := range append(append([]string{}, readiness.BlockingContracts...), readiness.UnverifiedContracts...) {
		if unverified[contract] {
			continue
		}
		unverified[contract] = true
		report.Findings = append(report.Findings, VerifyFinding{
			Kind: "semantic-unverified", Path: contract,
			Detail: "contract is not semantically verified: bind requirement → claim → evidence (W15)",
		})
	}
	sortFindings(report.Findings)
	report.OK = len(report.Findings) == 0
	return report, nil
}

// sortFindings keeps every report in a stable, reviewable order.
func sortFindings(findings []VerifyFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Line < findings[j].Line
	})
}

// ScanDocuments lists the Markdown documents the verification checks apply to:
// the tracked root documents plus everything under docs/. It is exported so
// adjacent planes (terminology, version policy) check exactly the same set
// instead of drifting into their own file list.
func ScanDocuments(root string) ([]string, error) { return verificationFiles(root) }

// verificationFiles lists the Markdown documents under verification: the
// tracked root documents plus everything under docs/.
func verificationFiles(root string) ([]string, error) {
	files := []string{}
	for _, rel := range rootDocs {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			files = append(files, rel)
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

// checkLinks validates relative Markdown links to .md files.
func checkLinks(root, rel, content string) []VerifyFinding {
	findings := []VerifyFinding{}
	base := filepath.Dir(rel)
	for i, line := range strings.Split(content, "\n") {
		for _, match := range mdLinkPattern.FindAllStringSubmatch(line, -1) {
			target := match[1]
			if strings.HasPrefix(target, "/") || strings.Contains(target, "://") {
				continue
			}
			resolved := filepath.ToSlash(filepath.Clean(filepath.Join(base, target)))
			if strings.HasPrefix(resolved, "..") {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(resolved))); err != nil {
				findings = append(findings, VerifyFinding{
					Kind: "broken-link", Path: rel, Line: i + 1,
					Detail: "relative link target does not exist: " + resolved,
				})
			}
		}
	}
	return findings
}

// checkRegions enforces managed-region marker integrity: a begin marker without
// an end marker silently swallows curated content on the next build.
func checkRegions(rel, content string) []VerifyFinding {
	findings := []VerifyFinding{}
	open := []string{}
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "<!-- prumo:begin "):
			open = append(open, strings.TrimSuffix(strings.TrimPrefix(trimmed, "<!-- prumo:begin "), " -->"))
		case strings.HasPrefix(trimmed, "<!-- prumo:end "):
			id := strings.TrimSuffix(strings.TrimPrefix(trimmed, "<!-- prumo:end "), " -->")
			if len(open) == 0 || open[len(open)-1] != id {
				findings = append(findings, VerifyFinding{Kind: "region-marker-mismatch", Path: rel, Line: i + 1,
					Detail: "end marker " + id + " does not close the open region"})
				continue
			}
			open = open[:len(open)-1]
		}
	}
	for _, id := range open {
		findings = append(findings, VerifyFinding{Kind: "region-marker-unclosed", Path: rel,
			Detail: "managed region " + id + " is never closed"})
	}
	return findings
}

// checkClaimDrift flags statements that call something future work while the
// named implementation exists. This is the audit's dogfood case: a claim that
// says "not yet implemented" about code that is already in the tree.
func checkClaimDrift(rel, content string, implemented map[string]bool) []VerifyFinding {
	findings := []VerifyFinding{}
	for i, line := range strings.Split(content, "\n") {
		if !futureClaims.MatchString(line) {
			continue
		}
		for _, match := range inlineCodePattern.FindAllStringSubmatch(line, -1) {
			symbol := strings.TrimSuffix(match[1], "()")
			symbol = symbol[strings.LastIndex(symbol, ".")+1:]
			if !implemented[symbol] {
				continue
			}
			findings = append(findings, VerifyFinding{
				Kind: "claim-drift", Path: rel, Line: i + 1,
				Detail: "line claims future work but `" + match[1] + "` exists in the repository",
			})
			break
		}
	}
	return findings
}

// implementationWords indexes identifiers that exist in the repository's Go
// sources, so a "future work" claim can be checked against reality.
func implementationWords(root string) map[string]bool {
	implemented := map[string]bool{}
	for _, dir := range []string{"cmd", "internal"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, word := range regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`).FindAllString(string(data), -1) {
				implemented[word] = true
			}
			return nil
		})
	}
	return implemented
}
