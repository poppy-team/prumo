package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/raillen/prumo/tui/theme"
)

// Styles paints the view from a resolved token palette.
//
// No colour value is written down here. Every one is looked up in the palette
// produced from `docs/ui-ux/design-tokens.json`, and `theme.Resolve` has already
// folded the theme's override chain, so switching to `theme.no-color` needs no
// code path in this file: `Palette.Paints` simply reports false and styling
// degrades to plain text. That is the token contract doing its job instead of
// being restated.
type Styles struct {
	palette theme.Palette

	title     lipgloss.Style
	muted     lipgloss.Style
	accent    lipgloss.Style
	success   lipgloss.Style
	warning   lipgloss.Style
	failure   lipgloss.Style
	focus     lipgloss.Style
	selected  lipgloss.Style
	panel     lipgloss.Style
	statusbar lipgloss.Style
	prompt    lipgloss.Style
}

// NewStyles resolves a theme and returns the styles painted from it.
func NewStyles(themeID string) (Styles, error) {
	palette, err := theme.Resolve(themeID)
	if err != nil {
		return Styles{}, err
	}
	s := Styles{palette: palette}
	paint := func(token string) lipgloss.Style {
		style := lipgloss.NewStyle()
		if palette.Paints(token) {
			style = style.Foreground(lipgloss.Color(palette.MustValue(token)))
		}
		return style
	}
	s.title = paint(theme.Text).Bold(true)
	s.muted = paint(theme.TextMuted)
	s.accent = paint(theme.Accent)
	s.success = paint(theme.Success)
	s.warning = paint(theme.Warning)
	s.failure = paint(theme.Error)
	s.focus = paint(theme.Focus).Bold(true)
	s.selected = paint(theme.TUICommandPaletteHighlight).Bold(true)
	s.prompt = paint(theme.TUIApprovalPrompt).Bold(true)

	// Panels carry the surface background and the border token.
	//
	// The border is structural, so it is drawn whenever the token *is* a
	// border token — including in a monochrome theme, where the frame is the
	// only structure left. Its colour is applied only when the palette paints
	// it, which is how `theme.no-color` keeps its frame and loses its colour.
	s.panel = paint(theme.Surface)
	if palette.Kind(theme.TUIPanelBorder) == "border" {
		s.panel = s.panel.Border(lipgloss.RoundedBorder())
		if palette.Paints(theme.TUIPanelBorder) {
			s.panel = s.panel.BorderForeground(lipgloss.Color(palette.MustValue(theme.TUIPanelBorder)))
		}
	}
	s.statusbar = paint(theme.TUIStatusline)
	return s, nil
}

// ThemeID is the resolved theme, for the statusline.
func (s Styles) ThemeID() string { return s.palette.ID() }

// PaintedTokens lists the tokens this theme actually paints, in the contractual
// order.
//
// It reports a decision rather than the escape sequences it produces, because
// the bytes depend on the terminal's colour profile while the decision does
// not. That makes "does this theme paint colour?" checkable in a test regardless
// of where the test runs.
func (s Styles) PaintedTokens() []string {
	painted := make([]string, 0, len(theme.ContractualTokens()))
	for _, token := range theme.ContractualTokens() {
		if s.palette.Paints(token) {
			painted = append(painted, token)
		}
	}
	return painted
}

// SourceDigest is the canonical token-set digest this projection was built from.
func (s Styles) SourceDigest() string { return theme.SourceDigest() }

// NoColor reports whether the theme suppresses colour.
func (s Styles) NoColor() bool { return s.palette.NoColor() }

// HighContrast reports whether the theme is a high-contrast variant.
func (s Styles) HighContrast() bool { return s.palette.HighContrast() }

// Severity paints a timeline row according to its classification.
func (s Styles) Severity(severity Severity) lipgloss.Style {
	switch severity {
	case SeveritySuccess:
		return s.success
	case SeverityWarning:
		return s.warning
	case SeverityError:
		return s.failure
	default:
		return lipgloss.NewStyle()
	}
}

// Title styles a section heading.
func (s Styles) Title() lipgloss.Style { return s.title }

// Muted styles secondary text.
func (s Styles) Muted() lipgloss.Style { return s.muted }

// Accent styles the emphasised value of a row.
func (s Styles) Accent() lipgloss.Style { return s.accent }

// Focus styles the focused element.
func (s Styles) Focus() lipgloss.Style { return s.focus }

// Selected styles the highlighted palette row.
func (s Styles) Selected() lipgloss.Style { return s.selected }

// Prompt styles an approval question.
func (s Styles) Prompt() lipgloss.Style { return s.prompt }

// Panel frames a region.
func (s Styles) Panel() lipgloss.Style { return s.panel }

// Statusbar styles the bottom line.
func (s Styles) Statusbar() lipgloss.Style { return s.statusbar }

// Highlight marks the matched span of a palette row. It returns the text
// unchanged when the query is not a contiguous substring — a fuzzy match has no
// single span to highlight, and inventing one would misreport why the row
// matched.
func (s Styles) Highlight(text, query string) string {
	if query == "" || !s.palette.Paints(theme.Accent) {
		return text
	}
	idx := strings.Index(strings.ToLower(text), strings.ToLower(query))
	if idx < 0 {
		return text
	}
	before, match, after := text[:idx], text[idx:idx+len(query)], text[idx+len(query):]
	return before + s.accent.Render(match) + after
}
