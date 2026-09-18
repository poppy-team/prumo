package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/commands"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/logging"
	"github.com/raillen/prumo-tui/internal/onboard"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/components/core"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
	"github.com/raillen/prumo-tui/internal/tui/layout"
	"github.com/raillen/prumo-tui/internal/tui/page"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

type keyMap struct {
	Cancel        key.Binding
	Logs          key.Binding
	Quit          key.Binding
	Help          key.Binding
	SwitchSession key.Binding
	Commands      key.Binding
	Filepicker    key.Binding
	Models        key.Binding
	SwitchTheme   key.Binding
	ChangedFiles  key.Binding
	Sidebar       key.Binding
}

const (
	quitKey = "q"
)

var keys = keyMap{
	Logs: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "logs"),
	),

	// The interaction contract reserves these two chords: ctrl+c cancels the
	// active operation and never exits silently, ctrl+q quits. The client obeys
	// the reservation rather than its own convenience, so a user coming from
	// another terminal tool finds the chords where they are documented.
	Cancel: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("ctrl+_", "ctrl+h"),
		key.WithHelp("ctrl+?", "toggle help"),
	),

	SwitchSession: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "switch session"),
	),

	Commands: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "commands"),
	),
	Filepicker: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "select files to upload"),
	),
	Models: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "model selection"),
	),

	SwitchTheme: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "switch theme"),
	),

	ChangedFiles: key.NewBinding(
		key.WithKeys("ctrl+g"),
		key.WithHelp("ctrl+g", "files this run changed"),
	),

	Sidebar: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "toggle sidebar"),
	),
}

var helpEsc = key.NewBinding(
	key.WithKeys("?"),
	key.WithHelp("?", "toggle help"),
)

var returnKey = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "close"),
)

var logsKeyReturnKey = key.NewBinding(
	key.WithKeys("esc", "backspace", quitKey),
	key.WithHelp("esc/q", "go back"),
)

type appModel struct {
	width, height   int
	currentPage     page.PageID
	previousPage    page.PageID
	pages           map[page.PageID]tea.Model
	loadedPages     map[page.PageID]bool
	status          core.StatusCmp
	app             *app.App
	selectedSession session.Session

	showPermissions bool
	permissions     dialog.PermissionDialogCmp

	showHelp bool
	help     dialog.HelpCmp

	showQuit      bool
	showOnboard   bool
	startupNotice string
	onboard       dialog.OnboardDialogCmp
	quit          dialog.QuitDialog

	showSessionDialog bool
	sessionDialog     dialog.SessionDialog

	showCommandDialog bool
	commandDialog     dialog.CommandDialog
	commands          []dialog.Command

	showModelDialog bool
	modelDialog     dialog.ModelDialog

	showInitDialog bool
	initDialog     dialog.InitDialogCmp

	showFilepicker bool
	filepicker     dialog.FilepickerCmp

	showThemeDialog bool
	themeDialog     dialog.ThemeDialog

	showMultiArgumentsDialog bool
	multiArgumentsDialog     dialog.MultiArgumentsDialogCmp

	showFiles bool
	files     dialog.FilesDialog

	showJobs bool
	jobs     dialog.JobsDialog

	showProviderDialog bool
	providerDialog     dialog.ProviderDialog

	// composerFocused is the focus the page was last told about, kept so the
	// shell tells it when the answer changes rather than on every message.
	composerFocused bool
	// commandsError is the command directory failing to read, reported once in
	// Init: the palette is built before there is a statusline to say it in.
	commandsError error
}

// openProviderDialogMsg asks for the provider configuration dialog to be shown.
type openProviderDialogMsg struct{}

func (a appModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmd := a.pages[a.currentPage].Init()
	a.loadedPages[a.currentPage] = true
	cmds = append(cmds, cmd)
	cmd = a.status.Init()
	cmds = append(cmds, cmd)
	cmd = a.quit.Init()
	if a.startupNotice != "" {
		notice := a.startupNotice
		a.startupNotice = ""
		cmds = append(cmds, util.ReportInfo(notice))
	}
	if a.onboard != nil {
		cmd = a.onboard.Init()
	}
	cmds = append(cmds, cmd)
	cmd = a.help.Init()
	cmds = append(cmds, cmd)
	cmd = a.sessionDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.commandDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.modelDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.initDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.filepicker.Init()
	cmds = append(cmds, cmd)
	cmd = a.themeDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.files.Init()
	cmds = append(cmds, cmd)
	cmd = a.jobs.Init()
	cmds = append(cmds, cmd)
	cmd = a.providerDialog.Init()
	cmds = append(cmds, cmd)

	// Check if we should show the init dialog
	cmds = append(cmds, func() tea.Msg {
		shouldShow, err := config.ShouldShowInitDialog()
		if err != nil {
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  "Failed to check init status: " + err.Error(),
			}
		}
		return dialog.ShowInitDialogMsg{Show: shouldShow}
	})

	cmds = append(cmds, a.reconcile())

	if a.commandsError != nil {
		cmds = append(cmds, util.ReportFailure("Reading your commands", "the palette still has its own entries", a.commandsError))
	}

	return tea.Batch(cmds...)
}

// Update applies a message and, when the answer changes who holds the keyboard,
// tells the composer.
//
// Focus is a decision of the shell rather than of the composer: a dialog takes
// the keyboard away, and a surface has no way to know that one opened. Deciding
// it here keeps that in one place instead of in every branch that opens or
// closes a layer.
func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := a.update(msg)
	next, ok := updated.(appModel)
	if !ok {
		return updated, cmd
	}
	if focused := !next.dialogOpen(); focused != next.composerFocused {
		next.composerFocused = focused
		return next, tea.Batch(cmd, util.CmdHandler(layout.FocusMsg{Focused: focused}))
	}
	return next, cmd
}

