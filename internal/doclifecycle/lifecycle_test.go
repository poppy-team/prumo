package doclifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeJSON(t *testing.T, root, rel string, value any) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeText(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTranslationStalenessFollowsSourceDigest(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "docs/manual/usage.md", "# Usage\n\nRun the command.\n")
	digest := DigestOfFile(root, "docs/manual/usage.md")
	writeJSON(t, root, TranslationsPath, []TranslationRecord{{
		UnitID: "docs.manual.usage", Locale: "pt-BR", SourceLocale: "en",
		SourcePath: "docs/manual/usage.md", SourceDigest: digest, State: "current",
	}})

	statuses, err := TranslationStatuses(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Stale {
		t.Fatalf("expected a current translation: %#v", statuses)
	}
	if statuses[0].RecordedDigest != statuses[0].CurrentDigest {
		t.Fatalf("digests must match: %#v", statuses[0])
	}

	writeText(t, root, "docs/manual/usage.md", "# Usage\n\nRun the new command.\n")
	statuses, err = TranslationStatuses(root)
	if err != nil {
		t.Fatal(err)
	}
	if !statuses[0].Stale || statuses[0].State != "needs-update" {
		t.Fatalf("expected needs-update after a source change: %#v", statuses[0])
	}
	if !strings.Contains(statuses[0].Describe(), "needs-update") {
		t.Fatalf("unexpected description: %s", statuses[0].Describe())
	}
}

func TestTranslationWithoutSourceOrDigestIsStale(t *testing.T) {
	root := t.TempDir()
	writeJSON(t, root, TranslationsPath, []TranslationRecord{
		{UnitID: "u1", Locale: "pt-BR", SourceLocale: "en", State: "current"},
		{UnitID: "u2", Locale: "pt-BR", SourceLocale: "en", SourcePath: "docs/missing.md", SourceDigest: "sha256:x", State: "current"},
	})
	statuses, err := TranslationStatuses(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range statuses {
		if !status.Stale {
			t.Fatalf("record %s must be stale: %#v", status.ID, status)
		}
	}
}

func TestPseudoLocalizationExpandsAndPreservesShape(t *testing.T) {
	out := PseudoLocalize("Open file {name}")
	if !strings.HasPrefix(out, "⟦") || !strings.HasSuffix(out, "⟧") {
		t.Fatalf("pseudo-localization must be visibly marked: %s", out)
	}
	if len(out) <= len("Open file {name}") {
		t.Fatalf("pseudo-localization must expand the text: %s", out)
	}
	if !strings.Contains(out, "{name}") {
		t.Fatalf("placeholders must survive: %s", out)
	}
	if PseudoLocalize("ab") == PseudoLocalize("ab") == false {
		t.Fatal("pseudo-localization must be deterministic")
	}
}

func TestVerifyPseudoLocFindsGeneratorProblems(t *testing.T) {
	root := t.TempDir()
	writeJSON(t, root, TranslationsPath, []TranslationRecord{{
		UnitID: "u", Locale: "pt-BR", SourceLocale: "en", SourcePath: "docs/x.md", SourceDigest: "sha256:y", State: "current",
		Segments: []struct {
			ID       string `json:"id"`
			Source   string `json:"source_segment"`
			Text     string `json:"text"`
			State    string `json:"state,omitempty"`
			Reviewed bool   `json:"reviewed,omitempty"`
		}{
			{ID: "s1", Source: "Hello {user}", Text: ""},
			{ID: "s2", Source: "Bye {user}", Text: "Tchau {usuario}"},
		},
	}})
	findings, err := VerifyPseudoLoc(root)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, f := range findings {
		kinds[f.Kind]++
	}
	if kinds["translation-empty-segment"] != 1 {
		t.Fatalf("expected one empty segment finding: %#v", findings)
	}
	if kinds["translation-placeholder-mismatch"] != 1 {
		t.Fatalf("expected one placeholder mismatch: %#v", findings)
	}
	if kinds["translation-no-fallback"] != 1 {
		t.Fatalf("a locale surface must declare a fallback: %#v", findings)
	}
}

func TestMediaGoesStaleWhenItsUnitChanges(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "docs/ui-ux/screens.md", "# Screens\n\nOld capture.\n")
	revision := DigestOfFile(root, "docs/ui-ux/screens.md")
	writeText(t, root, "docs/media/cli.png", "binary-ish")
	writeJSON(t, root, MediaPath, []MediaRecord{{
		MediaID: "media:cli-overview", Kind: "terminal-frame", SourceUnit: "docs/ui-ux/screens.md",
		SourceRevision: revision, Hash: "sha256:abc", Status: "current", Path: "docs/media/cli.png",
	}})
	statuses, err := MediaStatuses(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Stale {
		t.Fatalf("expected current media: %#v", statuses)
	}

	writeText(t, root, "docs/ui-ux/screens.md", "# Screens\n\nNew capture states.\n")
	statuses, err = MediaStatuses(root)
	if err != nil {
		t.Fatal(err)
	}
	if !statuses[0].Stale || statuses[0].State != "affected-by-ui-change" {
		t.Fatalf("expected affected-by-ui-change: %#v", statuses[0])
	}

	if err := os.Remove(filepath.Join(root, "docs", "media", "cli.png")); err != nil {
		t.Fatal(err)
	}
	statuses, err = MediaStatuses(root)
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].State != "missing" {
		t.Fatalf("a missing artifact must be reported: %#v", statuses[0])
	}
}

func TestDeprecationLifecycleRequiresReplacementAndNotice(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "docs/old-guide.md", "# Old guide\n\nDeprecated: use the new guide instead.\n")
	writeText(t, root, "docs/new-guide.md", "# New guide\n")
	writeJSON(t, root, LifecyclePath, Lifecycle{Version: "0.6.0", Deprecated: []Deprecation{
		{Path: "docs/old-guide.md", Replacement: "docs/new-guide.md", Reason: "replaced"},
		{Path: "docs/new-guide.md", Reason: "no replacement and no removal"},
	}})
	findings, err := DeprecationFindings(root)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, f := range findings {
		kinds[f.Kind]++
	}
	if kinds["deprecation-without-replacement"] != 1 {
		t.Fatalf("a deprecation needs a replacement or removal: %#v", findings)
	}
	if kinds["deprecation-not-annotated"] != 1 {
		t.Fatalf("a deprecated document must carry a notice: %#v", findings)
	}
}

