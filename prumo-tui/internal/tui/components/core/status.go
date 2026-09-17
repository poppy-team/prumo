package core

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

type StatusCmp interface {
	tea.Model
}

type statusCmp struct {
	info       util.InfoMsg
	width      int
	messageTTL time.Duration
	session    session.Session
}

// clearMessageCmd is a command that clears status messages after a timeout
func (m statusCmp) clearMessageCmd(ttl time.Duration) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return util.ClearStatusMsg{}
	})
}

func (m statusCmp) Init() tea.Cmd {
	return nil
}

func (m statusCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case chat.SessionSelectedMsg:
		m.session = msg
	case chat.SessionClearedMsg:
		m.session = session.Session{}
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent {
			if m.session.ID == msg.Payload.ID {
				m.session = msg.Payload
			}
		}
	case util.InfoMsg:
		m.info = msg
		ttl := msg.TTL
		if ttl == 0 {
			ttl = m.messageTTL
		}
		return m, m.clearMessageCmd(ttl)
	case util.ClearStatusMsg:
		m.info = util.InfoMsg{}
	}
	return m, nil
}

var helpWidget = ""

// getHelpWidget returns the help widget with current theme colors
func getHelpWidget() string {
	t := theme.CurrentTheme()
	helpText := "ctrl+? help"

	return styles.Padded().
		Background(t.TextMuted()).
		Foreground(t.BackgroundDarker()).
		Bold(true).
		Render(helpText)
}

func formatTokensAndCost(tokens, contextWindow int64, cost float64) string {
	// Format tokens in human-readable format (e.g., 110K, 1.2M)
	var formattedTokens string
	switch {
	case tokens >= 1_000_000:
		formattedTokens = fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	case tokens >= 1_000:
		formattedTokens = fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	default:
		formattedTokens = fmt.Sprintf("%d", tokens)
	}

	// Remove .0 suffix if present
	if strings.HasSuffix(formattedTokens, ".0K") {
		formattedTokens = strings.Replace(formattedTokens, ".0K", "K", 1)
	}
	if strings.HasSuffix(formattedTokens, ".0M") {
		formattedTokens = strings.Replace(formattedTokens, ".0M", "M", 1)
	}

	// Format cost with $ symbol and 2 decimal places
	formattedCost := fmt.Sprintf("$%.2f", cost)

	percentage := (float64(tokens) / float64(contextWindow)) * 100
	if percentage > 80 {
		// add the warning icon and percentage
		formattedTokens = fmt.Sprintf("%s(%d%%)", styles.WarningIcon, int(percentage))
	}

	return fmt.Sprintf("Context: %s, Cost: %s", formattedTokens, formattedCost)
}

// View renders the component for the terminal.
func (m statusCmp) View() tea.View { return tea.NewView(m.viewString()) }
func (m statusCmp) viewString() string {
	t := theme.CurrentTheme()
	// The client shows the model it asked for, not a catalogue entry: what
	// actually served the run is the harness's answer, reported per run.
	model := models.Model{ID: models.ModelID(config.Get().Model), Name: config.Get().Model}

	// Initialize the help widget
	status := getHelpWidget()

	tokenInfoWidth := 0
	if m.session.ID != "" {
		totalTokens := m.session.PromptTokens + m.session.CompletionTokens
		tokens := formatTokensAndCost(totalTokens, model.ContextWindow, m.session.Cost)
		tokensStyle := styles.Padded().
			Background(t.Text()).
			Foreground(t.BackgroundSecondary())
		percentage := (float64(totalTokens) / float64(model.ContextWindow)) * 100
		if percentage > 80 {
			tokensStyle = tokensStyle.Background(t.Warning())
		}
		tokenInfoWidth = lipgloss.Width(tokens) + 2
		status += tokensStyle.Render(tokens)
	}

	diagnostics := styles.Padded().
		Background(t.BackgroundDarker()).
		Render(m.projectDiagnostics())

	availableWidht := max(0, m.width-lipgloss.Width(helpWidget)-lipgloss.Width(m.model())-lipgloss.Width(diagnostics)-tokenInfoWidth)

	if m.info.Msg != "" {
		infoStyle := styles.Padded().
			Foreground(t.Background()).
			Width(availableWidht)

		switch m.info.Type {
		case util.InfoTypeInfo:
			infoStyle = infoStyle.Background(t.Info())
		case util.InfoTypeWarn:
			infoStyle = infoStyle.Background(t.Warning())
		case util.InfoTypeError:
			infoStyle = infoStyle.Background(t.Error())
		}

		infoWidth := availableWidht - 10
		// Truncate message if it's longer than available width
		msg := m.info.Msg
		if len(msg) > infoWidth && infoWidth > 0 {
			msg = msg[:infoWidth] + "..."
		}
		status += infoStyle.Render(msg)
	} else {
		status += styles.Padded().
			Foreground(t.Text()).
			Background(t.BackgroundSecondary()).
			Width(availableWidht).
			Render("")
	}

	status += diagnostics
	status += m.model()
	return status
}

func (m *statusCmp) projectDiagnostics() string {
	// The harness owns diagnostics: it runs the language tooling behind the ACI
	// catalog, and the client cannot see a language server that it does not
	// start. Rendering nothing is the honest answer — an empty indicator means
	// "not reported", not "clean", and the statusline already says what the run
	// is doing.
	return ""
}

func (m statusCmp) availableFooterMsgWidth(diagnostics, tokenInfo string) int {
	tokensWidth := 0
	if m.session.ID != "" {
		tokensWidth = lipgloss.Width(tokenInfo) + 2
	}
	return max(0, m.width-lipgloss.Width(helpWidget)-lipgloss.Width(m.model())-lipgloss.Width(diagnostics)-tokensWidth)
}

func (m statusCmp) model() string {
	t := theme.CurrentTheme()

	cfg := config.Get()

	if cfg.Model == "" {
		return cfg.Provider
	}
	model := models.Model{ID: models.ModelID(cfg.Model), Name: cfg.Model}

	return styles.Padded().
		Background(t.Secondary()).
		Foreground(t.Background()).
		Render(model.Name)
}

func NewStatusCmp() StatusCmp {
	helpWidget = getHelpWidget()

	return &statusCmp{
		messageTTL: 10 * time.Second,
	}
}
