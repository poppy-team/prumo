package dialog

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// ModelSelectedMsg reports the model the user chose.
type ModelSelectedMsg struct{ Model string }

// CloseModelDialogMsg asks the shell to stop showing the dialog.
type CloseModelDialogMsg struct{}

// LoadingModelsMsg indicates that models are currently being fetched.
type LoadingModelsMsg struct{}

// ModelsLoadedMsg carries what the harness said a provider can serve.
type ModelsLoadedMsg struct {
	Info   map[string]runtime.ModelCapabilities
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
	// info is what each model declares it can do, by id. A model with no entry
	// is one nobody declared — not one that was measured and found wanting.
	info    map[string]runtime.ModelCapabilities
	cursor  int
	err     error
	loading bool
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
	case LoadingModelsMsg:
		m.loading = true
		m.err = nil
		return m, nil
	case ModelsLoadedMsg:
		m.loading = false
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

// features renders what a model declares, or says plainly that nobody declared
// anything about it.
func features(info runtime.ModelCapabilities) string {
	if !info.Declared {
		return "(not declared)"
	}
	if len(info.Features) == 0 {
		return "(declared, nothing claimed)"
	}
	return "[" + strings.Join(info.Features, " ") + "]"
}

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
	case m.loading:
		rows = append(rows, baseStyle.Foreground(t.TextMuted()).Width(width).
			Render("Loading models from provider (may take 10-20 seconds)..."))
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
			rows = append(rows, row.Render(model+" "+features(m.info[model])))
		}
	}

	// The legend exists because a model whose features nobody declared must not
	// read as a model without features.
	hint := baseStyle.Foreground(t.TextMuted()).Width(width).
		Render("text reason vision tools audio · \"not declared\" = nobody said")

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
