//go:build ignore

package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo/tui/theme"
)

// Stage is which surface currently owns the keyboard. Exactly one at a time:
// the whole point of palette-first navigation is that the user never has to
// guess where input goes.
type Stage string

const (
	// StagePalette is the command palette with the query field focused.
	StagePalette Stage = "palette"
	// StageGoal is the goal prompt opened by the "run goal" command.
	StageGoal Stage = "goal"
	// StageRun is the live event stream while the run is in flight.
	StageRun Stage = "run"
	// StageEvidence is the terminal accounting shown once the run stops.
	StageEvidence Stage = "evidence"
)

// Command IDs the palette can run.
const (
	CommandRunGoal    = "run-goal"
	CommandEvidence   = "show-evidence"
	CommandCancel     = "cancel-run"
	CommandNextTheme  = "next-theme"
	CommandReconnect  = "reconnect"
	CommandQuit       = "quit"
	CommandClosePanel = "close-panel"
)

// Model is the whole view state.
//
// It holds *no* decision logic: which command applies is `palette.go`, what a
// given event looks like is `timeline.go`, what a colour is, is `styles.go`.
// This file routes keys and messages into those and nothing else, which is why
// the vertical slice can be tested without a terminal.
type Model struct {
	Styles    Styles
	Palette   *Palette
	Timeline  *Timeline
	Session   *Session
	Workspace string

	Config StartConfig

	stage      Stage
	goalInput  string
	status     string
	err        error
	width      int
	height     int
	themeIndex int
	themeIDs   []string
	// store dir is reported, never parsed: the daemon's private artifacts are
	// not part of the protocol.
	storeDir string
	// sessionFactory is how a new run is started. It is injected so the
	// palette flow can be driven in a test without a daemon process.
	sessionFactory func(context.Context, StartConfig) (*Session, error)
}

// NewModel builds the initial view: palette open, nothing run yet.
func NewModel(styles Styles, workspace, storeDir string) *Model {
	themes := theme.Names()
	index := 0
	for i, id := range themes {
		if id == styles.ThemeID() {
			index = i
		}
	}
	return &Model{
		Styles:     styles,
		Palette:    NewPalette(commands()),
		Timeline:   NewTimeline(DefaultTimelineCapacity),
		Workspace:  workspace,
		Config:     StartConfig{Provider: "fake", MaxTurns: 5, Workspace: workspace},
		stage:      StagePalette,
		status:     "palette-first: type to filter, enter to run",
		themeIndex: index,
		themeIDs:   themes,
		storeDir:   storeDir,
	}
}

func commands() []Command {
	return []Command{
		{ID: CommandRunGoal, Title: "Run goal", Description: "start a headless run and stream its events",
			Keywords: []string{"start", "agent", "prompt"}},
		{ID: CommandEvidence, Title: "Show evidence", Description: "how the last run ended",
			Keywords: []string{"result", "gate", "status"}},
		{ID: CommandCancel, Title: "Cancel run", Description: "stop the run in flight",
			Keywords: []string{"stop", "abort", "interrupt"}},
		{ID: CommandReconnect, Title: "Reconnect and replay", Description: "re-read the run's timeline from the daemon",
			Keywords: []string{"replay", "resume", "events"}},
		{ID: CommandNextTheme, Title: "Next theme", Description: "cycle default, high-contrast, no-color, reduced-motion",
			Keywords: []string{"color", "contrast", "accessibility"}},
		{ID: CommandClosePanel, Title: "Close panel", Description: "return to the palette",
			Keywords: []string{"back", "escape", "dismiss"}},
		{ID: CommandQuit, Title: "Quit", Description: "leave the TUI and stop the daemon",
			Keywords: []string{"exit", "leave", "bye"}},
	}
}

// --- messages ---------------------------------------------------------------

type pollMsg struct{ result PollResult }

type pollErrMsg struct{ err error }

// permissionErrMsg carries a failed approval/denial. It is separate from
// pollErrMsg because the run is fine: the decision did not reach it, and saying
// "lost the daemon" would be a different (and wrong) story.
type permissionErrMsg struct{ err error }

type runErrMsg struct{ err error }

