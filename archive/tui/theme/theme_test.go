package theme

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sourcePath = "../../docs/ui-ux/design-tokens.json"

// TestProjectionMatchesSource is the freshness check: editing the canonical
// token set without regenerating the projection fails here instead of rendering
// stale colours. This is the same digest discipline evidence and translations
// use, applied to a compile-time projection.
func TestProjectionMatchesSource(t *testing.T) {
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("canonical token set is missing: %v", err)
	}
	sum := sha256.Sum256(raw)
	want := "sha256:" + hex.EncodeToString(sum[:])[:32]
	if got := SourceDigest(); got != want {
		t.Fatalf("token projection is stale: generated from %s, source is %s.\n"+
			"Run `go generate ./tui/theme` and commit the result.", got, want)
	}
}

// TestContractualTokensExist pins the set the visual constitution requires. A
// rename or removal in the token set breaks the build here rather than leaving
// a surface unstyled at runtime.
func TestContractualTokensExist(t *testing.T) {
	palette, err := Resolve(DefaultTheme)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ContractualTokens() {
		if _, ok := palette.Value(id); !ok {
			t.Errorf("contractual token %s is not in the canonical set", id)
		}
	}
}

// TestEveryThemeResolves keeps a theme from shipping in a state that cannot be
// painted: every declared theme must resolve every token, and a cycle or a
// dangling reference must fail rather than resolve to nothing.
func TestEveryThemeResolves(t *testing.T) {
	names := Names()
	if len(names) < 4 {
		t.Fatalf("expected the declared theme set, got %v", names)
	}
	for _, name := range names {
		palette, err := Resolve(name)
		if err != nil {
			t.Errorf("theme %s does not resolve: %v", name, err)
			continue
		}
		if palette.ID() != name {
			t.Errorf("theme %s resolved as %s", name, palette.ID())
		}
		for _, id := range ContractualTokens() {
			if !palette.Paints(id) && palette.NoColor() {
				continue // a monochrome theme legitimately paints nothing
			}
			if !palette.Paints(id) {
				t.Errorf("theme %s: token %s resolves to nothing", name, id)
			}
		}
	}
}

// TestNoColorThemeIsDerivedFromData pins that monochrome is detected from the
// token values rather than from the theme's name, so a theme cannot claim to be
// monochrome while painting colour.
func TestNoColorThemeIsDerivedFromData(t *testing.T) {
	mono, err := Resolve(NoColorTheme)
	if err != nil {
		t.Fatal(err)
	}
	if !mono.NoColor() {
		t.Fatal("the no-color theme must report itself as monochrome")
	}
	for _, id := range []string{Accent, Success, Warning, Error} {
		if mono.Paints(id) {
			t.Errorf("no-color theme still paints %s = %s", id, mono.MustValue(id))
		}
	}
	plain, err := Resolve(DefaultTheme)
	if err != nil {
		t.Fatal(err)
	}
	if plain.NoColor() {
		t.Fatal("the default theme must not report itself as monochrome")
	}
}

// TestHighContrastAndReducedMotionFlags replaces the lower-contrast text token,
// which is the point of having the variant at all.
func TestHighContrastFlagsAreHonoured(t *testing.T) {
	high, err := Resolve(HighContrastTheme)
	if err != nil {
		t.Fatal(err)
	}
	if !high.HighContrast() {
		t.Fatal("the high-contrast theme must report itself as such")
	}
	plain, err := Resolve(DefaultTheme)
	if err != nil {
		t.Fatal(err)
	}
	if high.MustValue(TextMuted) == plain.MustValue(TextMuted) {
		t.Fatal("high contrast must change the muted text token, otherwise it does nothing")
	}
	reduced, err := Resolve(ReducedMotionTheme)
	if err != nil {
		t.Fatal(err)
	}
	if !reduced.ReducedMotion() {
		t.Fatal("the reduced-motion theme must report itself as such")
	}
}

// TestUnknownThemeFailsClosed pins that a typo is an error, not a silent
// fallback to a default palette that hides the mistake.
func TestUnknownThemeFailsClosed(t *testing.T) {
	if _, err := Resolve("theme.does-not-exist"); err == nil {
		t.Fatal("an unknown theme must fail rather than fall back")
	}
}

func TestContrastEvidenceIsReadable(t *testing.T) {
	palette, err := Resolve(DefaultTheme)
	if err != nil {
		t.Fatal(err)
	}
	against, ratio, wcag, ok := palette.Contrast(Text)
	if !ok {
		t.Fatal("the text token must declare measured contrast evidence")
	}
	if against == "" || ratio <= 0 || wcag == "" {
		t.Fatalf("incomplete contrast evidence: against=%q ratio=%g wcag=%q", against, ratio, wcag)
	}
}

func TestUnicodeFallbackIsDeclaredForBorders(t *testing.T) {
	palette, err := Resolve(DefaultTheme)
	if err != nil {
		t.Fatal(err)
	}
	fallback := palette.Fallback(TUIPanelBorder)
	if !strings.Contains(strings.ToLower(fallback), "ascii") {
		t.Fatalf("the panel border must declare an ASCII fallback, got %q", fallback)
	}
}

func TestTiersAreTheContractualThree(t *testing.T) {
	tiers := Tiers()
	if len(tiers) != 3 {
		t.Fatalf("the token system is three tiers, got %v", tiers)
	}
}

// TestGeneratedFileIsPresent guards against a projection that was never
// generated, which would otherwise look like an empty token set.
func TestGeneratedFileIsPresent(t *testing.T) {
	if _, err := os.Stat(filepath.Join("tokens_gen.go")); err != nil {
		t.Fatalf("tokens_gen.go is missing: run `go generate ./tui/theme` (%v)", err)
	}
}
