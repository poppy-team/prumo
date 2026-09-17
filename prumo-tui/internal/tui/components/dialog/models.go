package dialog

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// ModelSelectedMsg reports the model the user chose.
type ModelSelectedMsg struct{ Model string }

// CloseModelDialogMsg asks the shell to stop showing the dialog.
type CloseModelDialogMsg struct{}

// ModelsLoadedMsg carries what the harness said a provider can serve.
type ModelsLoadedMsg struct {
	Models []string
	Err    error
}

// ModelDialog chooses which model the harness should run with.
//
// The list is not the client's. It comes from the harness, which asked the
// provider — a picker with a baked-in catalogue would offer models the endpoint
// may not serve, and would have to be edited whenever a vendor shipped one.
type ModelDialog interface {
	tea.Model
	SetSize(width, height int) tea.Cmd
}

type modelDialogCmp struct {
	width, height int
	models        []string
	cursor        int
	err           error
}

// NewModelDialogCmp returns an empty picker; the list arrives as a message.
func NewModelDialogCmp() ModelDialog { return &modelDialogCmp{} }

func (m *modelDialogCmp) Init() tea.Cmd { return nil }

// SetSize records the space the dialog may use.
func (m *modelDialogCmp) SetSize(width, height int) tea.Cmd {
	m.width, m.height = width, height
	return nil
}

func (m *modelDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ModelsLoadedMsg:
		m.err = msg.Err
		m.models = msg.Models
		m.cursor = 0
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "ctrl+o", "q":
			return m, util.CmdHandler(CloseModelDialogMsg{})
		case "up", "k", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j", "ctrl+n":
			if m.cursor < len(m.models)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.models) {
				return m, util.CmdHandler(ModelSelectedMsg{Model: m.models[m.cursor]})
			}
		}
	}
	return m, nil
}

// View renders the component for the terminal.
func (m *modelDialogCmp) View() tea.View { return tea.NewView(m.viewString()) }
func (m *modelDialogCmp) viewString() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	width := 50
	if m.width > 0 && m.width-10 < width {
		width = max(20, m.width-10)
	}

	title := baseStyle.
		Bold(true).
		Foreground(t.Primary()).
		Width(width).
		Render("Model")

	rows := make([]string, 0, len(m.models)+1)
	switch {
	case m.err != nil:
		rows = append(rows, baseStyle.Foreground(t.Error()).Width(width).
			Render("Could not read the model list: "+m.err.Error()))
	case len(m.models) == 0:
		rows = append(rows, baseStyle.Foreground(t.TextMuted()).Width(width).
			Render("The harness reported no models for this provider."))
	default:
		for i, model := range m.models {
			row := baseStyle.Width(width).Padding(0, 1)
			if i == m.cursor {
				row = row.Background(t.Primary()).Foreground(t.Background()).Bold(true)
			}
			rows = append(rows, row.Render(model))
		}
	}

	hint := baseStyle.Foreground(t.TextMuted()).Width(width).
		Render("enter select · esc close")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		baseStyle.Render(strings.Repeat(" ", width)),
		lipgloss.JoinVertical(lipgloss.Left, rows...),
		baseStyle.Render(strings.Repeat(" ", width)),
		hint,
	)

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.TextMuted()).
		Width(width + 4).
		Render(content)
}
