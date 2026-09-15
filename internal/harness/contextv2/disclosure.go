package contextv2

import (
	"strings"

	"github.com/raillen/prumo/internal/harness/model"
)

// Level is a progressive-disclosure level: it decides how much of an item's
// content a compilation is allowed to reveal (W4.4).
//
//	L0  pointer   the reference only; no content at all
//	L1  summary   the item's first meaningful line
//	L2  outline   every heading of the item
//	L3  excerpt   the opening block of the item, line-capped
//	L4  full      the whole item
//
// The levels are monotonic: for the same item, a higher level never reveals
// less text than a lower one.
type Level string

const (
	LevelPointer Level = "L0"
	LevelSummary Level = "L1"
	LevelOutline Level = "L2"
	LevelExcerpt Level = "L3"
	LevelFull    Level = "L4"
)

// DefaultLevel is used when a caller does not choose one. It is deliberately
// the cheapest level that still shows the reader something.
const DefaultLevel = LevelSummary

// excerptLines bounds L3 so the level has a fixed, predictable cost.
const excerptLines = 40

// Levels lists every level from cheapest to most expensive.
func Levels() []Level {
	return []Level{LevelPointer, LevelSummary, LevelOutline, LevelExcerpt, LevelFull}
}

// ParseLevel maps a caller-supplied string to a level. An unknown value falls
// back to DefaultLevel rather than failing a compilation: the level is a
// presentation choice, not a correctness claim.
func ParseLevel(s string) Level {
	switch Level(strings.TrimSpace(strings.ToUpper(s))) {
	case LevelPointer:
		return LevelPointer
	case LevelSummary:
		return LevelSummary
	case LevelOutline:
		return LevelOutline
	case LevelExcerpt:
		return LevelExcerpt
	case LevelFull:
		return LevelFull
	}
	return DefaultLevel
}

// Valid reports whether the level is one of the five defined levels.
func (l Level) Valid() bool {
	for _, known := range Levels() {
		if l == known {
			return true
		}
	}
	return false
}

// Rank orders the levels; an unknown level ranks as the default.
func (l Level) Rank() int {
	for i, known := range Levels() {
		if l == known {
			return i
		}
	}
	return 1
}

// Disclose returns the content a level is allowed to reveal and whether that is
// the item's full content. The second result lets a caller report honestly that
// a level truncated an item instead of implying completeness.
func (l Level) Disclose(content string) (string, bool) {
	if strings.TrimSpace(content) == "" {
		return "", true
	}
	switch l {
	case LevelPointer:
		return "", false
	case LevelOutline:
		headings := headingsOf(content)
		if len(headings) == 0 {
			return l.summary(content), false
		}
		return strings.Join(headings, "\n"), false
	case LevelExcerpt:
		lines := strings.Split(content, "\n")
		if len(lines) <= excerptLines {
			return content, true
		}
		return strings.Join(lines[:excerptLines], "\n"), false
	case LevelFull:
		return content, true
	default: // LevelSummary and anything unrecognized
		return l.summary(content), false
	}
}

// summary is the first meaningful line: leading blanks are skipped so the level
// never discloses an empty prefix.
func (Level) summary(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// headingsOf collects Markdown ATX headings in document order.
func headingsOf(content string) []string {
	out := []string{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			out = append(out, trimmed)
		}
	}
	return out
}

// DiscloseItem applies the level to one item and records the resulting cost, so
// a manifest's token accounting matches what was actually revealed.
//
// Cost rule: L0 is a pointer and pays for its reference only. An item that
// carries no content cannot be disclosed less than it already is, so its
// producer's own estimate stands. Otherwise the cost is the reference plus the
// text the level actually revealed.
func DiscloseItem(it Item, level Level) Item {
	content, full := level.Disclose(it.Content)
	it.Content = content
	it.Level = level
	it.Truncated = !full
	switch {
	case level == LevelPointer:
		it.TokenCost = model.EstimateTokens(it.Ref, "")
	case it.Content == "":
		// keep the producer's estimate
	default:
		it.TokenCost = model.EstimateTokens(it.Ref, "") + model.EstimateTokens(content, "")
	}
	return it
}
