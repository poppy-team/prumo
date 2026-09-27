package main

import (
	"testing"

	"github.com/raillen/prumo/internal/cliops"
)

func TestRunExplain_NewTopics(t *testing.T) {
	svc := cliops.New(t.TempDir())

	// 1. explain run
	if code := runExplain(svc, false, []string{"run"}); code != exitOK {
		t.Fatalf("expected exitOK for explain run, got %d", code)
	}
	if code := runExplain(svc, true, []string{"run", "R-100"}); code != exitOK {
		t.Fatalf("expected exitOK for explain run --json, got %d", code)
	}

	// 2. explain route
	if code := runExplain(svc, false, []string{"route"}); code != exitOK {
		t.Fatalf("expected exitOK for explain route, got %d", code)
	}
	if code := runExplain(svc, true, []string{"route", "ROUTE-PRIMARY"}); code != exitOK {
		t.Fatalf("expected exitOK for explain route ROUTE-PRIMARY --json, got %d", code)
	}

	// 3. explain budget
	if code := runExplain(svc, false, []string{"budget"}); code != exitOK {
		t.Fatalf("expected exitOK for explain budget, got %d", code)
	}
	if code := runExplain(svc, true, []string{"budget"}); code != exitOK {
		t.Fatalf("expected exitOK for explain budget --json, got %d", code)
	}

	// 4. explain decision
	if code := runExplain(svc, false, []string{"decision"}); code != exitOK {
		t.Fatalf("expected exitOK for explain decision, got %d", code)
	}
	if code := runExplain(svc, true, []string{"decision"}); code != exitOK {
		t.Fatalf("expected exitOK for explain decision --json, got %d", code)
	}

	// 5. unknown topic
	if code := runExplain(svc, false, []string{"unsupported_topic"}); code != exitUsage {
		t.Fatalf("expected exitUsage for unknown explain topic, got %d", code)
	}
}
