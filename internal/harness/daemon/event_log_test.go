package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// The event log rotated on every append, and RotateLog reads and rewrites the
// whole file. Past the cap that meant every event paid for the entire log, so
// producing N events cost O(N²) of disk work — from the synchronous stream loop
// of a run, which is the worst place for it (GAP-153).

func TestAppendingIsCheapPastTheCap(t *testing.T) {
	// The property is about behaviour over a run, not about one call: writing
	// three times the cap must still be a normal amount of work, and the file
	// must stay bounded.
	dir := t.TempDir()
	srv := New("s.sock", filepath.Join(dir, "store"), Deps{Workspace: dir})
	total := 20000
	begin := time.Now()
	for i := 0; i < total; i++ {
		srv.appendEvent("R-perf", agent.AgentEvent{
			ID: "ev-" + strconv.Itoa(i), RunID: "R-perf", Kind: "text_delta",
			Payload: map[string]any{"text": "x"}, CreatedAt: agent.Now(),
		})
	}
	elapsed := time.Since(begin)

	path, err := srv.eventPath("R-perf")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Count(string(data), "\n")
	if lines > eventLogMax {
		t.Fatalf("the log grew to %d lines, past its cap of %d", lines, eventLogMax)
	}
	if lines == 0 {
		t.Fatal("the log is empty; rotation lost everything")
	}
	// Reading the whole log on every event was measured at 12.6s for 20k
	// appends, against 0.6s for counting, with the same output. The bound sits
	// between them with room for a slower machine, and a log read per event
	// cannot pass it.
	if elapsed > 5*time.Second {
		t.Fatalf("writing %d events took %s; the per-event log read is back", total, elapsed)
	}
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
