package tui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/styles"
	"github.com/raillen/prumo-tui/internal/tui/theme"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// updateGolden regenerates the captured frames instead of comparing them.
//
// The frames are evidence a reader reviews in a diff, so the same code writes
// and checks them: a frame nobody can regenerate is a fixture nobody can
// maintain.
var updateGolden = flag.Bool("update-golden", false, "rewrite the golden frames from the current rendering")

const goldenDir = "testdata/golden"

// goldenWidths are the three size classes the interaction contract declares,
// each captured at the widest terminal of its class so a frame shows the class
// at its most demanding.
var goldenWidths = []struct {
	class string
	width int
}{
	{"compact", 79},
	{"standard", 100},
	{"wide", 140},
}

// builder drives the shell the way the program's event loop does, so a state is
// reached by the same messages a user produces.
type builder struct {
	t     testing.TB
	model tea.Model
	app   *app.App
}

func newBuilder(t testing.TB, harness *stubHarness) *builder {
	t.Helper()
	application := app.New(app.Options{Client: harness})
	model, _ := New(application).Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return &builder{t: t, model: model, app: application}
}

// send applies a message and runs whatever it returns, bounded so a
// self-renewing command cannot loop the test.
func (b *builder) send(msg tea.Msg) *builder {
	updated, cmd := b.model.Update(msg)
	b.model = drive(b.t, updated, cmd, 4)
	return b
}

// post applies a message without running what it returns, which is how a state
// is caught mid-transition — while the transcript is still rendering, for one.
func (b *builder) post(msg tea.Msg) *builder {
	updated, _ := b.model.Update(msg)
	b.model = updated
	return b
}

// seed fills a session's transcript without going through a run.
func (b *builder) seed(sessionID string, rows int) *builder {
	b.t.Helper()
	for i := 0; i < rows; i++ {
		role := message.User
		if i%2 == 1 {
			role = message.Assistant
		}
		if _, err := b.app.Messages.Create(context.Background(), sessionID, message.CreateMessageParams{
			Role:  role,
			Parts: []message.ContentPart{message.TextContent{Text: fmt.Sprintf("row %d of the conversation", i)}},
		}); err != nil {
			b.t.Fatalf("cannot seed the transcript: %v", err)
		}
	}
	return b
}

// selectSession selects from the store, which is what the picker hands the view:
// selecting a hand-built value would drop the usage the run already reported.
func (b *builder) selectSession(sessionID string) *builder {
	b.t.Helper()
	selected, err := b.app.Sessions.Get(context.Background(), sessionID)
	if err != nil {
		selected = session.Session{ID: sessionID, Title: sessionID}
	}
	return b.send(chat.SessionSelectedMsg(selected))
}

// runToCompletion starts a run and waits for the client to fold it, so the
// frame is of a finished run rather than of one still arriving.
func (b *builder) runToCompletion(goal string) *builder {
	b.t.Helper()
	out, err := b.app.Runner.Run(context.Background(), "S1", goal)
	if err != nil {
		b.t.Fatalf("cannot start the run: %v", err)
	}
	for range out {
	}
	return b
}

// applyOnce runs a command and applies what it produces, without following the
// commands those messages return.
//
// A state whose update returns a timer — the statusline clears a message after a
// TTL — would otherwise make the test sleep through the frame it is capturing.
func applyOnce(t testing.TB, model tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	if cmd == nil {
		return model
	}
	msg := cmd()
	if msg == nil {
		return model
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, inner := range batch {
			model = applyOnce(t, model, inner)
		}
		return model
	}
	updated, _ := model.Update(msg)
	return updated
}

// approvalRequest is one gate a run stopped on.
func approvalRequest() permission.PermissionRequest {
	return permission.PermissionRequest{
		ID:          "req-7",
		SessionID:   "S1",
		ToolName:    "process.exec",
		Description: "run the repository's test suite",
		Action:      "process.exec",
		// The arguments are what the run stopped on, and what the dialog draws
		// its preview from: a gate frame without them would be evidence of a
		// dialog nobody can answer.
		Params: map[string]any{"command": "go test ./..."},
	}
}

// approvalOpened drives the client to the moment a permission gate is on screen.
func approvalOpened(t testing.TB) *builder {
	request := approvalRequest()
	b := newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1")
	// Raising the request is what puts the gate in the client's pending set,
	// which is what makes a later answer land rather than being refused as stale.
	b.app.Permissions.Raise(request)

	// The announcement the gate leaves on the statusline is applied rather than
	// waited for: running the timer that clears it would capture the frame after
	// the message had already gone.
	updated, cmd := b.model.Update(pubsub.Event[permission.PermissionRequest]{Type: pubsub.CreatedEvent, Payload: request})
	b.model = applyOnce(t, updated, cmd)
	return b
}

// connectionEvent is what the runtime publishes when the daemon stops answering.
func connectionEvent(state agent.ConnectionState, attempt, of int) pubsub.Event[agent.AgentEvent] {
	return pubsub.Event[agent.AgentEvent]{
		Type: pubsub.UpdatedEvent,
		Payload: agent.AgentEvent{
			Type:       agent.AgentEventTypeConnection,
			SessionID:  "S1",
			Connection: agent.Connection{State: state, Attempt: attempt, Of: of, Reason: "the daemon did not answer"},
		},
	}
}

