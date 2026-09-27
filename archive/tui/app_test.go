//go:build ignore

package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	prumo "github.com/raillen/prumo/sdk/prumo"
	"github.com/raillen/prumo/tui/theme"
)

var errTestRefused = errors.New("quota exhausted")

// newTestModel wires the model to a scripted endpoint. The whole point of the
// split is visible here: the flow is driven by messages, so it needs neither a
// terminal nor a daemon process.
func newTestModel(t testing.TB, ops Ops) *Model {
	t.Helper()
	styles, err := NewStyles(theme.DefaultTheme)
	if err != nil {
		t.Fatalf("styles: %v", err)
	}
	model := NewModel(styles, "/tmp/workspace", "/tmp/workspace/.prumo/runtime/harness")
	model.SetSessionFactory(func(ctx context.Context, cfg StartConfig) (*Session, error) {
		return StartSession(ctx, ops, cfg)
	})
	return model
}

func typeString(t *testing.T, model *Model, text string) *Model {
	t.Helper()
	for _, r := range text {
		next, _ := model.Update(tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
		model = next.(*Model)
	}
	return model
}

// update feeds one message and adapts the tea.Model result back to the concrete
// model the tests hold.
func update(t testing.TB, model *Model, msg tea.Msg) *Model {
	t.Helper()
	next, _ := model.Update(msg)
	concrete, ok := next.(*Model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	return concrete
}

// run invokes a palette command and adapts the tea.Model result back to the
// concrete model the tests hold.
func run(t *testing.T, model *Model, id string) (*Model, tea.Cmd) {
	t.Helper()
	next, cmd := model.runCommand(id)
	concrete, ok := next.(*Model)
	if !ok {
		t.Fatalf("runCommand(%q) returned %T", id, next)
	}
	return concrete, cmd
}

func press(t *testing.T, model *Model, key string) (*Model, tea.Cmd) {
	t.Helper()
	var press tea.KeyPressMsg
	switch key {
	case "enter":
		press = tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "esc":
		press = tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	case "backspace":
		press = tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace})
	case "down":
		press = tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	case "up":
		press = tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})
	default:
		press = tea.KeyPressMsg(tea.Key{Text: key, Code: rune(key[0])})
	}
	next, cmd := model.Update(press)
	return next.(*Model), cmd
}

// TestVerticalSliceFlow drives palette → goal → run → stream → approval →
// evidence, which is H10 acceptance criterion 2's shape without the terminal.
func TestVerticalSliceFlow(t *testing.T) {
	ops := &stubOps{}
	model := newTestModel(t, ops)

	if model.Stage() != StagePalette {
		t.Fatalf("initial stage = %q, want the palette", model.Stage())
	}

	// 1. palette: filter to the run action and pick it.
	model = typeString(t, model, "run goal")
	model, _ = press(t, model, "enter")
	if model.Stage() != StageGoal {
		t.Fatalf("after enter stage = %q, want the goal prompt", model.Stage())
	}

	// 2. goal: type it and start the run.
	model = typeString(t, model, "list the files")
	model, cmd := press(t, model, "enter")
	if cmd == nil {
		t.Fatal("starting a run produced no command")
	}
	message := cmd()
	sessionMsgValue, ok := message.(sessionMsg)
	if !ok {
		t.Fatalf("start produced %T, want sessionMsg (err=%v)", message, message)
	}
	model = update(t, model, sessionMsgValue)
	if model.Stage() != StageRun {
		t.Fatalf("after start stage = %q, want the run panel", model.Stage())
	}
	if ops.started[0].Goal != "list the files" {
		t.Fatalf("goal sent to the daemon = %q", ops.started[0].Goal)
	}

	// 3. stream: one poll adds the run's events.
	ops.events = []prumo.Event{
		ev("1", "run.started", map[string]any{"goal": "list the files"}),
		ev("2", "context.compiled", map[string]any{"included": float64(3), "tokens": float64(120)}),
		ev("3", "text_delta", map[string]any{"text": "thinking"}),
	}
	result, err := model.Session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	model = update(t, model, pollMsg{result: result})
	if model.Timeline.Len() != 3 {
		t.Fatalf("timeline has %d rows, want 3", model.Timeline.Len())
	}
	rendered := model.View().Content
	for _, want := range []string{"run R-test-1", "context compiled (120 tokens)", "thinking"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("run view is missing %q:\n%s", want, rendered)
		}
	}

	// 4. approval: the run stops on a permission request. The view must stay on
	// the run panel (it has a question to ask), say so, and answer the request
	// the daemon named.
	ops.events = append(ops.events,
		ev("4", "permission_wait", map[string]any{"tool": "edit.delete", "request_id": "perm-c1"}))
	ops.status = prumo.RunStatus{
		RunID: "R-test-1", Status: "awaiting_approval", Phase: "yield",
		PendingPermissions: []string{"perm-c1"},
	}
	result, err = model.Session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	model = update(t, model, pollMsg{result: result})
	if model.Stage() != StageRun {
		t.Fatalf("stage while waiting for approval = %q, want the run panel", model.Stage())
	}
	if rendered := model.View().Content; !strings.Contains(rendered, "a to approve") {
		t.Errorf("the run panel does not offer the decision:\n%s", rendered)
	}
	model, cmd = press(t, model, "a")
	if cmd == nil {
		t.Fatal("approving produced no command")
	}
	model = update(t, model, cmd())
	if len(ops.approved) != 1 || ops.approved[0] != "perm-c1" {
		t.Fatalf("approval did not reach the protocol: %v", ops.approved)
	}

	// 5. evidence: a terminal status moves the view there with no restart.
	ops.status = prumo.RunStatus{RunID: "R-test-1", Status: "complete", Phase: "complete", StopReason: "max turns reached"}
	result, err = model.Session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	model = update(t, model, pollMsg{result: result})
	if model.Stage() != StageEvidence {
		t.Fatalf("after the run stopped stage = %q, want the evidence panel", model.Stage())
	}
	evidence := model.View().Content
	if !strings.Contains(evidence, "complete") || !strings.Contains(evidence, "max turns reached") {
		t.Errorf("evidence view is missing the terminal accounting:\n%s", evidence)
	}
}

