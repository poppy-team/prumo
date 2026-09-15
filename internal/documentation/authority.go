package docengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
)

// AuthorityRole classifies how a document relates to canonical truth.
//
// Canonical documents own project facts. Projections are derived surfaces that
// must resolve back to a canonical source. Historical documents are records
// (ADRs, migration notes, progress logs) that legitimately reference obsolete
// versions and are excluded from drift checks.
type AuthorityRole string

const (
	RoleCanonical  AuthorityRole = "canonical"
	RoleProjection AuthorityRole = "projection"
	RoleHistorical AuthorityRole = "historical"
)

// AuthorityEntry is one classification rule. Path may be an exact repo-relative
// path, a glob (filepath.Match syntax) or a directory prefix ending in "/**".
// Rules are evaluated in order; the first match wins.
type AuthorityEntry struct {
	Path              string        `json:"path"`
	Role              AuthorityRole `json:"role"`
	Owner             string        `json:"owner,omitempty"`
	CanonicalSource   string        `json:"canonical_source,omitempty"`
	DriftExempt       bool          `json:"drift_exempt,omitempty"`
	DriftExemptReason string        `json:"drift_exempt_reason,omitempty"`
	Note              string        `json:"note,omitempty"`
}

type AuthorityMap struct {
	Version        int              `json:"version"`
	CurrentVersion string           `json:"current_version"`
	GeneratedFrom  []string         `json:"generated_from,omitempty"`
	Documents      []AuthorityEntry `json:"documents"`
}

type AuthorityFinding struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

type AuthorityReport struct {
	CurrentVersion string             `json:"current_version"`
	Files          int                `json:"files"`
	Canonical      int                `json:"canonical"`
	Projection     int                `json:"projection"`
	Historical     int                `json:"historical"`
	Findings       []AuthorityFinding `json:"findings"`
	OK             bool               `json:"ok"`
}

// routingFiles are the entrypoint/routing surfaces whose pointers must always
// resolve (W0.10). Broader link checking is W19.
var routingFiles = []string{"AGENTS.md", "ENTRYPOINT.md", "README.md", "FRAMEWORK.md", "docs/PRUMO.md"}

// rootDocs are the tracked top-level Markdown files included in the scan.
// Untracked input artifacts (e.g. gitignored audit reports) stay out.
var rootDocs = []string{
	"AGENTS.md", "ENTRYPOINT.md", "README.md", "FRAMEWORK.md",
	"CONTRIBUTING.md", "SECURITY.md", "CHANGELOG.md",
	"HARNESS_IMPLEMENTATION_REPORT.md",
}

// driftExemptAnnotations mark a line as an intentional historical reference.
var driftExemptAnnotations = regexp.MustCompile(`(?i)historical|legacy|migration|retired|superseded|deprecated|no longer|oracle`)

var versionToken = regexp.MustCompile(`\bv0\.(\d+)\b`)
var markdownLink = regexp.MustCompile(`\]\(([^)]+)\)`)

func LoadAuthorityMap(root string) (AuthorityMap, error) {
	data, err := os.ReadFile(filepath.Join(root, "docs", "AUTHORITY_MAP.json"))
	if err != nil {
		return AuthorityMap{}, err
	}
	var m AuthorityMap
	if err := json.Unmarshal(data, &m); err != nil {
		return AuthorityMap{}, err
	}
	if len(m.Documents) == 0 {
		return AuthorityMap{}, fmt.Errorf("authority map has no documents")
	}
	return m, nil
}

// RoleOf reports the authority role of a repo-relative document, so other
// projections (publishing, retrieval) classify documents with exactly the
// authority rules instead of re-implementing them.
func RoleOf(root, rel string) (AuthorityRole, bool) {
	m, err := LoadAuthorityMap(root)
	if err != nil {
		return "", false
	}
	entry, ok := m.classify(rel)
	if !ok {
		return "", false
	}
	return entry.Role, true
}

// classify returns the first matching rule for a repo-relative path.
func (m AuthorityMap) classify(rel string) (AuthorityEntry, bool) {
	rel = filepath.ToSlash(rel)
	for _, e := range m.Documents {
		if patternMatches(e.Path, rel) {
			return e, true
		}
	}
	return AuthorityEntry{}, false
}

func patternMatches(pattern, rel string) bool {
	pattern = filepath.ToSlash(pattern)
	if pattern == rel {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**") + "/"
		return strings.HasPrefix(rel, prefix)
	}
	if strings.ContainsAny(pattern, "*?[") {
		ok, err := filepath.Match(pattern, rel)
		return err == nil && ok
	}
	return false
}

func scanAuthorityFiles(root string) ([]string, error) {
	var files []string
	for _, f := range rootDocs {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			files = append(files, f)
		}
	}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
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

