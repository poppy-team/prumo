package tui

import (
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// A gate that exists only as a dialog cannot be read from a frame, and the
// contract asks for it to be legible without interacting with it: the request,
// and every key that answers it, are said in words as well.
func TestAGateIsAnnouncedInWords(t *testing.T) {
	b := newBuilder(t, &stubHarness{}).seed("S1", 4).selectSession("S1")
	request := approvalRequest()
	b.app.Permissions.Raise(request)

	updated, cmd := b.model.Update(pubsub.Event[permission.PermissionRequest]{
		Type: pubsub.CreatedEvent, Payload: request,
	})

	notice := noticeIn(t, cmd)
	if notice.Type != util.InfoTypeWarn {
		t.Fatalf("a gate waiting for an answer was reported as %v", notice.Type)
	}
	for _, want := range []string{"process.exec", "a to allow", "d to deny"} {
		if !strings.Contains(notice.Msg, want) {
			t.Errorf("the announcement lost %q: %q", want, notice.Msg)
		}
	}

	// And it reaches the frame, which is what "without interacting" means.
	shown := applyOnce(t, updated, cmd)
	if frame := plain(shown.View().Content); !strings.Contains(frame, "process.exec") {
		t.Fatalf("the frame does not say what is waiting:\n%s", frame)
	}
}

// The palette's export entry asks the shell which run it means, so the handler
// itself carries no state that could go stale.
func TestTheExportCommandAsksTheShellForTheSession(t *testing.T) {
	model, ok := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)
	if !ok {
		t.Fatal("New did not return the shell")
	}

	found := false
	for _, command := range model.commands {
		if command.ID == "export" {
			found = true
		}
	}
	if !found {
		t.Fatal("the palette offers no way to export what a run recorded")
	}

	// With nothing selected there is nothing to write, and the client says so
	// rather than writing an empty file.
	_, cmd := model.Update(exportTimelineMsg{})
	notice := noticeIn(t, cmd)
	if !strings.Contains(notice.Msg, "Nothing to export") {
		t.Fatalf("exporting without a session said %q", notice.Msg)
	}
}
