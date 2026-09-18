package chat

import (
	"context"
	"fmt"
	"math"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

type cacheItem struct {
	width   int
	content []uiMessage
}
type messagesCmp struct {
	app           *app.App
	width, height int
	viewport      viewport.Model
	session       session.Session
	messages      []message.Message
	uiMessages    []uiMessage
	currentMsgID  string
	cachedContent map[string]cacheItem
	spinner       spinner.Model
	rendering     bool
	attachments   viewport.Model
	// pendingWidth is the width the terminal last asked for, applied once it
	// stops asking. Zero means nothing is waiting.
	pendingWidth int
	// renderedFrom is the index of the oldest message that is drawn. Everything
	// before it is in the conversation and not on screen yet.
	renderedFrom int
}
type renderFinishedMsg struct{}

type MessageKeys struct {
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
}

var messageKeys = MessageKeys{
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("f/pgdn", "page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("b/pgup", "page up"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "½ page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	),
}

func (m *messagesCmp) Init() tea.Cmd {
	return tea.Batch(m.viewport.Init(), m.spinner.Tick)
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case dialog.ThemeChangedMsg:
		m.rerender()
		return m, nil
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			cmd := m.SetSession(msg)
			return m, cmd
		}
		return m, nil
	case SessionClearedMsg:
		m.session = session.Session{}
		m.messages = make([]message.Message, 0)
		m.currentMsgID = ""
		m.rendering = false
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, messageKeys.PageUp) || key.Matches(msg, messageKeys.PageDown) ||
			key.Matches(msg, messageKeys.HalfPageUp) || key.Matches(msg, messageKeys.HalfPageDown) {
			u, cmd := m.viewport.Update(msg)
			m.viewport = u
			cmds = append(cmds, cmd)
			// A reader who reaches the top of what is drawn is asking for what
			// is above it.
			if m.viewport.YOffset() == 0 {
				m.renderMore()
			}
		}

	case renderFinishedMsg:
		m.rendering = false
		m.viewport.GotoBottom()
	case resizeSettledMsg:
		// A settle that has been overtaken by a later size is not the last word:
		// rendering for it would draw the transcript at a width the terminal no
		// longer has.
		if msg.width != m.pendingWidth || msg.width != m.width {
			return m, nil
		}
		m.pendingWidth = 0
		m.rerender()
		m.viewport.GotoBottom()
		return m, nil
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.session.ID {
			m.session = msg.Payload
			if m.session.SummaryMessageID == m.currentMsgID {
				delete(m.cachedContent, m.currentMsgID)
				m.renderView()
			}
		}
	case pubsub.Event[message.Message]:
		needsRerender := false
		if msg.Type == pubsub.CreatedEvent {
			if msg.Payload.SessionID == m.session.ID {

				messageExists := false
				for _, v := range m.messages {
					if v.ID == msg.Payload.ID {
						messageExists = true
						break
					}
				}

				if !messageExists {
					if len(m.messages) > 0 {
						lastMsgID := m.messages[len(m.messages)-1].ID
						delete(m.cachedContent, lastMsgID)
					}

					m.messages = append(m.messages, msg.Payload)
					delete(m.cachedContent, m.currentMsgID)
					m.currentMsgID = msg.Payload.ID
					needsRerender = true
				}
			}
			// There are tool calls from the child task
			for _, v := range m.messages {
				for _, c := range v.ToolCalls() {
					if c.ID == msg.Payload.SessionID {
						delete(m.cachedContent, v.ID)
						needsRerender = true
					}
				}
			}
		} else if msg.Type == pubsub.UpdatedEvent && msg.Payload.SessionID == m.session.ID {
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					m.messages[i] = msg.Payload
					delete(m.cachedContent, msg.Payload.ID)
					needsRerender = true
					break
				}
			}
		}
		if needsRerender {
			m.renderView()
			if len(m.messages) > 0 {
				if (msg.Type == pubsub.CreatedEvent) ||
					(msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.messages[len(m.messages)-1].ID) {
					m.viewport.GotoBottom()
				}
			}
		}
	}

	spinner, cmd := m.spinner.Update(msg)
	m.spinner = spinner
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *messagesCmp) IsAgentWorking() bool {
	return m.app.CoderAgent.IsSessionBusy(m.session.ID)
}

