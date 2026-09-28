package dialog

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

type CloseQuitMsg struct{}

// QuitWithSaveMsg indicates that the session should be saved with the specified title before quitting.
type QuitWithSaveMsg struct {
	SessionID string
	Title     string
}

type QuitDialog interface {
	tea.Model
	layout.Bindings
	SetSession(id string, title string)
}

type quitDialogCmp struct {
	sessionID   string
	input       textinput.Model
	selectedBtn int // 0: Save & Quit, 1: Quit, 2: Cancel
	focusInput  bool
}

type quitKeyMap struct {
	LeftRight key.Binding
	UpDown    key.Binding
	Enter     key.Binding
	Esc       key.Binding
	Tab       key.Binding
	Save      key.Binding
	Quit      key.Binding
}

var quitKeys = quitKeyMap{
	LeftRight: key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←/→", "switch buttons"),
	),
	UpDown: key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑/↓", "toggle input/buttons"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "next field/button"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save & quit"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "quit without saving"),
	),
}

func (q *quitDialogCmp) Init() tea.Cmd {
	return textinput.Blink
}

func (q *quitDialogCmp) SetSession(id string, title string) {
	q.sessionID = id
	if title == "" {
		title = "New Session"
	}
	q.input.SetValue(title)
	q.input.CursorEnd()
	q.focusInput = true
	q.input.Focus()
	q.selectedBtn = 0
}

func (q *quitDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, quitKeys.Esc):
			return q, util.CmdHandler(CloseQuitMsg{})

		case key.Matches(msg, quitKeys.Save):
			title := q.input.Value()
			if title == "" {
				title = "New Session"
			}
			return q, util.CmdHandler(QuitWithSaveMsg{SessionID: q.sessionID, Title: title})

		case key.Matches(msg, quitKeys.Quit):
			return q, tea.Quit
		}

		if q.focusInput {
			switch {
			case key.Matches(msg, quitKeys.UpDown):
				q.focusInput = false
				q.input.Blur()
				return q, nil
			case key.Matches(msg, quitKeys.Tab):
				q.focusInput = false
				q.input.Blur()
				return q, nil
			case key.Matches(msg, quitKeys.Enter):
				title := q.input.Value()
				if title == "" {
					title = "New Session"
				}
				return q, util.CmdHandler(QuitWithSaveMsg{SessionID: q.sessionID, Title: title})
			default:
				var cmd tea.Cmd
				q.input, cmd = q.input.Update(msg)
				return q, cmd
			}
		}

		// Button navigation mode
		switch {
		case key.Matches(msg, quitKeys.UpDown):
			q.focusInput = true
			q.input.Focus()
			return q, textinput.Blink

		case key.Matches(msg, quitKeys.LeftRight):
			if msg.String() == "left" {
				q.selectedBtn = (q.selectedBtn + 2) % 3
			} else {
				q.selectedBtn = (q.selectedBtn + 1) % 3
			}
			return q, nil

		case key.Matches(msg, quitKeys.Tab):
			if msg.String() == "shift+tab" {
				q.selectedBtn = (q.selectedBtn + 2) % 3
			} else {
				q.selectedBtn = (q.selectedBtn + 1) % 3
			}
			return q, nil

		case key.Matches(msg, quitKeys.Enter) || msg.String() == " ":
			switch q.selectedBtn {
			case 0: // Save & Quit
				title := q.input.Value()
				if title == "" {
					title = "New Session"
				}
				return q, util.CmdHandler(QuitWithSaveMsg{SessionID: q.sessionID, Title: title})
			case 1: // Quit
				return q, tea.Quit
			case 2: // Cancel
				return q, util.CmdHandler(CloseQuitMsg{})
			}

		case msg.String() == "s" || msg.String() == "y":
			title := q.input.Value()
			if title == "" {
				title = "New Session"
			}
			return q, util.CmdHandler(QuitWithSaveMsg{SessionID: q.sessionID, Title: title})

		case msg.String() == "q":
			return q, tea.Quit

		case msg.String() == "n":
			return q, util.CmdHandler(CloseQuitMsg{})
		}

	default:
		if q.focusInput {
			var cmd tea.Cmd
			q.input, cmd = q.input.Update(msg)
			return q, cmd
		}
	}
	return q, nil
}

// View renders the component for the terminal.
func (q *quitDialogCmp) View() tea.View { return tea.NewView(q.viewString()) }
func (q *quitDialogCmp) viewString() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	const dialogWidth = 48

	titleHeader := baseStyle.Bold(true).Foreground(t.Primary()).Render("Quit Session")
	questionText := baseStyle.Foreground(t.Text()).Render("Save conversation before quitting?")

	inputLabel := baseStyle.Foreground(t.TextMuted()).Render("Session name (editable):")

	inputBorderColor := t.TextMuted()
	if q.focusInput {
		inputBorderColor = t.Primary()
	}
	inputBox := baseStyle.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorderColor).
		Width(dialogWidth-4).
		Padding(0, 1).
		Render(q.input.View())

	// Buttons
	btnLabels := []string{"Save & Quit", "Quit", "Cancel"}
	renderedBtns := make([]string, len(btnLabels))

	for i, label := range btnLabels {
		btnStyle := baseStyle.Padding(0, 1)
		if !q.focusInput && q.selectedBtn == i {
			btnStyle = btnStyle.Background(t.Primary()).Foreground(t.Background()).Bold(true)
		} else {
			btnStyle = btnStyle.Background(t.BackgroundSecondary()).Foreground(t.Text())
		}
		renderedBtns[i] = btnStyle.Render(label)
	}

	btnRow := lipgloss.JoinHorizontal(lipgloss.Left,
		renderedBtns[0],
		"  ",
		renderedBtns[1],
		"  ",
		renderedBtns[2],
	)

	// Clean right alignment on a single line
	alignedButtons := baseStyle.Width(dialogWidth - 2).Align(lipgloss.Right).Render(btnRow)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleHeader,
		"",
		questionText,
		"",
		inputLabel,
		inputBox,
		"",
		alignedButtons,
	)

	return baseStyle.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.Primary()).
		Render(content)
}

func (q *quitDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(quitKeys)
}

func NewQuitCmp() QuitDialog {
	ti := textinput.New()
	ti.Placeholder = "Session title..."
	ti.CharLimit = 64
	ti.SetWidth(40)
	ti.Focus()

	return &quitDialogCmp{
		input:       ti,
		focusInput:  true,
		selectedBtn: 0,
	}
}