// Init starts the poll loop when a session is already attached. The normal path
// starts a session from the palette, which schedules its own polling.
func (m *Model) Init() tea.Cmd {
	if m.Session != nil {
		return m.pollCmd()
	}
	return nil
}

func (m *Model) pollCmd() tea.Cmd {
	session := m.Session
	if session == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := session.Poll(ctx)
		if err != nil {
			return pollErrMsg{err: err}
		}
		return pollMsg{result: result}
	}
}

func nextTick() tea.Cmd {
	return tea.Tick(PollInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

type tickMsg struct{}

type sessionMsg struct{ session *Session }

// --- update -----------------------------------------------------------------

// Update routes one message. Keys are handled per stage so the same rune means
// "filter the palette" or "type a goal" depending on the focused surface, and
// never both.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
		return m, nil

	case runErrMsg:
		m.err = message.err
		m.status = "run failed to start"
		return m, nil

	case sessionMsg:
		m.Session = message.session
		m.Timeline = NewTimeline(DefaultTimelineCapacity)
		m.err = nil
		m.stage = StageRun
		m.status = "run " + message.session.RunID + " started"
		return m, m.pollCmd()

	case pollMsg:
		for _, ev := range message.result.Events {
			m.Timeline.Append(ev)
		}
		if _, waiting := m.Session.PendingPermission(); waiting {
			m.status = "approval requested — a to approve, d to deny"
		}
		if message.result.Finished {
			m.stage = StageEvidence
			m.status = m.Session.Evidence(m.Timeline.Len(), m.Timeline.Dropped()).Summary()
			return m, nil
		}
		return m, nextTick()

	case permissionErrMsg:
		m.err = message.err
		m.status = "could not answer the permission: " + message.err.Error()
		return m, nil

	case pollErrMsg:
		// A dropped connection is a state, not a crash: say so and keep the
		// view alive so "reconnect and replay" can be chosen.
		m.err = message.err
		m.status = "lost the daemon — reconnect and replay to re-read the run"
		return m, nil

	case tickMsg:
		if m.Session != nil && !m.Session.Finished() {
			return m, m.pollCmd()
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(message)
	}
	return m, nil
}

func (m *Model) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "ctrl+c":
		return m, tea.Quit
	}
	switch m.stage {
	case StagePalette:
		return m.handlePaletteKey(key)
	case StageGoal:
		return m.handleGoalKey(key)
	case StageRun, StageEvidence:
		return m.handleRunKey(key)
	}
	return m, nil
}

func (m *Model) handlePaletteKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up", "ctrl+p":
		m.Palette.Move(-1)
	case "down", "ctrl+n":
		m.Palette.Move(1)
	case "esc":
		m.Palette.SetQuery("")
	case "enter":
		command, ok := m.Palette.Selected()
		if !ok {
			return m, nil
		}
		return m.runCommand(command.ID)
	default:
		if text := key.Text; text != "" {
			for _, r := range text {
				m.Palette.AppendRune(r)
			}
		} else if key.String() == "backspace" {
			m.Palette.Backspace()
		}
	}
	return m, nil
}

func (m *Model) handleGoalKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.stage = StagePalette
		m.status = "goal discarded"
	case "enter":
		goal := strings.TrimSpace(m.goalInput)
		if goal == "" {
			m.status = "a goal is required"
			return m, nil
		}
		return m.startRun(goal)
	case "backspace":
		if m.goalInput != "" {
			runes := []rune(m.goalInput)
			m.goalInput = string(runes[:len(runes)-1])
		}
	default:
		m.goalInput += key.Text
	}
	return m, nil
}

func (m *Model) handleRunKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.stage = StagePalette
		m.status = "back to the palette"
	case "ctrl+r":
		return m.reconnect()
	case "a":
		return m.answerPermission(true)
	case "d":
		return m.answerPermission(false)
	}
	return m, nil
}

// answerPermission sends the user's decision for the request the run is waiting
// on. With nothing pending the key only says so: there is no such thing as
// approving a request that was never made.
func (m *Model) answerPermission(allow bool) (tea.Model, tea.Cmd) {
	if m.Session == nil {
		m.status = "no run yet"
		return m, nil
	}
	if _, waiting := m.Session.PendingPermission(); !waiting {
		m.status = "nothing is waiting for approval"
		return m, nil
	}
	session := m.Session
	if allow {
		m.status = "approving…"
	} else {
		m.status = "denying…"
	}
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var err error
		if allow {
			err = session.Approve(ctx)
		} else {
			err = session.Deny(ctx, "denied in the run panel")
		}
		if err != nil {
			return permissionErrMsg{err: err}
		}
		return tickMsg{}
	}
}