func formatTimeDifference(unixTime1, unixTime2 int64) string {
	diffSeconds := float64(math.Abs(float64(unixTime2 - unixTime1)))

	if diffSeconds < 60 {
		return fmt.Sprintf("%.1fs", diffSeconds)
	}

	minutes := int(diffSeconds / 60)
	seconds := int(diffSeconds) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// margin is how many viewports of transcript are rendered beyond the visible
// one, so that scrolling a few pages does not immediately ask for more.
const renderMargin = 3

// renderView renders the part of the transcript the viewport can reach, and
// leaves the rest until a reader scrolls to it.
//
// Rendering every message to show forty rows is work nobody sees: a long
// conversation cost seconds before the first line appeared, and the same again
// on every width change. What is rendered is the tail — where a reader starts,
// and where a run's newest words are — and `renderMore` walks backwards from
// there when the viewport reaches its top.
func (m *messagesCmp) renderView() {
	if m.width == 0 || len(m.messages) == 0 {
		m.uiMessages = nil
		m.setViewportContent()
		return
	}

	m.renderedFrom = m.renderBackwards(len(m.messages), m.renderBudget())
	m.uiMessages = m.uiFrom(m.renderedFrom)
	m.setViewportContent()
}

// renderBudget is how many rows of transcript are worth rendering.
func (m *messagesCmp) renderBudget() int {
	visible := max(m.height, 10)
	return visible * renderMargin
}

// renderBackwards renders from `to` backwards until the budget is spent, and
// returns the index of the oldest message rendered.
//
// Each message is rendered once and kept in the cache, so this is the only place
// the expensive part happens.
func (m *messagesCmp) renderBackwards(to, budget int) int {
	height := 0
	from := to
	for from > 0 {
		msg := m.messages[from-1]
		rendered := m.rendered(msg, from-1)
		// The newest message is always rendered, however tall it is: a single
		// long answer has to be readable.
		if height > 0 && height+heightOf(rendered) > budget {
			break
		}
		height += heightOf(rendered) + 1
		from--
	}
	return from
}

// rendered returns the cached rendering of a message, rendering it if the cache
// has nothing for this width.
func (m *messagesCmp) rendered(msg message.Message, index int) []uiMessage {
	if cache, ok := m.cachedContent[msg.ID]; ok && cache.width == m.width {
		return cache.content
	}

	var rendered []uiMessage
	switch msg.Role {
	case message.User:
		rendered = []uiMessage{renderUserMessage(msg, msg.ID == m.currentMsgID, m.width, index)}
	case message.Assistant:
		rendered = renderAssistantMessage(
			msg,
			index,
			m.messages,
			m.app.Messages,
			m.currentMsgID,
			m.session.SummaryMessageID == msg.ID,
			m.width,
			index,
		)
	}
	m.cachedContent[msg.ID] = cacheItem{width: m.width, content: rendered}
	return rendered
}

// uiFrom collects the rendered rows from an index onwards. It does not render:
// everything in the range came from `renderBackwards` or from the cache it read.
func (m *messagesCmp) uiFrom(from int) []uiMessage {
	out := make([]uiMessage, 0, len(m.messages)-from)
	for inx := from; inx < len(m.messages); inx++ {
		if cache, ok := m.cachedContent[m.messages[inx].ID]; ok && cache.width == m.width {
			out = append(out, cache.content...)
			continue
		}
		out = append(out, m.rendered(m.messages[inx], inx)...)
	}
	return out
}

// renderMore renders the transcript above what is already drawn, and keeps the
// reader's place by shifting the viewport by exactly what was added.
func (m *messagesCmp) renderMore() {
	if m.renderedFrom == 0 {
		return
	}
	before := m.renderedHeight()
	from := m.renderBackwards(m.renderedFrom, m.renderBudget())
	if from == m.renderedFrom {
		return
	}
	m.renderedFrom = from
	m.uiMessages = m.uiFrom(from)
	m.setViewportContent()
	// The added rows are all above the reader, so the offset moves by the same
	// amount: the line being read stays on the line it was read on.
	if added := m.renderedHeight() - before; added > 0 {
		m.viewport.SetYOffset(m.viewport.YOffset() + added)
	}
}

func (m *messagesCmp) renderedHeight() int {
	height := 0
	for _, ui := range m.uiMessages {
		height += ui.height + 1
	}
	return height
}

func heightOf(rendered []uiMessage) int {
	height := 0
	for _, ui := range rendered {
		height += ui.height + 1
	}
	return height
}

func (m *messagesCmp) setViewportContent() {
	baseStyle := styles.BaseStyle()

	messages := make([]string, 0)
	for _, v := range m.uiMessages {
		messages = append(messages, lipgloss.JoinVertical(lipgloss.Left, v.content),
			baseStyle.
				Width(m.width).
				Render(
					"",
				),
		)
	}

	m.viewport.SetContent(
		baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					messages...,
				),
			),
	)
}

