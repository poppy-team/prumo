package dialog

import (
	"fmt"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// CloseDirtyDialogMsg indicates the dirty preflight dialog was closed.
type CloseDirtyDialogMsg struct {
	Action string // "stash", "commit", "proceed", "abort"
	Prompt string
}

// DirtyDialog is the modal pre-flight check when the workspace has uncommitted changes.
type DirtyDialog interface {
	tea.Model
	layout.Bindings
	SetDirty(prompt, workspace, summary string)
}

type dirtyDialogCmp struct {
	prompt      string
	workspace   string
	summary     string
	selectedBtn int // 0: Stash, 1: Commit, 2: Proceed, 3: Abort
}

type dirtyKeyMap struct {
	LeftRight key.Binding
	Tab       key.Binding
	Enter     key.Binding
	Esc       key.Binding
	Stash     key.Binding
	Commit    key.Binding
	Proceed   key.Binding
	Abort     key.Binding
}

var dirtyKeys = dirtyKeyMap{
	LeftRight: key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←/→", "switch buttons"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "next button"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm selection"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "abort"),
	),
	Stash: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stash changes"),
	),
	Commit: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "commit changes"),
	),
	Proceed: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "proceed anyway"),
	),
	Abort: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "abort"),
	),
}

// CheckDirtyStatus reports whether git workspace has uncommitted changes and returns a summary.
func CheckDirtyStatus(workspace string) (bool, string, error) {
	if workspace == "" {
		workspace = "."
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return false, "", nil
	}
	lines := strings.Split(trimmed, "\n")
	var summary string
	if len(lines) == 1 {
		summary = fmt.Sprintf("1 uncommitted change: %s", strings.TrimSpace(lines[0]))
	} else {
		summary = fmt.Sprintf("%d uncommitted changes (%s, ...)", len(lines), strings.TrimSpace(lines[0]))
	}
	return true, summary, nil
}

// NewDirtyDialog creates a new preflight dirty working tree dialog.
func NewDirtyDialog() DirtyDialog {
	return &dirtyDialogCmp{
		selectedBtn: 0,
	}
}

func (d *dirtyDialogCmp) Init() tea.Cmd {
	return nil
}

func (d *dirtyDialogCmp) SetDirty(prompt, workspace, summary string) {
	d.prompt = prompt
	d.workspace = workspace
	d.summary = summary
	d.selectedBtn = 0
}

func (d *dirtyDialogCmp) performStash() tea.Cmd {
	ws := d.workspace
	if ws == "" {
		ws = "."
	}
	cmd := exec.Command("git", "stash", "push", "-u", "-m", "prumo: preflight auto-stash")
	cmd.Dir = ws
	if out, err := cmd.CombinedOutput(); err != nil {
		return util.ReportFailure("Pre-flight Stash", string(out), err)
	}
	return util.CmdHandler(CloseDirtyDialogMsg{Action: "stash", Prompt: d.prompt})
}

func (d *dirtyDialogCmp) performCommit() tea.Cmd {
	ws := d.workspace
	if ws == "" {
		ws = "."
	}
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = ws
	if out, err := addCmd.CombinedOutput(); err != nil {
		return util.ReportFailure("Pre-flight Git Add", string(out), err)
	}
	commitCmd := exec.Command("git", "commit", "-m", "prumo: preflight auto-commit")
	commitCmd.Dir = ws
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return util.ReportFailure("Pre-flight Git Commit", string(out), err)
	}
	return util.CmdHandler(CloseDirtyDialogMsg{Action: "commit", Prompt: d.prompt})
}

func (d *dirtyDialogCmp) performProceed() tea.Cmd {
	return util.CmdHandler(CloseDirtyDialogMsg{Action: "proceed", Prompt: d.prompt})
}

func (d *dirtyDialogCmp) performAbort() tea.Cmd {
	return util.CmdHandler(CloseDirtyDialogMsg{Action: "abort", Prompt: d.prompt})
}

func (d *dirtyDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, dirtyKeys.Esc), key.Matches(msg, dirtyKeys.Abort):
			return d, d.performAbort()

		case key.Matches(msg, dirtyKeys.Stash):
			return d, d.performStash()

		case key.Matches(msg, dirtyKeys.Commit):
			return d, d.performCommit()

		case key.Matches(msg, dirtyKeys.Proceed):
			return d, d.performProceed()

		case key.Matches(msg, dirtyKeys.LeftRight):
			if msg.String() == "left" {
				d.selectedBtn = (d.selectedBtn + 3) % 4
			} else {
				d.selectedBtn = (d.selectedBtn + 1) % 4
			}
			return d, nil

		case key.Matches(msg, dirtyKeys.Tab):
			if msg.String() == "shift+tab" {
				d.selectedBtn = (d.selectedBtn + 3) % 4
			} else {
				d.selectedBtn = (d.selectedBtn + 1) % 4
			}
			return d, nil

		case key.Matches(msg, dirtyKeys.Enter) || msg.String() == " ":
			switch d.selectedBtn {
			case 0:
				return d, d.performStash()
			case 1:
				return d, d.performCommit()
			case 2:
				return d, d.performProceed()
			case 3:
				return d, d.performAbort()
			}
		}
	}
	return d, nil
}

func (d *dirtyDialogCmp) View() tea.View {
	return tea.NewView(d.viewString())
}

func (d *dirtyDialogCmp) viewString() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	const dialogWidth = 56

	titleHeader := baseStyle.Bold(true).Foreground(t.Warning()).Render("⚠  Uncommitted Workspace Changes")
	descText := baseStyle.Foreground(t.Text()).Render("The working tree contains uncommitted changes.")

	var summaryText string
	if d.summary != "" {
		summaryText = baseStyle.Foreground(t.TextMuted()).Italic(true).Render(d.summary)
	}

	hintText := baseStyle.Foreground(t.TextMuted()).Render("Running the agent now may overwrite or interleave with your changes.")

	btnLabels := []string{"[s] Stash", "[c] Commit", "[p] Proceed", "[a] Abort"}
	renderedBtns := make([]string, len(btnLabels))

	for i, label := range btnLabels {
		btnStyle := baseStyle.Padding(0, 1)
		if d.selectedBtn == i {
			btnStyle = btnStyle.Background(t.Primary()).Foreground(t.Background()).Bold(true)
		} else {
			btnStyle = btnStyle.Background(t.BackgroundSecondary()).Foreground(t.Text())
		}
		renderedBtns[i] = btnStyle.Render(label)
	}

	btnRow := lipgloss.JoinHorizontal(lipgloss.Left,
		renderedBtns[0], "  ",
		renderedBtns[1], "  ",
		renderedBtns[2], "  ",
		renderedBtns[3],
	)

	alignedButtons := baseStyle.Width(dialogWidth - 2).Align(lipgloss.Center).Render(btnRow)

	contentLines := []string{
		titleHeader,
		"",
		descText,
	}
	if summaryText != "" {
		contentLines = append(contentLines, summaryText)
	}
	contentLines = append(contentLines,
		"",
		hintText,
		"",
		alignedButtons,
	)

	content := lipgloss.JoinVertical(lipgloss.Left, contentLines...)

	return baseStyle.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.Warning()).
		Render(content)
}

func (d *dirtyDialogCmp) BindingKeys() []key.Binding {
	return []key.Binding{
		dirtyKeys.Stash,
		dirtyKeys.Commit,
		dirtyKeys.Proceed,
		dirtyKeys.Abort,
	}
}