func (m *Model) runCommand(id string) (tea.Model, tea.Cmd) {
	switch id {
	case CommandRunGoal:
		m.stage = StageGoal
		m.goalInput = ""
		m.status = "type the goal, enter to run, esc to cancel"
	case CommandEvidence:
		if m.Session == nil {
			m.status = "no run yet"
			return m, nil
		}
		m.stage = StageEvidence
		m.status = m.Session.Evidence(m.Timeline.Len(), m.Timeline.Dropped()).Summary()
	case CommandCancel:
		if m.Session == nil || m.Session.Finished() {
			m.status = "nothing in flight"
			return m, nil
		}
		session := m.Session
		m.status = "cancelling…"
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := session.Cancel(ctx); err != nil {
				return pollErrMsg{err: err}
			}
			return tickMsg{}
		}
	case CommandReconnect:
		return m.reconnect()
	case CommandNextTheme:
		return m.nextTheme()
	case CommandClosePanel:
		m.stage = StagePalette
		m.status = "palette-first: type to filter, enter to run"
	case CommandQuit:
		return m, tea.Quit
	}
	return m, nil
}

// reconnect re-reads the run's timeline from the daemon and rebuilds the view
// from it. This is criterion 3's "replay without UI state corruption": the
// timeline is derived from the daemon's record, so a restart mid-run re-renders
// the same rows rather than merging two half-views.
func (m *Model) reconnect() (tea.Model, tea.Cmd) {
	if m.Session == nil {
		m.status = "no run to replay"
		return m, nil
	}
	session := m.Session
	m.Timeline = NewTimeline(DefaultTimelineCapacity)
	// Rewinding the cursor is what makes the replay full rather than
	// incremental; Poll already handles a log that shrank.
	session.cursor = 0
	m.status = "replaying the run's events"
	m.stage = StageRun
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result, err := session.Poll(ctx)
		if err != nil {
			return pollErrMsg{err: err}
		}
		return pollMsg{result: result}
	}
}

func (m *Model) nextTheme() (tea.Model, tea.Cmd) {
	if len(m.themeIDs) == 0 {
		return m, nil
	}
	m.themeIndex = (m.themeIndex + 1) % len(m.themeIDs)
	styles, err := NewStyles(m.themeIDs[m.themeIndex])
	if err != nil {
		m.err = err
		m.status = "theme not applied: " + err.Error()
		return m, nil
	}
	m.Styles = styles
	m.status = "theme " + styles.ThemeID()
	return m, nil
}

func (m *Model) startRun(goal string) (tea.Model, tea.Cmd) {
	if m.sessionFactory == nil {
		m.err = fmt.Errorf("tui: no daemon attached")
		m.status = "not connected to a daemon"
		return m, nil
	}
	cfg := m.Config
	cfg.Goal = goal
	factory := m.sessionFactory
	m.status = "starting the run…"
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		session, err := factory(ctx, cfg)
		if err != nil {
			return runErrMsg{err: err}
		}
		return sessionMsg{session: session}
	}
}

// SetSessionFactory installs how a new run is started. The vertical slice wires
// it to a supervised daemon; a test wires it to a stub, which is what keeps the
// palette flow testable without a process.
func (m *Model) SetSessionFactory(factory func(context.Context, StartConfig) (*Session, error)) {
	m.sessionFactory = factory
}

// --- view -------------------------------------------------------------------

// View renders the current stage plus the always-present statusline.
func (m *Model) View() tea.View {
	var body string
	switch m.stage {
	case StagePalette:
		body = m.paletteView()
	case StageGoal:
		body = m.goalView()
	case StageRun:
		body = m.runView()
	case StageEvidence:
		body = m.evidenceView()
	}
	return tea.NewView(body + "\n" + m.statusView())
}

