package chat

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
)

// SidebarCmp is the collapsible right sidebar panel displaying session details and changed files.
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

	var lines []string

	// Section 1: Session Status & Details
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

	lines = append(lines, "")

	// Section 2: Changed Files
	lines = append(lines, headerStyle.Render("CHANGED FILES"))

	var changes []runtime.Change
	if s.app != nil && s.app.Runner != nil && s.session.ID != "" {
		changes = s.app.Runner.Changes(s.session.ID)
	}

	if len(changes) == 0 {
		lines = append(lines, labelStyle.Render("None in this run"))
	} else {
		maxFiles := s.height - len(lines) - 8
		if maxFiles < 1 {
			maxFiles = 1
		}
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

	// Section 3: Shortcuts Cheat-Sheet
	lines = append(lines, headerStyle.Render("QUICK SHORTCUTS"))
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"/", "commands"},
		{"@", "files"},
		{"ctrl+b", "toggle panel"},
		{"ctrl+k", "palette"},
		{"ctrl+o", "models"},
		{"ctrl+s", "sessions"},
		{"ctrl+g", "diffs"},
		{"ctrl+l", "logs"},
		{"ctrl+q", "quit"},
	}

	for _, sc := range shortcuts {
		if len(lines) >= s.height-1 {
			break
		}
		lines = append(lines, labelStyle.Render(fmt.Sprintf("%-7s %s", sc.key, sc.desc)))
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
