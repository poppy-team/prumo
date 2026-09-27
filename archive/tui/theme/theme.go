//go:build ignore

// Package theme resolves the canonical design-token set into concrete values
// for one theme.
//
// The canonical set is `docs/ui-ux/design-tokens.json`; this package holds a
// generated projection of it (`tokens_gen.go`) and never reads the file at
// runtime. Editing the token set without regenerating fails
// `TestProjectionMatchesSource` rather than rendering stale values.
//
// This package depends only on the standard library. It answers "what is the
// value of this token in this theme" and nothing about how a value is painted;
// the renderer adapts it (see `tui/styles.go`). Keeping the decision separate
// from the painting is what makes the token contract testable without a
// terminal.
package theme

//go:generate go run gen_tokens.go

import (
	"fmt"
	"sort"
	"strings"
)

// tokenDecl is one token in the canonical set.
type tokenDecl struct {
	ID        string
	Tier      string
	Kind      string
	Value     string
	Platforms []string
	Fallback  string
	Against   string
	Ratio     float64
	WCAG      string
}

// themeDecl is one theme in the canonical set.
type themeDecl struct {
	ID            string
	Extends       string
	HighContrast  bool
	ReducedMotion bool
	Overrides     map[string]string
}

// Contractual token IDs. `docs/architecture/visual-constitution.md` names these
// as required, so the TUI refers to them by constant rather than by string and
// a rename in the token set breaks the build here instead of silently rendering
// an unstyled surface.
const (
	Surface      = "token:color.surface"
	Panel        = "token:color.panel"
	BorderSubtle = "token:color.border-subtle"
	Text         = "token:color.text"
	TextMuted    = "token:color.text-muted"
	Accent       = "token:color.accent"
	Success      = "token:color.success"
	Warning      = "token:color.warning"
	Error        = "token:color.error"
	Focus        = "token:color.focus"

	TUIPanelBorder             = "token:tui.panel.border"
	TUIStatusline              = "token:tui.statusline"
	TUICommandPaletteHighlight = "token:tui.command-palette.highlight"
	TUIApprovalPrompt          = "token:tui.approval.prompt"
)

// Theme IDs.
const (
	DefaultTheme       = "theme.default"
	NoColorTheme       = "theme.no-color"
	HighContrastTheme  = "theme.high-contrast"
	ReducedMotionTheme = "theme.reduced-motion"
)

// NoColorValue is the canonical value that means "apply no colour at all".
const NoColorValue = "none"

// ContractualTokens returns the tokens the visual constitution requires, in a
// stable order. A test asserts every one of them exists in the canonical set.
func ContractualTokens() []string {
	return []string{Surface, Panel, BorderSubtle, Text, TextMuted, Accent,
		Success, Warning, Error, Focus,
		TUIPanelBorder, TUIStatusline, TUICommandPaletteHighlight, TUIApprovalPrompt}
}

// SourceDigest returns the digest of the token set this projection was built
// from. It is exported so a build or a report can state which token set is in
// force.
func SourceDigest() string { return sourceDigest }

// Version returns the canonical token set version.
func Version() int { return sourceVersion }

// Tiers returns the enforced tier vocabulary, outermost first.
func Tiers() []string { return append([]string{}, declaredTiers...) }

// Names returns every declared theme ID, sorted.
func Names() []string {
	names := make([]string, 0, len(declaredThemes))
	for _, t := range declaredThemes {
		names = append(names, t.ID)
	}
	sort.Strings(names)
	return names
}

// Palette is a resolved token set for one theme.
type Palette struct {
	id            string
	declared      map[string]tokenDecl
	values        map[string]string
	highContrast  bool
	reducedMotion bool
}

