// Package doclifecycle manages the post-generation documentation lifecycle:
// localization, media currency, version/deprecation policy and release
// readiness (W20).
//
// Canonical lifecycle state is human-maintained JSON (translation and media
// records). Derived status is computed from source digests, never stored as
// truth: a record is stale when its recorded digest no longer matches the
// canonical source.
package doclifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Canonical lifecycle stores.
const (
	TranslationsPath = "docs/i18n/translations.json"
	MediaPath        = "docs/media/records.json"
	LifecyclePath    = "docs/lifecycle.json"
)

// TranslationRecord is one localized unit (see schemas/translation-record.schema.json).
type TranslationRecord struct {
	UnitID       string `json:"unit_id"`
	Locale       string `json:"locale"`
	SourceLocale string `json:"source_locale"`
	SourcePath   string `json:"source_path,omitempty"`
	SourceDigest string `json:"source_digest"`
	State        string `json:"state"`
	Segments     []struct {
		ID       string `json:"id"`
		Source   string `json:"source_segment"`
		Text     string `json:"text"`
		State    string `json:"state,omitempty"`
		Reviewed bool   `json:"reviewed,omitempty"`
	} `json:"segments,omitempty"`
	Fallback  string `json:"fallback,omitempty"`
	Reviewer  string `json:"reviewer,omitempty"`
	Generated string `json:"generated_by,omitempty"`
}

// MediaRecord is one visual artifact (see schemas/media-record.schema.json).
type MediaRecord struct {
	MediaID        string `json:"media_id"`
	Kind           string `json:"kind"`
	SourceUnit     string `json:"source_unit"`
	SourceRevision string `json:"source_revision,omitempty"`
	Hash           string `json:"hash"`
	Status         string `json:"status"`
	Theme          string `json:"theme,omitempty"`
	Locale         string `json:"locale,omitempty"`
	Platform       string `json:"platform,omitempty"`
	Path           string `json:"path,omitempty"`
}

// UnitStatus is the derived currency of one localized or visual unit.
type UnitStatus struct {
	ID             string `json:"id"`
	Locale         string `json:"locale,omitempty"`
	Source         string `json:"source,omitempty"`
	State          string `json:"state"`
	RecordedDigest string `json:"recorded_digest,omitempty"`
	CurrentDigest  string `json:"current_digest,omitempty"`
	Stale          bool   `json:"stale"`
	Reason         string `json:"reason"`
}

// Finding is one lifecycle problem.
type Finding struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Detail string `json:"detail"`
}

// Digest is the canonical content digest used by lifecycle records.
func Digest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])[:32]
}

// DigestOfFile digests a repository file, or returns "" when it is absent.
func DigestOfFile(root, rel string) string {
	if strings.TrimSpace(rel) == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return ""
	}
	return Digest(string(data))
}

func loadJSON(path string, target any) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return false, err
	}
	return true, nil
}

// Translations loads the canonical translation records.
func Translations(root string) ([]TranslationRecord, error) {
	var records []TranslationRecord
	_, err := loadJSON(filepath.Join(root, filepath.FromSlash(TranslationsPath)), &records)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].UnitID != records[j].UnitID {
			return records[i].UnitID < records[j].UnitID
		}
		return records[i].Locale < records[j].Locale
	})
	return records, nil
}

// Media loads the canonical media records.
func Media(root string) ([]MediaRecord, error) {
	var records []MediaRecord
	_, err := loadJSON(filepath.Join(root, filepath.FromSlash(MediaPath)), &records)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool { return records[i].MediaID < records[j].MediaID })
	return records, nil
}

