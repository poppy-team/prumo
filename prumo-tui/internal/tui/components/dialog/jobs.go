package dialog

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// CloseJobsDialogMsg is sent when the scheduled-runs panel is closed.
type CloseJobsDialogMsg struct{}

// UnscheduleJobMsg asks for a recurring run to be stopped.
type UnscheduleJobMsg struct{ JobID string }

// ScheduledJob is one run the daemon repeats.
type ScheduledJob struct {
	ID        string
	Goal      string
	EverySecs int64
	NextRun   int64
}

// JobsDialog is the panel listing what the daemon runs on a schedule.
type JobsDialog interface {
	tea.Model
	layout.Bindings
	SetJobs(jobs []ScheduledJob)
}

type jobsDialogCmp struct {
	jobs        []ScheduledJob
	selectedIdx int
	width       int
	height      int
}

type jobsKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Cancel key.Binding
	Escape key.Binding
	J      key.Binding
	K      key.Binding
}

var jobsKeys = jobsKeyMap{
	Up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "previous")),
	Down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next")),
	Cancel: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "stop repeating it")),
	Escape: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
	J:      key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "next")),
	K:      key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "previous")),
}

func (j *jobsDialogCmp) Init() tea.Cmd { return nil }

func (j *jobsDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, jobsKeys.Up) || key.Matches(msg, jobsKeys.K):
			if j.selectedIdx > 0 {
				j.selectedIdx--
			}
		case key.Matches(msg, jobsKeys.Down) || key.Matches(msg, jobsKeys.J):
			if j.selectedIdx < len(j.jobs)-1 {
				j.selectedIdx++
			}
		case key.Matches(msg, jobsKeys.Cancel):
			if len(j.jobs) > 0 {
				return j, util.CmdHandler(UnscheduleJobMsg{JobID: j.jobs[j.selectedIdx].ID})
			}
		case key.Matches(msg, jobsKeys.Escape):
			return j, util.CmdHandler(CloseJobsDialogMsg{})
		}
	case tea.WindowSizeMsg:
		j.width, j.height = msg.Width, msg.Height
	}
	return j, nil
}

// View renders the component for the terminal.
func (j *jobsDialogCmp) View() tea.View { return tea.NewView(j.viewString()) }
func (j *jobsDialogCmp) viewString() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	title := baseStyle.Foreground(t.Primary()).Bold(true).Padding(0, 1).Render("Runs on a schedule")

	if len(j.jobs) == 0 {
		return baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(t.Background()).
			BorderForeground(t.TextMuted()).
			Width(52).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				title,
				baseStyle.Width(50).Render(""),
				baseStyle.Foreground(t.TextMuted()).Width(50).Padding(0, 1).
					Render("nothing is scheduled; `prumo agent schedule` adds one"),
			))
	}

	maxWidth := 52
	rows := make([]string, 0, len(j.jobs))
	for i, job := range j.jobs {
		// The row says what a reader needs to decide whether to stop it: what it
		// runs, how often, and when it runs next.
		row := fmt.Sprintf("%-9s %s", every(job.EverySecs), truncate(job.Goal, maxWidth-30))
		if job.NextRun > 0 {
			row = fmt.Sprintf("%s  next %s", row, time.Unix(job.NextRun, 0).Format("15:04"))
		}
		rowStyle := baseStyle.Width(maxWidth)
		if i == j.selectedIdx {
			rowStyle = rowStyle.Background(t.Primary()).Foreground(t.Background()).Bold(true)
		}
		rows = append(rows, rowStyle.Padding(0, 1).Render(row))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		baseStyle.Width(maxWidth).Render(""),
		lipgloss.JoinVertical(lipgloss.Left, rows...),
		baseStyle.Width(maxWidth).Render(""),
		baseStyle.Foreground(t.TextMuted()).Width(maxWidth).Padding(0, 1).Render("d stops the selected one; esc closes"),
	)

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (j *jobsDialogCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(jobsKeys)
}

func (j *jobsDialogCmp) SetJobs(jobs []ScheduledJob) {
	j.jobs = jobs
	if j.selectedIdx >= len(jobs) {
		j.selectedIdx = max(0, len(jobs)-1)
	}
}

// every renders an interval the way a person reads one.
func every(secs int64) string {
	switch {
	case secs <= 0:
		return "always"
	case secs%3600 == 0:
		return fmt.Sprintf("%dh", secs/3600)
	case secs%60 == 0:
		return fmt.Sprintf("%dm", secs/60)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

func truncate(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	return s[:width] + "…"
}

// NewJobsDialogCmp creates the scheduled-runs panel.
func NewJobsDialogCmp() JobsDialog {
	return &jobsDialogCmp{jobs: []ScheduledJob{}}
}
