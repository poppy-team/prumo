package dialog

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/diff"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// CloseFilesDialogMsg is sent when the changed-files panel is closed.
type CloseFilesDialogMsg struct{}

// ViewDiffMsg requests on-demand file diff from the harness (ADR 014).
type ViewDiffMsg struct {
	Path string
}

// DiffLoadedMsg delivers the fetched diff content.
type DiffLoadedMsg struct {
	Path    string
	Kind    string
	Content string
	Err     error
}

// ChangedFile is one file a run touched, as the harness reported it.
//
// It is the dialog's own type rather than the runtime's, so the view layer does
// not depend on the transport: the panel draws what it is handed.
type ChangedFile struct {
	Path      string
	Operation string
}

// FilesDialog is the panel listing what the current run changed and viewing their diffs.
type FilesDialog interface {
	tea.Model
	layout.Bindings
	SetFiles(files []ChangedFile)
	SetDiff(msg DiffLoadedMsg)
}

type filesDialogCmp struct {
	files       []ChangedFile
	selectedIdx int
	width       int
	height      int

	// Diff view state (ADR 014)
	viewingDiff bool
	diffLoading bool
	diffPath    string
	diffKind    string
	diffErr     error
	viewport    viewport.Model
}

type filesKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Diff   key.Binding
	Escape key.Binding
	Back   key.Binding
	J      key.Binding
	K      key.Binding
}

var filesKeys = filesKeyMap{
	Up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "previous file")),
	Down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next file")),
	Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view diff")),
	Diff:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "view diff")),
	Escape: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
	Back:   key.NewBinding(key.WithKeys("esc", "q", "backspace"), key.WithHelp("esc/q", "back to files")),
	J:      key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "next file")),
	K:      key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "previous file")),
}

func (f *filesDialogCmp) Init() tea.Cmd { return nil }

func (f *filesDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if f.viewingDiff {
			switch {
			case key.Matches(msg, filesKeys.Back):
				f.viewingDiff = false
				f.diffLoading = false
				return f, nil
			default:
				vp, cmd := f.viewport.Update(msg)
				f.viewport = vp
				return f, cmd
			}
		}

		switch {
		case key.Matches(msg, filesKeys.Up) || key.Matches(msg, filesKeys.K):
			if f.selectedIdx > 0 {
				f.selectedIdx--
			}
		case key.Matches(msg, filesKeys.Down) || key.Matches(msg, filesKeys.J):
			if f.selectedIdx < len(f.files)-1 {
				f.selectedIdx++
			}
		case key.Matches(msg, filesKeys.Enter) || key.Matches(msg, filesKeys.Diff):
			if len(f.files) > 0 && f.selectedIdx < len(f.files) {
				path := f.files[f.selectedIdx].Path
				f.viewingDiff = true
				f.diffLoading = true
				f.diffPath = path
				f.diffKind = ""
				f.diffErr = nil
				f.viewport.SetContent("Loading diff...")
				return f, util.CmdHandler(ViewDiffMsg{Path: path})
			}
		case key.Matches(msg, filesKeys.Escape):
			return f, util.CmdHandler(CloseFilesDialogMsg{})
		}
	case tea.WindowSizeMsg:
		f.width, f.height = msg.Width, msg.Height
		f.viewport.SetWidth(max(40, f.width-12))
		f.viewport.SetHeight(max(10, f.height-10))
	}
	return f, nil
}