// streamingShell is a run in flight over a transcript, which is where the client
// animates anything at all.
func streamingShell(t testing.TB) tea.Model {
	b := newBuilder(t, &stubHarness{status: "running"}).seed("S1", 4).selectSession("S1")
	if _, err := b.app.CoderAgent.Run(context.Background(), "S1", "carry on"); err != nil {
		t.Fatalf("cannot start a run: %v", err)
	}
	return b.model
}

// frameStreaming is the animated state, captured with the animation off.
func frameStreaming(t testing.TB) tea.Model {
	return withReducedMotion(t, streamingShell(t))
}

// withReducedMotion renders a state with the animation off.
//
// A moving glyph cannot be captured: the spinner's shape is a function of the
// wall clock, so a frame with the animation on would change on every run.
// Everything such a frame is evidence for — the layout, the verb, the escape
// hatch — is there, and the difference the setting makes is asserted by
// `TestReducedMotionStatesProgressWithoutAnimatingIt`.
func withReducedMotion(t testing.TB, model tea.Model) tea.Model {
	previous := config.Get().ReducedMotion
	t.Cleanup(func() {
		cfg := *config.Get()
		cfg.ReducedMotion = previous
		config.Set(cfg)
	})
	cfg := *config.Get()
	cfg.ReducedMotion = true
	config.Set(cfg)
	return model
}

// stateFrames maps a state in the canonical matrix to the frame it draws. A
// state here must produce a frame for every width class.
var stateFrames = map[string]func(t testing.TB) tea.Model{
	"default": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").model
	},
	"empty": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).model
	},
	"loading": func(t testing.TB) tea.Model {
		// The render command is deliberately not run: this is the frame a user
		// sees between selecting a session and the transcript arriving.
		return newBuilder(t, &stubHarness{}).seed("S1", 4).post(chat.SessionSelectedMsg(session.Session{ID: "S1", Title: "S1"})).model
	},
	"active":  frameStreaming,
	"partial": frameStreaming,
	"success": func(t testing.TB) tea.Model {
		harness := &stubHarness{status: "complete", events: []runtime.Event{
			{Kind: "text_delta", Payload: map[string]any{"text": "The client folds the run's own log and draws what it says."}},
			{Kind: "usage", Payload: map[string]any{
				"prompt_tokens": float64(18400), "completion_tokens": float64(760),
				"cache_read_tokens": float64(9200), "cache_write_tokens": float64(2100), "cost_usd": 0.0412,
			}},
			{Kind: "file.changed", Payload: map[string]any{"path": "prumo-tui/internal/runtime/runner.go", "operation": "modified", "tool": "edit.patch"}},
			{Kind: "file.changed", Payload: map[string]any{"path": "prumo-tui/internal/stream/stream.go", "operation": "added", "tool": "fs.write"}},
		}}
		b := newBuilder(t, harness).runToCompletion("how does the client fold a run?")
		return b.selectSession("S1").model
	},
	// The two message states are posted rather than sent: the message is what
	// the frame is of, and running the command it returns would only wait out
	// the timer that clears it again.
	"warning": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").
			post(util.InfoMsg{Type: util.InfoTypeWarn, Msg: "the harness reported no budget left for this goal"}).model
	},
	"error": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").
			post(util.InfoMsg{Type: util.InfoTypeError, Msg: "no harness daemon answered within 20s"}).model
	},
	"selected": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").
			send(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl}).model
	},
	"read-only": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").
			send(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}).model
	},
	"permission-requested": func(t testing.TB) tea.Model {
		return approvalOpened(t).model
	},
	"permission-denied": func(t testing.TB) tea.Model {
		b := approvalOpened(t)
		// Answering with the key is what carries the denial through to the
		// service. The sentence it leaves on the statusline is posted rather than
		// waited for, so the frame does not sleep through the timer that clears
		// it — and it is the same function the handler calls, so the two cannot
		// disagree about what the client says.
		updated, cmd := b.model.Update(tea.KeyPressMsg{Code: 'd'})
		b.model = applyOnce(t, updated, cmd)
		return b.post(util.InfoMsg{Type: util.InfoTypeWarn, Msg: denialNotice("process.exec")}).model
	},
	"reconnecting": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").
			send(connectionEvent(agent.ConnectionReconnecting, 3, 8)).model
	},
	"offline": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").
			send(connectionEvent(agent.ConnectionOffline, 8, 8)).model
	},
	"destructive-confirmation": func(t testing.TB) tea.Model {
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").
			send(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl}).model
	},
	"alternate-theme": func(t testing.TB) tea.Model {
		previous := theme.CurrentThemeName()
		t.Cleanup(func() { _ = theme.SetTheme(previous) })
		if err := theme.SetTheme("gruvbox"); err != nil {
			t.Skipf("the alternate palette is not registered: %v", err)
		}
		return newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").model
	},
	"high-contrast": func(t testing.TB) tea.Model {
		// The frame a terminal without colour and without a font for the
		// Unicode set draws: the ASCII glyphs, which is a different frame rather
		// than the same one with the colour taken out.
		styles.UseASCII()
		t.Cleanup(styles.UseUnicode)
		inner := newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1").model
		return frameColourless{inner: inner}
	},
}

