package core

import (
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/session"
)

// The accounting keeps the kinds apart because they are priced apart: a cached
// token is not an input token, and a total that hid the difference would report
// what a session cost without saying why.
func TestTheAccountingShowsEveryKindOfToken(t *testing.T) {
	cmp := statusCmp{session: session.Session{
		ID:               "S1",
		PromptTokens:     19_200,
		CacheReadTokens:  8_100,
		CompletionTokens: 760,
		Cost:             0.0412,
		UsageReports:     3,
	}}

	text := cmp.accountingText()
	for _, want := range []string{"in 19.2K", "cache 8.1K", "out 760", "$0.0412", "~$0.0137/req"} {
		if !strings.Contains(text, want) {
			t.Errorf("the accounting lost %q: %q", want, text)
		}
	}
}

// A session that never used a cache says nothing about one: a zero that appears
// without a reason is a field a reader has to decode.
func TestNoCacheIsNotShown(t *testing.T) {
	cmp := statusCmp{session: session.Session{ID: "S1", PromptTokens: 120, CompletionTokens: 30, Cost: 0.001}}

	if text := cmp.accountingText(); strings.Contains(text, "cache") {
		t.Fatalf("a session with no cache reported one: %q", text)
	}
}

// Before the harness has reported anything there is no average: a request count
// of zero is not a request that cost nothing.
func TestNoAverageBeforeTheFirstReport(t *testing.T) {
	cmp := statusCmp{session: session.Session{ID: "S1", PromptTokens: 10}}

	if text := cmp.accountingText(); strings.Contains(text, "/req") {
		t.Fatalf("an average was invented from no reports: %q", text)
	}
}

// The client has no context window for a model it only knows by name, so it
// never claims a share of one.
func TestNoShareOfAWindowIsInvented(t *testing.T) {
	cmp := statusCmp{session: session.Session{ID: "S1", PromptTokens: 900_000, CompletionTokens: 100_000, Cost: 1.2}}

	if text := cmp.accountingText(); strings.Contains(text, "%") {
		t.Fatalf("the accounting invented a share of the context: %q", text)
	}
}

func TestTokenCountsReadTheWayPeopleReadThem(t *testing.T) {
	for _, tc := range []struct {
		tokens int64
		want   string
	}{
		{0, "0"},
		{999, "999"},
		{1_000, "1K"},
		{19_200, "19.2K"},
		{1_000_000, "1M"},
		{2_450_000, "2.5M"},
	} {
		if got := formatTokens(tc.tokens); got != tc.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tc.tokens, got, tc.want)
		}
	}
}

func TestTruncateMarksTheCut(t *testing.T) {
	if got := truncate("a short label", 40); got != "a short label" {
		t.Fatalf("a label that fits was changed: %q", got)
	}
	got := truncate("a label that does not fit at all", 10)
	if got == "" {
		t.Fatal("truncate dropped the whole label")
	}
	if !strings.Contains(got, "...") && !strings.Contains(got, "…") {
		t.Fatalf("the cut is invisible, which reads as a label that ended: %q", got)
	}
}