func (m *Model) paletteView() string {
	var b strings.Builder
	b.WriteString(m.Styles.Title().Render("Prumo") + "\n\n")
	if m.Palette.Empty() {
		b.WriteString(m.Styles.Muted().Render("no command matches "+fmt.Sprintf("%q", m.Palette.Query())) + "\n")
		return b.String()
	}
	for i, match := range m.Palette.Matches() {
		line := m.Styles.Highlight(match.Command.Title, m.Palette.Query())
		if i == m.Palette.Cursor() {
			b.WriteString(m.Styles.Selected().Render("› "+line) + "\n")
			b.WriteString("  " + m.Styles.Muted().Render(match.Command.Description) + "\n")
			continue
		}
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n" + m.Styles.Muted().Render("query: "+m.Palette.Query()) + "\n")
	return b.String()
}

func (m *Model) goalView() string {
	return m.Styles.Title().Render("Prumo") + "\n\n" +
		m.Styles.Focus().Render("goal") + "\n" +
		"  " + m.goalInput + "▏\n\n" +
		m.Styles.Muted().Render(fmt.Sprintf("provider %s · max turns %d", m.Config.Provider, m.Config.MaxTurns)) + "\n"
}

func (m *Model) runView() string {
	var b strings.Builder
	b.WriteString(m.Styles.Title().Render("run "+m.Session.RunID) + "\n")
	b.WriteString(m.Styles.Muted().Render("goal: "+m.Session.Goal) + "\n\n")
	b.WriteString(m.Styles.Muted().Render(fmt.Sprintf("timeline (%d rows", m.Timeline.Len())) +
		droppedSuffix(m.Timeline.Dropped()) + ")\n")
	for _, entry := range m.Timeline.Entries() {
		style := m.Styles.Severity(entry.Severity)
		b.WriteString(fmt.Sprintf("%s %s\n", m.Styles.Muted().Render(fmt.Sprintf("%3d", entry.Index)),
			style.Render(entry.Title)))
		if entry.Detail != "" {
			b.WriteString("    " + m.Styles.Muted().Render(entry.Detail) + "\n")
		}
	}
	// A key nobody is told about is a key nobody presses: the panel says what
	// it is waiting for and how to answer it.
	if _, waiting := m.Session.PendingPermission(); waiting {
		b.WriteString("\n" + m.Styles.Focus().Render("approval requested — a to approve, d to deny") + "\n")
	}
	return b.String()
}

func (m *Model) evidenceView() string {
	evidence := m.Session.Evidence(m.Timeline.Len(), m.Timeline.Dropped())
	var b strings.Builder
	b.WriteString(m.Styles.Title().Render("evidence") + "\n\n")
	rows := [][2]string{
		{"run", evidence.RunID},
		{"status", evidence.Status},
		{"phase", evidence.Phase},
		{"stop reason", evidence.StopReason},
		{"events seen", fmt.Sprintf("%d", evidence.Events)},
		{"aged out of view", fmt.Sprintf("%d", evidence.Dropped)},
		{"store", m.storeDir},
	}
	for _, row := range rows {
		b.WriteString("  " + m.Styles.Muted().Render(row[0]) + ": " + m.Styles.Accent().Render(row[1]) + "\n")
	}
	b.WriteString("\n" + m.Styles.Muted().Render(
		"the richer artifacts (evidence, budget, permissions) live in the store above; this panel reports what the protocol states") + "\n")
	return b.String()
}

func droppedSuffix(dropped int) string {
	if dropped == 0 {
		return ""
	}
	return fmt.Sprintf(", %d aged out", dropped)
}

func (m *Model) statusView() string {
	line := fmt.Sprintf("%s · %s · theme %s", m.stage, m.status, m.Styles.ThemeID())
	if m.err != nil {
		line = m.err.Error() + " · " + line
	}
	style := m.Styles.Statusbar()
	if m.width > 0 {
		style = style.Width(m.width)
	}
	return style.Render(line)
}

// Stage reports which surface currently owns the keyboard.
func (m *Model) Stage() Stage { return m.stage }

// Ops is the transport the current session runs over, for callers that need to
// reconnect with a fresh one.
func (m *Model) Ops() Ops {
	if m.Session == nil {
		return nil
	}
	return m.Session.Ops()
}
