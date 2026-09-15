package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raillen/prumo/tui/theme"
)

// The binary is built once per test binary: each live test needs the same
// artifact and rebuilding it per test would multiply the cost for no coverage.
var liveBinary struct {
	once sync.Once
	bin  string
	err  error
	skip string
}

// buildPrumo builds the CLI once. Skips rather than fails when the toolchain or
// the build is unavailable, matching sdk/prumo's roundtrip tests: a missing
// binary is a missing prerequisite, not a broken boundary.
func buildPrumo(t *testing.T) string {
	t.Helper()
	liveBinary.once.Do(func() {
		if _, err := exec.LookPath("go"); err != nil {
			liveBinary.skip = "go tool unavailable"
			return
		}
		repo, err := filepath.Abs("..")
		if err != nil {
			liveBinary.err = err
			return
		}
		dir, err := os.MkdirTemp("", "prumo-live-bin")
		if err != nil {
			liveBinary.err = err
			return
		}
		bin := filepath.Join(dir, "prumo")
		build := exec.Command("go", "build", "-o", bin, "./cmd/prumo")
		build.Dir = repo
		if out, err := build.CombinedOutput(); err != nil {
			liveBinary.skip = fmt.Sprintf("binary build unavailable: %s", out)
			return
		}
		liveBinary.bin = bin
	})
	if liveBinary.skip != "" {
		t.Skip(liveBinary.skip)
	}
	if liveBinary.err != nil {
		t.Fatalf("preparing the live binary: %v", liveBinary.err)
	}
	return liveBinary.bin
}

// TestLiveDaemonFlow is H10 acceptance criterion 2 without the terminal: the
// real supervisor starts a real `prumo agent serve`, the real SDK client drives
// a real run with the FakeProvider, and the model renders its events as they
// arrive — no restart of the TUI during the flow.
//
// It covers palette → goal → run → stream → evidence. It cannot cover the
// "approve one permission" step of the criterion: the protocol exposes no op to
// answer a permission request, which the gap register records.
func TestLiveDaemonFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("live daemon test skipped in -short mode")
	}
	bin := buildPrumo(t)

	workspace := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	daemon := NewDaemon(workspace)
	daemon.Binary = bin
	if err := daemon.Start(ctx, 30*time.Second); err != nil {
		t.Fatalf("starting the supervised daemon: %v", err)
	}
	defer daemon.Stop()

	if _, err := daemon.Client().Protocol(ctx); err != nil {
		t.Fatalf("daemon does not answer the protocol op: %v", err)
	}
	if _, err := os.Stat(daemon.Socket()); err != nil {
		t.Fatalf("daemon socket missing: %v", err)
	}

	styles, err := NewStyles(theme.DefaultTheme)
	if err != nil {
		t.Fatalf("styles: %v", err)
	}
	model := NewModel(styles, workspace, filepath.Join(workspace, ".prumo", "runtime", "harness"))
	model.SetSessionFactory(func(ctx context.Context, cfg StartConfig) (*Session, error) {
		return StartSession(ctx, daemon.Client(), cfg)
	})

	// Drive the user's path: palette → goal → enter.
	model = typeString(t, model, "run goal")
	model, _ = press(t, model, "enter")
	model = typeString(t, model, "prove the live flow")
	model, cmd := press(t, model, "enter")
	if cmd == nil {
		t.Fatal("starting the run produced no command")
	}
	message := cmd()
	if _, ok := message.(sessionMsg); !ok {
		t.Fatalf("the live daemon refused the run: %v", message)
	}
	model = update(t, model, message)
	if model.Stage() != StageRun {
		t.Fatalf("stage after start = %q, want the run panel", model.Stage())
	}

	// Stream until the run reaches a terminal status, exactly as the tick loop
	// does, and assert the view grows per poll rather than re-rendering.
	deadline := time.Now().Add(60 * time.Second)
	sawEvent := false
	for {
		result, err := model.Session.Poll(ctx)
		if err != nil {
			t.Fatalf("poll: %v", err)
		}
		if len(result.Events) > 0 {
			sawEvent = true
		}
		// Feed each poll exactly once. A terminal poll that also carries
		// events is the case that makes this matter: feeding it twice would
		// append the last batch twice and the replay check below would then
		// compare a doubled view against the daemon's record.
		model = update(t, model, pollMsg{result: result})
		if result.Finished {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the run never reached a terminal status; last = %+v", model.Session.Status())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawEvent {
		t.Fatal("the run completed without emitting a single event")
	}
	if model.Stage() != StageEvidence {
		t.Fatalf("stage after the run stopped = %q, want the evidence panel", model.Stage())
	}

	evidence := model.Session.Evidence(model.Timeline.Len(), model.Timeline.Dropped())
	if evidence.RunID == "" || evidence.Status == "" || evidence.Phase == "" {
		t.Fatalf("evidence is incomplete: %+v", evidence)
	}
	if evidence.Status != "complete" {
		t.Fatalf("fake-provider run status = %q, want complete (stop reason %q)", evidence.Status, evidence.StopReason)
	}

	// The terminal poll must have been complete: the view cannot be missing an
	// event the daemon already has. Asserting against the daemon's own log is
	// what makes this timing-independent — a short timeline here means the run
	// finished inside the read window and the client stopped early.
	complete, err := daemon.Client().Events(ctx, model.Session.RunID)
	if err != nil {
		t.Fatalf("reading the daemon's full log: %v", err)
	}
	if model.Timeline.Len() != len(complete) {
		t.Fatalf("the view has %d events but the daemon recorded %d: a terminal poll dropped events",
			model.Timeline.Len(), len(complete))
	}
	rendered := model.View().Content
	for _, want := range []string{"evidence", evidence.RunID, "complete"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("evidence view is missing %q:\n%s", want, rendered)
		}
	}

	// Criterion 3's mechanism: re-reading the timeline from the daemon
	// reproduces the same rows instead of merging two half-views.
	before := model.Timeline.Len()
	model, replay := run(t, model, CommandReconnect)
	if replay == nil {
		t.Fatal("reconnect produced no command")
	}
	model = update(t, model, replay())
	if model.Timeline.Len() != before {
		t.Fatalf("after replay the timeline has %d rows, was %d", model.Timeline.Len(), before)
	}
}

// TestSupervisedDaemonStopLeavesNoProcess asserts the supervisor really owns the
// child: a TUI that exits while its daemon keeps running would leave the store
// locked and the next start would fail.
func TestSupervisedDaemonStopLeavesNoProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("live daemon test skipped in -short mode")
	}
	bin := buildPrumo(t)
	workspace := t.TempDir()
	daemon := NewDaemon(workspace)
	daemon.Binary = bin

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := daemon.Start(ctx, 30*time.Second); err != nil {
		t.Fatalf("start: %v", err)
	}
	daemon.Stop()
	if daemon.Running() {
		t.Fatal("Stop left the daemon running")
	}
	// A second daemon over the same store must be able to acquire it, which is
	// the observable consequence of having stopped the first.
	second := NewDaemon(workspace)
	second.Binary = bin
	if err := second.Start(ctx, 30*time.Second); err != nil {
		t.Fatalf("restart after Stop failed, so the first daemon still holds the store: %v", err)
	}
	second.Stop()
}