// statesWithoutFrames are states the matrix applies to this surface that draw
// no frame of their own, each with the reason a reader would otherwise have to
// guess at.
var statesWithoutFrames = map[string]string{
	"focus":           "the client has one editable surface, so its frame is the `default` frame: the composer carries the `>` prompt and its border is the focus marker, and a dialog owns the keyboard until it closes. Focus is asserted rather than captured a second time by `TestTheComposerShowsWhetherItHoldsTheKeyboard`.",
	"narrow-viewport": "every state is captured at all three width classes, so the compact capture of each state is this state's evidence rather than a frame of its own.",
}

// frameColourless is the frame a terminal without colour support draws: the
// escape sequences are gone and what is left has to say the same thing.
type frameColourless struct{ inner tea.Model }

func (f frameColourless) Init() tea.Cmd { return nil }

func (f frameColourless) Update(tea.Msg) (tea.Model, tea.Cmd) { return f, nil }

// View renders the component for the terminal, without its escapes.
func (f frameColourless) View() tea.View { return tea.NewView(plain(f.inner.View().Content)) }

type matrixEntry struct {
	ID      string `json:"id"`
	Surface string `json:"surface"`
	States  []struct {
		State      string   `json:"state"`
		Applicable bool     `json:"applicable"`
		Evidence   []string `json:"evidence"`
	} `json:"states"`
}

func loadMatrix(t testing.TB) matrixEntry {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "ui-ux", "state-matrix.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the state vocabulary is what these frames are evidence for: %v", err)
	}
	var matrix matrixEntry
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatalf("the state matrix is not valid JSON: %v", err)
	}
	return matrix
}

// The frames are derived from the canonical matrix, so a state that becomes
// applicable without a frame fails here instead of going unnoticed. This is the
// half that keeps the evidence set complete rather than merely green.
func TestEveryApplicableStateIsCapturedOrExcused(t *testing.T) {
	matrix := loadMatrix(t)
	if len(matrix.States) == 0 {
		t.Fatal("the matrix declares no state")
	}
	declared := map[string]bool{}
	for _, state := range matrix.States {
		declared[state.State] = true
		if !state.Applicable {
			continue
		}
		_, drawn := stateFrames[state.State]
		_, excused := statesWithoutFrames[state.State]
		switch {
		case drawn && excused:
			t.Errorf("state %q is both captured and excused; it must be one or the other", state.State)
		case !drawn && !excused:
			t.Errorf("state %q is applicable and neither captured nor given a reason", state.State)
		}
	}
	for name := range stateFrames {
		if !declared[name] {
			t.Errorf("a frame is captured for %q, which the matrix does not declare", name)
		}
	}
	for name := range statesWithoutFrames {
		if !declared[name] {
			t.Errorf("state %q is excused but the matrix does not declare it", name)
		}
	}
}

// The visual evidence set: one frame per captured state per width class,
// compared exactly. A frame that changes without being reviewed fails here,
// which is what makes the diff the review surface.
func TestGoldenFramesMatchByteForByte(t *testing.T) {
	names := make([]string, 0, len(stateFrames))
	for name := range stateFrames {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			for _, class := range goldenWidths {
				model := stateFrames[name](t)
				sized, cmd := model.Update(tea.WindowSizeMsg{Width: class.width, Height: 30})
				model = drive(t, sized, cmd, 3)
				checkGolden(t, filepath.Join(goldenDir, name+"."+class.class+".txt"), model.View().Content)
			}
		})
	}
}

// checkGolden compares a frame byte for byte.
//
// The frame is captured without its escape sequences: colour is a dimension of
// its own, covered by the token set, and what these frames are evidence for is
// the layout, the content and the state. A frame a reviewer cannot read in a
// diff would be a fixture, not evidence.
func checkGolden(t *testing.T, path, frame string) {
	t.Helper()
	frame = plain(frame)
	body := frameHeader(frame) + frame
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("cannot create the golden directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("cannot write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing: run `go test ./internal/tui -update-golden` and review the diff", path)
	}
	if string(want) != body {
		t.Errorf("%s does not match the current rendering.\nRegenerate with `go test ./internal/tui -update-golden` and review the diff.\n%s",
			path, firstDifference(string(want), body))
	}
}

// frameHeader carries the digest of the frame it precedes, so the file states
// what it is evidence for and a reader can tell whether it still corresponds.
func frameHeader(frame string) string {
	sum := sha256.Sum256([]byte(frame))
	return fmt.Sprintf("# digest sha256:%s\n", hex.EncodeToString(sum[:])[:16])
}

func firstDifference(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return fmt.Sprintf("first difference at line %d:\n  captured: %q\n  current:  %q", i+1, w, g)
		}
	}
	return "the frames differ in length only"
}