// View renders the component for the terminal.
func (m *messagesCmp) View() tea.View { return tea.NewView(m.viewString()) }
func (m *messagesCmp) viewString() string {
	baseStyle := styles.BaseStyle()

	if m.rendering {
		return baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					"Loading...",
					m.working(),
					m.help(),
				),
			)
	}
	if len(m.messages) == 0 {
		content := baseStyle.
			Width(m.width).
			Height(m.height - 1).
			Render(
				m.initialScreen(),
			)

		return baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					content,
					"",
					m.help(),
				),
			)
	}

	// The viewport's content is styled where it is built, so wrapping it in a
	// block again would restyle every cell of the whole transcript on every
	// frame — and a transcript is the largest thing on screen. Only the two
	// small lines are styled here.
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.viewport.View(),
		baseStyle.Width(m.width).Render(m.working()),
		baseStyle.Width(m.width).Render(m.help()),
	)
}

func hasToolsWithoutResponse(messages []message.Message) bool {
	toolCalls := make([]message.ToolCall, 0)
	toolResults := make([]message.ToolResult, 0)
	for _, m := range messages {
		toolCalls = append(toolCalls, m.ToolCalls()...)
		toolResults = append(toolResults, m.ToolResults()...)
	}

	for _, v := range toolCalls {
		found := false
		for _, r := range toolResults {
			if v.ID == r.ToolCallID {
				found = true
				break
			}
		}
		if !found && v.Finished {
			return true
		}
	}
	return false
}

func hasUnfinishedToolCalls(messages []message.Message) bool {
	toolCalls := make([]message.ToolCall, 0)
	for _, m := range messages {
		toolCalls = append(toolCalls, m.ToolCalls()...)
	}
	for _, v := range toolCalls {
		if !v.Finished {
			return true
		}
	}
	return false
}

func (m *messagesCmp) working() string {
	text := ""
	if m.IsAgentWorking() && len(m.messages) > 0 {
		t := theme.CurrentTheme()
		baseStyle := styles.BaseStyle()

		task := "Thinking..."
		lastMessage := m.messages[len(m.messages)-1]
		if hasToolsWithoutResponse(m.messages) {
			task = "Waiting for tool response..."
		} else if hasUnfinishedToolCalls(m.messages) {
			task = "Building tool call..."
		} else if !lastMessage.IsFinished() {
			task = "Generating..."
		}
		if task != "" {
			// Motion is what reduced motion drops, never the information the
			// motion was carrying: the verb stays and the marker holds still.
			marker := m.spinner.View()
			if config.Get().ReducedMotion {
				marker = styles.LoadingIcon
			}
			text += baseStyle.
				Width(m.width).
				Foreground(t.Primary()).
				Bold(true).
				Render(fmt.Sprintf("%s %s ", marker, task))
		}
	}
	return text
}

