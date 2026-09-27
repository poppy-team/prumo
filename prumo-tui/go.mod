// Module prumo-tui is the Prumo terminal client.
//
// It is a client of the Prumo harness, never part of it: it reaches the daemon
// through the public SDK (github.com/raillen/prumo/sdk/prumo) and through the
// `prumo` binary it supervises as a subprocess. The view layer is derived from
// the archived OpenCode Go implementation (MIT) — see THIRD_PARTY_NOTICES.md
// and docs/adr/013-tui-foundation.md.
//
// The core module does not depend on this one. This module carries the heavy
// terminal dependencies precisely so the core does not have to.
module github.com/raillen/prumo-tui

go 1.26.6

require (
	charm.land/bubbles/v2 v2.2.1
	charm.land/bubbletea/v2 v2.0.9
	charm.land/glamour/v2 v2.0.1
	charm.land/lipgloss/v2 v2.0.6
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/aymanbagabas/go-udiff v0.4.1
	github.com/bmatcuk/doublestar/v4 v4.10.0
	github.com/catppuccin/go v0.3.0
	github.com/charmbracelet/x/ansi v0.11.8
	github.com/disintegration/imaging v1.6.2
	github.com/go-logfmt/logfmt v0.6.1
	github.com/google/uuid v1.6.0
	github.com/lithammer/fuzzysearch v1.1.8
	github.com/lucasb-eyer/go-colorful v1.4.1
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6
	github.com/muesli/reflow v0.3.0
	github.com/muesli/termenv v0.16.0
	github.com/raillen/prumo v0.0.0
	github.com/sergi/go-diff v1.4.0
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260811164956-006e29f97886 // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20250327172914-2fdc97757edf // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.27 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/goldmark v1.7.13 // indirect
	github.com/yuin/goldmark-emoji v1.0.6 // indirect
	golang.org/x/image v0.0.0-20191009234506-e7c1f5e7dbb8 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)

replace github.com/raillen/prumo => ../