// dialogOpen reports whether a layer is drawn over the page.
func (a appModel) dialogOpen() bool {
	return a.showQuit || a.showOnboard || a.showPermissions || a.showHelp || a.showSessionDialog ||
		a.showCommandDialog || a.showModelDialog || a.showInitDialog || a.showFilepicker ||
		a.showThemeDialog || a.showMultiArgumentsDialog || a.showFiles || a.showJobs || a.showProviderDialog
}

func (a appModel) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		msg.Height -= 1 // Make space for the status bar
		a.width, a.height = msg.Width, msg.Height

		s, _ := a.status.Update(msg)
		a.status = s.(core.StatusCmp)
		a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
		cmds = append(cmds, cmd)

		prm, permCmd := a.permissions.Update(msg)
		a.permissions = prm.(dialog.PermissionDialogCmp)
		cmds = append(cmds, permCmd)

		help, helpCmd := a.help.Update(msg)
		a.help = help.(dialog.HelpCmp)
		cmds = append(cmds, helpCmd)

		session, sessionCmd := a.sessionDialog.Update(msg)
		a.sessionDialog = session.(dialog.SessionDialog)
		cmds = append(cmds, sessionCmd)

		command, commandCmd := a.commandDialog.Update(msg)
		a.commandDialog = command.(dialog.CommandDialog)
		cmds = append(cmds, commandCmd)

		filepicker, filepickerCmd := a.filepicker.Update(msg)
		a.filepicker = filepicker.(dialog.FilepickerCmp)
		cmds = append(cmds, filepickerCmd)

		// Every layer that draws is told the new size, not only the ones that
		// happened to be open: a dialog that resizes itself only after being
		// opened a second time is a dialog that draws for the previous terminal.
		model, modelCmd := a.modelDialog.Update(msg)
		a.modelDialog = model.(dialog.ModelDialog)
		cmds = append(cmds, modelCmd)

		theme, themeCmd := a.themeDialog.Update(msg)
		a.themeDialog = theme.(dialog.ThemeDialog)
		cmds = append(cmds, themeCmd)

		quit, quitCmd := a.quit.Update(msg)
		a.quit = quit.(dialog.QuitDialog)
		if a.onboard != nil {
			ob, obCmd := a.onboard.Update(msg)
			a.onboard = ob.(dialog.OnboardDialogCmp)
			cmds = append(cmds, obCmd)
		}
		cmds = append(cmds, quitCmd)

		files, filesCmd := a.files.Update(msg)
		a.files = files.(dialog.FilesDialog)
		cmds = append(cmds, filesCmd)

		jobs, jobsCmd := a.jobs.Update(msg)
		a.jobs = jobs.(dialog.JobsDialog)
		cmds = append(cmds, jobsCmd)

		prov, provCmd := a.providerDialog.Update(msg)
		a.providerDialog = prov.(dialog.ProviderDialog)
		cmds = append(cmds, provCmd)

		a.initDialog.SetSize(msg.Width, msg.Height)

		if a.showMultiArgumentsDialog {
			a.multiArgumentsDialog.SetSize(msg.Width, msg.Height)
			args, argsCmd := a.multiArgumentsDialog.Update(msg)
			a.multiArgumentsDialog = args.(dialog.MultiArgumentsDialogCmp)
			cmds = append(cmds, argsCmd, a.multiArgumentsDialog.Init())
		}

		return a, tea.Batch(cmds...)
	// Status
	case util.InfoMsg:
		s, cmd := a.status.Update(msg)
		a.status = s.(core.StatusCmp)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	case pubsub.Event[logging.LogMessage]:
		if msg.Payload.Persist {
			switch msg.Payload.Level {
			case "error":
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeError,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)
			case "info":
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeInfo,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)

			case "warn":
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeWarn,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})

				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)
			default:
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeInfo,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)
			}
		}
	case util.ClearStatusMsg:
		s, _ := a.status.Update(msg)
		a.status = s.(core.StatusCmp)

	// Permission
	case pubsub.Event[permission.PermissionRequest]:
		a.showPermissions = true
		// The gate is announced in words as well as drawn: a request that exists
		// only as a dialog cannot be read from a frame, and the contract asks for
		// it to be legible without interacting with it.
		return a, tea.Batch(
			a.permissions.SetPermissions(msg.Payload),
			util.ReportWarn(gateNotice(msg.Payload.ToolName)),
		)
	case openJobsMsg:
		return a, a.openJobs()
	case exportTimelineMsg:
		if a.selectedSession.ID == "" {
			return a, util.ReportWarn("Nothing to export: no session has been selected")
		}
		return a, a.exportTimeline(a.selectedSession.ID)
	case dialog.PermissionResponseMsg:
		var cmd tea.Cmd
		switch msg.Action {
		case dialog.PermissionAllow:
			if err := a.app.Permissions.Grant(context.Background(), msg.Permission); err != nil {
				cmd = util.ReportFailure("Allowing the tool", "answer the gate again, or deny it with d", err)
			}
		case dialog.PermissionAllowForSession:
			if err := a.app.Permissions.GrantPersistant(context.Background(), msg.Permission); err != nil {
				cmd = util.ReportFailure("Allowing the tool for the session", "answer the gate again, or deny it with d", err)
			}
		case dialog.PermissionDeny:
			if err := a.app.Permissions.Deny(context.Background(), msg.Permission); err != nil {
				cmd = util.ReportFailure("Denying the tool", "answer the gate again from the run's own state", err)
			} else {
				// The daemon reports the run as failed, and from the protocol
				// alone a denial is indistinguishable from any other stop. The
				// one place that difference exists is here, so it is said here.
				cmd = util.ReportWarn(denialNotice(msg.Permission.ToolName))
			}
		}
		a.showPermissions = false
		return a, cmd

	case page.PageChangeMsg:
		return a, a.moveToPage(msg.ID)

	case dialog.OnboardChoiceMsg:
		return a.answerOnboard(msg)

	case dialog.OnboardInstallFinishedMsg:
		return a.finishOnboardInstall(msg)

	case dialog.CloseQuitMsg:
		a.showQuit = false
		return a, nil

	case dialog.CloseSessionDialogMsg:
		a.showSessionDialog = false
		return a, nil

	case dialog.CloseCommandDialogMsg:
		a.showCommandDialog = false
		return a, nil

	case pubsub.Event[agent.AgentEvent]:
		payload := msg.Payload
		if payload.Error != nil {
			return a, util.ReportFailure("The run stopped", "send the goal again, or read the log with ctrl+l", payload.Error)
		}
		if payload.Type == agent.AgentEventTypeConnection {
			// The link is the client's own condition, so it reaches the
			// statusline rather than the transcript: a conversation that folded
			// its transport into its answers would be describing the wrong thing.
			s, cmd := a.status.Update(core.ConnectionMsg{
				State:   string(payload.Connection.State),
				Attempt: payload.Connection.Attempt,
				Of:      payload.Connection.Of,
				Reason:  payload.Connection.Reason,
			})
			a.status = s.(core.StatusCmp)
			return a, cmd
		}
		// A run event that is not an error needs nothing from this model: the
		// message store publishes what the conversation gained, and the
		// transcript and statusline redraw from that. Compaction is not
		// triggered here either — the harness compacts a run as it approaches
		// its budget, and the client has no operation to ask with.
		return a, nil

	case dialog.CloseThemeDialogMsg:
		a.showThemeDialog = false
		return a, nil

	case dialog.CloseFilesDialogMsg:
		a.showFiles = false
		return a, nil

	case dialog.ViewDiffMsg:
		return a, a.loadDiff(msg.Path)

	case dialog.DiffLoadedMsg:
		a.files.SetDiff(msg)
		return a, nil

	case dialog.CloseJobsDialogMsg:
		a.showJobs = false
		return a, nil

	case jobsLoadedMsg:
		jobs := make([]dialog.ScheduledJob, 0, len(msg.jobs))
		for _, job := range msg.jobs {
			jobs = append(jobs, dialog.ScheduledJob{ID: job.ID, Goal: job.Goal, EverySecs: job.EverySecs, NextRun: job.NextRun})
		}
		a.jobs.SetJobs(jobs)
		if msg.err != nil {
			return a, util.ReportFailure("Reading the daemon's schedule", "the runs themselves still work", msg.err)
		}
		return a, nil

	case dialog.UnscheduleJobMsg:
		return a, a.unschedule(msg.JobID)

	case dialog.ThemeChangedMsg:
		a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
		a.showThemeDialog = false
		return a, tea.Batch(cmd, util.ReportInfo("Theme changed to: "+msg.ThemeName))

	case dialog.ShowInitDialogMsg:
		a.showInitDialog = msg.Show
		return a, nil

	case dialog.LoadingModelsMsg, dialog.ModelsLoadedMsg:
		d, cmd := a.modelDialog.Update(msg)
		a.modelDialog = d.(dialog.ModelDialog)
		return a, cmd

	case dialog.CloseModelDialogMsg:
		a.showModelDialog = false
		return a, nil

	case dialog.ModelSelectedMsg:
		a.showModelDialog = false
		model, err := a.app.CoderAgent.Update(models.ModelID(msg.Model))
		if err != nil {
			return a, util.ReportFailure("Choosing the model", "pick another one with ctrl+o", err)
		}
		if err := config.UpdateModel(msg.Model); err != nil {
			logging.WarnPersist("could not save chosen model", "err", err)
		}
		return a, util.ReportInfo(fmt.Sprintf("Model selected: %s", model.Name))

	case dialog.CloseInitDialogMsg:
		a.showInitDialog = false
		if msg.Initialize {
			// Run the initialization command
			for _, cmd := range a.commands {
				if cmd.ID == "init" {
					// Mark the project as initialized
					if err := config.MarkProjectInitialized(); err != nil {
						return a, util.ReportError(err)
					}
					return a, cmd.Handler(cmd)
				}
			}
		} else {
			// Mark the project as initialized without running the command
			if err := config.MarkProjectInitialized(); err != nil {
				return a, util.ReportError(err)
			}
		}
		return a, nil

	case chat.SessionSelectedMsg:
		a.selectedSession = msg
		a.sessionDialog.SetSelectedSession(msg.ID)

	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == a.selectedSession.ID {
			a.selectedSession = msg.Payload
		}
	case dialog.SessionSelectedMsg:
		a.showSessionDialog = false
		if a.currentPage == page.ChatPage {
			return a, tea.Batch(
				util.CmdHandler(chat.SessionSelectedMsg(msg.Session)),
				a.attach(msg.Session.ID),
			)
		}
		return a, nil

	case dialog.CommandSelectedMsg:
		a.showCommandDialog = false
		// Execute the command handler if available
		if msg.Command.Handler != nil {
			return a, msg.Command.Handler(msg.Command)
		}
		return a, util.ReportInfo("Command selected: " + msg.Command.Title)

	case dialog.ShowMultiArgumentsDialogMsg:
		// Show multi-arguments dialog
		a.multiArgumentsDialog = dialog.NewMultiArgumentsDialogCmp(msg.CommandID, msg.Content, msg.ArgNames)
		a.showMultiArgumentsDialog = true
		return a, a.multiArgumentsDialog.Init()

	case dialog.CloseMultiArgumentsDialogMsg:
		a.showMultiArgumentsDialog = false
		if !msg.Submit {
			return a, nil
		}
		// A submitted dialog is a command whose arguments are now known, so the
		// prompt is filled in and sent the way any goal is.
		command, ok := a.findCommand(msg.CommandID)
		if !ok || command.Prompt == "" {
			return a, util.ReportWarn("That command is no longer available")
		}
		return a, util.CmdHandler(chat.SendMsg{Text: commands.Expand(command.Prompt, msg.Args)})

	case dialog.ProviderSelectedMsg:
		a.showProviderDialog = false
		if a.app != nil {
			a.app.SetProvider(msg.Provider)
		}
		// After switching, automatically show the model picker so the user can
		// choose which model to run with under the new provider.
		a.showModelDialog = true
		return a, tea.Batch(
			util.ReportInfo(fmt.Sprintf("Provider switched to: %s. Loading its models…", msg.Provider)),
			a.modelDialog.Init(),
			a.loadModels(),
		)

	case dialog.CloseProviderDialogMsg:
		a.showProviderDialog = false
		return a, nil

	case openProviderDialogMsg:
		if a.app != nil {
			a.providerDialog.SetCurrentProvider(a.app.CurrentProvider())
		}
		a.showProviderDialog = true
		return a, nil

	case chat.SendMsg:
		trimmed := strings.TrimSpace(msg.Text)
		if strings.HasPrefix(trimmed, "/") {
			return a.handleSlashCommand(trimmed)
		}

	case tea.KeyPressMsg:
		// If model dialog is open, let it handle the key press first
		if a.showModelDialog {
			d, modelCmd := a.modelDialog.Update(msg)
			a.modelDialog = d.(dialog.ModelDialog)
			return a, modelCmd
		}

		// If provider dialog is open, let it handle the key press first
		if a.showProviderDialog {
			d, provCmd := a.providerDialog.Update(msg)
			a.providerDialog = d.(dialog.ProviderDialog)
			return a, provCmd
		}

		// If multi-arguments dialog is open, let it handle the key press first
		if a.showMultiArgumentsDialog {
			args, cmd := a.multiArgumentsDialog.Update(msg)
			a.multiArgumentsDialog = args.(dialog.MultiArgumentsDialogCmp)
			return a, cmd
		}

		switch {

		case key.Matches(msg, keys.Cancel):
			// Cancelling stops the run the daemon owns. It never exits, and with
			// nothing in flight it opens the quit prompt rather than doing
			// nothing: a chord that silently does nothing reads as a broken key.
			if a.app.CoderAgent.IsBusy() {
				a.app.CoderAgent.Cancel(a.selectedSession.ID)
				return a, util.ReportInfo("Cancelling the run...")
			}
			return a, a.promptQuit()
		case key.Matches(msg, keys.Quit):
			return a, a.promptQuit()
		case key.Matches(msg, keys.SwitchSession):
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showCommandDialog {
				// Load sessions and show the dialog
				sessions, err := a.app.Sessions.List(context.Background())
				if err != nil {
					return a, util.ReportFailure("Listing the runs", "send a goal to start a new one", err)
				}
				if len(sessions) == 0 {
					return a, util.ReportWarn("No sessions available")
				}
				a.sessionDialog.SetSessions(sessions)
				a.showSessionDialog = true
				return a, nil
			}
			return a, nil
		case key.Matches(msg, keys.Commands):
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showThemeDialog && !a.showFilepicker {
				// Show commands dialog
				if len(a.commands) == 0 {
					return a, util.ReportWarn("No commands available")
				}
				a.commandDialog.SetCommands(a.commands)
				a.showCommandDialog = true
				return a, nil
			}
			return a, nil
		// The picker asks the harness what the provider serves: the catalogue is
		// the provider's, and a client that kept its own list would be asserting
		// what it cannot verify.
		case key.Matches(msg, keys.Models):
			if a.showModelDialog {
				a.showModelDialog = false
				return a, nil
			}
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showCommandDialog {
				a.showModelDialog = true
				return a, tea.Batch(a.modelDialog.Init(), a.loadModels())
			}
			return a, nil
		case key.Matches(msg, keys.ChangedFiles):
			if a.showFiles {
				a.showFiles = false
				return a, nil
			}
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showCommandDialog {
				a.recordChanges()
				a.showFiles = true
				return a, nil
			}
			return a, nil
		case key.Matches(msg, keys.SwitchTheme):
			if !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showCommandDialog {
				// Show theme switcher dialog
				a.showThemeDialog = true
				// Theme list is dynamically loaded by the dialog component
				return a, a.themeDialog.Init()
			}
			return a, nil
		case key.Matches(msg, keys.Sidebar):
			if a.currentPage == page.ChatPage && !a.dialogOpen() {
				return a, util.CmdHandler(chat.ToggleSidebarMsg{})
			}
			return a, nil
		case key.Matches(msg, returnKey) || key.Matches(msg):
			if msg.String() == quitKey {
				if a.currentPage == page.LogsPage {
					return a, a.moveToPage(page.ChatPage)
				}
			} else if !a.filepicker.IsCWDFocused() {
				if a.showQuit {
					a.showQuit = !a.showQuit
					return a, nil
				}
				if a.showHelp {
					a.showHelp = !a.showHelp
					return a, nil
				}
				if a.showInitDialog {
					a.showInitDialog = false
					// Mark the project as initialized without running the command
					if err := config.MarkProjectInitialized(); err != nil {
						return a, util.ReportError(err)
					}
					return a, nil
				}
				if a.showFilepicker {
					a.showFilepicker = false
					a.filepicker.ToggleFilepicker(a.showFilepicker)
					return a, nil
				}
				if a.currentPage == page.LogsPage {
					return a, a.moveToPage(page.ChatPage)
				}
			}
		case key.Matches(msg, keys.Logs):
			return a, a.moveToPage(page.LogsPage)
		case key.Matches(msg, keys.Help):
			if a.showQuit {
				return a, nil
			}
			a.showHelp = !a.showHelp
			return a, nil
		case key.Matches(msg, helpEsc):
			if a.app.CoderAgent.IsBusy() {
				if a.showQuit {
					return a, nil
				}
				a.showHelp = !a.showHelp
				return a, nil
			}
		case key.Matches(msg, keys.Filepicker):
			a.showFilepicker = !a.showFilepicker
			a.filepicker.ToggleFilepicker(a.showFilepicker)
			return a, nil
		}
	default:
		f, filepickerCmd := a.filepicker.Update(msg)
		a.filepicker = f.(dialog.FilepickerCmp)
		cmds = append(cmds, filepickerCmd)

	}

	if a.showFilepicker {
		f, filepickerCmd := a.filepicker.Update(msg)
		a.filepicker = f.(dialog.FilepickerCmp)
		cmds = append(cmds, filepickerCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showQuit {
		q, quitCmd := a.quit.Update(msg)
		a.quit = q.(dialog.QuitDialog)
		cmds = append(cmds, quitCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}
	if a.showPermissions {
		d, permissionsCmd := a.permissions.Update(msg)
		a.permissions = d.(dialog.PermissionDialogCmp)
		cmds = append(cmds, permissionsCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showSessionDialog {
		d, sessionCmd := a.sessionDialog.Update(msg)
		a.sessionDialog = d.(dialog.SessionDialog)
		cmds = append(cmds, sessionCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showModelDialog {
		d, modelCmd := a.modelDialog.Update(msg)
		a.modelDialog = d.(dialog.ModelDialog)
		cmds = append(cmds, modelCmd)
		// Only block key messages; everything else flows down.
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showCommandDialog {
		d, commandCmd := a.commandDialog.Update(msg)
		a.commandDialog = d.(dialog.CommandDialog)
		cmds = append(cmds, commandCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showJobs {
		d, jobsCmd := a.jobs.Update(msg)
		a.jobs = d.(dialog.JobsDialog)
		cmds = append(cmds, jobsCmd)
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showFiles {
		d, filesCmd := a.files.Update(msg)
		a.files = d.(dialog.FilesDialog)
		cmds = append(cmds, filesCmd)
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showInitDialog {
		d, initCmd := a.initDialog.Update(msg)
		a.initDialog = d.(dialog.InitDialogCmp)
		cmds = append(cmds, initCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showThemeDialog {
		d, themeCmd := a.themeDialog.Update(msg)
		a.themeDialog = d.(dialog.ThemeDialog)
		cmds = append(cmds, themeCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showProviderDialog {
		d, provCmd := a.providerDialog.Update(msg)
		a.providerDialog = d.(dialog.ProviderDialog)
		cmds = append(cmds, provCmd)
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	s, _ := a.status.Update(msg)
	a.status = s.(core.StatusCmp)
	a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
}

// RegisterCommand adds a command to the command dialog
func (a *appModel) RegisterCommand(cmd dialog.Command) {
	a.commands = append(a.commands, cmd)
}

func (a *appModel) findCommand(id string) (dialog.Command, bool) {
	for _, cmd := range a.commands {
		if cmd.ID == id {
			return cmd, true
		}
	}
	return dialog.Command{}, false
}

func (a *appModel) moveToPage(pageID page.PageID) tea.Cmd {
	if a.app.CoderAgent.IsBusy() {
		// For now we don't move to any page if the agent is busy
		return util.ReportWarn("Agent is busy, please wait...")
	}

	var cmds []tea.Cmd
	if _, ok := a.loadedPages[pageID]; !ok {
		cmd := a.pages[pageID].Init()
		cmds = append(cmds, cmd)
		a.loadedPages[pageID] = true
	}
	a.previousPage = a.currentPage
	a.currentPage = pageID
	if sizable, ok := a.pages[a.currentPage].(layout.Sizeable); ok {
		cmd := sizable.SetSize(a.width, a.height)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// exportTimelineMsg asks for the current run's record to be written out.
type exportTimelineMsg struct{}

// openJobsMsg asks for the daemon's schedule to be shown.
type openJobsMsg struct{}

// registerUserCommands adds what the user wrote to the palette.
//
// A directory that cannot be read is reported once, in the statusline, rather
// than silently producing an empty palette: a command the author wrote and
// cannot find is a mistake they would look for in the wrong place.
func (a *appModel) registerUserCommands() {
	loaded, err := commands.Load()
	if err != nil && !errors.Is(err, commands.ErrNoDirectory) {
		a.commandsError = err
	}
	for _, command := range loaded {
		command := command
		a.RegisterCommand(dialog.Command{
			ID:          "user:" + command.ID,
			Title:       command.Title,
			Description: command.Description,
			Prompt:      command.Body,
			Handler: func(dialog.Command) tea.Cmd {
				return runUserCommand(command)
			},
		})
	}
}

// runUserCommand either asks for the arguments the prompt declares, or sends it
// as it was written.
func runUserCommand(command commands.Command) tea.Cmd {
	if len(command.Args) == 0 {
		return util.CmdHandler(chat.SendMsg{Text: command.Body})
	}
	return util.CmdHandler(dialog.ShowMultiArgumentsDialogMsg{
		CommandID: "user:" + command.ID,
		Content:   command.Body,
		ArgNames:  command.Args,
	})
}

// jobsLoadedMsg carries the daemon's schedule, or the failure to read it.
type jobsLoadedMsg struct {
	jobs []runtime.Job
	err  error
}

// openJobs asks the daemon what it repeats and shows the answer.
//
// The list is read when the panel opens rather than kept in step: the schedule
// belongs to the daemon, and a client that mirrored it would be describing a
// queue it cannot keep current.
func (a *appModel) openJobs() tea.Cmd {
	runner := a.app.Runner
	a.showJobs = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		jobs, err := runner.Scheduled(ctx)
		return jobsLoadedMsg{jobs: jobs, err: err}
	}
}

// unschedule stops a recurring run and re-reads the list, so what is drawn is
// what the daemon now has.
func (a *appModel) unschedule(jobID string) tea.Cmd {
	runner := a.app.Runner
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := runner.Unschedule(ctx, jobID); err != nil {
			return util.ReportFailure("Stopping the scheduled run", "check the daemon's schedule again", err)()
		}
		jobs, err := runner.Scheduled(ctx)
		if err != nil {
			return util.ReportFailure("Reading the daemon's schedule", "the stop took effect", err)()
		}
		return jobsLoadedMsg{jobs: jobs}
	}
}

// recordChanges hands the panel what the harness reported for this session.
//
// The panel is filled when it opens rather than kept in step: the changes are
// the run's own record, so reading them on demand is what keeps the panel from
// becoming a second list that can disagree with the statusline.
func (a *appModel) recordChanges() {
	if a.app.Runner == nil || a.selectedSession.ID == "" {
		a.files.SetFiles(nil)
		return
	}
	changes := a.app.Runner.Changes(a.selectedSession.ID)
	files := make([]dialog.ChangedFile, 0, len(changes))
	for _, change := range changes {
		files = append(files, dialog.ChangedFile{Path: change.Path, Operation: change.Operation})
	}
	a.files.SetFiles(files)
}

// loadDiff asks the harness for what a run changed in one file (ADR 014).
func (a *appModel) loadDiff(path string) tea.Cmd {
	sessionID := a.selectedSession.ID
	return func() tea.Msg {
		if a.app.Runner == nil || sessionID == "" {
			return dialog.DiffLoadedMsg{Path: path, Err: errors.New("no session active")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := a.app.Runner.Diff(ctx, sessionID, path)
		if err != nil {
			return dialog.DiffLoadedMsg{Path: path, Err: err}
		}
		return dialog.DiffLoadedMsg{
			Path:    resp.Path,
			Kind:    resp.Kind,
			Content: resp.Content,
		}
	}
}

// gateNotice is how a permission gate reads without a dialog open: what is
// waiting, and every key that answers it.
func gateNotice(tool string) string {
	return fmt.Sprintf("Permission required: %s — a to allow, s for the session, d to deny", tool)
}

// exportTimeline writes what the harness recorded for a run, where a reader or a
// script can pick it up without the terminal.
func (a *appModel) exportTimeline(sessionID string) tea.Cmd {
	runner := a.app.Runner
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		path, err := runner.ExportTimeline(ctx, sessionID)
		if err != nil {
			return util.ReportFailure("Exporting the run's record", "try again once the daemon answers", err)()
		}
		return util.ReportInfo("Exported to " + path)()
	}
}

// denialNotice is what the client says when the user answers a gate with deny.
//
// It is a function rather than a literal because the golden frame of the denied
// state has to show the same words the client produces, and two copies of a
// sentence drift.
func denialNotice(tool string) string {
	return fmt.Sprintf("Denied %s. The run stops.", tool)
}

// promptQuit opens the confirmation, dismissing whatever layer is on top.
//
// Quitting is never silent: the dialog is the only route out of the client, so
// a stray chord cannot end a session that is mid-run.
func (a *appModel) promptQuit() tea.Cmd {
	a.showQuit = !a.showQuit
	if a.showHelp {
		a.showHelp = false
	}
	if a.showSessionDialog {
		a.showSessionDialog = false
	}
	if a.showCommandDialog {
		a.showCommandDialog = false
	}
	if a.showFilepicker {
		a.showFilepicker = false
		a.filepicker.ToggleFilepicker(a.showFilepicker)
	}
	if a.showModelDialog {
		a.showModelDialog = false
	}
	if a.showMultiArgumentsDialog {
		a.showMultiArgumentsDialog = false
	}
	return nil
}

// reconcile folds the daemon's runs into the session index.
//
// It runs at start-up because the client keeps no history of its own: a client
// that started after the runs did has to learn about them from the harness, or
// the session picker offers an empty list as if nothing had ever run.
func (a *appModel) reconcile() tea.Cmd {
	runner := a.app.Runner
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := runner.ListRuns(ctx); err != nil {
			cmd := util.ReportWarn("Reading the harness's runs: " + err.Error() + " — new goals still work")
			return cmd()
		}
		return nil
	}
}

// attach re-reads a run's timeline and keeps following it.
//
// Selecting a session is a reconnect, not a reset: the transcript and what the
// run spent are rebuilt from the run's own log, and a run still in flight keeps
// streaming. A failure is reported rather than swallowed, because a view that
// looks current but is not is worse than an error.
func (a *appModel) attach(sessionID string) tea.Cmd {
	runner := a.app.Runner
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := runner.Attach(ctx, sessionID); err != nil {
			cmd := util.ReportFailure("Re-attaching to the run", "pick another session, or send the goal again", err)
			return cmd()
		}
		return nil
	}
}

// loadModels asks the harness what the provider can serve.
//
// It is a command rather than a call because the answer crosses a socket: a
// view that blocked on it would freeze the client while the daemon thinks.
func (a *appModel) loadModels() tea.Cmd {
	runner := a.app.Runner
	provider := config.Get().Provider
	return tea.Sequence(
		func() tea.Msg {
			return dialog.LoadingModelsMsg{}
		},
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// The list and what each model can do come from the same operation: a
			// picker that offered names without features would ask a user to know
			// which models see by heart.
			models, info, err := runner.ModelCatalogue(ctx, provider)
			return dialog.ModelsLoadedMsg{Models: models, Info: info, Err: err}
		},
	)
}

// fitToTerminal trims every line to the terminal and marks each cut.
//
// Nothing drawn above this point can be trusted to fit: a width in lipgloss is
// a minimum, so content wider than the terminal keeps its width and the frame
// grows past the screen. No surface scrolls horizontally, which makes fitting
// the frame the shell's job rather than a rule each component has to remember.
func fitToTerminal(view string, width int) string {
	if width <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) <= width {
			continue
		}
		lines[i] = ansi.Truncate(line, width, styles.TruncationMarker)
	}
	return strings.Join(lines, "\n")
}

// View renders the component for the terminal.
func (a appModel) View() tea.View {
	v := tea.NewView(a.viewString())
	v.AltScreen = true
	return v
}
func (a appModel) viewString() string {
	components := []string{
		a.pages[a.currentPage].View().Content,
	}

	components = append(components, a.status.View().Content)

	appView := lipgloss.JoinVertical(lipgloss.Top, components...)

	if a.showPermissions {
		overlay := a.permissions.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showFilepicker {
		overlay := a.filepicker.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)

	}

	if a.showHelp {
		bindings := layout.KeyMapToSlice(keys)
		if p, ok := a.pages[a.currentPage].(layout.Bindings); ok {
			bindings = append(bindings, p.BindingKeys()...)
		}
		if a.showPermissions {
			bindings = append(bindings, a.permissions.BindingKeys()...)
		}
		if a.currentPage == page.LogsPage {
			bindings = append(bindings, logsKeyReturnKey)
		}
		if !a.app.CoderAgent.IsBusy() {
			bindings = append(bindings, helpEsc)
		}
		a.help.SetBindings(bindings)

		overlay := a.help.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showOnboard {
		overlay := a.onboard.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showQuit {
		overlay := a.quit.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showSessionDialog {
		overlay := a.sessionDialog.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showModelDialog {
		overlay := a.modelDialog.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showCommandDialog {
		overlay := a.commandDialog.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showJobs {
		overlay := a.jobs.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(col, row, overlay, appView, true)
	}

	if a.showFiles {
		overlay := a.files.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(col, row, overlay, appView, true)
	}

	if a.showInitDialog {
		overlay := a.initDialog.View().Content
		appView = layout.PlaceOverlay(
			a.width/2-lipgloss.Width(overlay)/2,
			a.height/2-lipgloss.Height(overlay)/2,
			overlay,
			appView,
			true,
		)
	}

	if a.showThemeDialog {
		overlay := a.themeDialog.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showProviderDialog {
		overlay := a.providerDialog.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	if a.showMultiArgumentsDialog {
		overlay := a.multiArgumentsDialog.View().Content
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	// The frame is fitted last so overlays are subject to it too: a dialog that
	// runs past the terminal is exactly the surface a user cannot dismiss.
	return fitToTerminal(appView, a.width)
}

// answerOnboard carries out the first-run choice.
//
// The answer is recorded either way: "not now" is an answer, and a client that
// asks again on every start is a client that nags instead of helping.
func (a appModel) answerOnboard(msg dialog.OnboardChoiceMsg) (tea.Model, tea.Cmd) {
	a.showOnboard = false
	if err := config.UpdateOnboarded(); err != nil {
		logging.WarnPersist("could not record the first-run answer", "err", err)
	}
	switch msg.ID {
	case "install-opencode":
		return a, installOpenCode()
	case "api-key":
		return a, util.CmdHandler(chat.SendMsg{Text: config.APIKeyHelp()})
	}
	return a, nil
}

// installOpenCode runs the installer and reports what it said.
func installOpenCode() tea.Cmd {
	return func() tea.Msg {
		out, err := onboard.Install(context.Background())
		return dialog.OnboardInstallFinishedMsg{Err: err, Output: out}
	}
}

// finishOnboardInstall tells the person what happened, and only claims opencode
// was installed when it was.
func (a appModel) finishOnboardInstall(msg dialog.OnboardInstallFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		text := "opencode was not installed: " + msg.Err.Error()
		if tail := lastLines(msg.Output, 3); tail != "" {
			text += "\n\n" + tail
		}
		return a, util.CmdHandler(chat.SendMsg{Text: text})
	}
	if path, err := exec.LookPath("opencode"); err == nil {
		if err := config.UpdateProvider("opencode"); err != nil {
			logging.WarnPersist("could not record the provider", "err", err)
		}
		text := "opencode is installed (" + path + "). Prumo will run with it from now on: " +
			"its models need no api-key of ours, and the model picker lists what it serves."
		return a, util.CmdHandler(chat.SendMsg{Text: text})
	}
	return a, util.CmdHandler(chat.SendMsg{Text: "the installer finished but opencode is not on PATH yet; " +
		"it may need a new shell before this session can use it"})
}

func lastLines(text string, n int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) <= n {
		return strings.TrimSpace(text)
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func New(app *app.App) tea.Model {
	// The palette is applied where the interface is built, not where the flags
	// are parsed: it is the client's own setting, it survives a restart, and a
	// theme that only a flag could set would be one the user re-chooses forever.
	if name := config.Get().Theme; name != "" {
		if err := theme.SetTheme(name); err != nil {
			logging.WarnPersist("unknown theme, keeping the one in use", "theme", name, "err", err)
		}
	}

	startPage := page.ChatPage
	model := &appModel{
		currentPage:     startPage,
		composerFocused: true,
		loadedPages:     make(map[page.PageID]bool),
		status:          core.NewStatusCmp(app),
		modelDialog:     dialog.NewModelDialogCmp(),
		help:            dialog.NewHelpCmp(),
		quit:            dialog.NewQuitCmp(),
		sessionDialog:   dialog.NewSessionDialogCmp(),
		commandDialog:   dialog.NewCommandDialogCmp(),
		permissions:     dialog.NewPermissionDialogCmp(),
		initDialog:      dialog.NewInitDialogCmp(),
		themeDialog:     dialog.NewThemeDialogCmp(),
		files:           dialog.NewFilesDialogCmp(),
		jobs:            dialog.NewJobsDialogCmp(),
		providerDialog:  dialog.NewProviderDialogCmp(app.CurrentProvider()),
		app:             app,
		commands:        []dialog.Command{},
		pages: map[page.PageID]tea.Model{
			page.ChatPage: page.NewChatPage(app),
			page.LogsPage: page.NewLogsPage(),
		},
		filepicker: dialog.NewFilepickerCmp(app),
	}

	// The first-run offer: asked once, and only when there is nothing to run
	// with. opencode serves models for free and authenticates itself, which is a
	// path nobody discovers from an empty model list.
	decision := onboard.Detect(config.Get().Provider, config.Get().Onboarded)
	if decision.Ask {
		model.onboard = dialog.NewOnboardDialogCmp(onboard.Options(decision))
		model.showOnboard = true
	} else if decision.Notice != "" {
		// Someone who already has opencode is not asked anything, which would
		// leave them with the one thing they need to know and no way to learn
		// it: that a real provider is one flag away. Said once, then recorded.
		model.startupNotice = decision.Notice
		if err := config.UpdateOnboarded(); err != nil {
			logging.WarnPersist("could not record the startup notice", "err", err)
		}
	}

	model.RegisterCommand(dialog.Command{
		ID:          "init",
		Title:       "Initialize Project",
		Description: "Create or update AGENTS.md, the agent memory file",
		Handler: func(cmd dialog.Command) tea.Cmd {
			prompt := `Please analyze this codebase and create an AGENTS.md file containing:
1. Build/lint/test commands - especially for running a single test
2. Code style guidelines including imports, formatting, types, naming conventions, error handling, etc.

The file you create will be given to agentic coding agents (such as yourself) that operate in this repository. Make it about 20 lines long.
If AGENTS.md already exists, improve it.
If there are Cursor rules (in .cursor/rules/ or .cursorrules) or Copilot rules (in .github/copilot-instructions.md), make sure to include them.`
			return tea.Batch(
				util.CmdHandler(chat.SendMsg{
					Text: prompt,
				}),
			)
		},
	})

	model.RegisterCommand(dialog.Command{
		ID:          "schedule",
		Title:       "Scheduled runs",
		Description: "See what the daemon repeats on a schedule, and stop one",
		Handler: func(cmd dialog.Command) tea.Cmd {
			return func() tea.Msg { return openJobsMsg{} }
		},
	})

	model.RegisterCommand(dialog.Command{
		ID:          "export",
		Title:       "Export the session's timeline",
		Description: "Write the run's own record to .prumo/runtime/exports/ as plain text",
		Handler: func(cmd dialog.Command) tea.Cmd {
			// The handler cannot see the selected session, so it asks the shell
			// for the export and lets the shell resolve which run it means.
			return util.CmdHandler(exportTimelineMsg{})
		},
	})

	// Compaction is the harness's: it summarizes a run as the run's context
	// budget requires, and the protocol exposes no operation that asks it for
	// one. The command states that boundary rather than simulating a job the
	// client cannot start.
	model.RegisterCommand(dialog.Command{
		ID:          "compact",
		Title:       "Compact Session",
		Description: "Who compacts this session's history",
		Handler: func(cmd dialog.Command) tea.Cmd {
			return util.ReportInfo("The harness compacts a run as it approaches its context budget; the client asks for nothing.")
		},
	})

	model.RegisterCommand(dialog.Command{
		ID:          "provider",
		Title:       "Configure Provider",
		Description: "Switch or view active LLM provider (opencode, anthropic, openai-compat, fake)",
		Handler: func(cmd dialog.Command) tea.Cmd {
			return func() tea.Msg { return openProviderDialogMsg{} }
		},
	})

	// The command surface is the user's own directory of markdown prompts. It is
	// theirs rather than the project's on purpose: a command is a prompt its
	// author owns, and the client reads what the person running it wrote.
	model.registerUserCommands()

	return model
}

func (a appModel) handleSlashCommand(raw string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return a, nil
	}
	name := strings.ToLower(parts[0])
	args := parts[1:]

	switch name {
	case "/help", "/h", "/?":
		a.showHelp = !a.showHelp
		return a, nil

	case "/model", "/models", "/m":
		if len(args) > 0 {
			targetModel := args[0]
			model, err := a.app.CoderAgent.Update(models.ModelID(targetModel))
			if err != nil {
				return a, util.ReportFailure("Choosing the model", "pick another one with /models or ctrl+o", err)
			}
			_ = config.UpdateModel(targetModel)
			return a, util.ReportInfo(fmt.Sprintf("Model switched to: %s", model.Name))
		}
		a.showModelDialog = true
		return a, tea.Batch(a.modelDialog.Init(), a.loadModels())

	case "/provider", "/providers", "/p":
		if len(args) > 0 {
			targetProvider := strings.ToLower(args[0])
			if err := config.UpdateProvider(targetProvider); err != nil {
				return a, util.ReportFailure("Updating provider", "", err)
			}
			if a.app != nil {
				a.app.SetProvider(targetProvider)
			}
			return a, util.ReportInfo(fmt.Sprintf("Provider switched to: %s", targetProvider))
		}
		if a.app != nil {
			a.providerDialog.SetCurrentProvider(a.app.CurrentProvider())
		}
		a.showProviderDialog = true
		return a, nil

	case "/session", "/sessions", "/s":
		sessions, err := a.app.Sessions.List(context.Background())
		if err != nil {
			return a, util.ReportFailure("Listing the runs", "send a goal to start a new one", err)
		}
		if len(sessions) == 0 {
			return a, util.ReportWarn("No sessions available")
		}
		a.sessionDialog.SetSessions(sessions)
		a.showSessionDialog = true
		return a, nil

	case "/theme", "/themes", "/t":
		a.showThemeDialog = true
		return a, a.themeDialog.Init()

	case "/file", "/files", "/f":
		a.showFilepicker = true
		return a, nil

	case "/diff", "/changes", "/c":
		a.recordChanges()
		a.showFiles = true
		return a, nil

	case "/job", "/jobs", "/schedule":
		return a, func() tea.Msg { return openJobsMsg{} }

	case "/sidebar", "/b":
		return a, util.CmdHandler(chat.ToggleSidebarMsg{})

	case "/init":
		for _, cmd := range a.commands {
			if cmd.ID == "init" {
				return a, cmd.Handler(cmd)
			}
		}
		return a, util.ReportWarn("Init command not available")

	case "/export":
		return a, util.CmdHandler(exportTimelineMsg{})

	case "/compact":
		return a, util.ReportInfo("The harness compacts a run as it approaches its context budget; the client asks for nothing.")

	case "/new", "/clear":
		a.selectedSession = session.Session{}
		return a, tea.Batch(
			util.CmdHandler(chat.SessionClearedMsg{}),
			util.ReportInfo("Started new session"),
		)

	case "/log", "/logs", "/l":
		return a, a.moveToPage(page.LogsPage)

	case "/quit", "/exit", "/q":
		a.showQuit = true
		return a, nil

	default:
		trimmedName := strings.TrimPrefix(name, "/")
		for _, cmd := range a.commands {
			if cmd.ID == trimmedName || cmd.ID == "user:"+trimmedName {
				if cmd.Handler != nil {
					return a, cmd.Handler(cmd)
				}
				if cmd.Prompt != "" {
					return a, util.CmdHandler(chat.SendMsg{Text: cmd.Prompt})
				}
			}
		}
		return a, util.ReportWarn(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", name))
	}
}