func TestReleaseReadinessBlocksStaleUnits(t *testing.T) {
	root := t.TempDir()
	writeText(t, root, "docs/manual/usage.md", "# Usage\n\nChanged since translation.\n")
	writeJSON(t, root, TranslationsPath, []TranslationRecord{{
		UnitID: "docs.manual.usage", Locale: "pt-BR", SourceLocale: "en",
		SourcePath: "docs/manual/usage.md", SourceDigest: "sha256:old", State: "current", Fallback: "source-locale",
	}})
	report, err := CheckRelease(root, "0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready {
		t.Fatalf("release must not be ready with a stale translation: %#v", report)
	}
	if len(report.StaleUnits) != 1 || report.StaleUnits[0] != "docs.manual.usage/pt-BR" {
		t.Fatalf("unexpected stale units: %#v", report.StaleUnits)
	}

	writeJSON(t, root, TranslationsPath, []TranslationRecord{{
		UnitID: "docs.manual.usage", Locale: "pt-BR", SourceLocale: "en",
		SourcePath: "docs/manual/usage.md", SourceDigest: DigestOfFile(root, "docs/manual/usage.md"),
		State: "current", Fallback: "source-locale",
	}})
	report, err = CheckRelease(root, "0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready {
		t.Fatalf("expected a ready release: %#v", report.Findings)
	}

	empty, err := CheckRelease(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if empty.Ready {
		t.Fatal("a release check without a version must fail")
	}
}

func TestRepositoryLifecycleStoresAreValid(t *testing.T) {
	root, _ := filepath.Abs("../..")
	if _, err := Translations(root); err != nil {
		t.Fatal(err)
	}
	if _, err := Media(root); err != nil {
		t.Fatal(err)
	}
	report, err := CheckRelease(root, "0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready {
		t.Fatalf("repository lifecycle must be consistent: %#v", report.Findings)
	}
}
