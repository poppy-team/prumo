package chat

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
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
}

type sidebarCmp struct {
	app     *app.App
	session session.Session
	width   int
	height  int
}

func (s *sidebarCmp) Init() tea.Cmd {
	return nil
}

func (s *sidebarCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
	case SessionSelectedMsg:
		s.session = msg
	case SessionClearedMsg:
		s.session = session.Session{}
	}
	return s, nil
}

func (s *sidebarCmp) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height
	return nil
}

func (s *sidebarCmp) GetSize() (int, int) {
	return s.width, s.height
}

func (s *sidebarCmp) BindingKeys() []key.Binding {
	return nil
}

func (s *sidebarCmp) UpdateSession(sess session.Session) {
	s.session = sess
}

func (s *sidebarCmp) View() tea.View {
	return tea.NewView(s.viewString())
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

	// Section 1: Session Status & Model
	lines = append(lines, headerStyle.Render("SESSION STATUS"))

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

	lines = append(lines, "")

	// Section 2: Context & Token Occupancy (OpenCode v2 Telemetry)
	lines = append(lines, headerStyle.Render("CONTEXT TOKENS"))

	totalTokens := s.session.PromptTokens + s.session.CompletionTokens
	contextLimit := int64(128_000)
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

	lines = append(lines, "")

	// Section 3: Workforce & Subagents (Prumo Exclusive)
	lines = append(lines, headerStyle.Render("WORKFORCE"))
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
			lines = append(lines, labelStyle.Render(fmt.Sprintf("%s %s [%s]", prefix, role, sub.Status)))
		}
	}

	lines = append(lines, "")

	// Section 4: Changed Files
	lines = append(lines, headerStyle.Render("CHANGED FILES"))

	var changes []runtime.Change
	if s.app != nil && s.app.Runner != nil && s.session.ID != "" {
		changes = s.app.Runner.Changes(s.session.ID)
	}

	if len(changes) == 0 {
		lines = append(lines, labelStyle.Render("None in this run"))
	} else {
		maxFiles := max(2, s.height-len(lines)-9)
		for i, change := range changes {
			if i >= maxFiles {
				lines = append(lines, labelStyle.Render(fmt.Sprintf("... and %d more", len(changes)-i)))
				break
			}
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

	lines = append(lines, "")

	// Section 5: Shortcuts Cheat-Sheet
	if len(lines) < s.height-4 {
		lines = append(lines, headerStyle.Render("SHORTCUTS"))
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
			if len(lines) >= s.height-1 {
				break
			}
			lines = append(lines, labelStyle.Render(fmt.Sprintf("%-7s %s", sc.key, sc.desc)))
		}
	}

	// Truncate to height to guarantee no terminal overflow
	if len(lines) > s.height {
		lines = lines[:s.height]
	}

	content := strings.Join(lines, "\n")
	return baseStyle.Height(s.height).Render(content)
}

// NewSidebarCmp creates a new sidebar panel component.
func NewSidebarCmp(app *app.App) SidebarCmp {
	return &sidebarCmp{
		app: app,
	}
}
