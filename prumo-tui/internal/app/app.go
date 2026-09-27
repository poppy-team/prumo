// Package app wires the client's services together.
//
// The imported view layer expects an App with four services on it. In the
// upstream implementation those were database-backed and owned the agent loop.
// Here they are views over the harness: the client holds no state the daemon
// cannot recreate.
package app

import (
	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
)

// App is the surface the view layer reads.
type App struct {
	Sessions    session.Service
	Messages    message.Service
	Permissions *permission.Service
	CoderAgent  agent.Service
	Provider    string
	Workspace   string

	// Runner is the harness-backed runner, kept for callers that need the
	// transport itself (listing runs, for instance).
	Runner *runtime.Runner
}

// Options configures the app.
type Options struct {
	Client    runtime.Client
	Provider  string
	Model     models.Model
	MaxTurns  int
	Workspace string
}

// New builds an app over a harness transport.
func New(opts Options) *App {
	sessions := session.NewStore()
	messages := message.NewStore()

	runner := runtime.NewRunner(runtime.Options{
		Client:    opts.Client,
		Sessions:  sessions,
		Messages:  messages,
		Provider:  opts.Provider,
		Model:     opts.Model,
		MaxTurns:  opts.MaxTurns,
		Workspace: opts.Workspace,
	})
	permissions := permission.NewService(runner)
	runner.SetPermissions(permissions)

	return &App{
		Sessions:    sessions,
		Messages:    messages,
		Permissions: permissions,
		CoderAgent:  runner,
		Provider:    opts.Provider,
		Workspace:   opts.Workspace,
		Runner:      runner,
	}
}

// SetProvider changes the active provider across app and runner.
func (a *App) SetProvider(provider string) {
	a.Provider = provider
	if a.Runner != nil {
		a.Runner.SetProvider(provider)
	}
}

// CurrentProvider returns the currently active provider.
func (a *App) CurrentProvider() string {
	if a.Runner != nil {
		return a.Runner.Provider()
	}
	return a.Provider
}
