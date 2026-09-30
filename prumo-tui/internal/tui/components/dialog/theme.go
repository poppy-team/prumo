package dialog

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// ThemeChangedMsg is sent when the theme is changed
type ThemeChangedMsg struct {
	ThemeName string
}

// CloseThemeDialogMsg is sent when the theme dialog is closed
type CloseThemeDialogMsg struct{}

// ThemeDialog interface for the theme switching dialog
type ThemeDialog interface {
	tea.Model
	layout.Bindings
}

type themeDialogCmp struct {
	themes        []string
	selectedIdx   int
	width         int
	height        int
	originalTheme string
}

type themeKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
	J      key.Binding
	K      key.Binding
}

var themeKeys = themeKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous theme"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next theme"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "apply theme"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "revert & close"),
	),
	J: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "next theme"),
	),
	K: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "previous theme"),
	),
}

func (t *themeDialogCmp) Init() tea.Cmd {
	t.themes = theme.AvailableThemes()
	t.originalTheme = theme.CurrentThemeName()

	for i, name := range t.themes {
		if name == t.originalTheme {
			t.selectedIdx = i
			break
		}
	}

	return nil
}

func (t *themeDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, themeKeys.Up) || key.Matches(msg, themeKeys.K):
			if t.selectedIdx > 0 {
				t.selectedIdx--
				_ = theme.PreviewTheme(t.themes[t.selectedIdx])
			}
			return t, nil

		case key.Matches(msg, themeKeys.Down) || key.Matches(msg, themeKeys.J):
			if t.selectedIdx < len(t.themes)-1 {
				t.selectedIdx++
				_ = theme.PreviewTheme(t.themes[t.selectedIdx])
			}
			return t, nil

		case key.Matches(msg, themeKeys.Enter):
			if len(t.themes) > 0 {
				selectedTheme := t.themes[t.selectedIdx]
				if err := theme.SetTheme(selectedTheme); err != nil {
					return t, util.ReportError(err)
				}
				return t, util.CmdHandler(ThemeChangedMsg{
					ThemeName: selectedTheme,
				})
			}

		case key.Matches(msg, themeKeys.Escape):
			if t.originalTheme != "" {
				_ = theme.PreviewTheme(t.originalTheme)
			}
			return t, util.CmdHandler(CloseThemeDialogMsg{})
		}

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	}
	return t, nil
}

// View renders the component for the terminal.
func (t *themeDialogCmp) View() tea.View { return tea.NewView(t.viewString()) }
func (t *themeDialogCmp) viewString() string {
	currentTheme := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	if len(t.themes) == 0 {
		return baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(currentTheme.Background()).
			BorderForeground(currentTheme.TextMuted()).
			Render("No themes available")
	}

	const dialogWidth = 44

	title := baseStyle.
		Foreground(currentTheme.Primary()).
		Bold(true).
		Width(dialogWidth).
		Padding(0, 1).
		Render("Select Theme")

	// Build the theme list with real-time palette swatches
	themeItems := make([]string, 0, len(t.themes))
	for i, themeName := range t.themes {
		selected := i == t.selectedIdx
		isOriginal := themeName == t.originalTheme

		th := theme.GetTheme(themeName)
		swatches := ""
		if th != nil {
			s1 := lipgloss.NewStyle().Foreground(th.Primary()).Render("■")
			s2 := lipgloss.NewStyle().Foreground(th.Secondary()).Render("■")
			s3 := lipgloss.NewStyle().Foreground(th.Accent()).Render("■")
			swatches = fmt.Sprintf("%s %s %s", s1, s2, s3)
		}

		cursorPrefix := "  "
		if selected {
			cursorPrefix = "▸ "
		} else if isOriginal {
			cursorPrefix = "● "
		}

		nameLabel := cursorPrefix + themeName
		remainingSpaces := max(1, dialogWidth-lipgloss.Width(nameLabel)-lipgloss.Width(swatches)-3)
		rowContent := nameLabel + strings.Repeat(" ", remainingSpaces) + swatches

		itemStyle := baseStyle.Width(dialogWidth).Padding(0, 1)
		if selected {
			itemStyle = itemStyle.
				Background(currentTheme.Primary()).
				Foreground(currentTheme.Background()).
				Bold(true)
		} else {
			itemStyle = itemStyle.Foreground(currentTheme.Text())
		}

		themeItems = append(themeItems, itemStyle.Render(rowContent))
	}

	hint := baseStyle.
		Foreground(currentTheme.TextMuted()).
		Width(dialogWidth).
		Padding(1, 1, 0, 1).
		Render("↑/↓ live preview · enter apply · esc revert")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.JoinVertical(lipgloss.Left, themeItems...),
		hint,
	)

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(currentTheme.Background()).
		BorderForeground(currentTheme.Primary()).
		Render(content)
}

func (t *themeDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(themeKeys)
}

// NewThemeDialogCmp creates a new theme switching dialog
func NewThemeDialogCmp() ThemeDialog {
	return &themeDialogCmp{
		themes:      []string{},
		selectedIdx: 0,
	}
}