// Resolve returns the palette for a theme, applying the theme's override chain
// and resolving every token reference to a concrete value.
//
// Failure is explicit. An unknown theme, a reference to an undeclared token, a
// reference cycle and an override of an undeclared token are all errors rather
// than silent fallbacks: a palette that quietly resolves to nothing renders an
// unstyleable surface, which is harder to diagnose than a failed start.
func Resolve(themeID string) (Palette, error) {
	if themeID == "" {
		themeID = DefaultTheme
	}
	byID := make(map[string]themeDecl, len(declaredThemes))
	for _, t := range declaredThemes {
		byID[t.ID] = t
	}
	theme, ok := byID[themeID]
	if !ok {
		return Palette{}, fmt.Errorf("unknown theme %q (declared: %s)", themeID, strings.Join(Names(), ", "))
	}

	palette := Palette{id: theme.ID, declared: map[string]tokenDecl{}, values: map[string]string{},
		highContrast: theme.HighContrast, reducedMotion: theme.ReducedMotion}
	for _, token := range declaredTokens {
		palette.declared[token.ID] = token
		palette.values[token.ID] = token.Value
	}

	// Override chains are applied root-first so a derived theme wins over the
	// theme it extends.
	chain, err := overrideChain(theme, byID)
	if err != nil {
		return Palette{}, err
	}
	for _, entry := range chain {
		for tokenID, value := range entry.Overrides {
			if _, exists := palette.declared[tokenID]; !exists {
				return Palette{}, fmt.Errorf("theme %s overrides undeclared token %s", entry.ID, tokenID)
			}
			palette.values[tokenID] = value
		}
		if entry.HighContrast {
			palette.highContrast = true
		}
		if entry.ReducedMotion {
			palette.reducedMotion = true
		}
	}

	if err := palette.resolveReferences(); err != nil {
		return Palette{}, err
	}
	if err := palette.validateNoColorIsComplete(); err != nil {
		return Palette{}, err
	}
	return palette, nil
}

// validateNoColorIsComplete fails closed when a theme neutralises its accent —
// the declaration that means "paint nothing" — but leaves other colour tokens
// resolving to real colours.
//
// Without this check a theme can claim to be monochrome and still paint: the
// token set shipped that way, with only four of the ten colour tokens
// neutralised, and nothing noticed because a partially monochrome theme renders
// instead of failing. The error names the offending tokens so the fix is a data
// edit rather than an investigation.
func (p Palette) validateNoColorIsComplete() error {
	if p.values[Accent] != NoColorValue {
		return nil
	}
	var offenders []string
	for _, token := range declaredTokens {
		// Primitives are the raw palette, not a usage: a monochrome theme
		// neutralises what the product paints, and a primitive nobody paints
		// directly is not part of that surface. Any non-primitive token that
		// still resolves to a colour — including one that reaches a primitive
		// by reference — is.
		if token.Kind != "color" || token.Tier == "primitive" {
			continue
		}
		if value := p.values[token.ID]; value != NoColorValue {
			offenders = append(offenders, token.ID+"="+value)
		}
	}
	if len(offenders) == 0 {
		return nil
	}
	sort.Strings(offenders)
	return fmt.Errorf("theme %s suppresses colour but still paints %s", p.id, strings.Join(offenders, ", "))
}

// overrideChain walks a theme's extends chain, returning it root-first and
// refusing a cycle. `extends` is honoured only when it names a declared theme:
// `theme.default` names a token instead, which is a defect in the data rather
// than a parent theme, and treating it as a parent would invent a relationship
// that does not exist.
func overrideChain(theme themeDecl, byID map[string]themeDecl) ([]themeDecl, error) {
	chain := []themeDecl{}
	seen := map[string]bool{}
	for current := theme; ; {
		if seen[current.ID] {
			return nil, fmt.Errorf("theme override cycle at %s", current.ID)
		}
		seen[current.ID] = true
		chain = append([]themeDecl{current}, chain...)
		parent, ok := byID[current.Extends]
		if !ok {
			return chain, nil
		}
		current = parent
	}
}

