package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// The event log rotated on every append, and RotateLog reads and rewrites the
// whole file. Past the cap that meant every event paid for the entire log, so
// producing N events cost O(N²) of disk work — from the synchronous stream loop
// of a run, which is the worst place for it (GAP-153).

// The rotation used to run on every append, and RotateLog opens and reads the
// whole log to find out whether it needs trimming. It did that for every single
// event, including the overwhelming majority that needed nothing (GAP-153).
//
// This asserts the behaviour rather than a duration: a timing bound on a shared
// machine is a flaky test, and the property is about how often the file is
// rewritten, which is observable without a clock.

// The rotation used to run on every append, and RotateLog opens and reads the
// whole log to find out whether it needs trimming. It did that for every single
// event, including the overwhelming majority that needed nothing (GAP-153).
//
// This asserts the rate rather than a duration: a timing bound on a shared
// machine is a flaky test, and the property is how often the file is rewritten,
// which is observable without a clock.

func TestTheLogIsRewrittenRarelyRatherThanPerEvent(t *testing.T) {
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	path, _ := srv.eventPath("R-freq")

	const appends = eventLogMax * 4
	rewrites := 0
	previous := ""
	for i := 0; i < appends; i++ {
		srv.appendEvent("R-freq", agent.AgentEvent{
			ID: "ev-" + strconv.Itoa(i), RunID: "R-freq", Kind: "text_delta", CreatedAt: agent.Now(),
		})
		if i%64 != 63 {
			// Sampling: a rewrite between samples is still counted at the next
			// sample, which is all the rate needs.
			continue
		}
		current := firstLineID(t, path)
		if current != previous {
			rewrites++
			previous = current
		}
	}
	// Trimming happens on the order of once per cap, so the count tracks
	// appends/cap. Reading the log per event would rewrite or re-read it on every
	// append instead, which is three orders of magnitude more — so any bound
	// between the two rates catches the regression without being sensitive to
	// the exact schedule.
	maxRewrites := appends / 100
	if rewrites > maxRewrites {
		t.Fatalf("the log was rewritten %d times over %d appends; trimming should be on the order of once per cap (%d), not this often", rewrites, appends, maxRewrites)
	}
	if rewrites == 0 {
		t.Fatal("the log was never trimmed, so the cap is not being enforced")
	}
	if lines := countLogLines(t, path); lines > eventLogMax*2 {
		t.Fatalf("the log grew to %d lines, far past its cap of %d", lines, eventLogMax)
	}
}

func TestAppendsAfterATrimDoNotEachTriggerAnother(t *testing.T) {
	// The specific shape of the old bug: trim, append one event, and the very
	// next append rewrites again. After a real trim the file holds cap/2 lines
	// and the count matches it, so the next cap/2 appends must not rewrite.
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	path, _ := srv.eventPath("R-one")

	// Append until the log has actually been trimmed.
	before := ""
	trimmed := false
	for i := 0; i < eventLogMax*3 && !trimmed; i++ {
		srv.appendEvent("R-one", agent.AgentEvent{
			ID: "ev-" + strconv.Itoa(i), RunID: "R-one", Kind: "text_delta", CreatedAt: agent.Now(),
		})
		after := firstLineID(t, path)
		if i == 0 {
			before = after
		} else if after != before {
			trimmed = true
		}
	}
	if !trimmed {
		t.Fatal("the log was never trimmed, so this proves nothing about what follows")
	}
	settled := firstLineID(t, path)
	linesAtTrim := countLogLines(t, path)
	if linesAtTrim > eventLogMax {
		t.Fatalf("a trimmed log must be well under its cap, got %d lines", linesAtTrim)
	}
	for i := 0; i < 10; i++ {
		srv.appendEvent("R-one", agent.AgentEvent{
			ID: "late-" + strconv.Itoa(i), RunID: "R-one", Kind: "text_delta", CreatedAt: agent.Now(),
		})
	}
	if got := firstLineID(t, path); got != settled {
		t.Fatalf("ten appends inside the cap rewrote the log (first line moved from %q to %q)", settled, got)
	}
}

func TestTheCounterResetsToWhatTheFileNowHolds(t *testing.T) {
	// Resetting to the cap rather than to the half that survived is the exact
	// mistake that makes every append rewrite: the condition is true on the next
	// event, and on every one after it.
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	path, _ := srv.eventPath("R-reset")
	for i := 0; i <= eventLogMax; i++ {
		srv.appendEvent("R-reset", agent.AgentEvent{
			ID: "ev-" + strconv.Itoa(i), RunID: "R-reset", Kind: "text_delta", CreatedAt: agent.Now(),
		})
	}
	srv.eventsMu.Lock()
	tracked := srv.eventLines[path]
	srv.eventsMu.Unlock()
	lines := countLogLines(t, path)
	if tracked != lines {
		t.Fatalf("the counter says %d but the file holds %d lines; a drifting counter either never rotates or rotates constantly", tracked, lines)
	}
	if tracked >= eventLogMax {
		t.Fatalf("after a rotation the count should be the %d lines that were kept, not the cap of %d", eventLogMax/2, eventLogMax)
	}
}

func firstLineID(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(strings.TrimRight(string(data), "\n"), "\n", 2)[0]
	return first
}

func countLogLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(strings.TrimRight(string(data), "\n"), "\n") + 1
}

