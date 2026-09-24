package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// `opList` read every run record and returned every run; `opEvents` read the
// whole log file and kept the last 500. Both were unbounded responses to a
// client that had asked for a window, and a daemon that had run for a month
// spent the memory to show a client the last twenty (GAP-162).

func writeRunRecords(t *testing.T, s *Server, count int) []string {
	t.Helper()
	ids := make([]string, 0, count)
	for i := range count {
		id := fmt.Sprintf("R-%04d", i)
		record := RunRecord{RunID: id, Status: "complete", UpdatedAt: "2026-01-01T00:00:00Z"}
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(s.StoreDir, "daemon-run-"+id+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestTheRunListingIsBounded(t *testing.T) {
	s := newScheduleServer(t)
	writeRunRecords(t, s, 500)
	res := s.opList("", 25)
	rows, _ := res["runs"].([]any)
	if len(rows) != 25 {
		t.Fatalf("a limit of 25 returned %d runs", len(rows))
	}
	if res["has_more"] != true {
		t.Fatal("500 runs were reported as the whole list")
	}
	cursor, _ := res["next_after_id"].(string)
	if cursor == "" {
		t.Fatal("no cursor was offered, so the client cannot ask for the rest")
	}
}

func TestPagingTheRunListingVisitsEveryRunOnce(t *testing.T) {
	s := newScheduleServer(t)
	ids := writeRunRecords(t, s, 47)
	seen := map[string]bool{}
	cursor := ""
	for page := range 20 {
		res := s.opList(cursor, 10)
		rows, _ := res["runs"].([]any)
		for _, row := range rows {
			m, _ := row.(map[string]any)
			id, _ := m["run_id"].(string)
			if seen[id] {
				t.Fatalf("run %s appeared on two pages", id)
			}
			seen[id] = true
		}
		if res["has_more"] != true {
			break
		}
		cursor, _ = res["next_after_id"].(string)
		if page == 19 {
			t.Fatal("paging did not terminate")
		}
	}
	if len(seen) != len(ids) {
		t.Fatalf("paging visited %d of %d runs", len(seen), len(ids))
	}
}

func TestAnUnknownCursorDoesNotReplayTheList(t *testing.T) {
	s := newScheduleServer(t)
	writeRunRecords(t, s, 20)
	// A cursor the daemon has never issued must not quietly become "start from
	// the beginning", which would hand the client rows it already saw and call
	// them new.
	res := s.opList("R-does-not-exist", 10)
	rows, _ := res["runs"].([]any)
	if len(rows) != 0 {
		t.Fatalf("an unknown cursor returned %d rows as if they were new", len(rows))
	}
}

func writeEventLog(t *testing.T, s *Server, runID string, count int) {
	t.Helper()
	path, err := s.eventPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	body := ""
	for i := range count {
		event, _ := json.Marshal(map[string]any{
			"id": fmt.Sprintf("ev-%05d", i), "run_id": runID, "kind": "text_delta",
		})
		body += string(event) + "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTheEventLogIsPagedNotTruncated(t *testing.T) {
	s := newScheduleServer(t)
	writeEventLog(t, s, "R-big", 2000)
	// The old behaviour returned the LAST 500. A client that wanted the
	// beginning of a run could not have it, and had no way to know that what it
	// received was the tail.
	res := s.opEvents("R-big", "", 10)
	evs, _ := res["events"].([]any)
	if len(evs) != 10 {
		t.Fatalf("a limit of 10 returned %d events", len(evs))
	}
	first, _ := evs[0].(map[string]any)
	if first["id"] != "ev-00000" {
		t.Fatalf("the first page started at %v, not at the beginning of the log", first["id"])
	}
	if res["has_more"] != true {
		t.Fatal("2000 events were reported as the whole log")
	}
}

func TestPagingTheEventLogVisitsEveryEventOnce(t *testing.T) {
	s := newScheduleServer(t)
	writeEventLog(t, s, "R-big", 137)
	seen := map[string]bool{}
	cursor := ""
	for range 30 {
		res := s.opEvents("R-big", cursor, 25)
		evs, _ := res["events"].([]any)
		for _, raw := range evs {
			m, _ := raw.(map[string]any)
			id, _ := m["id"].(string)
			if seen[id] {
				t.Fatalf("event %s appeared on two pages", id)
			}
			seen[id] = true
		}
		if res["has_more"] != true {
			break
		}
		cursor, _ = res["next_after_id"].(string)
	}
	if len(seen) != 137 {
		t.Fatalf("paging visited %d of 137 events", len(seen))
	}
}

func TestALimitCannotBeUsedToAskForEverything(t *testing.T) {
	s := newScheduleServer(t)
	writeRunRecords(t, s, 50)
	res := s.opList("", 1_000_000)
	rows, _ := res["runs"].([]any)
	if len(rows) > maxPageLimit {
		t.Fatalf("a limit of a million returned %d rows; the cap does not hold", len(rows))
	}
}

func TestAnUnknownEventCursorIsEmptyRatherThanTheWholeLog(t *testing.T) {
	s := newScheduleServer(t)
	writeEventLog(t, s, "R-big", 50)
	res := s.opEvents("R-big", "ev-does-not-exist", 10)
	evs, _ := res["events"].([]any)
	if len(evs) != 0 {
		t.Fatalf("an unknown cursor returned %d events", len(evs))
	}
	if res["ok"] != true {
		t.Fatal("an unknown cursor is an error; the log exists, the cursor does not")
	}
}