// resolveReferences folds token→token references into concrete values. A token
// whose value names another token inherits that token's resolved value, which is
// what makes a three-tier set composable.
//
// It takes a pointer receiver because it replaces the value map. With a value
// receiver the fold landed on a copy and was discarded, so every consumer read
// raw references (`token:color.panel`) instead of resolved ones — and the bug
// was invisible because `theme.no-color` overrides its accent with a literal,
// so the one check that ran still agreed.
func (p *Palette) resolveReferences() error {
	resolved := map[string]string{}
	var walk func(id string, path []string) (string, error)
	walk = func(id string, path []string) (string, error) {
		if value, ok := resolved[id]; ok {
			return value, nil
		}
		for _, seen := range path {
			if seen == id {
				return "", fmt.Errorf("token reference cycle: %s", strings.Join(append(path, id), " → "))
			}
		}
		raw, ok := p.values[id]
		if !ok {
			return "", fmt.Errorf("reference to undeclared token %s (from %s)", id, strings.Join(path, " → "))
		}
		if !strings.HasPrefix(raw, "token:") {
			resolved[id] = raw
			return raw, nil
		}
		value, err := walk(raw, append(path, id))
		if err != nil {
			return "", err
		}
		resolved[id] = value
		return value, nil
	}
	for id := range p.values {
		if _, err := walk(id, nil); err != nil {
			return err
		}
	}
	p.values = resolved
	return nil
}

// ID returns the theme this palette was resolved for.
func (p Palette) ID() string { return p.id }

// NoColor reports whether the theme suppresses colour. It is derived from the
// data — the accent token resolving to `none` — rather than from the theme's
// name, so a theme cannot claim to be monochrome while painting colour.
func (p Palette) NoColor() bool { return p.values[Accent] == NoColorValue }

// HighContrast reports whether the theme declares a high-contrast variant.
func (p Palette) HighContrast() bool { return p.highContrast }

// ReducedMotion reports whether the theme suppresses motion.
func (p Palette) ReducedMotion() bool { return p.reducedMotion }

// Value returns the resolved value of a token.
func (p Palette) Value(tokenID string) (string, bool) {
	value, ok := p.values[tokenID]
	return value, ok
}

// MustValue returns a resolved value or panics. It exists for the renderer,
// where a missing contractual token is a programming error rather than an input
// condition: the token set is compiled in, so it cannot vary at runtime.
func (p Palette) MustValue(tokenID string) string {
	value, ok := p.values[tokenID]
	if !ok {
		panic("theme: unresolved token " + tokenID + " in theme " + p.id)
	}
	return value
}

// Paints reports whether a token resolves to a paintable value. It is the check
// a renderer makes before applying a colour, so a no-color theme degrades to
// plain text instead of painting the literal string `none`.
func (p Palette) Paints(tokenID string) bool {
	value, ok := p.values[tokenID]
	return ok && value != NoColorValue && value != ""
}

// Fallback returns the declared terminal-capability fallback for a token, for
// example the ASCII border used when the terminal lacks Unicode box drawing.
func (p Palette) Fallback(tokenID string) string {
	return p.declared[tokenID].Fallback
}

// Kind returns a token's declared kind (`color`, `border`, `motion`, …).
//
// It answers a different question from Paints: a `border` token in a monochrome
// theme resolves to no colour but the border is still structural and must be
// drawn, so the renderer asks for the kind and not for the paintability.
func (p Palette) Kind(tokenID string) string {
	return p.declared[tokenID].Kind
}

// Contrast returns the measured contrast evidence for a token: the token it was
// measured against, the WCAG ratio and the resulting level.
func (p Palette) Contrast(tokenID string) (against string, ratio float64, wcag string, ok bool) {
	token, exists := p.declared[tokenID]
	if !exists || token.Against == "" {
		return "", 0, "", false
	}
	return token.Against, token.Ratio, token.WCAG, true
}

// Resolved returns a copy of every resolved token value, for reporting.
func (p Palette) Resolved() map[string]string {
	out := make(map[string]string, len(p.values))
	for id, value := range p.values {
		out[id] = value
	}
	return out
}
