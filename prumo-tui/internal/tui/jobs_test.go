package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
)

// The schedule belongs to the daemon, so the panel reads it when it opens rather
// than keeping a list of its own that could go stale.
func TestTheScheduleIsReadWhenThePanelOpens(t *testing.T) {
	harness := &stubHarness{jobs: []prumo.Job{
		{ID: "job-1", Goal: "check the flaky test", EverySecs: 3600, NextRun: 1800000000},
	}}
	model := New(app.New(app.Options{Client: harness}))

	updated, cmd := model.Update(openJobsMsg{})
	shell, ok := updated.(appModel)
	if !ok {
		t.Fatal("the shell changed shape")
	}
	if !shell.showJobs {
		t.Fatal("the panel did not open")
	}

	// The list arrives as a message, which is what the panel draws from.
	loaded := drive(t, updated, cmd, 3)
	frame := plain(loaded.View().Content)
	for _, want := range []string{"Runs on a schedule", "check the flaky test", "1h"} {
		if !strings.Contains(frame, want) {
			t.Errorf("the panel does not show %q:\n%s", want, frame)
		}
	}
}

// Stopping a recurring run is the one change the panel makes, and it re-reads
// the list so what is drawn is what the daemon now has.
func TestStoppingAScheduledRunAsksTheDaemonAndReReads(t *testing.T) {
	harness := &stubHarness{jobs: []prumo.Job{{ID: "job-7", Goal: "nightly", EverySecs: 86400}}}
	model := New(app.New(app.Options{Client: harness}))

	opened, openCmd := model.Update(openJobsMsg{})
	loaded := drive(t, opened, openCmd, 3)

	updated, cmd := loaded.Update(dialog.UnscheduleJobMsg{JobID: "job-7"})
	drive(t, updated, cmd, 3)

	if len(harness.unscheduled) != 1 || harness.unscheduled[0] != "job-7" {
		t.Fatalf("the daemon was told to stop %v, want the selected job", harness.unscheduled)
	}
}

// scheduleFrom reads the answer out of whatever the update returned. Opening the
// panel also moves the keyboard, so the answer travels in a batch rather than on
// its own.
func scheduleFrom(t *testing.T, cmd tea.Cmd) jobsLoadedMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("opening the panel produced nothing")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, inner := range batch {
			if inner == nil {
				continue
			}
			if loaded, ok := inner().(jobsLoadedMsg); ok {
				return loaded
			}
		}
	}
	loaded, ok := msg.(jobsLoadedMsg)
	if !ok {
		t.Fatalf("no schedule came back: %T", msg)
	}
	return loaded
}

// A schedule that cannot be read is reported in the shape every failure takes:
// what failed, and what still works.
func TestAScheduleThatCannotBeReadIsReported(t *testing.T) {
	model := New(app.New(app.Options{Client: nil}))

	_, cmd := model.Update(openJobsMsg{})
	loaded := scheduleFrom(t, cmd)
	if loaded.err == nil {
		t.Fatal("a client with no harness was asked for a schedule and answered")
	}

	_, noticeCmd := model.Update(loaded)
	notice := noticeIn(t, noticeCmd)
	if notice.Msg == "" {
		t.Fatal("reading a schedule that failed said nothing")
	}
	if !strings.Contains(notice.Msg, "schedule") {
		t.Fatalf("the failure does not name the operation: %q", notice.Msg)
	}
}

// The palette is how a user reaches the panel: a surface nobody can find is a
// surface that does not exist.
func TestTheSchedulePanelIsReachableFromThePalette(t *testing.T) {
	model, ok := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)
	if !ok {
		t.Fatal("New did not return the shell")
	}
	for _, command := range model.commands {
		if command.ID == "schedule" {
			if command.Handler == nil || command.Description == "" {
				t.Fatalf("the entry explains nothing: %+v", command)
			}
			// Selecting it asks the shell to open the panel.
			if msg := command.Handler(command)(); msg == nil {
				t.Fatal("selecting the entry does nothing")
			}
			return
		}
	}
	t.Fatal("the palette offers no way to see the daemon's schedule")
}