// TestDenyKeyAnswersThePermission is the other half of the surface: refusing
// must reach the protocol too, and the key must not invent a request.
func TestDenyKeyAnswersThePermission(t *testing.T) {
	ops := &stubOps{}
	model := newTestModel(t, ops)
	session, err := StartSession(context.Background(), ops, StartConfig{Goal: "deny me"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	model = update(t, model, sessionMsg{session: session})

	// Nothing pending: the key reports that instead of answering nothing.
	model, cmd := press(t, model, "d")
	if cmd != nil {
		t.Fatal("denying with nothing pending must not reach the protocol")
	}
	if len(ops.denied) != 0 || !strings.Contains(model.View().Content, "nothing is waiting for approval") {
		t.Fatalf("the panel did not say why nothing happened:\n%s", model.View().Content)
	}

	ops.status = prumo.RunStatus{
		RunID: "R-test-1", Status: "awaiting_approval", Phase: "yield",
		PendingPermissions: []string{"perm-c1"},
	}
	if _, err := session.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	model, cmd = press(t, model, "d")
	if cmd == nil {
		t.Fatal("denying a pending request produced no command")
	}
	model = update(t, model, cmd())
	if len(ops.denied) != 1 || !strings.HasPrefix(ops.denied[0], "perm-c1|") {
		t.Fatalf("denial did not reach the protocol: %v", ops.denied)
	}
}

func TestPaletteEscapeClearsTheFilter(t *testing.T) {
	model := newTestModel(t, &stubOps{})
	model = typeString(t, model, "zz")
	model, _ = press(t, model, "esc")
	if model.Palette.Query() != "" {
		t.Fatalf("escape left the filter at %q", model.Palette.Query())
	}
}

func TestEmptyGoalIsRefused(t *testing.T) {
	ops := &stubOps{}
	model := newTestModel(t, ops)
	model = typeString(t, model, "run")
	model, _ = press(t, model, "enter")
	model, _ = press(t, model, "enter") // empty goal
	if model.Stage() != StageGoal {
		t.Fatalf("an empty goal must keep the prompt open, stage = %q", model.Stage())
	}
	if len(ops.started) != 0 {
		t.Fatal("an empty goal reached the daemon")
	}
}

func TestStartFailureStaysInTheGoalPrompt(t *testing.T) {
	ops := &stubOps{startErr: errTestRefused}
	model := newTestModel(t, ops)
	model = typeString(t, model, "run")
	model, _ = press(t, model, "enter")
	model = typeString(t, model, "g")
	model, cmd := press(t, model, "enter")
	model = update(t, model, cmd())
	if !strings.Contains(model.View().Content, "quota exhausted") {
		t.Fatalf("the refusal is not visible in the view:\n%s", model.View().Content)
	}
}

func TestPollFailureIsAStateNotACrash(t *testing.T) {
	model := newTestModel(t, &stubOps{})
	model = update(t, model, pollErrMsg{err: errTestRefused})
	if !strings.Contains(model.View().Content, "reconnect") {
		t.Fatalf("a lost connection must be recoverable from the view:\n%s", model.View().Content)
	}
}

func TestReconnectRebuildsTheTimeline(t *testing.T) {
	ops := &stubOps{}
	model := newTestModel(t, ops)
	model = typeString(t, model, "run")
	model, _ = press(t, model, "enter")
	model = typeString(t, model, "g")
	_, cmd := press(t, model, "enter")
	model = update(t, model, cmd())

	ops.events = []prumo.Event{ev("1", "run.started", nil), ev("2", "text_delta", map[string]any{"text": "hi"})}
	result, _ := model.Session.Poll(context.Background())
	model = update(t, model, pollMsg{result: result})
	if model.Timeline.Len() != 2 {
		t.Fatalf("timeline = %d rows", model.Timeline.Len())
	}

	// Reconnecting must reproduce the same rows from the daemon's record, not
	// append a second copy — criterion 3's "no UI state corruption".
	model, cmd = run(t, model, CommandReconnect)
	if cmd == nil {
		t.Fatal("reconnect produced no command")
	}
	model = update(t, model, cmd())
	if model.Timeline.Len() != 2 {
		t.Fatalf("after replay the timeline has %d rows, want 2", model.Timeline.Len())
	}
	if entry := model.Timeline.Entries()[0]; entry.Index != 1 {
		t.Fatalf("replayed timeline did not restart numbering: %d", entry.Index)
	}
}

func TestThemeCyclingKeepsTheViewRenderable(t *testing.T) {
	model := newTestModel(t, &stubOps{})
	seen := map[string]bool{}
	for range len(theme.Names()) {
		var ignored tea.Cmd
		model, ignored = run(t, model, CommandNextTheme)
		_ = ignored
		seen[model.Styles.ThemeID()] = true
		if model.View().Content == "" {
			t.Fatalf("theme %s rendered nothing", model.Styles.ThemeID())
		}
	}
	if len(seen) != len(theme.Names()) {
		t.Fatalf("cycling reached %d of %d themes: %v", len(seen), len(theme.Names()), seen)
	}
}

func TestCancelRefusesWhenNothingIsInFlight(t *testing.T) {
	model := newTestModel(t, &stubOps{})
	model, cmd := run(t, model, CommandCancel)
	if cmd != nil {
		t.Fatal("cancelling with no run must not call the daemon")
	}
	if !strings.Contains(model.View().Content, "nothing in flight") {
		t.Fatalf("no feedback for a refused cancel:\n%s", model.View().Content)
	}
}

// TestTerminalPollIsFedOnce pins the invariant the live test tripped on: a poll
// that both carries events and reports a terminal status must add those events
// exactly once. Feeding it twice is the natural shape of a loop that handles
// the "new events" and "finished" cases in separate branches, and it silently
// doubles the last batch — which then disagrees with the daemon's own record on
// replay.
func TestTerminalPollIsFedOnce(t *testing.T) {
	ops := &stubOps{}
	model := newTestModel(t, ops)
	model = typeString(t, model, "run")
	model, _ = press(t, model, "enter")
	model = typeString(t, model, "g")
	_, cmd := press(t, model, "enter")
	model = update(t, model, cmd())

	ops.status = prumo.RunStatus{RunID: "R-test-1", Status: "complete", Phase: "complete"}
	ops.events = []prumo.Event{ev("1", "run.started", nil), ev("2", "run.finished", nil)}
	result, err := model.Session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("terminal poll returned %d events, want 2", len(result.Events))
	}
	model = update(t, model, pollMsg{result: result})
	if model.Timeline.Len() != 2 {
		t.Fatalf("timeline has %d rows, want 2 — the terminal batch was applied more than once",
			model.Timeline.Len())
	}
	if model.Stage() != StageEvidence {
		t.Fatalf("stage = %q, want the evidence panel", model.Stage())
	}
}

func TestRunViewReportsRowsAgedOutOfTheWindow(t *testing.T) {
	ops := &stubOps{}
	model := newTestModel(t, ops)
	model = typeString(t, model, "run")
	model, _ = press(t, model, "enter")
	model = typeString(t, model, "g")
	_, cmd := press(t, model, "enter")
	model = update(t, model, cmd())
	model.Timeline = NewTimeline(2)
	for _, id := range []string{"1", "2", "3", "4"} {
		model.Timeline.Append(ev(id, "text_delta", map[string]any{"text": id}))
	}
	if !strings.Contains(model.View().Content, "aged out") {
		t.Fatalf("the view hid that rows were dropped:\n%s", model.View().Content)
	}
}
