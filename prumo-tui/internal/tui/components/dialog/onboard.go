package dialog

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/raillen/prumo-tui/internal/onboard"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// OnboardChoiceMsg is the answer to the first-run offer.
type OnboardChoiceMsg struct {
	ID string
}

// OnboardInstallFinishedMsg reports how the installer went.
//
// The output travels with it because an installer that fails is a person who
// needs to see why, not a person who needs to see "failed".
type OnboardInstallFinishedMsg struct {
	Err    error
	Output string
}

// OnboardDialogCmp is the first-run offer.
//
// It exists because of one asymmetry: opencode serves models for free and
// authenticates itself, so someone who has it needs no key — and someone who
// does not would never guess that from an empty model list. Every option says
// what it will do before it is chosen.
type OnboardDialogCmp interface {
	tea.Model
	layout.Bindings
}

type onboardDialogCmp struct {
	options []onboard.Option
	cursor  int
	width   int
}

var onboardKeys = struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Esc   key.Binding
}{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "previous option")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "next option")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")),
	Esc:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "decide later")),
}

func NewOnboardDialogCmp(options []onboard.Option) OnboardDialogCmp {
	return &onboardDialogCmp{options: options, width: 76}
}

func (o *onboardDialogCmp) Init() tea.Cmd { return nil }

func (o *onboardDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.width = min(76, max(40, msg.Width-4))
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, onboardKeys.Up):
			if o.cursor > 0 {
				o.cursor--
			}
		case key.Matches(msg, onboardKeys.Down):
			if o.cursor < len(o.options)-1 {
				o.cursor++
			}
		case key.Matches(msg, onboardKeys.Enter):
			if len(o.options) > 0 {
				return o, util.CmdHandler(OnboardChoiceMsg{ID: o.options[o.cursor].ID})
			}
		case key.Matches(msg, onboardKeys.Esc):
			// Dismissing is an answer, and it is the "not now" one: a dialog
			// that cannot be left is a dialog that blocks the work it offered
			// to help with.
			return o, util.CmdHandler(OnboardChoiceMsg{ID: "later"})
		}
	}
	return o, nil
}

func (o *onboardDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()
	title := base.Foreground(t.Primary()).Bold(true).Render("Which models should Prumo run with?")
	intro := base.Foreground(t.TextMuted()).Render(
		"No provider is configured yet. opencode serves models for free and authenticates\n" +
			"itself, so it needs no api-key from us.")

	var lines []string
	lines = append(lines, title, "", intro, "")
	for i, opt := range o.options {
		marker := base.Render("  ")
		label := base.Foreground(t.Text()).Render(opt.Label)
		detail := base.Foreground(t.TextMuted()).Render("     " + opt.Detail)
		if i == o.cursor {
			marker = base.Foreground(t.Primary()).Bold(true).Render("> ")
			label = base.Foreground(t.Primary()).Bold(true).Render(opt.Label)
		}
		if opt.Automatic {
			label += base.Foreground(t.TextMuted()).Render("  (automatic)")
		}
		lines = append(lines, marker+label, detail, "")
	}
	lines = append(lines, base.Foreground(t.TextMuted()).Render("↑/↓ choose · enter confirm · esc decide later"))

	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocused()).
		Padding(1, 2).
		Width(o.width).
		Render(strings.Join(lines, "\n"))

	return tea.NewView(body)
}

func (o *onboardDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(onboardKeys)
}
