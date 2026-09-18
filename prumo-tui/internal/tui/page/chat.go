package page

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/completions"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

var ChatPage PageID = "chat"

type chatPage struct {
	app                  *app.App
	editor               layout.Container
	editorCmp            chat.EditorCmp
	messages             layout.Container
	sidebar              layout.Container
	sidebarCmp           chat.SidebarCmp
	showSidebar          bool
	layout               layout.SplitPaneLayout
	session              session.Session
	completionDialog     dialog.CompletionDialog
	filesProvider        dialog.CompletionProvider
	slashProvider        dialog.CompletionProvider
	showCompletionDialog bool
}

type ChatKeyMap struct {
	ShowCompletionDialog   key.Binding
	ShowCommandsCompletion key.Binding
	ToggleSidebar          key.Binding
	NewSession             key.Binding
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
		cmd := p.layout.SetSize(msg.Width, msg.Height)
		cmds = append(cmds, cmd)
	case dialog.CompletionDialogCloseMsg:
		p.showCompletionDialog = false
	case layout.FocusMsg:
		// The composer is the surface that carries a border, so it is the one
		// that shows whether the keyboard is on the page or on a dialog.
		p.editor.SetFocused(msg.Focused)
		return p, nil
	case chat.ToggleSidebarMsg:
		return p, p.toggleSidebar()
	case chat.SendMsg:
		return p, p.sendMessage(msg.Text)
	case chat.SessionSelectedMsg:
		p.session = msg
		p.sidebarCmp.UpdateSession(p.session)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.ShowCompletionDialog):
			p.completionDialog.SetProvider(p.filesProvider)
			p.showCompletionDialog = true
			// Continue sending keys to layout->chat
		case key.Matches(msg, keyMap.ShowCommandsCompletion):
			val := strings.TrimSpace(p.editorCmp.Value())
			if val == "" || val == "/" || strings.HasSuffix(val, "\n") {
				p.completionDialog.SetProvider(p.slashProvider)
				p.showCompletionDialog = true
			}
		case key.Matches(msg, keyMap.ToggleSidebar):
			return p, p.toggleSidebar()
		case key.Matches(msg, keyMap.NewSession):
			p.session = session.Session{}
			p.sidebarCmp.UpdateSession(p.session)
			return p, tea.Batch(
				util.CmdHandler(chat.SessionClearedMsg{}),
			)
		case key.Matches(msg, keyMap.Cancel):
			if p.session.ID != "" {
				// Cancelling is a protocol call: the daemon owns the loop and
				// stops it. The client only asks.
				p.app.CoderAgent.Cancel(p.session.ID)
				return p, nil
			}
		}
	}
	if p.showCompletionDialog {
		context, contextCmd := p.completionDialog.Update(msg)
		p.completionDialog = context.(dialog.CompletionDialog)
		cmds = append(cmds, contextCmd)

		// Enter closes the dialog rather than also sending what it selected.
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

func (p *chatPage) toggleSidebar() tea.Cmd {
	p.showSidebar = !p.showSidebar
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

// sendMessage starts a run for what was typed.
//
// It returns a command rather than doing the work inline because starting a run
// crosses a socket: a view that called it directly would freeze on every
// message it sent. What the run produces does not come back through this
// command either — the runner publishes it and the stream bridge carries it to
// the program, which is the only path the conversation travels.
func (p *chatPage) sendMessage(text string) tea.Cmd {
	app := p.app
	session := p.session
	return func() tea.Msg {
		ctx := context.Background()
		if session.ID == "" {
			created, err := app.Sessions.Create(ctx, "New Session")
			if err != nil {
				// Failures name the operation and the next action, so the reader
				// is not left to work out what the client was trying to do.
				return util.ReportFailure("Starting the session", "press enter to try again", err)()
			}
			session = created
		}

		if app.CoderAgent.IsSessionBusy(session.ID) {
			// A run is already going, so this is not a new goal: it is something
			// said to the run that is running.
			if err := app.CoderAgent.Steer(ctx, session.ID, text); err != nil {
				return util.ReportFailure("Adding to the run in flight", "wait for it to finish, or press esc to cancel it", err)()
			}
			return chat.SessionSelectedMsg(session)
		}

		if _, err := app.CoderAgent.Run(ctx, session.ID, text); err != nil {
			return util.ReportFailure("Starting the run", "press enter to try again, or read the log with ctrl+l", err)()
		}
		// Selecting the session is what tells the view which conversation it is
		// now drawing, and it is what the editor keys off to know a run exists.
		return chat.SessionSelectedMsg(session)
	}
}

func (p *chatPage) SetSize(width, height int) tea.Cmd {
	return p.layout.SetSize(width, height)
}

func (p *chatPage) GetSize() (int, int) {
	return p.layout.GetSize()
}

// View renders the component for the terminal.
func (p *chatPage) View() tea.View { return tea.NewView(p.viewString()) }
func (p *chatPage) viewString() string {
	layoutView := p.layout.View().Content

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

	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(1, 1, 0, 1),
	)

	editorCmp := chat.NewEditorCmp(app).(chat.EditorCmp)
	editorContainer := layout.NewContainer(
		editorCmp,
		layout.WithBorder(true, false, false, false),
	)
	// The composer holds the keyboard when the page is first drawn; a dialog
	// opening is what takes it away.
	editorContainer.SetFocused(true)

	sidebarCmp := chat.NewSidebarCmp(app)
	sidebarContainer := layout.NewContainer(
		sidebarCmp,
		layout.WithBorder(false, false, false, true),
		layout.WithPadding(0, 1, 0, 1),
	)

	return &chatPage{
		app:                  app,
		editor:               editorContainer,
		editorCmp:            editorCmp,
		messages:             messagesContainer,
		sidebar:              sidebarContainer,
		sidebarCmp:           sidebarCmp,
		showSidebar:          false,
		completionDialog:     completionDialog,
		filesProvider:        filesProvider,
		slashProvider:        slashProvider,
		layout: layout.NewSplitPane(
			layout.WithLeftPanel(messagesContainer),
			layout.WithBottomPanel(editorContainer),
		),
	}
}
