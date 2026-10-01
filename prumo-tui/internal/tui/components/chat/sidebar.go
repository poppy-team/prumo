package chat

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/goals"
	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
)

// SidebarCmp is the collapsible right sidebar panel displaying session details,
// context token usage, workforce & subagents tree, and changed files.
type SidebarCmp interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
	UpdateSession(s session.Session)
	Focus()
	Blur()
	Focused() bool
}

type sidebarCmp struct {
	app       *app.App
	session   session.Session
	width     int
	height    int
	viewport  viewport.Model
	focused   bool
	selected  int
	workspace string
	goals     []goals.Goal
	expanded  map[string]bool
	sections  map[string]bool
	rows      []sidebarRow
}

type sidebarRow struct {
	id   string
	line int
}

func (s *sidebarCmp) Init() tea.Cmd {
	return nil
}

func (s *sidebarCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.viewport.SetWidth(max(0, msg.Width))
		s.viewport.SetHeight(msg.Height)
	case SessionSelectedMsg:
		s.session = msg
	case SessionClearedMsg:
		s.session = session.Session{}
	case tea.KeyPressMsg:
		if !s.focused {
			return s, nil
		}
		switch msg.String() {
		case "up", "k":
			s.moveSelection(-1)
		case "down", "j":
			s.moveSelection(1)
		case "enter", " ":
			s.toggleSelected()
		case "pgup", "ctrl+u":
			s.viewport.PageUp()
		case "pgdown", "ctrl+d":
			s.viewport.PageDown()
		}
	}
	return s, nil
}

func (s *sidebarCmp) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height
	s.viewport.SetWidth(max(0, width))
	s.viewport.SetHeight(height)
	return nil
}

func (s *sidebarCmp) GetSize() (int, int) {
	return s.width, s.height
}

type SidebarKeyMap struct {
	Navigate key.Binding
	Toggle   key.Binding
	Scroll   key.Binding
}

var sidebarKeyMap = SidebarKeyMap{
	Navigate: key.NewBinding(
		key.WithKeys("up", "down", "k", "j"),
		key.WithHelp("↑/↓", "move between sections"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "expand/collapse"),
	),
	Scroll: key.NewBinding(
		key.WithKeys("pgup", "pgdown"),
		key.WithHelp("pgup/pgdn", "scroll panel"),
	),
}

func (s *sidebarCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(sidebarKeyMap)
}

func (s *sidebarCmp) Focus() {
	s.focused = true
}

func (s *sidebarCmp) Blur() {
	s.focused = false
}

func (s *sidebarCmp) Focused() bool {
	return s.focused
}

func (s *sidebarCmp) UpdateSession(sess session.Session) {
	s.session = sess
}

func (s *sidebarCmp) View() tea.View {
	return tea.NewView(s.viewString())
}

func (s *sidebarCmp) refreshGoals() {
	ws := ""
	if s.app != nil {
		ws = s.app.CurrentWorkspace()
	}
	if ws == s.workspace {
		return
	}
	s.workspace = ws
	s.goals = goals.ReadWorkspace(ws)
	s.expanded = make(map[string]bool)
	active := -1
	for i, g := range s.goals {
		if g.State == "EXECUTING" || g.State == "VERIFYING" || g.State == "REVIEWING" {
			active = i
			break
		}
	}
	if active < 0 {
		for i, g := range s.goals {
			if g.State == "LOCKED" {
				active = i
				break
			}
		}
	}
	if active >= 0 {
		g := s.goals[active]
		copy(s.goals[1:active+1], s.goals[:active])
		s.goals[0] = g
		s.expanded[g.ID] = true
	}
}

func (s *sidebarCmp) moveSelection(delta int) {
	if len(s.rows) == 0 {
		s.viewString()
	}
	if len(s.rows) == 0 {
		return
	}
	s.selected = max(0, min(len(s.rows)-1, s.selected+delta))
	s.viewport.EnsureVisible(s.rows[s.selected].line, 0, 1)
}

