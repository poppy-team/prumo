package daemon

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func eventsOf(t *testing.T, c Client, runID string) []any {
	t.Helper()
	res, err := c.Events(runID)
	if err != nil {
		t.Fatalf("events failed: %v", err)
	}
	if res["ok"] != true {
		t.Fatalf("events not ok: %v", res)
	}
	list, _ := res["events"].([]any)
	if len(list) == 0 {
		t.Fatalf("expected a replayable timeline for %s", runID)
	}
	return list
}

func eventIDs(t *testing.T, events []any) []string {
	t.Helper()
	ids := make([]string, 0, len(events))
	seen := map[string]bool{}
	for _, e := range events {
		m, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("event is not an object: %v", e)
		}
		id, _ := m["id"].(string)
		if id == "" {
			t.Fatalf("event without id: %v", m)
		}
		if seen[id] {
			t.Fatalf("duplicate event id %q in replayed timeline", id)
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func sameOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestClientReplayConformance is the W12.1 client-facing contract: a fresh
// client must replay the same JSONL timeline, in the same order, with no
// duplication and no corruption. The TUI/Desktop reconnect path relies on it.
func TestClientReplayConformance(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "agentd.sock")
	_, c1, cancel := serveForTest(t, dir, false)
	defer cancel()

	if _, err := c1.Start("replay me", "fake", "R-replay", 1); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, c1, "R-replay", "complete")

	firstIDs := eventIDs(t, eventsOf(t, c1, "R-replay"))

	// Simulate a client reconnect: a brand-new client over the same socket.
	c2 := Client{SocketPath: sock}
	secondIDs := eventIDs(t, eventsOf(t, c2, "R-replay"))
	if !sameOrder(firstIDs, secondIDs) {
		t.Fatalf("replay must be stable across clients:\n first:  %v\n second: %v", firstIDs, secondIDs)
	}

	// A third read must not mutate the timeline (no duplicate appends).
	thirdIDs := eventIDs(t, eventsOf(t, c2, "R-replay"))
	if !sameOrder(firstIDs, thirdIDs) {
		t.Fatalf("repeated replay must be idempotent:\n first: %v\n third: %v", firstIDs, thirdIDs)
	}
}

// TestReconnectSurvivesServerRestart is W12.2: the timeline is durable JSONL,
// so restarting the daemon over the same store preserves client-visible state.
func TestReconnectSurvivesServerRestart(t *testing.T) {
	dir := t.TempDir()
	_, c1, cancel := serveForTest(t, dir, false)
	if _, err := c1.Start("survive restart", "fake", "R-restart", 1); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, c1, "R-restart", "complete")
	before := eventIDs(t, eventsOf(t, c1, "R-restart"))
	cancel()
	time.Sleep(100 * time.Millisecond)

	_, c2, cancel2 := serveForTest(t, dir, false)
	defer cancel2()

	after := eventIDs(t, eventsOf(t, c2, "R-restart"))
	if !sameOrder(before, after) {
		t.Fatalf("restart must not change the client-visible timeline:\n before: %v\n after:  %v", before, after)
	}
}

// TestReconnectStateIsSerialisable is W12.3: the replayed payload must be
// JSON-encodable so a client can persist and restore its own view of the run.
func TestReconnectStateIsSerialisable(t *testing.T) {
	dir := t.TempDir()
	_, c, cancel := serveForTest(t, dir, false)
	defer cancel()
	if _, err := c.Start("serialise me", "fake", "R-serial", 1); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, c, "R-serial", "complete")

	state, err := c.Status("R-serial")
	if err != nil {
		t.Fatal(err)
	}
	events := eventsOf(t, c, "R-serial")

	blob, err := json.Marshal(map[string]any{"status": state, "events": events})
	if err != nil {
		t.Fatalf("reconnect state must be serialisable: %v", err)
	}
	var restored map[string]any
	if err := json.Unmarshal(blob, &restored); err != nil {
		t.Fatalf("reconnect state must round-trip: %v", err)
	}
	if restored["events"] == nil || restored["status"] == nil {
		t.Fatalf("restored state lost fields: %v", restored)
	}
}