func (f *filesDialogCmp) SetDiff(msg DiffLoadedMsg) {
	f.diffLoading = false
	f.diffPath = msg.Path
	f.diffKind = msg.Kind
	f.diffErr = msg.Err
	if msg.Err != nil {
		f.viewport.SetContent(fmt.Sprintf("Error loading diff: %v", msg.Err))
		return
	}
	vpWidth := max(40, f.width-12)
	var body string
	switch msg.Kind {
	case "patch":
		body, _ = diff.FormatDiff(msg.Content, diff.WithTotalWidth(vpWidth))
	case "created":
		body = "(new file)\n\n" + msg.Content
	case "deleted":
		body = "(deleted file)"
	case "moved":
		body = msg.Content
	default:
		if msg.Content != "" {
			body, _ = diff.FormatDiff(msg.Content, diff.WithTotalWidth(vpWidth))
		} else {
			body = "(" + msg.Kind + ")"
		}
	}
	f.viewport.SetContent(body)
	f.viewport.GotoTop()
}

// View renders the component for the terminal.
func (f *filesDialogCmp) View() tea.View { return tea.NewView(f.viewString()) }

func (f *filesDialogCmp) viewString() string {
	if f.viewingDiff {
		return f.viewDiffString()
	}
	return f.viewListString()
}

func (f *filesDialogCmp) viewDiffString() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	kindLabel := f.diffKind
	if kindLabel == "" {
		kindLabel = "diff"
	}
	title := baseStyle.Foreground(t.Primary()).Bold(true).Padding(0, 1).Render(fmt.Sprintf("Diff: %s (%s)", f.diffPath, kindLabel))
	help := baseStyle.Foreground(t.TextMuted()).Padding(0, 1).Render("[esc/q] back to file list  [↑/↓] scroll")

	contentWidth := max(40, min(f.width-10, 90))
	contentHeight := max(10, min(f.height-8, 24))
	f.viewport.SetWidth(contentWidth)
	f.viewport.SetHeight(contentHeight)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		help,
		baseStyle.Width(contentWidth).Render(""),
		f.viewport.View(),
	)

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (f *filesDialogCmp) viewListString() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	title := baseStyle.Foreground(t.Primary()).Bold(true).Padding(0, 1).Render("Files this run changed")

	if len(f.files) == 0 {
		return baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(t.Background()).
			BorderForeground(t.TextMuted()).
			Width(46).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				title,
				baseStyle.Width(44).Render(""),
				baseStyle.Foreground(t.TextMuted()).Width(44).Padding(0, 1).Render("no file has changed yet"),
			))
	}

	maxWidth := 46
	for _, file := range f.files {
		if len(file.Path)+len(file.Operation)+4 > maxWidth {
			maxWidth = min(len(file.Path)+len(file.Operation)+4, max(30, f.width-15))
		}
	}

	rows := make([]string, 0, len(f.files))
	for i, file := range f.files {
		row := fmt.Sprintf("%-9s %s", file.Operation, file.Path)
		rowStyle := baseStyle.Width(maxWidth)
		if i == f.selectedIdx {
			rowStyle = rowStyle.Background(t.Primary()).Foreground(t.Background()).Bold(true)
		}
		rows = append(rows, rowStyle.Padding(0, 1).Render(row))
	}

	help := baseStyle.Foreground(t.TextMuted()).Padding(0, 1).Render("[enter/d] view diff  [esc] close")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		help,
		baseStyle.Width(maxWidth).Render(""),
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(lipgloss.Width(content) + 4).
		Render(content)
}

func (f *filesDialogCmp) BindingKeys() []key.Binding {
	if f.viewingDiff {
		return []key.Binding{filesKeys.Back, filesKeys.Up, filesKeys.Down}
	}
	return []key.Binding{filesKeys.Up, filesKeys.Down, filesKeys.Enter, filesKeys.Escape}
}

func (f *filesDialogCmp) SetFiles(files []ChangedFile) {
	f.files = files
	f.viewingDiff = false
	f.diffLoading = false
	if f.selectedIdx >= len(files) {
		f.selectedIdx = max(0, len(files)-1)
	}
}

// NewFilesDialogCmp creates the changed-files panel.
func NewFilesDialogCmp() FilesDialog {
	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	return &filesDialogCmp{
		files:    []ChangedFile{},
		viewport: vp,
	}
}