func (m *messagesCmp) help() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	text := ""

	if m.app.CoderAgent.IsBusy() {
		// A run in flight accepts a follow-up, so the hint says so rather than
		// leaving the reader to find out by trying.
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render("press "),
			baseStyle.Foreground(t.Text()).Bold(true).Render("enter"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" to add to the run, "),
			baseStyle.Foreground(t.Text()).Bold(true).Render("esc"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" to cancel it"),
		)
	} else {
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render("press "),
			baseStyle.Foreground(t.Text()).Bold(true).Render("enter"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" to send the message,"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" write"),
			baseStyle.Foreground(t.Text()).Bold(true).Render(" \\"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" and enter to add a new line"),
		)
	}
	return baseStyle.
		Width(m.width).
		Render(text)
}

func (m *messagesCmp) initialScreen() string {
	baseStyle := styles.BaseStyle()

	return baseStyle.Width(m.width).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			header(m.width),
			"",
			lspsConfigured(m.width),
		),
	)
}

func (m *messagesCmp) rerender() {
	for _, msg := range m.messages {
		delete(m.cachedContent, msg.ID)
	}
	m.renderView()
}

// resizeSettle is how long a width change waits for the terminal to stop moving.
//
// Dragging a window emits a size per frame, and every one of them invalidates
// every rendered row: re-rendering the transcript per frame is work that the
// next frame throws away. The wait is short enough to read as immediate and long
// enough that a drag settles into one render.
const resizeSettle = 80 * time.Millisecond

// resizeSettledMsg is the terminal having stopped moving.
type resizeSettledMsg struct{ width int }

func (m *messagesCmp) SetSize(width, height int) tea.Cmd {
	if m.width == width && m.height == height {
		return nil
	}
	// Every row is rendered to a width, so a width change invalidates the
	// transcript. A height change does not: the rows are already drawn, and
	// re-rendering them for one more visible line is work nobody sees.
	widthChanged := m.width != width

	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height - 2)
	m.attachments.SetWidth(width + 40)
	m.attachments.SetHeight(3)

	if !widthChanged {
		return nil
	}
	// The rows keep the wrapping they had for a moment; the layout does not.
	// What a drag multiplies is the number of renders, not their cost, so this
	// is where a storm stops costing seconds.
	m.pendingWidth = width
	return tea.Tick(resizeSettle, func(time.Time) tea.Msg {
		return resizeSettledMsg{width: width}
	})
}

func (m *messagesCmp) GetSize() (int, int) {
	return m.width, m.height
}

func (m *messagesCmp) SetSession(session session.Session) tea.Cmd {
	if m.session.ID == session.ID {
		return nil
	}
	m.session = session
	messages, err := m.app.Messages.List(context.Background(), session.ID)
	if err != nil {
		return util.ReportError(err)
	}
	m.messages = messages
	if len(m.messages) > 0 {
		m.currentMsgID = m.messages[len(m.messages)-1].ID
	}
	delete(m.cachedContent, m.currentMsgID)
	m.rendering = true
	return func() tea.Msg {
		m.renderView()
		return renderFinishedMsg{}
	}
}

func (m *messagesCmp) BindingKeys() []key.Binding {
	return []key.Binding{
		m.viewport.KeyMap.PageDown,
		m.viewport.KeyMap.PageUp,
		m.viewport.KeyMap.HalfPageUp,
		m.viewport.KeyMap.HalfPageDown,
	}
}

func NewMessagesCmp(app *app.App) tea.Model {
	s := spinner.New()
	s.Spinner = spinner.Pulse
	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	attachmets := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	vp.KeyMap.PageUp = messageKeys.PageUp
	vp.KeyMap.PageDown = messageKeys.PageDown
	vp.KeyMap.HalfPageUp = messageKeys.HalfPageUp
	vp.KeyMap.HalfPageDown = messageKeys.HalfPageDown
	return &messagesCmp{
		app:           app,
		cachedContent: make(map[string]cacheItem),
		viewport:      vp,
		spinner:       s,
		attachments:   attachmets,
	}
}
