// Package config holds the client's own settings.
//
// The imported view layer expected a large, file-and-environment configuration
// service. This client needs very little: where the workspace is, which theme
// to draw with, and what to call the run. Everything else — providers, models,
// budgets, permissions — belongs to the harness and reaches the client through
// the protocol, never through a local file.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config is the client's resolved settings.
type Config struct {
	// WorkingDir is the workspace root the client was pointed at.
	WorkingDir string
	// Theme is the active theme id.
	Theme string
	// Debug and DebugLSP are accepted for compatibility with the imported
	// view layer; neither enables anything that talks to a provider.
	Debug    bool
	DebugLSP bool
	// AutoCompact is accepted for compatibility. The harness owns compaction.
	AutoCompact bool
	// ReducedMotion drops animation — the spinner's turn and the cursor's blink
	// — without dropping what the animation was carrying, which is stated in
	// words instead. It is the user's setting, from `--reduced-motion` or
	// PRUMO_REDUCED_MOTION.
	ReducedMotion bool
	// Provider and Model name what the client asks the harness to run with.
	Provider string
	Model    string
	// Onboarded records that the first-run offer was answered. It is written
	// after the answer, so an answer that never reached disk is asked again
	// rather than assumed.
	Onboarded bool
	// chosenProvider is true only when the client itself settled the provider
	// (the first-run offer). A provider that arrived in an invocation is that
	// invocation's decision and is not this file's to remember.
	chosenProvider bool
	// MaxTurns bounds a run; zero means the harness default.
	MaxTurns int
	// SocketPath / RemoteAddr select how the client reaches the daemon.
	SocketPath string
	RemoteAddr string
	Token      string
	TLSCert    string
}

// defaultProvider is what a run uses when nothing chose for it: deterministic
// and offline, so a first run cannot fail for reasons the person cannot see.
const defaultProvider = "fake"

var (
	mu      sync.RWMutex
	current = Config{
		WorkingDir:  ".",
		Theme:       "prumo",
		AutoCompact: true,
		Provider:    defaultProvider,
		MaxTurns:    5,
	}
)

// Get returns the active configuration.
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	cfg := current
	return &cfg
}

// Set replaces the active configuration.
func Set(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	current = cfg
}

// WorkingDirectory is the resolved workspace root.
func WorkingDirectory() string {
	mu.RLock()
	defer mu.RUnlock()
	abs, err := filepath.Abs(current.WorkingDir)
	if err != nil {
		return current.WorkingDir
	}
	return abs
}

// UpdateTheme records the active theme and writes it down, so a restart keeps
// it. The write is what makes that sentence true: a theme that only lives in
// memory is a theme the client forgets the moment it is useful.
func UpdateTheme(theme string) error {
	if theme == "" {
		return errors.New("a theme name is required")
	}
	mu.Lock()
	current.Theme = theme
	mu.Unlock()
	return save()
}

// persisted is the part of the configuration the client writes for itself.
//
// Only its own preferences live here. Where the workspace is, which provider to
// ask and how to reach the daemon are decisions of the invocation, and
// everything about a run belongs to the harness.
type persisted struct {
	Theme string `json:"theme"`
	// Provider is written only when the first-run offer chose one for the
	// person; --provider still wins, because a flag is said now and this file
	// was read earlier.
	Provider string `json:"provider,omitempty"`

	Onboarded bool `json:"onboarded,omitempty"`
}

// ConfigPath is where the client's own settings live.
//
// PRUMO_TUI_CONFIG_DIR exists so a test can point the client at a directory it
// owns: a test that wrote to the developer's home is a test nobody runs twice.
func ConfigPath() string {
	if dir := os.Getenv("PRUMO_TUI_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.json")
	}
	home, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "prumo-agent-tui", "config.json")
}

// Load reads what the client wrote for itself, if anything.
//
// A missing file is not an error: a client that has never been configured is
// the ordinary case, not a broken one.
func Load() error {
	path := ConfigPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var saved persisted
	if err := json.Unmarshal(data, &saved); err != nil {
		return fmt.Errorf("the client's own settings are not readable: %w", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if saved.Theme != "" {
		current.Theme = saved.Theme
	}
	if saved.Provider != "" && current.Provider == defaultProvider {
		current.Provider = saved.Provider
		current.chosenProvider = true
	}
	current.Onboarded = saved.Onboarded
	return nil
}

func save() error {
	path := ConfigPath()
	if path == "" {
		return errors.New("no configuration directory is available to write to")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mu.RLock()
	saved := persisted{Theme: current.Theme, Onboarded: current.Onboarded}
	if current.chosenProvider {
		saved.Provider = current.Provider
	}
	data, err := json.Marshal(saved)
	mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// UpdateOnboarded records that the first-run offer was answered.
func UpdateOnboarded() error {
	mu.Lock()
	current.Onboarded = true
	mu.Unlock()
	return save()
}

// UpdateProvider records the provider a choice settled on, so the next start
// runs with it.
func UpdateProvider(name string) error {
	if name == "" {
		return errors.New("a provider name is required")
	}
	mu.Lock()
	current.Provider = name
	current.chosenProvider = true
	mu.Unlock()
	return save()
}

// APIKeyHelp is what the client says when someone prefers a key to opencode.
//
// It is instructions rather than a prompt for a secret, and that is deliberate:
// this client keeps preferences, not credentials, so a key typed into it would
// be a secret written to a file nobody audited. The variables are the harness's
// own, and a shell that exports them is a shell the person can see.
func APIKeyHelp() string {
	return "To use an api-key provider instead of opencode, export these before starting the client:\n\n" +
		"  export PRUMO_MODEL_API_KEY='...'                      # the key\n" +
		"  export PRUMO_MODEL_BASE_URL='https://...'             # only for an openai-compatible endpoint\n" +
		"  prumo-agent tui --provider anthropic                  # or --provider openai-compat\n\n" +
		"The key is read from the environment, never written down by this client."
}

// MarkProjectInitialized is accepted for compatibility with the imported view
// layer: the harness decides what a workspace needs, not the client. It reports
// success because there is nothing here that can fail.
func MarkProjectInitialized() error { return nil }

// ShouldShowInitDialog reports whether the client should offer first-run help.
// It is false once the workspace has a harness store, which is the harness's
// own signal that it has been used here before.
func ShouldShowInitDialog() (bool, error) {
	_, err := os.Stat(filepath.Join(WorkingDirectory(), ".prumo", "runtime", "harness"))
	if os.IsNotExist(err) {
		return true, nil
	}
	return err != nil, nil
}
