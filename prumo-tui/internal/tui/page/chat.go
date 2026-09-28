package page

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/completions"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

var ChatPage PageID = "chat"

type SessionTab struct {
	ID              string
	Title           string
	Provider        string
	Model           string
	ReasoningEffort string
	Workspace       string
}

type chatPage struct {
	app                  *app.App
	editor               layout.Container
	editorCmp            chat.EditorCmp
	messages             layout.Container
	messagesCmp          chat.MessagesCmp
	lastVimChord         rune
	sidebar              layout.Container
	sidebarCmp           chat.SidebarCmp
	showSidebar          bool
	layout               layout.SplitPaneLayout
	session              session.Session
	completionDialog     dialog.CompletionDialog
	filesProvider        dialog.CompletionProvider
	slashProvider        dialog.CompletionProvider
	showCompletionDialog bool

	tabs           []SessionTab
	activeTabIndex int
	width          int
	height         int
}

type ChatKeyMap struct {
	ShowCompletionDialog   key.Binding
	ShowCommandsCompletion key.Binding
	ToggleSidebar          key.Binding
	NewSession             key.Binding
	NewTab                 key.Binding
	CloseTab               key.Binding
	NextTab                key.Binding
	PrevTab                key.Binding
	Cancel                 key.Binding
}

var keyMap = ChatKeyMap{
	ShowCompletionDialog: key.NewBinding(
		key.WithKeys("@"),
		key.WithHelp("@", "complete files"),
	),
	ShowCommandsCompletion: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "slash commands"),
	),
	ToggleSidebar: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "toggle sidebar"),
	),
	NewSession: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new session"),
	),
	NewTab: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "new tab"),
	),
	CloseTab: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("ctrl+w", "close tab"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("alt+]", "ctrl+pgdown"),
		key.WithHelp("alt+]", "next tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("alt+[", "ctrl+pgup"),
		key.WithHelp("alt+[", "prev tab"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (p *chatPage) Init() tea.Cmd {
	cmds := []tea.Cmd{
		p.layout.Init(),
		p.completionDialog.Init(),
		p.sidebar.Init(),
	}
	return tea.Batch(cmds...)
}

func (p *chatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		cmds = append(cmds, p.updateLayoutSize())

	case dialog.CompletionDialogCloseMsg:
		p.showCompletionDialog = false

	case layout.FocusMsg:
		p.editor.SetFocused(msg.Focused)
		if msg.Focused {
			return p, p.editorCmp.Focus()
		}
		p.editorCmp.Blur()
		return p, nil

	case chat.ToggleSidebarMsg:
		return p, p.toggleSidebar()

	case chat.NewTabMsg:
		return p, p.newTab(msg)

	case chat.UpdateActiveTabMsg:
		if p.activeTabIndex >= 0 && p.activeTabIndex < len(p.tabs) {
			if msg.Title != "" {
				p.tabs[p.activeTabIndex].Title = msg.Title
			}
			if msg.Provider != "" {
				p.tabs[p.activeTabIndex].Provider = msg.Provider
			}
			if msg.Model != "" {
				p.tabs[p.activeTabIndex].Model = msg.Model
			}
			if msg.ReasoningEffort != "" {
				p.tabs[p.activeTabIndex].ReasoningEffort = msg.ReasoningEffort
			}
			if msg.Workspace != "" {
				p.tabs[p.activeTabIndex].Workspace = msg.Workspace
			}
			p.sidebarCmp.UpdateSession(p.session)
		}
		return p, nil

	case chat.SwitchTabMsg:
		return p, p.switchTab(msg.Index)

	case chat.NextTabMsg:
		return p, p.nextTab()

	case chat.PrevTabMsg:
		return p, p.prevTab()

	case chat.CloseTabMsg:
		return p, p.closeTab(msg.Index)

	case chat.SendMsg:
		return p, p.sendMessage(msg.Text)

	case chat.SessionSelectedMsg:
		p.session = msg
		p.sidebarCmp.UpdateSession(p.session)
		if p.activeTabIndex >= 0 && p.activeTabIndex < len(p.tabs) {
			p.tabs[p.activeTabIndex].ID = msg.ID
			if msg.Title != "" {
				p.tabs[p.activeTabIndex].Title = msg.Title
			}
		}

	case tea.KeyPressMsg:
		// Modal Vim navigation when composer is unfocused/unblurred
		if !p.editorCmp.Focused() {
			switch msg.String() {
			case "i", "a", "enter":
				p.lastVimChord = 0
				p.editor.SetFocused(true)
				return p, p.editorCmp.Focus()
			case "j", "down":
				p.lastVimChord = 0
				if p.messagesCmp != nil {
					p.messagesCmp.ScrollDown(1)
				}
				return p, nil
			case "k", "up":
				p.lastVimChord = 0
				if p.messagesCmp != nil {
					p.messagesCmp.ScrollUp(1)
				}
				return p, nil
			case "g":
				if p.lastVimChord == 'g' {
					p.lastVimChord = 0
					if p.messagesCmp != nil {
						p.messagesCmp.ScrollToTop()
					}
				} else {
					p.lastVimChord = 'g'
				}
				return p, nil
			case "G":
				p.lastVimChord = 0
				if p.messagesCmp != nil {
					p.messagesCmp.ScrollToBottom()
				}
				return p, nil
			case "ctrl+d":
				p.lastVimChord = 0
				if p.messagesCmp != nil {
					p.messagesCmp.ScrollDown(10)
				}
				return p, nil
			case "ctrl+u":
				p.lastVimChord = 0
				if p.messagesCmp != nil {
					p.messagesCmp.ScrollUp(10)
				}
				return p, nil
			}
			p.lastVimChord = 0
		}

		switch {
		case key.Matches(msg, keyMap.ShowCompletionDialog):
			p.completionDialog.SetProvider(p.filesProvider)
			p.showCompletionDialog = true

		case key.Matches(msg, keyMap.ShowCommandsCompletion):
			val := strings.TrimSpace(p.editorCmp.Value())
			if val == "" || val == "/" || strings.HasSuffix(val, "\n") {
				p.completionDialog.SetProvider(p.slashProvider)
				p.showCompletionDialog = true
			}

		case key.Matches(msg, keyMap.ToggleSidebar):
			return p, p.toggleSidebar()

		case key.Matches(msg, keyMap.NewTab):
			return p, p.newTab(chat.NewTabMsg{})

		case key.Matches(msg, keyMap.CloseTab):
			return p, p.closeTab(p.activeTabIndex)

		case key.Matches(msg, keyMap.NextTab):
			return p, p.nextTab()

		case key.Matches(msg, keyMap.PrevTab):
			return p, p.prevTab()

		case key.Matches(msg, keyMap.NewSession):
			p.session = session.Session{}
			p.sidebarCmp.UpdateSession(p.session)
			if p.activeTabIndex >= 0 && p.activeTabIndex < len(p.tabs) {
				ws := ""
				if p.app != nil {
					ws = p.app.Workspace
				}
				p.tabs[p.activeTabIndex].ID = ""
				p.tabs[p.activeTabIndex].Title = util.DefaultSessionTitle(ws)
			}
			return p, tea.Batch(
				util.CmdHandler(chat.SessionClearedMsg{}),
			)

		case key.Matches(msg, keyMap.Cancel):
			if p.session.ID != "" && p.app != nil && p.app.CoderAgent != nil && p.app.CoderAgent.IsSessionBusy(p.session.ID) {
				p.app.CoderAgent.Cancel(p.session.ID)
				return p, nil
			}
			if p.editorCmp.Focused() {
				p.editorCmp.Blur()
				p.editor.SetFocused(false)
				return p, nil
			}
		}
	}

	if p.showCompletionDialog {
		context, contextCmd := p.completionDialog.Update(msg)
		p.completionDialog = context.(dialog.CompletionDialog)
		cmds = append(cmds, contextCmd)

		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if keyMsg.String() == "enter" {
				return p, tea.Batch(cmds...)
			}
		}
	}

	u, cmd := p.layout.Update(msg)
	cmds = append(cmds, cmd)
	p.layout = u.(layout.SplitPaneLayout)

	return p, tea.Batch(cmds...)
}

func (p *chatPage) newTab(msg chat.NewTabMsg) tea.Cmd {
	title := msg.Title
	if title == "" {
		title = fmt.Sprintf("Tab %d", len(p.tabs)+1)
	}
	provider := msg.Provider
	if provider == "" && p.app != nil {
		provider = p.app.CurrentProvider()
	}
	model := msg.Model
	if model == "" && p.app != nil && p.app.CoderAgent != nil {
		model = string(p.app.CoderAgent.Model().ID)
	}
	effort := msg.ReasoningEffort
	if effort == "" && p.app != nil {
		effort = p.app.CurrentReasoningEffort()
	}
	ws := msg.Workspace
	if ws == "" && p.app != nil {
		ws = p.app.CurrentWorkspace()
	}

	p.saveActiveTabState()

	p.tabs = append(p.tabs, SessionTab{
		ID:              "",
		Title:           title,
		Provider:        provider,
		Model:           model,
		ReasoningEffort: effort,
		Workspace:       ws,
	})
	p.activeTabIndex = len(p.tabs) - 1
	p.applyTabState(p.tabs[p.activeTabIndex])

	p.session = session.Session{Title: title}
	p.sidebarCmp.UpdateSession(p.session)
	return tea.Batch(
		util.CmdHandler(chat.SessionClearedMsg{}),
		p.updateLayoutSize(),
	)
}

func (p *chatPage) switchTab(index int) tea.Cmd {
	if len(p.tabs) == 0 {
		return nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(p.tabs) {
		index = len(p.tabs) - 1
	}

	p.saveActiveTabState()

	p.activeTabIndex = index
	tab := p.tabs[index]
	p.applyTabState(tab)

	if tab.ID != "" && p.app != nil && p.app.Sessions != nil {
		sess, err := p.app.Sessions.Get(context.Background(), tab.ID)
		if err == nil {
			p.session = sess
		} else {
			p.session = session.Session{ID: tab.ID, Title: tab.Title}
		}
		p.sidebarCmp.UpdateSession(p.session)
		return tea.Batch(
			util.CmdHandler(chat.SessionSelectedMsg(p.session)),
			p.updateLayoutSize(),
		)
	}

	p.session = session.Session{Title: tab.Title}
	p.sidebarCmp.UpdateSession(p.session)
	return tea.Batch(
		util.CmdHandler(chat.SessionClearedMsg{}),
		p.updateLayoutSize(),
	)
}

func (p *chatPage) nextTab() tea.Cmd {
	if len(p.tabs) > 1 {
		return p.switchTab((p.activeTabIndex + 1) % len(p.tabs))
	}
	return nil
}

func (p *chatPage) prevTab() tea.Cmd {
	if len(p.tabs) > 1 {
		return p.switchTab((p.activeTabIndex + len(p.tabs) - 1) % len(p.tabs))
	}
	return nil
}

func (p *chatPage) closeTab(index int) tea.Cmd {
	if len(p.tabs) <= 1 {
		ws := ""
		prov := "fake"
		if p.app != nil {
			ws = p.app.Workspace
			prov = p.app.CurrentProvider()
		}
		p.tabs[0] = SessionTab{
			Title:     util.DefaultSessionTitle(ws),
			Provider:  prov,
			Workspace: ws,
		}
		p.activeTabIndex = 0
		p.session = session.Session{Title: p.tabs[0].Title}
		p.sidebarCmp.UpdateSession(p.session)
		return tea.Batch(util.CmdHandler(chat.SessionClearedMsg{}), p.updateLayoutSize())
	}

	p.tabs = append(p.tabs[:index], p.tabs[index+1:]...)
	if p.activeTabIndex >= len(p.tabs) {
		p.activeTabIndex = len(p.tabs) - 1
	}
	tab := p.tabs[p.activeTabIndex]
	p.applyTabState(tab)
	return p.switchTab(p.activeTabIndex)
}

func (p *chatPage) saveActiveTabState() {
	if p.activeTabIndex >= 0 && p.activeTabIndex < len(p.tabs) && p.app != nil {
		p.tabs[p.activeTabIndex].Provider = p.app.CurrentProvider()
		if p.app.CoderAgent != nil {
			p.tabs[p.activeTabIndex].Model = string(p.app.CoderAgent.Model().ID)
		}
		p.tabs[p.activeTabIndex].ReasoningEffort = p.app.CurrentReasoningEffort()
		p.tabs[p.activeTabIndex].Workspace = p.app.CurrentWorkspace()
	}
}

func (p *chatPage) applyTabState(tab SessionTab) {
	if p.app == nil {
		return
	}
	if tab.Provider != "" {
		p.app.SetProvider(tab.Provider)
	}
	if tab.Model != "" {
		p.app.SetModel(models.ModelID(tab.Model))
	}
	if tab.ReasoningEffort != "" {
		p.app.SetReasoningEffort(tab.ReasoningEffort)
	}
	if tab.Workspace != "" {
		p.app.SetWorkspace(tab.Workspace)
	}
}

func (p *chatPage) updateLayoutSize() tea.Cmd {
	h := p.height
	if len(p.tabs) > 1 {
		h = max(0, p.height-1)
	}
	return p.layout.SetSize(p.width, h)
}

func (p *chatPage) toggleSidebar() tea.Cmd {
	p.showSidebar = !p.showSidebar
	_ = config.UpdateSidebarVisibility(p.showSidebar)
	if p.showSidebar {
		return p.setSidebar()
	}
	return p.clearSidebar()
}

func (p *chatPage) setSidebar() tea.Cmd {
	p.sidebarCmp.UpdateSession(p.session)
	return p.layout.SetRightPanel(p.sidebar)
}

func (p *chatPage) clearSidebar() tea.Cmd {
	return p.layout.ClearRightPanel()
}

func (p *chatPage) sendMessage(text string) tea.Cmd {
	app := p.app
	session := p.session
	return func() tea.Msg {
		ctx := context.Background()
		if session.ID == "" {
			title := session.Title
			if title == "" {
				ws := ""
				if app != nil {
					ws = app.Workspace
				}
				title = util.DefaultSessionTitle(ws)
			}
			created, err := app.Sessions.Create(ctx, title)
			if err != nil {
				return util.ReportFailure("Starting the session", "press enter to try again", err)()
			}
			session = created
		}

		if app.CoderAgent.IsSessionBusy(session.ID) {
			if err := app.CoderAgent.Steer(ctx, session.ID, text); err != nil {
				return util.ReportFailure("Adding to the run in flight", "wait for it to finish, or press esc to cancel it", err)()
			}
			return chat.SessionSelectedMsg(session)
		}

		if _, err := app.CoderAgent.Run(ctx, session.ID, text); err != nil {
			return util.ReportFailure("Starting the run", "press enter to try again, or read the log with ctrl+l", err)()
		}
		return chat.SessionSelectedMsg(session)
	}
}

func (p *chatPage) SetSize(width, height int) tea.Cmd {
	p.width = width
	p.height = height
	return p.updateLayoutSize()
}

func (p *chatPage) GetSize() (int, int) {
	return p.layout.GetSize()
}

func (p *chatPage) renderTabBar() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	var tabElements []string
	for i, tab := range p.tabs {
		tabTitle := tab.Title
		if tabTitle == "" {
			tabTitle = fmt.Sprintf("Tab %d", i+1)
		}
		if len(tabTitle) > 18 {
			tabTitle = tabTitle[:15] + "..."
		}
		label := fmt.Sprintf(" %d: %s ", i+1, tabTitle)
		var style lipgloss.Style
		if i == p.activeTabIndex {
			style = baseStyle.
				Background(t.Primary()).
				Foreground(t.Background()).
				Bold(true)
		} else {
			style = baseStyle.
				Background(t.BackgroundSecondary()).
				Foreground(t.TextMuted())
		}
		tabElements = append(tabElements, style.Render(label))
	}

	tabBarContent := lipgloss.JoinHorizontal(lipgloss.Left, tabElements...)
	totalWidth, _ := p.layout.GetSize()
	if totalWidth <= 0 {
		totalWidth = p.width
	}
	if totalWidth <= 0 {
		totalWidth = 80
	}
	remaining := max(0, totalWidth-lipgloss.Width(tabBarContent))
	filler := baseStyle.
		Background(t.BackgroundDarker()).
		Render(strings.Repeat(" ", remaining))

	return lipgloss.JoinHorizontal(lipgloss.Left, tabBarContent, filler)
}

// View renders the component for the terminal.
func (p *chatPage) View() tea.View { return tea.NewView(p.viewString()) }
func (p *chatPage) viewString() string {
	layoutView := p.layout.View().Content

	if len(p.tabs) > 1 {
		tabBar := p.renderTabBar()
		layoutView = lipgloss.JoinVertical(lipgloss.Left, tabBar, layoutView)
	}

	if p.showCompletionDialog {
		_, layoutHeight := p.layout.GetSize()
		editorWidth, editorHeight := p.editor.GetSize()

		p.completionDialog.SetWidth(editorWidth)
		overlay := p.completionDialog.View().Content

		layoutView = layout.PlaceOverlay(
			0,
			layoutHeight-editorHeight-lipgloss.Height(overlay),
			overlay,
			layoutView,
			false,
		)
	}

	return layoutView
}

func (p *chatPage) BindingKeys() []key.Binding {
	bindings := layout.KeyMapToSlice(keyMap)
	bindings = append(bindings, p.messages.BindingKeys()...)
	bindings = append(bindings, p.editor.BindingKeys()...)
	return bindings
}

func NewChatPage(app *app.App) tea.Model {
	filesProvider := completions.NewFileAndFolderContextGroup()
	slashProvider := completions.NewSlashCommandContextGroup()
	completionDialog := dialog.NewCompletionDialogCmp(filesProvider)

	messagesModel := chat.NewMessagesCmp(app)
	messagesCmp, _ := messagesModel.(chat.MessagesCmp)
	messagesContainer := layout.NewContainer(
		messagesModel,
		layout.WithPadding(1, 1, 0, 1),
	)

	editorCmp := chat.NewEditorCmp(app).(chat.EditorCmp)
	editorContainer := layout.NewContainer(
		editorCmp,
		layout.WithBorder(true, false, false, false),
	)
	editorContainer.SetFocused(true)

	sidebarCmp := chat.NewSidebarCmp(app)
	sidebarContainer := layout.NewContainer(
		sidebarCmp,
		layout.WithBorder(false, false, false, true),
		layout.WithPadding(0, 1, 0, 1),
	)

	initShowSidebar := config.Get().ShowSidebar
	var splitOpts []layout.SplitPaneOption
	splitOpts = append(splitOpts,
		layout.WithLeftPanel(messagesContainer),
		layout.WithBottomPanel(editorContainer),
	)
	if initShowSidebar {
		splitOpts = append(splitOpts, layout.WithRightPanel(sidebarContainer))
	}

	initTitle := "project"
	initProvider := "fake"
	initModel := ""
	initEffort := ""
	initWorkspace := ""
	if app != nil {
		initTitle = util.DefaultSessionTitle(app.Workspace)
		initProvider = app.CurrentProvider()
		if app.CoderAgent != nil {
			initModel = string(app.CoderAgent.Model().ID)
		}
		initEffort = app.CurrentReasoningEffort()
		initWorkspace = app.CurrentWorkspace()
	}

	return &chatPage{
		app:              app,
		editor:           editorContainer,
		editorCmp:        editorCmp,
		messages:         messagesContainer,
		messagesCmp:      messagesCmp,
		sidebar:          sidebarContainer,
		sidebarCmp:       sidebarCmp,
		showSidebar:      initShowSidebar,
		completionDialog: completionDialog,
		filesProvider:    filesProvider,
		slashProvider:    slashProvider,
		layout:           layout.NewSplitPane(splitOpts...),
		tabs: []SessionTab{
			{
				ID:              "",
				Title:           initTitle,
				Provider:        initProvider,
				Model:           initModel,
				ReasoningEffort: initEffort,
				Workspace:       initWorkspace,
			},
		},
		activeTabIndex: 0,
	}
}