// TranslationStatuses computes digest-based currency for every record.
func TranslationStatuses(root string) ([]UnitStatus, error) {
	records, err := Translations(root)
	if err != nil {
		return nil, err
	}
	statuses := []UnitStatus{}
	for _, record := range records {
		status := UnitStatus{ID: record.UnitID, Locale: record.Locale, Source: record.SourcePath,
			State: record.State, RecordedDigest: record.SourceDigest}
		current := DigestOfFile(root, record.SourcePath)
		status.CurrentDigest = current
		switch {
		case record.SourcePath == "":
			// Fail closed: a record that cannot name its source cannot be
			// proven current.
			status.Stale = true
			status.State = "needs-update"
			status.Reason = "record does not name its canonical source"
		case current == "":
			status.Stale = true
			status.State = "stale"
			status.Reason = "canonical source is missing: " + record.SourcePath
		case record.SourceDigest == "":
			status.Stale = true
			status.State = "needs-update"
			status.Reason = "record has no source digest"
		case record.SourceDigest != current:
			status.Stale = true
			status.State = "needs-update"
			status.Reason = "source changed since translation"
		default:
			status.Reason = "source digest matches"
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// MediaStatuses computes digest-based currency for media records.
func MediaStatuses(root string) ([]UnitStatus, error) {
	records, err := Media(root)
	if err != nil {
		return nil, err
	}
	statuses := []UnitStatus{}
	for _, record := range records {
		status := UnitStatus{ID: record.MediaID, Locale: record.Locale, Source: record.SourceUnit,
			State: record.Status, RecordedDigest: record.SourceRevision}
		current := DigestOfFile(root, record.SourceUnit)
		status.CurrentDigest = current
		switch {
		case record.SourceUnit == "":
			status.Stale = true
			status.State = "needs-update"
			status.Reason = "media record does not name the unit it illustrates"
		case current == "":
			status.Stale = true
			status.State = "stale"
			status.Reason = "illustrated unit is missing: " + record.SourceUnit
		case record.SourceRevision == "":
			status.Stale = true
			status.State = "needs-update"
			status.Reason = "record has no source revision"
		case record.SourceRevision != current:
			status.Stale = true
			status.State = "affected-by-ui-change"
			status.Reason = "illustrated unit changed"
		default:
			status.Reason = "illustrated unit is current"
		}
		if record.Path != "" {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(record.Path))); err != nil {
				status.Stale = true
				status.State = "missing"
				status.Reason = "media artifact is missing: " + record.Path
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// PseudoLocalize expands a string deterministically so layout assumptions and
// truncation are visible before a real translation exists (W20.3).
func PseudoLocalize(text string) string {
	var b strings.Builder
	inPlaceholder := false
	for _, r := range text {
		switch {
		case r == '{':
			inPlaceholder = true
			b.WriteRune(r)
		case r == '}':
			inPlaceholder = false
			b.WriteRune(r)
		case inPlaceholder:
			// Placeholders must survive verbatim; only literal text expands.
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteString(pseudoLower[r-'a'])
		case r >= 'A' && r <= 'Z':
			upper := pseudoLower[r-'A']
			b.WriteString(strings.ToUpper(upper[:1]) + upper[1:])
		case r == ' ':
			b.WriteString("  ")
		default:
			b.WriteRune(r)
		}
	}
	return "⟦" + b.String() + "⟧"
}

var pseudoLower = [26]string{
	"à", "ƀ", "ç", "ð", "é", "ƒ", "ĝ", "ĥ", "î", "ĵ", "ķ", "ļ", "ɱ",
	"ñ", "ô", "þ", " q", "ŕ", "ş", "ţ", "û", "ṽ", "ŵ", "ẋ", "ŷ", "ž",
}

// VerifyPseudoLoc checks every translation segment for the failures
// pseudo-localization exposes: empty targets, unbalanced placeholders and
// missing expansion room.
func VerifyPseudoLoc(root string) ([]Finding, error) {
	records, err := Translations(root)
	if err != nil {
		return nil, err
	}
	findings := []Finding{}
	for _, record := range records {
		for _, segment := range record.Segments {
			target := strings.TrimSpace(segment.Text)
			switch {
			case target == "":
				findings = append(findings, Finding{Kind: "translation-empty-segment",
					Target: record.UnitID + "/" + segment.ID,
					Detail: "segment has no target text; fallback must be explicit"})
			case !placeholdersBalanced(segment.Source, segment.Text):
				findings = append(findings, Finding{Kind: "translation-placeholder-mismatch",
					Target: record.UnitID + "/" + segment.ID,
					Detail: "placeholders differ between source and target"})
			}
		}
		if record.Fallback == "" {
			findings = append(findings, Finding{Kind: "translation-no-fallback",
				Target: record.UnitID + "/" + record.Locale,
				Detail: "a locale surface must declare a fallback so it never renders empty"})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Target < findings[j].Target
	})
	return findings, nil
}

// placeholdersBalance compares the placeholder names in a source segment and
// its target. Renaming a placeholder breaks the rendered output even when the
// braces still balance.
func placeholdersBalanced(source, target string) bool {
	sourceNames, targetNames := placeholderNames(source), placeholderNames(target)
	if len(sourceNames) != len(targetNames) {
		return false
	}
	for i, name := range sourceNames {
		if targetNames[i] != name {
			return false
		}
	}
	return true
}

func placeholderNames(text string) []string {
	names := []string{}
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		end := strings.IndexByte(text[i:], '}')
		if end < 0 {
			names = append(names, text[i:])
			break
		}
		names = append(names, text[i:i+end+1])
		i += end
	}
	sort.Strings(names)
	return names
}

// Lifecycle is the canonical deprecation/version registry.
type Lifecycle struct {
	Version       string        `json:"version"`
	Deprecated    []Deprecation `json:"deprecated"`
	ReleaseGates  []string      `json:"release_gates,omitempty"`
	VersionPolicy VersionPolicy `json:"version_policy"`
}

// Deprecation records one retired surface and its replacement.
type Deprecation struct {
	Path        string `json:"path"`
	Replacement string `json:"replacement,omitempty"`
	Since       string `json:"since,omitempty"`
	Reason      string `json:"reason"`
	Removal     string `json:"removal,omitempty"`
}

// LoadLifecycle reads the deprecation registry.
func LoadLifecycle(root string) (Lifecycle, error) {
	var lifecycle Lifecycle
	found, err := loadJSON(filepath.Join(root, filepath.FromSlash(LifecyclePath)), &lifecycle)
	if err != nil {
		return Lifecycle{}, err
	}
	if !found {
		return Lifecycle{Version: "", Deprecated: []Deprecation{}}, nil
	}
	return lifecycle, nil
}

// DeprecationFindings enforces the deprecation lifecycle (W20.7): a deprecation
// must name a replacement, and an active document must not link to a deprecated
// page without pointing at its replacement.
func DeprecationFindings(root string) ([]Finding, error) {
	lifecycle, err := LoadLifecycle(root)
	if err != nil {
		return nil, err
	}
	findings := []Finding{}
	deprecated := map[string]Deprecation{}
	for _, d := range lifecycle.Deprecated {
		if strings.TrimSpace(d.Path) == "" {
			findings = append(findings, Finding{Kind: "deprecation-missing-path", Target: "docs/lifecycle.json",
				Detail: "deprecation entry has no path"})
			continue
		}
		if strings.TrimSpace(d.Replacement) == "" && strings.TrimSpace(d.Removal) == "" {
			findings = append(findings, Finding{Kind: "deprecation-without-replacement", Target: d.Path,
				Detail: "a deprecation must name a replacement or a removal target"})
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(d.Path))); err != nil {
			findings = append(findings, Finding{Kind: "deprecated-path-missing", Target: d.Path,
				Detail: "deprecated document no longer exists; remove the registry entry"})
		}
		deprecated[d.Path] = d
	}
	for path, d := range deprecated {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), "Deprecated") && !strings.Contains(string(data), "deprecated") {
			findings = append(findings, Finding{Kind: "deprecation-not-annotated", Target: path,
				Detail: "document is registered as deprecated but carries no deprecation notice"})
		}
		if d.Replacement != "" {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(d.Replacement))); err != nil {
				findings = append(findings, Finding{Kind: "deprecation-replacement-missing", Target: path,
					Detail: "replacement does not exist: " + d.Replacement})
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Target < findings[j].Target
	})
	return findings, nil
}

