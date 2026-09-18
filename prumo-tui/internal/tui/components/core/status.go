package core

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/raillen/prumo-tui/internal/app"
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

// ConnectionMsg is the client's link to the harness, reported to the user
// because a run that stopped and a daemon that went away are different facts.
type ConnectionMsg struct {
	State   string
	Attempt int
	Of      int
	Reason  string
}

type statusCmp struct {
	app        *app.App
	info       util.InfoMsg
	width      int
	messageTTL time.Duration
	session    session.Session
	connection ConnectionMsg
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
	case ConnectionMsg:
		m.connection = msg
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

// accounting is what the session spent, by kind, with the average request.
//
// The kinds are shown apart because they are priced apart: a cached token is not
// an input token, and a total that hid the difference would report what a
// session cost without saying why it cost that.
func (m statusCmp) accounting() string {
	if m.session.ID == "" {
		return ""
	}
	t := theme.CurrentTheme()
	style := styles.Padded().
		Background(t.Text()).
		Foreground(t.BackgroundSecondary())

	return style.Render(m.accountingText())
}

// accountingText is the accounting as words, so a test can read it and a narrow
// terminal can degrade to a part of it rather than to nothing.
func (m statusCmp) accountingText() string {
	parts := []string{
		fmt.Sprintf("in %s", formatTokens(m.session.PromptTokens)),
	}
	if m.session.CacheReadTokens > 0 || m.session.CacheWriteTokens > 0 {
		parts = append(parts, fmt.Sprintf("cache %s", formatTokens(m.session.CacheReadTokens+m.session.CacheWriteTokens)))
	}
	parts = append(parts, fmt.Sprintf("out %s", formatTokens(m.session.CompletionTokens)))
	parts = append(parts, fmt.Sprintf("$%.4f", m.session.Cost))
	if m.session.UsageReports > 0 {
		parts = append(parts, fmt.Sprintf("~$%.4f/req", m.session.Cost/float64(m.session.UsageReports)))
	}
	return strings.Join(parts, " · ")
}

// spend is the part of the accounting a narrow terminal keeps when the whole of
// it does not fit: what the session cost is the number a user acts on.
func (m statusCmp) spend() string {
	if m.session.ID == "" {
		return ""
	}
	t := theme.CurrentTheme()
	return styles.Padded().
		Background(t.Text()).
		Foreground(t.BackgroundSecondary()).
		Render(fmt.Sprintf("$%.4f", m.session.Cost))
}

// formatTokens renders a count the way a person reads one (110K, 1.2M).
func formatTokens(tokens int64) string {
	switch {
	case tokens >= 1_000_000:
		return trimZero(fmt.Sprintf("%.1fM", float64(tokens)/1_000_000), "M")
	case tokens >= 1_000:
		return trimZero(fmt.Sprintf("%.1fK", float64(tokens)/1_000), "K")
	default:
		return fmt.Sprintf("%d", tokens)
	}
}

func trimZero(value, suffix string) string {
	return strings.Replace(value, ".0"+suffix, suffix, 1)
}

// changeCount is what the run touched, from the harness's own file.changed
// events. It sits beside the accounting because both answer the same question:
// what did this run cost me. A count of zero is not shown — the absence of
// changes is not news.
func (m statusCmp) changeCount() string {
	if m.app == nil || m.app.Runner == nil || m.session.ID == "" {
		return ""
	}
	changes := m.app.Runner.Changes(m.session.ID)
	if len(changes) == 0 {
		return ""
	}
	t := theme.CurrentTheme()
	return styles.Padded().
		Background(t.BackgroundDarker()).
		Foreground(t.Text()).
		Render(fmt.Sprintf("%d changed", len(changes)))
}

// connection reports the state of the client's own link, in words.
//
// A live link says nothing: the statusline is for what a user has to know, and
// "everything is fine" is not news. A broken one says so with the attempt count,
// because a retry a user cannot count is indistinguishable from a hang.
func (m statusCmp) connectionState() string {
	switch m.connection.State {
	case "", "live":
		return ""
	}
	t := theme.CurrentTheme()
	label := m.connection.State
	if m.connection.Attempt > 0 {
		label = fmt.Sprintf("%s %d/%d", label, m.connection.Attempt, m.connection.Of)
	}
	style := styles.Padded().Foreground(t.Background()).Background(t.Warning())
	if m.connection.State == "offline" {
		style = style.Background(t.Error())
	}
	return style.Render(label)
}

// View renders the component for the terminal.
func (m statusCmp) View() tea.View { return tea.NewView(m.viewString()) }
func (m statusCmp) viewString() string {
	t := theme.CurrentTheme()

	// The model and the help hint are what the statusline always says. The
	// accounting and the change count take what is left, in that order, and are
	// dropped when the terminal cannot hold them: a row that runs past the
	// screen is a row nobody can read, and the parts that were dropped are the
	// ones a user can find elsewhere.
	model := m.model()
	help := getHelpWidget()
	available := max(0, m.width-lipgloss.Width(help)-lipgloss.Width(model))

	status := help

	// A link that is not healthy comes before what the run spent: a user who
	// cannot see the daemon needs to know that first.
	if link := m.connectionState(); link != "" && lipgloss.Width(link) <= available {
		status += link
		available -= lipgloss.Width(link)
	}

	// The whole accounting when it fits, what it cost when only that fits, and
	// nothing when even that does not: a number that is cut in half is worse
	// than a number that is absent.
	if tokens := m.accounting(); tokens != "" && lipgloss.Width(tokens) <= available {
		status += tokens
		available -= lipgloss.Width(tokens)
	} else if spend := m.spend(); spend != "" && lipgloss.Width(spend) <= available {
		status += spend
		available -= lipgloss.Width(spend)
	}
	if changed := m.changeCount(); changed != "" && lipgloss.Width(changed) <= available {
		status += changed
		available -= lipgloss.Width(changed)
	}

	// The segment is padded, so the text it can hold is narrower than the space
	// it was given.
	inner := max(0, available-2)
	if m.info.Msg != "" {
		infoStyle := styles.Padded().
			Foreground(t.Background()).
			Width(inner)

		switch m.info.Type {
		case util.InfoTypeInfo:
			infoStyle = infoStyle.Background(t.Info())
		case util.InfoTypeWarn:
			infoStyle = infoStyle.Background(t.Warning())
		case util.InfoTypeError:
			infoStyle = infoStyle.Background(t.Error())
		}

		status += infoStyle.Render(truncate(m.info.Msg, inner))
	} else {
		status += styles.Padded().
			Foreground(t.Text()).
			Background(t.BackgroundSecondary()).
			Width(inner).
			Render("")
	}

	// The harness owns diagnostics: it runs the language tooling behind the ACI
	// catalog, and the client cannot see a language server it does not start.
	// The segment appears only when there is something to report, because an
	// empty indicator reads as "clean" where the truth is "not reported".
	if diagnostics := m.projectDiagnostics(); diagnostics != "" {
		status += styles.Padded().Background(t.BackgroundDarker()).Render(diagnostics)
	}

	status += model
	return status
}

// truncate cuts a label to a width, keeping the marker that says it was cut.
func truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	return ansi.Truncate(text, width, styles.TruncationMarker)
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

func NewStatusCmp(app *app.App) StatusCmp {
	helpWidget = getHelpWidget()

	return &statusCmp{
		app:        app,
		messageTTL: 10 * time.Second,
	}
}