func TestRotationKeepsTheNewestEvents(t *testing.T) {
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	total := eventLogMax + 50
	for i := 0; i < total; i++ {
		srv.appendEvent("R-newest", agent.AgentEvent{
			ID: "ev-" + strconv.Itoa(i), RunID: "R-newest", Kind: "text_delta",
			Payload: map[string]any{"text": "x"}, CreatedAt: agent.Now(),
		})
	}
	path, _ := srv.eventPath("R-newest")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "ev-"+strconv.Itoa(total-1)) {
		t.Fatal("the newest event must survive rotation")
	}
	if strings.Contains(body, "ev-0\n") {
		t.Fatal("rotation must drop the oldest, not keep everything")
	}
}

func TestTheLineCountIsTrackedRatherThanMeasured(t *testing.T) {
	// Measuring means a read on every append; tracking means a counter. This
	// checks the counter agrees with the file rather than drifting from it,
	// because a wrong count either never rotates or rotates constantly.
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	path, _ := srv.eventPath("R-count")
	for i := 0; i < 10; i++ {
		srv.appendEvent("R-count", agent.AgentEvent{ID: "ev-" + strconv.Itoa(i), RunID: "R-count", Kind: "x"})
	}
	srv.eventsMu.Lock()
	tracked := srv.eventLines[path]
	srv.eventsMu.Unlock()
	if tracked != 10 {
		t.Fatalf("the counter says %d lines after 10 appends", tracked)
	}
	data, _ := os.ReadFile(path)
	if got := strings.Count(string(data), "\n"); got != 10 {
		t.Fatalf("the file has %d lines but the counter says %d", got, tracked)
	}
}

func TestConcurrentAppendsKeepTheCountHonest(t *testing.T) {
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	var wg sync.WaitGroup
	const writers = 16
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				srv.appendEvent("R-conc", agent.AgentEvent{
					ID:    "ev-" + strconv.Itoa(n) + "-" + strconv.Itoa(j),
					RunID: "R-conc", Kind: "text_delta", CreatedAt: agent.Now(),
				})
			}
		}(i)
	}
	wg.Wait()
	path, _ := srv.eventPath("R-conc")
	srv.eventsMu.Lock()
	tracked := srv.eventLines[path]
	srv.eventsMu.Unlock()
	if tracked != writers*20 {
		t.Fatalf("the counter says %d after %d concurrent appends", tracked, writers*20)
	}
}

// A subscriber that has already been sent an id must not get it twice. While ids
// collided, that check silently dropped live events as repeats of replayed ones
// and the client missed them — losing events, not just showing duplicates
// (GAP-118).

func TestASharedIDWithDifferentContentIsSentRatherThanDropped(t *testing.T) {
	// The old check was `if sentIDs[id] { continue }`: an id seen before meant
	// drop, whatever the event was. A new event under a colliding id disappeared.
	sent := map[string]string{}
	replayed := map[string]any{"id": "ev-2", "kind": "text_delta", "payload": map[string]any{"text": "old"}}
	live := map[string]any{"id": "ev-2", "kind": "text_delta", "payload": map[string]any{"text": "new"}}

	if alreadySent(sent, "ev-2", replayed) {
		t.Fatal("the first event has not been sent yet")
	}
	if alreadySent(sent, "ev-2", live) {
		t.Fatal("a new event sharing an id must be sent, not dropped as a repeat")
	}
}

func TestTheGenuineDuplicateIsStillCaught(t *testing.T) {
	// The replay/live race the dedup exists for is real, so a true repeat has to
	// be recognised.
	sent := map[string]string{}
	ev := map[string]any{"id": "ev-7", "kind": "text_delta", "payload": map[string]any{"text": "same"}}
	if alreadySent(sent, "ev-7", ev) {
		t.Fatal("the first delivery is not a repeat")
	}
	if !alreadySent(sent, "ev-7", ev) {
		t.Fatal("an identical redelivery must be recognised as a repeat")
	}
}

func TestAnEventWithoutAnIDIsNeverDeduped(t *testing.T) {
	// With no id there is nothing to compare on, and dropping events with no
	// identity would silently lose them.
	sent := map[string]string{}
	ev := map[string]any{"kind": "text_delta", "payload": map[string]any{"text": "x"}}
	alreadySent(sent, "", ev)
	if _, tracked := sent[""]; tracked {
		t.Fatal("an idless event must not enter the dedup table")
	}
}

func TestEventIDsOnDiskAreUniqueForEveryShapeTheRuntimeEmits(t *testing.T) {
	// End to end: the shapes the runtime actually emits, through the real
	// append path, must all land with distinct ids.
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	shapes := []agent.AgentEvent{
		{Kind: "text_delta", Payload: map[string]any{"text": "a"}},
		{Kind: "tool_call_ready", Payload: map[string]any{"id": "c1", "name": "fs.read"}},
		{Kind: "text_delta", Payload: map[string]any{"text": "b"}},
		{Kind: "usage", Payload: map[string]any{"prompt_tokens": 1, "completion_tokens": 2, "cache_read_tokens": 3, "cache_write_tokens": 4, "cost_usd": 0.5}},
		{Kind: "tool_call_ready", Payload: map[string]any{"id": "c2", "name": "fs.read"}},
		{Kind: "text_delta", Payload: map[string]any{"text": "c"}},
	}
	for i, shape := range shapes {
		shape.ID = "ev-" + strconv.Itoa(i+1)
		shape.RunID = "R-shapes"
		shape.CreatedAt = agent.Now()
		srv.appendEvent("R-shapes", shape)
	}
	events := srv.readRawEvents("R-shapes")
	seen := map[string]bool{}
	for _, ev := range events {
		id, _ := ev["id"].(string)
		if seen[id] {
			t.Fatalf("id %q appears twice in the log", id)
		}
		seen[id] = true
	}
	if len(seen) != len(shapes) {
		t.Fatalf("expected %d distinct ids on disk, got %d", len(shapes), len(seen))
	}
}
