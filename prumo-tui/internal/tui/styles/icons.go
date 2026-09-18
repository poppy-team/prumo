package styles

import (
	"os"
	"strings"
)

// The glyphs the client draws with.
//
// They are variables rather than constants because the terminal decides which
// set is legible: a glyph a font cannot render arrives as a replacement
// character, which is noise where a label was meant. The set is resolved once,
// at start-up, because the answer cannot change while the client runs.
var (
	PrumoIcon    = "◆"
	CheckIcon    = "✓"
	ErrorIcon    = "✖"
	WarningIcon  = "⚠"
	InfoIcon     = ""
	HintIcon     = "i"
	SpinnerIcon  = "..."
	LoadingIcon  = "⟳"
	DocumentIcon = "🖼"

	// TruncationMarker ends a line that lost its tail. A row that simply stops
	// is indistinguishable from a row that ended, so the cut has to be visible.
	TruncationMarker = "…"
)

// UseASCII swaps every glyph for one that any terminal can render.
//
// The replacements carry the same meaning in words rather than shapes, because
// a fallback that is still ambiguous is not a fallback: "!" for a warning and
// "ok" for a success read without knowing the original set.
func UseASCII() {
	PrumoIcon = "*"
	CheckIcon = "ok"
	ErrorIcon = "x"
	WarningIcon = "!"
	InfoIcon = "i"
	HintIcon = "?"
	SpinnerIcon = "..."
	LoadingIcon = "..."
	DocumentIcon = "file"
	TruncationMarker = ".."
}

// UseUnicode restores the default glyph set, which is what a test that swaps in
// the ASCII one has to put back.
func UseUnicode() {
	PrumoIcon = "◆"
	CheckIcon = "✓"
	ErrorIcon = "✖"
	WarningIcon = "⚠"
	InfoIcon = ""
	HintIcon = "i"
	SpinnerIcon = "..."
	LoadingIcon = "⟳"
	DocumentIcon = "🖼"
	TruncationMarker = "…"
}

// ResolveIcons picks the glyph set this terminal can render.
//
// The locale is the signal it can actually give: it states the terminal's
// encoding, and a UTF-8 locale is what a font with these glyphs comes with.
// Nothing set is the C locale, which is ASCII by definition — so an unset
// environment draws the fallback rather than glyphs that may arrive as boxes.
func ResolveIcons() {
	if localeIsUTF8() {
		return
	}
	UseASCII()
}

func localeIsUTF8() bool {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		value := strings.ToUpper(os.Getenv(key))
		if value == "" {
			continue
		}
		return strings.Contains(value, "UTF-8") || strings.Contains(value, "UTF8")
	}
	return false
}
