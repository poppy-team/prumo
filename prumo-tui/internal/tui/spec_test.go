package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// This file holds the specification to its own subject.
//
// `docs/ui-ux/interaction.md` is the canonical keyboard map, and until this test
// existed nothing checked it against the client: the document described a
// different interface for as long as it took someone to read both. The map is
// now verified in both directions — a chord the client answers without a row, and
// a row for a chord no binding answers, both fail.

var chordSpan = regexp.MustCompile("`([^`]+)`")

// chordAliases are the ways the map spells a chord the client matches under
// another name: the document is written for a reader and the terminal for a
// program, and the two spell the same key differently.
var chordAliases = map[string]string{
	"pgdn":  "pgdown",
	"space": " ",
}

func normalizeChord(chord string) string {
	c := strings.ToLower(strings.TrimSpace(chord))
	c = strings.ReplaceAll(c, " ", "")
	if alias, ok := chordAliases[c]; ok {
		return alias
	}
	return c
}

// shellChords are the chords the shell itself binds, read from the bindings so
// that adding one without documenting it fails here.
func shellChords() map[string]string {
	chords := map[string]string{}
	add := func(owner string, keys ...string) {
		for _, k := range keys {
			chords[normalizeChord(k)] = owner
		}
	}
	add("shell: cancel", keys.Cancel.Keys()...)
	add("shell: quit", keys.Quit.Keys()...)
	add("shell: help", keys.Help.Keys()...)
	add("shell: sessions", keys.SwitchSession.Keys()...)
	add("shell: commands", keys.Commands.Keys()...)
	add("shell: files", keys.Filepicker.Keys()...)
	add("shell: models", keys.Models.Keys()...)
	add("shell: theme", keys.SwitchTheme.Keys()...)
	add("shell: logs", keys.Logs.Keys()...)
	add("shell: changed files", keys.ChangedFiles.Keys()...)
	add("shell: sidebar", keys.Sidebar.Keys()...)
	add("shell: dismiss", returnKey.Keys()...)
	add("shell: leave the log page", logsKeyReturnKey.Keys()...)
	add("shell: toggle the keymap", helpEsc.Keys()...)
	return chords
}

// componentChords are the chords the embedded surfaces answer, named with the
// file that binds them.
//
// They are listed rather than read because the bindings live in components that
// keep them unexported — the imported view layer's own convention — and moving
// them for a test would be a worse trade than a reviewed list. The list is
// short, and every entry names where it comes from.
var componentChords = map[string]string{
	"ctrl+n":    "page/chat.go: start a new session",
	"@":         "page/chat.go: complete a path",
	"/":         "page/chat.go: slash command completion",
	"ctrl+s":    "components/chat/editor.go: send the message",
	"enter":     "components/chat/editor.go: send the message",
	"ctrl+e":    "components/chat/editor.go: open the goal in $EDITOR",
	"pgup":      "components/chat/list.go: scroll up a page",
	"b":         "components/chat/list.go: scroll up a page",
	"pgdown":    "components/chat/list.go: scroll down a page",
	"f":         "components/chat/list.go: scroll down a page",
	"ctrl+u":    "components/chat/list.go: half a page up",
	"ctrl+d":    "components/chat/list.go: half a page down",
	"esc":       "page/chat.go, dialogs: dismiss the topmost layer",
	"q":         "shell: leave the log page",
	"a":         "components/dialog/permission.go: approve",
	"s":         "components/dialog/permission.go: approve for the session",
	"d":         "components/dialog/permission.go: deny",
	"y":         "components/dialog/quit.go: confirm leaving",
	"n":         "components/dialog/quit.go: decline leaving",
	"up":        "dialogs and lists: move the selection",
	"down":      "dialogs and lists: move the selection",
	"j":         "components/dialog/filepicker.go, session.go: move the selection",
	"k":         "components/dialog/filepicker.go, session.go: move the selection",
	"h":         "components/dialog/filepicker.go: go up a directory",
	"l":         "components/dialog/filepicker.go: enter a directory",
	"left":      "components/dialog/quit.go, permission.go: move the selection",
	"right":     "components/dialog/quit.go, permission.go: move the selection",
	"tab":       "components/dialog/quit.go, permission.go, complete.go: move the selection",
	"backspace": "components/dialog/filepicker.go: go up a directory",
	" ":         "components/dialog/quit.go, permission.go: accept the selection",
	"i":         "components/dialog/filepicker.go: toggle hidden files",
	"shift+tab": "components/dialog/init.go: move the selection",
}

// documentedChords reads the chord column of the keyboard map.
func documentedChords(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "ui-ux", "interaction.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the interaction specification is what this test checks: %v", err)
	}

	chords := map[string]string{}
	inMap := false
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "## Keyboard map"):
			inMap = true
			continue
		case inMap && strings.HasPrefix(line, "## "):
			inMap = false
		}
		if !inMap || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			continue
		}
		action := strings.TrimSpace(cells[2])
		for _, span := range chordSpan.FindAllStringSubmatch(cells[1], -1) {
			chords[normalizeChord(span[1])] = action
		}
	}
	if len(chords) == 0 {
		t.Fatal("the keyboard map lists no chord, which cannot be right")
	}
	return chords
}

// boundChords is everything the client answers.
func boundChords() map[string]string {
	bound := shellChords()

	for chord, owner := range componentChords {
		bound[chord] = owner
	}
	return bound
}

func TestDocumentedKeymapMatchesTheBindings(t *testing.T) {
	documented := documentedChords(t)
	bound := boundChords()

	for chord, owner := range bound {
		if _, ok := documented[chord]; !ok {
			t.Errorf("the client binds %q (%s) and the keyboard map does not list it", chord, owner)
		}
	}
	for chord, action := range documented {
		if _, ok := bound[chord]; !ok {
			t.Errorf("the keyboard map lists %q (%s) and no binding answers it", chord, action)
		}
	}
}

// The two reserved chords are the exception the contract does not let a surface
// negotiate, so they are asserted against the document itself rather than only
// against the bindings.
func TestTheReservedChordsAreTheDocumentedOnes(t *testing.T) {
	for chord, wanted := range map[string]string{
		"ctrl+c": "Cancel the run in flight",
		"ctrl+q": "Ask before quitting",
	} {
		action, ok := documentedChords(t)[chord]
		if !ok {
			t.Errorf("the map does not document the reserved chord %q", chord)
			continue
		}
		if !strings.Contains(action, wanted) {
			t.Errorf("the reserved chord %q is documented as %q", chord, action)
		}
	}
}