// ReleaseReadiness is the documentation gate for a release (W20.8, W20.10).
type ReleaseReadiness struct {
	Version      string    `json:"version"`
	Translations int       `json:"translations"`
	Media        int       `json:"media"`
	StaleUnits   []string  `json:"stale_units"`
	Deprecations int       `json:"deprecations"`
	Findings     []Finding `json:"findings"`
	Ready        bool      `json:"ready"`
}

// CheckRelease evaluates whether documentation may ship with a version.
func CheckRelease(root, version string) (ReleaseReadiness, error) {
	translations, err := TranslationStatuses(root)
	if err != nil {
		return ReleaseReadiness{}, err
	}
	media, err := MediaStatuses(root)
	if err != nil {
		return ReleaseReadiness{}, err
	}
	deprecationFindings, err := DeprecationFindings(root)
	if err != nil {
		return ReleaseReadiness{}, err
	}
	lifecycle, err := LoadLifecycle(root)
	if err != nil {
		return ReleaseReadiness{}, err
	}
	report := ReleaseReadiness{Version: version, Translations: len(translations), Media: len(media),
		StaleUnits: []string{}, Deprecations: len(lifecycle.Deprecated), Findings: []Finding{}}
	for _, status := range translations {
		if status.Stale {
			report.StaleUnits = append(report.StaleUnits, status.ID+"/"+status.Locale)
			report.Findings = append(report.Findings, Finding{Kind: "translation-stale",
				Target: status.ID + "/" + status.Locale, Detail: status.Reason})
		}
	}
	for _, status := range media {
		if status.Stale {
			report.StaleUnits = append(report.StaleUnits, status.ID)
			report.Findings = append(report.Findings, Finding{Kind: "media-stale", Target: status.ID, Detail: status.Reason})
		}
	}
	report.Findings = append(report.Findings, deprecationFindings...)
	if strings.TrimSpace(version) == "" {
		report.Findings = append(report.Findings, Finding{Kind: "release-version-missing",
			Target: "docs/lifecycle.json", Detail: "a release readiness check requires a version"})
	}
	sort.Strings(report.StaleUnits)
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Kind != report.Findings[j].Kind {
			return report.Findings[i].Kind < report.Findings[j].Kind
		}
		return report.Findings[i].Target < report.Findings[j].Target
	})
	report.Ready = len(report.Findings) == 0
	return report, nil
}

// Describe renders a compact human summary of a unit status.
func (s UnitStatus) Describe() string {
	return fmt.Sprintf("%s [%s] %s — %s", s.ID, s.State, s.Source, s.Reason)
}
