package dialog

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// ProviderSelectedMsg is sent when a provider is selected.
type ProviderSelectedMsg struct {
	Provider string
}

// CloseProviderDialogMsg is sent when the provider dialog is closed.
type CloseProviderDialogMsg struct{}

// ProviderDialog interface for the provider selection dialog.
type ProviderDialog interface {
	tea.Model
	layout.Bindings
	SetCurrentProvider(provider string)
}

// ProviderOption represents one selectable LLM provider.
type ProviderOption struct {
	ID          string
	Name        string
	Description string
	EnvVar      string
	RequiresKey bool
}

var availableProviders = []ProviderOption{
	{
		ID:          "opencode",
		Name:        "OpenCode",
		Description: "Free models served via local OpenCode daemon (no API key needed)",
		RequiresKey: false,
	},
	{
		ID:          "anthropic",
		Name:        "Anthropic Claude",
		Description: "Claude 3.5 Sonnet / Haiku / Opus models",
		EnvVar:      "PRUMO_MODEL_API_KEY (or ANTHROPIC_API_KEY)",
		RequiresKey: true,
	},
	{
		ID:          "openai-compat",
		Name:        "OpenAI Compatible",
		Description: "OpenAI, Groq, Ollama, DeepSeek or compatible endpoints",
		EnvVar:      "PRUMO_MODEL_API_KEY (optional PRUMO_MODEL_BASE_URL)",
		RequiresKey: true,
	},
	{
		ID:          "fake",
		Name:        "Fake (Offline Mock)",
		Description: "Offline deterministic test provider (no network required)",
		RequiresKey: false,
	},
}

type providerDialogCmp struct {
	providers       []ProviderOption
	selectedIdx     int
	currentProvider string
	width           int
	height          int
}

type providerKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
	J      key.Binding
	K      key.Binding
}

var providerKeys = providerKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "previous provider"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next provider"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select provider"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
	J: key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "next provider"),
	),
	K: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "previous provider"),
	),
}

func (p *providerDialogCmp) Init() tea.Cmd {
	p.syncSelection()
	return nil
}

func (p *providerDialogCmp) syncSelection() {
	cur := strings.ToLower(strings.TrimSpace(p.currentProvider))
	if cur == "" {
		cur = config.Get().Provider
	}
	for i, opt := range p.providers {
		if strings.EqualFold(opt.ID, cur) {
			p.selectedIdx = i
			p.currentProvider = opt.ID
			break
		}
	}
}

func (p *providerDialogCmp) SetCurrentProvider(provider string) {
	p.currentProvider = provider
	p.syncSelection()
}

func hasEnvKey(opt ProviderOption) bool {
	if !opt.RequiresKey {
		return true
	}
	switch opt.ID {
	case "anthropic":
		return os.Getenv("PRUMO_MODEL_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != ""
	case "openai-compat":
		return os.Getenv("PRUMO_MODEL_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != ""
	default:
		return os.Getenv("PRUMO_MODEL_API_KEY") != ""
	}
}

func (p *providerDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, providerKeys.Up) || key.Matches(msg, providerKeys.K):
			if p.selectedIdx > 0 {
				p.selectedIdx--
			}
			return p, nil
		case key.Matches(msg, providerKeys.Down) || key.Matches(msg, providerKeys.J):
			if p.selectedIdx < len(p.providers)-1 {
				p.selectedIdx++
			}
			return p, nil
		case key.Matches(msg, providerKeys.Enter):
			if len(p.providers) > 0 {
				selected := p.providers[p.selectedIdx]
				p.currentProvider = selected.ID
				if err := config.UpdateProvider(selected.ID); err != nil {
					return p, util.ReportError(err)
				}
				return p, tea.Batch(
					util.CmdHandler(ProviderSelectedMsg{Provider: selected.ID}),
					util.CmdHandler(CloseProviderDialogMsg{}),
				)
			}
		case key.Matches(msg, providerKeys.Escape):
			return p, util.CmdHandler(CloseProviderDialogMsg{})
		}
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
	}
	return p, nil
}

// View renders the component for the terminal.
func (p *providerDialogCmp) View() tea.View { return tea.NewView(p.viewString()) }
func (p *providerDialogCmp) viewString() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	maxWidth := 60
	for _, opt := range p.providers {
		titleLen := len(opt.Name) + 16
		if titleLen > maxWidth {
			maxWidth = titleLen
		}
		if len(opt.Description)+4 > maxWidth {
			maxWidth = len(opt.Description) + 4
		}
	}

	title := baseStyle.
		Foreground(t.Primary()).
		Bold(true).
		Width(maxWidth).
		Padding(0, 1).
		Render("Configure LLM Provider")

	items := make([]string, 0, len(p.providers))
	for i, opt := range p.providers {
		selected := i == p.selectedIdx
		isActive := strings.EqualFold(opt.ID, p.currentProvider)

		status := ""
		if isActive {
			status = " (active)"
		}

		keyStatus := ""
		if opt.RequiresKey {
			if hasEnvKey(opt) {
				keyStatus = "  [env key found]"
			} else {
				keyStatus = "  [no env key]"
			}
		}

		headerText := opt.Name + status + keyStatus
		descText := opt.Description
		if opt.EnvVar != "" {
			descText += "\n  env: " + opt.EnvVar
		}

		itemStyle := baseStyle.Width(maxWidth).Padding(0, 1)
		descStyle := baseStyle.Width(maxWidth).Padding(0, 2).Foreground(t.TextMuted())

		if selected {
			itemStyle = itemStyle.
				Background(t.Primary()).
				Foreground(t.Background()).
				Bold(true)
			descStyle = descStyle.
				Background(t.Primary()).
				Foreground(t.Background())
		} else if isActive {
			itemStyle = itemStyle.Foreground(t.Primary()).Bold(true)
		}

		itemRender := itemStyle.Render(fmt.Sprintf("%s %s", func() string {
			if selected {
				return "▸"
			}
			if isActive {
				return "●"
			}
			return " "
		}(), headerText))

		descRender := descStyle.Render(descText)
		items = append(items, lipgloss.JoinVertical(lipgloss.Left, itemRender, descRender))
	}

	hint := baseStyle.
		Foreground(t.TextMuted()).
		Width(maxWidth).
		Padding(1, 1, 0, 1).
		Render("enter: select • esc: close • ↑/↓ or j/k: navigate\nKeys are read from environment variables, never saved to disk.")

	contentList := append([]string{title, ""}, items...)
	contentList = append(contentList, hint)

	content := lipgloss.JoinVertical(lipgloss.Left, contentList...)

	return baseStyle.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (p *providerDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(providerKeys)
}

// NewProviderDialogCmp creates a new provider configuration dialog.
func NewProviderDialogCmp(initialProvider string) ProviderDialog {
	if initialProvider == "" {
		initialProvider = config.Get().Provider
	}
	return &providerDialogCmp{
		providers:       availableProviders,
		currentProvider: initialProvider,
	}
}
