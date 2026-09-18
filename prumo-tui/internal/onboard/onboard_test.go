package onboard

import (
	"os"
	"path/filepath"
	"testing"
)

// The offer exists for one asymmetry: opencode serves models for free, so
// someone who has it needs no key — and someone who does not would never guess
// that from an empty model list.
func TestOpencodePresentMeansNothingToAsk(t *testing.T) {
	fakeBin(t, "opencode")

	d := Detect("fake", false)
	if d.Ask {
		t.Fatalf("asked anyway: %s", d.Reason)
	}
	if d.OpenCodePath == "" {
		t.Fatal("opencode was found but not reported")
	}
	if len(d.InstallCommand) != 0 {
		t.Fatalf("offered to install what is already there: %q", d.InstallCommand)
	}
}

// Having opencode and not being told is the one case where saying nothing
// leaves the person with no way to find out a real provider is one flag away.
func TestHavingOpencodeIsSaidOnce(t *testing.T) {
	fakeBin(t, "opencode")

	if d := Detect("fake", false); d.Notice == "" {
		t.Fatal("nothing was said about an installed opencode")
	}
	if d := Detect("fake", true); d.Notice != "" {
		t.Fatalf("said again: %q", d.Notice)
	}
	// Someone already running with a real provider needs to hear about another.
	if d := Detect("anthropic", false); d.Notice != "" {
		t.Fatalf("told someone who already chose: %q", d.Notice)
	}
}

// Asked once is asked: a client that re-asks every start is a client that
// nags, and a "not now" is an answer.
func TestAskedOnceIsAsked(t *testing.T) {
	fakeBin(t, "npm")

	d := Detect("fake", true)
	if d.Ask {
		t.Fatalf("asked twice: %s", d.Reason)
	}
}

// A provider chosen on purpose is an answer already.
func TestAConfiguredProviderIsNotSecondGuessed(t *testing.T) {
	fakeBin(t, "npm")

	if d := Detect("anthropic", false); d.Ask {
		t.Fatalf("second-guessed a configured provider: %s", d.Reason)
	}
	if d := Detect("opencode", false); d.Ask {
		t.Fatalf("asked to install what the provider already is: %s", d.Reason)
	}
}

func TestNothingInstalledAsksWithACommand(t *testing.T) {
	fakeBin(t, "npm")

	d := Detect("fake", false)
	if !d.Ask {
		t.Fatalf("did not ask: %s", d.Reason)
	}
	if d.InstallCommand == "" {
		t.Fatal("npm is present but no command was offered")
	}
	if got := d.InstallCommand; got != InstallCommand {
		t.Fatalf("offered %q", got)
	}
}

// Without a way to install it, the offer says so instead of running something
// that does not exist.
func TestNoInstallerMeansNoCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	d := Detect("fake", false)
	if !d.Ask {
		t.Fatalf("did not ask: %s", d.Reason)
	}
	if d.InstallCommand != "" {
		t.Fatalf("offered an installer this machine does not have: %q", d.InstallCommand)
	}
	options := Options(d)
	if options[0].Automatic {
		t.Fatal("an option that cannot run was presented as automatic")
	}
}

// Every option states its consequence: an offer whose choices do not say what
// they do is a choice made blind.
func TestEveryOptionStatesItsConsequence(t *testing.T) {
	fakeBin(t, "npm")

	for _, opt := range Options(Detect("fake", false)) {
		if opt.Label == "" || opt.Detail == "" {
			t.Fatalf("an option explained nothing: %+v", opt)
		}
	}
}

// fakeBin puts a script named `name` on PATH, plus the shells the lookup needs.
func fakeBin(t *testing.T, name string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the fake %s: %v", name, err)
	}
	t.Setenv("PATH", dir)
}
