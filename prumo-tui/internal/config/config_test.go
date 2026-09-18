package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// defaultConfig is what a fresh process starts from.
func defaultConfig() Config {
	return Config{WorkingDir: ".", Theme: "prumo", Provider: "fake", MaxTurns: 5}
}

// A theme that only lives in memory is a theme the client forgets the moment it
// is useful: the promise is that a restart keeps it, so a restart is what the
// test performs.
func TestTheChosenThemeSurvivesARestart(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	Set(defaultConfig())

	if err := UpdateTheme("gruvbox"); err != nil {
		t.Fatalf("choosing a theme failed: %v", err)
	}

	// A restart is a process that starts from the defaults and reads the file.
	Set(defaultConfig())
	if err := Load(); err != nil {
		t.Fatalf("reading the client's own settings failed: %v", err)
	}
	if got := Get().Theme; got != "gruvbox" {
		t.Fatalf("theme after a restart = %q, want the one that was chosen", got)
	}
}

// A client that has never been configured is the ordinary case, not a broken
// one: reading nothing is not a failure.
func TestAFreshClientHasNothingToReadAndSaysSoQuietly(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	if err := Load(); err != nil {
		t.Fatalf("reading settings that were never written failed: %v", err)
	}
}

// Settings the client cannot parse are reported rather than ignored: a file it
// silently skipped would be a preference the user set and never had applied.
func TestUnreadableSettingsAreReported(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PRUMO_TUI_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("cannot write the fixture: %v", err)
	}
	if err := Load(); err == nil {
		t.Fatal("a settings file that cannot be parsed was accepted silently")
	}
}

// The file the client writes is its own: it holds a preference, not a run.
func TestOnlyTheClientsOwnPreferenceIsWritten(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PRUMO_TUI_CONFIG_DIR", dir)

	Set(Config{WorkingDir: "/somewhere", Theme: "prumo", Provider: "anthropic", Model: "claude", SocketPath: "/tmp/x.sock"})
	if err := UpdateTheme("dracula"); err != nil {
		t.Fatalf("choosing a theme failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("nothing was written: %v", err)
	}
	for _, leaked := range []string{"anthropic", "claude", "somewhere", "x.sock"} {
		if strings.Contains(string(data), leaked) {
			t.Errorf("the client wrote %q down, which belongs to the invocation or the harness: %s", leaked, data)
		}
	}
}

// The first-run offer is the one provider choice that is the client's own, and
// a promise to use it next time is only kept if it is written down.
func TestTheChosenProviderIsRemembered(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PRUMO_TUI_CONFIG_DIR", dir)
	Set(Config{WorkingDir: ".", Provider: "fake"})

	if err := UpdateProvider("opencode"); err != nil {
		t.Fatalf("recording the choice failed: %v", err)
	}

	Set(Config{WorkingDir: ".", Provider: "fake"})
	if err := Load(); err != nil {
		t.Fatalf("reloading failed: %v", err)
	}
	if got := Get().Provider; got != "opencode" {
		t.Fatalf("the choice was forgotten: %q", got)
	}
}

// Answering the offer is recorded, so the client does not ask again on every
// start — a client that re-asks is a client that nags.
func TestTheAnswerIsRecorded(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PRUMO_TUI_CONFIG_DIR", dir)
	Set(Config{WorkingDir: ".", Provider: "fake"})

	if err := UpdateOnboarded(); err != nil {
		t.Fatalf("recording the answer failed: %v", err)
	}
	Set(Config{WorkingDir: ".", Provider: "fake"})
	if err := Load(); err != nil {
		t.Fatalf("reloading failed: %v", err)
	}
	if !Get().Onboarded {
		t.Fatal("the client would ask again")
	}
}
