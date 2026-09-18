package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

// A denial is the one stop the protocol cannot explain: the daemon reports the
// run as failed, and from its account a denial looks like any other failure. The
// client is the only place the difference exists, so it is the place that says
// it — otherwise a user who denied a command sees a run that "failed" for no
// reason they can find.
func TestDenyingReportsTheDecision(t *testing.T) {
	b := approvalOpened(t)

	_, cmd := b.model.Update(dialog.PermissionResponseMsg{
		Action:     dialog.PermissionDeny,
		Permission: approvalRequest(),
	})
	msg := noticeIn(t, cmd)
	if msg.Type != util.InfoTypeWarn {
		t.Fatalf("a denial is not good news: %v", msg.Type)
	}
	if !strings.Contains(msg.Msg, "process.exec") || !strings.Contains(msg.Msg, "stops") {
		t.Fatalf("the notice does not say what was denied and what happened: %q", msg.Msg)
	}
}

// noticeIn finds the statusline notice among the commands an answer returns.
//
// The shell batches what the answer produced with the focus it gives back, so
// the notice is one level down rather than at the top of the tree.
func noticeIn(t *testing.T, cmd tea.Cmd) util.InfoMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("the answer produced nothing for the user to read")
	}
	msg := cmd()
	batch, isBatch := msg.(tea.BatchMsg)
	if !isBatch {
		notice, ok := msg.(util.InfoMsg)
		if !ok {
			t.Fatalf("the answer produced %T", msg)
		}
		return notice
	}
	for _, inner := range batch {
		if inner == nil {
			continue
		}
		if notice, ok := inner().(util.InfoMsg); ok {
			return notice
		}
	}
	t.Fatal("the answer produced no notice")
	return util.InfoMsg{}
}

// The gate the dialog answers has to be the one the service is holding, or the
// answer lands on a request that is no longer open and the run waits forever.
func TestDenyingAnswersTheOpenGate(t *testing.T) {
	b := approvalOpened(t)
	if pending := b.app.Permissions.Pending(); len(pending) != 1 {
		t.Fatalf("the gate was not open: %v", pending)
	}

	updated, cmd := b.model.Update(tea.KeyPressMsg{Code: 'd'})
	applyOnce(t, updated, cmd)

	if pending := b.app.Permissions.Pending(); len(pending) != 0 {
		t.Fatalf("the gate is still waiting after it was answered: %v", pending)
	}
}
