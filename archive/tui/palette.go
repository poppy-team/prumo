//go:build ignore

package tui

import (
	"sort"
	"strings"
)

// Command is one action the palette can run. The palette decides *which*
// command applies; it never runs anything itself, so ranking is testable
// without a daemon.
type Command struct {
	ID          string
	Title       string
	Description string
	// Keywords are extra terms that should match without appearing in the
	// title, so "quit" can also be found by typing "exit".
	Keywords []string
}

// Palette is a fuzzy-filtered, keyboard-navigable command list.
type Palette struct {
	all      []Command
	query    string
	visible  []Match
	cursor   int
	capacity int
}

// Match pairs a command with the score that ranked it. Score exposes *why* a
// command ranked where it did, which is what makes the ranking arguable in a
// test instead of a matter of taste.
type Match struct {
	Command Command
	Score   int
}

// DefaultPaletteCapacity bounds how many matches are shown at once.
const DefaultPaletteCapacity = 8

// NewPalette builds a palette over commands, ordered by title.
func NewPalette(commands []Command) *Palette {
	all := append([]Command{}, commands...)
	sort.SliceStable(all, func(i, j int) bool { return all[i].Title < all[j].Title })
	p := &Palette{all: all, capacity: DefaultPaletteCapacity}
	p.refilter()
	return p
}

// SetQuery replaces the filter text and resets the cursor to the best match.
func (p *Palette) SetQuery(query string) {
	p.query = query
	p.refilter()
}

// Query returns the current filter text.
func (p *Palette) Query() string { return p.query }

// AppendRune extends the query by one character.
func (p *Palette) AppendRune(r rune) {
	p.SetQuery(p.query + string(r))
}

// Backspace removes the last character of the query.
func (p *Palette) Backspace() {
	if p.query == "" {
		return
	}
	runes := []rune(p.query)
	p.SetQuery(string(runes[:len(runes)-1]))
}

// Matches returns the visible matches, best first.
func (p *Palette) Matches() []Match { return append([]Match{}, p.visible...) }

// Cursor is the selected match index.
func (p *Palette) Cursor() int { return p.cursor }

// Move shifts the selection by delta, wrapping at both ends so navigation can
// never get stuck.
func (p *Palette) Move(delta int) {
	if len(p.visible) == 0 {
		p.cursor = 0
		return
	}
	p.cursor = (p.cursor + delta) % len(p.visible)
	if p.cursor < 0 {
		p.cursor += len(p.visible)
	}
}

// Selected returns the highlighted match, if the filter left any.
func (p *Palette) Selected() (Command, bool) {
	if len(p.visible) == 0 {
		return Command{}, false
	}
	return p.visible[p.cursor].Command, true
}

// Empty reports whether the query matched nothing — the state the view renders
// as "no command for this query" rather than as a blank panel.
func (p *Palette) Empty() bool { return len(p.visible) == 0 }

func (p *Palette) refilter() {
	query := strings.ToLower(strings.TrimSpace(p.query))
	if query == "" {
		p.visible = p.visible[:0]
		for _, cmd := range p.all {
			p.visible = append(p.visible, Match{Command: cmd})
		}
		p.truncate()
		p.cursor = 0
		return
	}
	matches := make([]Match, 0, len(p.all))
	for _, cmd := range p.all {
		score, ok := scoreCommand(cmd, query)
		if !ok {
			continue
		}
		matches = append(matches, Match{Command: cmd, Score: score})
	}
	// Highest score first; ties fall back to title order so the list never
	// reshuffles between two equally good matches.
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Command.Title < matches[j].Command.Title
	})
	p.visible = matches
	p.truncate()
	p.cursor = 0
}

func (p *Palette) truncate() {
	if len(p.visible) > p.capacity {
		p.visible = p.visible[:p.capacity]
	}
}

// scoreCommand ranks one command against a lowercased query.
//
// The ranking rewards a prefix match over a substring match over a subsequence
// match, and it prefers shorter titles among equals. That ordering is what makes
// typing "run" put "Run goal" above "Rerun last goal": both are subsequences,
// but only one starts with the query.
func scoreCommand(cmd Command, query string) (int, bool) {
	best := -1
	consider := func(term string, weight int) {
		term = strings.ToLower(term)
		if term == "" {
			return
		}
		score, ok := scoreTerm(term, query)
		if !ok {
			return
		}
		if total := score + weight; total > best {
			best = total
		}
	}
	consider(cmd.Title, 200)
	consider(cmd.Description, 0)
	for _, keyword := range cmd.Keywords {
		consider(keyword, 40)
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}

func scoreTerm(term, query string) (int, bool) {
	if strings.HasPrefix(term, query) {
		return 1000 - len(term), true
	}
	if idx := strings.Index(term, query); idx >= 0 {
		return 600 - idx*4 - len(term), true
	}
	if span, ok := subsequenceSpan(term, query); ok {
		return 300 - span, true
	}
	return 0, false
}

// subsequenceSpan reports whether query appears in term as an ordered
// subsequence, and over how many characters. A tighter span means a better
// match, which is what distinguishes "rg" → "Run goal" from "rg" → "Rerun a
// long goal".
func subsequenceSpan(term, query string) (int, bool) {
	queryRunes := []rune(query)
	if len(queryRunes) == 0 {
		return 0, true
	}
	first, last := -1, -1
	next := 0
	for i, r := range []rune(term) {
		if next >= len(queryRunes) {
			break
		}
		if r == queryRunes[next] {
			if first < 0 {
				first = i
			}
			last = i
			next++
		}
	}
	if next < len(queryRunes) {
		return 0, false
	}
	return last - first + 1, true
}
