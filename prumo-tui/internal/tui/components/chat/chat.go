package chat

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/version"
)

// SendMsg is a goal the composer wants sent.
//
// It carries text and nothing else: a file is *named* in the goal rather than
// attached to it, because what a model can see is the harness's decision.
type SendMsg struct {
	Text string
}

type SessionSelectedMsg = session.Session

type SessionClearedMsg struct{}

type EditorFocusMsg bool

func header(width int) string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		logo(width),
		repo(width),
		"",
		cwd(width),
	)
}

// lspsConfigured used to list the language servers the client would start.
//
// It does not any more, and not because the panel was dropped: the client never
// started a language server, the harness runs the language tooling behind its
// own tools. A panel listing servers this process does not own would be a claim
// the client cannot back.
func lspsConfigured(width int) string {
	return ""
}

func logo(width int) string {
	logo := fmt.Sprintf("%s %s", styles.PrumoIcon, "Prumo")
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	versionText := baseStyle.
		Foreground(t.TextMuted()).
		Render(version.Version)

	return baseStyle.
		Bold(true).
		Width(width).
		Render(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				logo,
				" ",
				versionText,
			),
		)
}

func repo(width int) string {
	repo := "https://github.com/raillen/prumo"
	t := theme.CurrentTheme()

	return styles.BaseStyle().
		Foreground(t.TextMuted()).
		Width(width).
		Render(repo)
}

func cwd(width int) string {
	cwd := fmt.Sprintf("cwd: %s", config.WorkingDirectory())
	t := theme.CurrentTheme()

	return styles.BaseStyle().
		Foreground(t.TextMuted()).
		Width(width).
		Render(cwd)
}