// CheckAuthority validates the authority map against repository reality:
// unclassified documents, missing exact targets, version drift in active
// documents, unresolvable routing links, and projection→projection chains.
func CheckAuthority(root string) (AuthorityReport, error) {
	m, err := LoadAuthorityMap(root)
	if err != nil {
		return AuthorityReport{}, err
	}
	report := AuthorityReport{CurrentVersion: protocol.CLIVersion, Findings: []AuthorityFinding{}}
	if m.CurrentVersion != protocol.CLIVersion {
		report.Findings = append(report.Findings, AuthorityFinding{
			Kind:   "version_mismatch",
			Path:   "docs/AUTHORITY_MAP.json",
			Detail: fmt.Sprintf("map current_version %q != protocol.CLIVersion %q", m.CurrentVersion, protocol.CLIVersion),
		})
	}

	files, err := scanAuthorityFiles(root)
	if err != nil {
		return AuthorityReport{}, err
	}
	report.Files = len(files)
	currentMinor := minorOf(protocol.CLIVersion)
	for _, rel := range files {
		entry, ok := m.classify(rel)
		if !ok {
			report.Findings = append(report.Findings, AuthorityFinding{
				Kind: "unclassified", Path: rel,
				Detail: "no authority rule matches this document",
			})
			continue
		}
		switch entry.Role {
		case RoleCanonical:
			report.Canonical++
		case RoleProjection:
			report.Projection++
		case RoleHistorical:
			report.Historical++
		default:
			report.Findings = append(report.Findings, AuthorityFinding{
				Kind: "invalid_role", Path: rel, Detail: fmt.Sprintf("unknown role %q", entry.Role),
			})
		}
		if !strings.ContainsAny(entry.Path, "*?[") {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				report.Findings = append(report.Findings, AuthorityFinding{
					Kind: "missing_file", Path: rel, Detail: "authority rule points to a missing file",
				})
			}
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		content := string(data)
		if entry.Role != RoleHistorical && !entry.DriftExempt {
			report.Findings = append(report.Findings, versionDriftFindings(rel, content, currentMinor)...)
		}
		if isRoutingFile(rel) {
			report.Findings = append(report.Findings, brokenLinkFindings(root, rel, content)...)
		}
	}
	report.Findings = append(report.Findings, projectionCycleFindings(m)...)
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Path != report.Findings[j].Path {
			return report.Findings[i].Path < report.Findings[j].Path
		}
		return report.Findings[i].Kind < report.Findings[j].Kind
	})
	report.OK = len(report.Findings) == 0
	return report, nil
}

func minorOf(version string) int {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return -1
	}
	n := 0
	for _, r := range parts[1] {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// versionDriftFindings flags references to project release lines older than the
// current CLI version, unless the line carries a historical annotation.
func versionDriftFindings(rel, content string, currentMinor int) []AuthorityFinding {
	if currentMinor < 0 {
		return nil
	}
	var findings []AuthorityFinding
	for i, line := range strings.Split(content, "\n") {
		if driftExemptAnnotations.MatchString(line) {
			continue
		}
		for _, loc := range versionToken.FindAllStringSubmatchIndex(line, -1) {
			// Skip component versions such as v0.1.0 (char after match is '.').
			end := loc[1]
			if end < len(line) && line[end] == '.' {
				continue
			}
			minor := 0
			for _, c := range line[loc[2]:loc[3]] {
				minor = minor*10 + int(c-'0')
			}
			if minor < currentMinor {
				findings = append(findings, AuthorityFinding{
					Kind:   "version_drift",
					Path:   rel,
					Detail: fmt.Sprintf("line %d references v0.%d (current v0.%d): %s", i+1, minor, currentMinor, strings.TrimSpace(line)),
				})
			}
		}
	}
	return findings
}

func isRoutingFile(rel string) bool {
	for _, f := range routingFiles {
		if f == rel {
			return true
		}
	}
	return false
}

func brokenLinkFindings(root, rel, content string) []AuthorityFinding {
	var findings []AuthorityFinding
	dir := filepath.Dir(filepath.Join(root, rel))
	for _, m := range markdownLink.FindAllStringSubmatch(content, -1) {
		target := strings.TrimSpace(m[1])
		if target == "" || strings.HasPrefix(target, "#") || strings.Contains(target, "://") ||
			strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "/") {
			continue
		}
		if i := strings.Index(target, "#"); i >= 0 {
			target = target[:i]
		}
		if target == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, target)); err != nil {
			findings = append(findings, AuthorityFinding{
				Kind: "broken_link", Path: rel, Detail: fmt.Sprintf("link target %q does not resolve", m[1]),
			})
		}
	}
	return findings
}

// projectionCycleFindings rejects projections whose declared canonical source is
// itself a projection (the audit's D009 case: no projection may become canonical).
func projectionCycleFindings(m AuthorityMap) []AuthorityFinding {
	roleByPath := map[string]AuthorityRole{}
	for _, e := range m.Documents {
		if !strings.ContainsAny(e.Path, "*?[") {
			roleByPath[filepath.ToSlash(e.Path)] = e.Role
		}
	}
	var findings []AuthorityFinding
	for _, e := range m.Documents {
		if e.Role != RoleProjection || e.CanonicalSource == "" {
			continue
		}
		if src, ok := roleByPath[filepath.ToSlash(e.CanonicalSource)]; ok && src == RoleProjection {
			findings = append(findings, AuthorityFinding{
				Kind: "projection_cycle", Path: e.Path,
				Detail: fmt.Sprintf("canonical_source %q is itself a projection", e.CanonicalSource),
			})
		}
	}
	return findings
}
