package dialog

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
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
	filtered      []string
	// info is what each model declares it can do, by id. A model with no entry
	// is one nobody declared — not one that was measured and found wanting.
	info    map[string]runtime.ModelCapabilities
	cursor  int
	err     error
	loading bool
	search  textinput.Model
}

var (
	modelKeysUp     = key.NewBinding(key.WithKeys("up", "ctrl+p"))
	modelKeysDown   = key.NewBinding(key.WithKeys("down", "ctrl+n"))
	modelKeysPgUp   = key.NewBinding(key.WithKeys("pgup", "ctrl+u"))
	modelKeysPgDown = key.NewBinding(key.WithKeys("pgdown", "ctrl+d"))
	modelKeysEnter  = key.NewBinding(key.WithKeys("enter"))
	modelKeysEsc    = key.NewBinding(key.WithKeys("esc", "ctrl+o"))
)

func isFree(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, "-free") ||
		strings.HasSuffix(lower, ":free") ||
		strings.Contains(lower, "free")
}

func sortModels(raw []string) []string {
	var free []string
	var other []string
	for _, m := range raw {
		if isFree(m) {
			free = append(free, m)
		} else {
			other = append(other, m)
		}
	}
	return append(free, other...)
}

// NewModelDialogCmp returns an empty picker with real-time search.
func NewModelDialogCmp() ModelDialog {
	ti := textinput.New()
	ti.Placeholder = "Type to search models (e.g. free, nemotron, sonnet)..."
	ti.Prompt = "Search: "
	ti.Focus()
	return &modelDialogCmp{
		search: ti,
	}
}

func (m *modelDialogCmp) Init() tea.Cmd {
	m.search.Focus()
	return textinput.Blink
}

// SetSize records the space the dialog may use.
func (m *modelDialogCmp) SetSize(width, height int) tea.Cmd {
	m.width, m.height = width, height
	return nil
}

func (m *modelDialogCmp) applyFilter() {
	query := strings.TrimSpace(strings.ToLower(m.search.Value()))
	if query == "" {
		m.filtered = m.models
		m.cursor = 0
		return
	}
	var freeMatches []string
	var otherMatches []string
	for _, model := range m.models {
		searchTarget := strings.ToLower(model)
		if info, ok := m.info[model]; ok {
			searchTarget += " " + strings.ToLower(strings.Join(info.Features, " "))
		}
		if strings.Contains(searchTarget, query) {
			if isFree(model) {
				freeMatches = append(freeMatches, model)
			} else {
				otherMatches = append(otherMatches, model)
			}
		}
	}
	m.filtered = append(freeMatches, otherMatches...)
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
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
		m.models = sortModels(msg.Models)
		m.info = msg.Info
		m.cursor = 0
		m.applyFilter()
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, modelKeysEsc):
			return m, util.CmdHandler(CloseModelDialogMsg{})
		case key.Matches(msg, modelKeysEnter):
			items := m.filtered
			if len(items) == 0 && len(m.models) > 0 && m.search.Value() == "" {
				items = m.models
			}
			if len(items) > 0 && m.cursor < len(items) {
				return m, util.CmdHandler(ModelSelectedMsg{Model: items[m.cursor]})
			}
		case key.Matches(msg, modelKeysUp):
			items := m.filtered
			if len(items) == 0 && len(m.models) > 0 && m.search.Value() == "" {
				items = m.models
			}
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, modelKeysDown):
			items := m.filtered
			if len(items) == 0 && len(m.models) > 0 && m.search.Value() == "" {
				items = m.models
			}
			if m.cursor < len(items)-1 {
				m.cursor++
			}
			return m, nil
		case key.Matches(msg, modelKeysPgUp):
			m.cursor = max(0, m.cursor-5)
			return m, nil
		case key.Matches(msg, modelKeysPgDown):
			items := m.filtered
			if len(items) == 0 && len(m.models) > 0 && m.search.Value() == "" {
				items = m.models
			}
			if len(items) > 0 {
				m.cursor = min(len(items)-1, m.cursor+5)
			}
			return m, nil
		default:
			prevVal := m.search.Value()
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			if m.search.Value() != prevVal {
				m.cursor = 0
				m.applyFilter()
			}
			return m, cmd
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

	width := 64
	if m.width > 0 && m.width-10 < width {
		width = max(35, m.width-10)
	}

	title := baseStyle.
		Bold(true).
		Foreground(t.Primary()).
		Width(width).
		Render("Select Model")

	searchView := baseStyle.Width(width).Padding(0, 1).Render(m.search.View())

	items := m.filtered
	if len(items) == 0 && len(m.models) > 0 && m.search.Value() == "" {
		items = m.models
	}

	rows := make([]string, 0)
	switch {
	case m.loading:
		rows = append(rows, baseStyle.Foreground(t.TextMuted()).Width(width).
			Render("Loading models from provider (may take a few seconds)..."))
	case m.err != nil:
		rows = append(rows, baseStyle.Foreground(t.Error()).Width(width).
			Render("Could not read the model list: "+m.err.Error()))
	case len(items) == 0:
		if len(m.models) == 0 {
			rows = append(rows, baseStyle.Foreground(t.TextMuted()).Width(width).
				Render("The harness reported no models for this provider."))
		} else {
			rows = append(rows, baseStyle.Foreground(t.TextMuted()).Width(width).
				Render("No models matched your search."))
		}
	default:
		maxVisible := 10
		if m.height > 0 {
			maxVisible = max(4, min(12, m.height-12))
		}
		startIdx := 0
		if len(items) > maxVisible {
			half := maxVisible / 2
			if m.cursor >= half && m.cursor < len(items)-half {
				startIdx = m.cursor - half
			} else if m.cursor >= len(items)-half {
				startIdx = len(items) - maxVisible
			}
		}
		endIdx := min(startIdx+maxVisible, len(items))

		for i := startIdx; i < endIdx; i++ {
			model := items[i]
			freeBadge := ""
			if isFree(model) {
				freeBadge = " [FREE]"
			}

			feat := features(m.info[model])
			lineText := model + freeBadge + " " + feat

			row := baseStyle.Width(width).Padding(0, 1)
			if i == m.cursor {
				row = row.Background(t.Primary()).Foreground(t.Background()).Bold(true)
			}
			rows = append(rows, row.Render(lineText))
		}
	}

	freeCount := 0
	for _, model := range m.models {
		if isFree(model) {
			freeCount++
		}
	}
	var hintText string
	if len(items) > 0 {
		cursorPos := fmt.Sprintf("[%d/%d]", m.cursor+1, len(items))
		if freeCount > 0 {
			hintText = fmt.Sprintf("%s (%d free) · ↑/↓ navigate · Enter select · Esc close", cursorPos, freeCount)
		} else {
			hintText = fmt.Sprintf("%s · ↑/↓ navigate · Enter select · Esc close", cursorPos)
		}
	} else {
		hintText = "Esc to close"
	}

	hint := baseStyle.Foreground(t.TextMuted()).Width(width).Render(hintText)
	divider := baseStyle.Foreground(t.TextMuted()).Width(width).Render(strings.Repeat("─", width))

	contentList := []string{
		title,
		"",
		searchView,
		divider,
		lipgloss.JoinVertical(lipgloss.Left, rows...),
		divider,
		hint,
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentList...)

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.TextMuted()).
		Width(width + 4).
		Render(content)
}
