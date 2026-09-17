// Package config holds the client's own settings.
//
// The imported view layer expected a large, file-and-environment configuration
// service. This client needs very little: where the workspace is, which theme
// to draw with, and what to call the run. Everything else — providers, models,
// budgets, permissions — belongs to the harness and reaches the client through
// the protocol, never through a local file.
package config

import (
	"errors"
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
	// Provider and Model name what the client asks the harness to run with.
	Provider string
	Model    string
	// MaxTurns bounds a run; zero means the harness default.
	MaxTurns int
	// SocketPath / RemoteAddr select how the client reaches the daemon.
	SocketPath string
	RemoteAddr string
	Token      string
	TLSCert    string
}

var (
	mu      sync.RWMutex
	current = Config{
		WorkingDir:  ".",
		Theme:       "prumo",
		AutoCompact: true,
		Provider:    "fake",
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

// UpdateTheme records the active theme so a restart keeps it. It returns an
// error for the same shape the view layer expects; this client has no config
// file to fail on, so the only failure is an empty theme name.
func UpdateTheme(theme string) error {
	if theme == "" {
		return errors.New("a theme name is required")
	}
	mu.Lock()
	defer mu.Unlock()
	current.Theme = theme
	return nil
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
