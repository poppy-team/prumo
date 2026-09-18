package headless

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
)

// The printer is written to from two places at once — the bridge that forwards
// the client's events, and the run's own tail — and a figure reported twice must
// still be one line. Without a lock on both entry points this is a data race
// (found by `-race`, and by a person reading a JSONL stream with half a line in
// it), so both are exercised together here.
func TestThePrinterSurvivesBothOfItsWriters(t *testing.T) {
	var out bytes.Buffer
	printer := NewPrinter(&out, ModeJSON)

	spent := session.Session{
		ID: "S1", PromptTokens: 120, CompletionTokens: 30,
		CacheReadTokens: 900, CacheWriteTokens: 50, Cost: 0.0042,
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			printer.Send(pubsub.Event[session.Session]{Type: pubsub.UpdatedEvent, Payload: spent})
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			printer.Spend(spent)
		}()
	}
	wg.Wait()
	printer.Close()

	// The same figure, however many times it arrived, is one report.
	if got := strings.Count(out.String(), "run.spend"); got != 1 {
		t.Fatalf("the same spend was reported %d times:\n%s", got, out.String())
	}
	if !strings.Contains(out.String(), `"cache_read_tokens":900`) {
		t.Fatalf("the spend lost what it cost:\n%s", out.String())
	}
}

// Two reports that are not the same are two reports: a session that gained
// tokens must not be mistaken for a repeat.
func TestASecondFigureIsASecondReport(t *testing.T) {
	var out bytes.Buffer
	printer := NewPrinter(&out, ModeJSON)

	printer.Spend(session.Session{ID: "S1", PromptTokens: 10, CompletionTokens: 1, Cost: 0.001})
	printer.Spend(session.Session{ID: "S1", PromptTokens: 40, CompletionTokens: 9, Cost: 0.002, CacheReadTokens: 6})

	if got := strings.Count(out.String(), "run.spend"); got != 2 {
		t.Fatalf("a session that grew was reported %d times:\n%s", got, out.String())
	}
}