func (s *sidebarCmp) toggleSelected() {
	if len(s.rows) == 0 {
		s.viewString()
	}
	if s.selected < 0 || s.selected >= len(s.rows) {
		return
	}
	id := s.rows[s.selected].id
	if strings.HasPrefix(id, "goal:") {
		id = strings.TrimPrefix(id, "goal:")
		s.expanded[id] = !s.expanded[id]
	} else {
		s.sections[id] = !s.sections[id]
	}
	s.viewString()
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// subagentElapsed names how long a running subagent has been going, coarsely
// enough that a frame stays readable: done work shows no duration, running
// work shows one. StartedAt is unix seconds; anything else means unknown.
func subagentElapsed(sub runtime.SubagentInfo) string {
	if sub.Status != "running" || sub.StartedAt <= 0 {
		return ""
	}
	secs := time.Now().Unix() - sub.StartedAt
	if secs < 0 {
		return ""
	}
	if secs < 90 {
		return fmt.Sprintf(" %ds", secs)
	}
	return fmt.Sprintf(" %dm", secs/60)
}

func renderProgressBar(ratio float64, width int, filledColor, emptyColor theme.AdaptiveColor) string {
	if width <= 0 {
		return ""
	}
	filledLen := int(ratio * float64(width))
	if filledLen < 0 {
		filledLen = 0
	}
	if filledLen > width {
		filledLen = width
	}
	emptyLen := width - filledLen

	filledPart := lipgloss.NewStyle().Foreground(filledColor).Render(strings.Repeat("█", filledLen))
	emptyPart := lipgloss.NewStyle().Foreground(emptyColor).Render(strings.Repeat("░", emptyLen))
	return filledPart + emptyPart
}

func (s *sidebarCmp) viewString() string {
	if s.width <= 0 || s.height <= 0 {
		return ""
	}

	s.refreshGoals()
	s.rows = s.rows[:0]

	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle().Width(s.width)

	headerStyle := baseStyle.
		Foreground(t.Primary()).
		Bold(true).
		Padding(0, 1)

	labelStyle := baseStyle.
		Foreground(t.TextMuted()).
		Padding(0, 1)

	valueStyle := baseStyle.
		Foreground(t.Text()).
		Padding(0, 1)

	accentStyle := baseStyle.
		Foreground(t.Accent()).
		Padding(0, 1)

	var lines []string

	renderHeader := func(id, title string) {
		rowIdx := len(s.rows)
		s.rows = append(s.rows, sidebarRow{id: id, line: len(lines)})
		expanded := s.sections[id]
		arrow := "▼"
		if !expanded {
			arrow = "▶"
		}
		hdr := fmt.Sprintf("%s %s", arrow, title)
		if s.focused && s.selected == rowIdx {
			lines = append(lines, baseStyle.Foreground(t.Accent()).Bold(true).Padding(0, 1).Render(hdr))
		} else {
			lines = append(lines, headerStyle.Render(hdr))
		}
	}

	// Section 1: Session Status & Model
	renderHeader("status", "SESSION STATUS")
	if s.sections["status"] {
		busy := false
		if s.app != nil && s.app.CoderAgent != nil {
			if s.session.ID != "" {
				busy = s.app.CoderAgent.IsSessionBusy(s.session.ID)
			} else {
				busy = s.app.CoderAgent.IsBusy()
			}
		}

		statusText := "● Idle"
		if busy {
			statusText = "◌ Working..."
		}
		lines = append(lines, valueStyle.Render(statusText))

		sessTitle := s.session.Title
		if sessTitle == "" {
			sessTitle = "New Session"
		}
		if len(sessTitle) > s.width-4 && s.width > 4 {
			sessTitle = sessTitle[:s.width-7] + "..."
		}
		lines = append(lines, labelStyle.Render("Title: "+sessTitle))

		provider := "fake"
		if s.app != nil {
			provider = s.app.CurrentProvider()
		}
		lines = append(lines, labelStyle.Render("Provider: "+provider))

		modelName := "default"
		if s.app != nil && s.app.CoderAgent != nil {
			if m := s.app.CoderAgent.Model(); m.Name != "" {
				modelName = m.Name
			}
		}
		if len(modelName) > s.width-4 && s.width > 4 {
			modelName = modelName[:s.width-7] + "..."
		}
		lines = append(lines, labelStyle.Render("Model: "+modelName))

		effort := "default"
		if s.app != nil && s.app.Runner != nil {
			if e := s.app.Runner.ReasoningEffort(); e != "" {
				effort = e
			}
		} else if e := config.Get().ReasoningEffort; e != "" {
			effort = e
		}
		lines = append(lines, labelStyle.Render("Effort: "+effort))

		if s.app != nil && s.app.Runner != nil {
			if vm := s.app.Runner.VisionModel(); vm != "" {
				visionLabel := vm
				if len(visionLabel) > s.width-4 && s.width > 4 {
					visionLabel = visionLabel[:s.width-7] + "..."
				}
				lines = append(lines, labelStyle.Render("Vision: "+visionLabel))
			}
		}

		ws := ""
		if s.app != nil {
			ws = s.app.CurrentWorkspace()
		}
		if ws != "" {
			wsLabel := filepath.Base(ws)
			if strings.Contains(ws, "worktrees") {
				wsLabel = "wt:" + filepath.Base(ws)
			}
			if len(wsLabel) > s.width-4 && s.width > 4 {
				wsLabel = wsLabel[:s.width-7] + "..."
			}
			lines = append(lines, labelStyle.Render("Workspace: "+wsLabel))
		}
	}

	lines = append(lines, "")

	// Section 2: Context & Token Occupancy
	renderHeader("tokens", "CONTEXT TOKENS")
	if s.sections["tokens"] {
		totalTokens := s.session.PromptTokens + s.session.CompletionTokens
		contextLimit := int64(128_000)
		if s.app != nil {
			if declared := s.app.ModelContextLength(); declared > 0 {
				contextLimit = declared
			}
		}
		ratio := float64(totalTokens) / float64(contextLimit)
		if ratio > 1.0 {
			ratio = 1.0
		}
		pct := int(ratio * 100)

		barWidth := max(8, s.width-4)
		bar := renderProgressBar(ratio, barWidth, t.Primary(), t.BackgroundSecondary())
		lines = append(lines, baseStyle.Padding(0, 1).Render(bar))

		occupancyText := fmt.Sprintf("%s / %s (%d%%)", formatTokens(totalTokens), formatTokens(contextLimit), pct)
		lines = append(lines, valueStyle.Render(occupancyText))

		tokensDetail := fmt.Sprintf("in: %s  out: %s", formatTokens(s.session.PromptTokens), formatTokens(s.session.CompletionTokens))
		lines = append(lines, labelStyle.Render(tokensDetail))

		if s.session.ReasoningTokens > 0 {
			lines = append(lines, labelStyle.Render(fmt.Sprintf("reasoning: %s", formatTokens(s.session.ReasoningTokens))))
		}

		if s.session.CacheReadTokens > 0 {
			lines = append(lines, labelStyle.Render(fmt.Sprintf("cache read: %s", formatTokens(s.session.CacheReadTokens))))
		}

		if s.session.Cost > 0 {
			lines = append(lines, accentStyle.Render(fmt.Sprintf("Cost: $%.4f", s.session.Cost)))
		}
	}

	lines = append(lines, "")

	// Section 3: Goals Accordion
	renderHeader("goals", "GOALS")
	if s.sections["goals"] {
		if len(s.goals) == 0 {
			lines = append(lines, labelStyle.Render("No canonical goals"))
		} else {
			for _, g := range s.goals {
				rowIdx := len(s.rows)
				s.rows = append(s.rows, sidebarRow{id: "goal:" + g.ID, line: len(lines)})
				exp := s.expanded[g.ID]
				arrow := "▶"
				if exp {
					arrow = "▼"
				}
				goalHdr := fmt.Sprintf(" %s %s [%s]", arrow, g.ID, g.State)
				if s.focused && s.selected == rowIdx {
					lines = append(lines, baseStyle.Foreground(t.Accent()).Bold(true).Padding(0, 1).Render(goalHdr))
				} else {
					lines = append(lines, valueStyle.Render(goalHdr))
				}
				if exp {
					if g.Title != "" && g.Title != g.ID {
						lines = append(lines, labelStyle.Render("   Title: "+g.Title))
					}
					if len(g.Acceptance) > 0 {
						lines = append(lines, labelStyle.Render("   Criteria:"))
						for _, a := range g.Acceptance {
							lines = append(lines, labelStyle.Render("   - "+a))
						}
					}
					if len(g.Constraints) > 0 {
						lines = append(lines, labelStyle.Render("   Constraints:"))
						for _, c := range g.Constraints {
							lines = append(lines, labelStyle.Render("   - "+c))
						}
					}
					if len(g.Gates) > 0 {
						lines = append(lines, labelStyle.Render("   Gates:"))
						keys := make([]string, 0, len(g.Gates))
						for k := range g.Gates {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						for _, k := range keys {
							lines = append(lines, labelStyle.Render(fmt.Sprintf("   - %s: %v", k, g.Gates[k])))
						}
					}
				}
			}
		}
	}

	lines = append(lines, "")

	// Section 4: Workforce & Subagents
	renderHeader("workforce", "WORKFORCE")
	if s.sections["workforce"] {
		activeAgent := "coder"
		if s.app != nil && s.app.Runner != nil {
			activeAgent = s.app.Runner.ActiveAgent()
		}
		lines = append(lines, valueStyle.Render("Primary: "+activeAgent))

		var subagents []runtime.SubagentInfo
		if s.app != nil && s.app.Runner != nil && s.session.ID != "" {
			subagents = s.app.Runner.Subagents(s.session.ID)
		}

		if len(subagents) == 0 {
			lines = append(lines, labelStyle.Render("└─ (single agent)"))
		} else {
			for i, sub := range subagents {
				prefix := "├─"
				if i == len(subagents)-1 {
					prefix = "└─"
				}
				role := sub.Role
				if role == "" {
					role = sub.ID
				}
				if len(role) > s.width-12 && s.width > 12 {
					role = role[:s.width-15] + "..."
				}
				lines = append(lines, labelStyle.Render(fmt.Sprintf("%s %s [%s]%s", prefix, role, sub.Status, subagentElapsed(sub))))
			}
		}
	}

	lines = append(lines, "")

	// Section 5: Changed Files
	renderHeader("changes", "CHANGED FILES")
	if s.sections["changes"] {
		var changes []runtime.Change
		if s.app != nil && s.app.Runner != nil && s.session.ID != "" {
			changes = s.app.Runner.Changes(s.session.ID)
		}

		if len(changes) == 0 {
			lines = append(lines, labelStyle.Render("None in this run"))
		} else {
			for _, change := range changes {
				icon := "●"
				if change.Operation == "created" {
					icon = "+"
				} else if change.Operation == "deleted" {
					icon = "✕"
				}
				path := change.Path
				if len(path) > s.width-6 && s.width > 6 {
					path = "..." + path[len(path)-(s.width-9):]
				}
				lines = append(lines, valueStyle.Render(fmt.Sprintf("%s %s", icon, path)))
			}
		}
	}

	lines = append(lines, "")

	// Section 6: Shortcuts Cheat-Sheet
	renderHeader("shortcuts", "SHORTCUTS")
	if s.sections["shortcuts"] {
		shortcuts := []struct {
			key  string
			desc string
		}{
			{"/", "commands"},
			{"@", "files"},
			{"ctrl+t", "new tab"},
			{"ctrl+w", "close tab"},
			{"ctrl+b", "toggle panel"},
			{"ctrl+o", "models"},
			{"ctrl+s", "sessions"},
			{"ctrl+q", "quit"},
		}

		for _, sc := range shortcuts {
			lines = append(lines, labelStyle.Render(fmt.Sprintf("%-7s %s", sc.key, sc.desc)))
		}
	}

	s.viewport.SetContent(strings.Join(lines, "\n"))
	return s.viewport.View()
}

// NewSidebarCmp creates a new sidebar panel component.
func NewSidebarCmp(app *app.App) SidebarCmp {
	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	return &sidebarCmp{
		app:      app,
		viewport: vp,
		expanded: make(map[string]bool),
		sections: map[string]bool{
			"status":    true,
			"tokens":    true,
			"goals":     true,
			"workforce": true,
			"changes":   true,
			"shortcuts": true,
		},
	}
}
