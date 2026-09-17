//go:build ignore

// Package tui is the Prumo terminal client.
//
// It is a *client* of the harness, never a part of it: it reaches the daemon
// through the public SDK (`sdk/prumo`) and through the `prumo` binary it
// supervises as a subprocess. `TestBoundaryNoInternalImports` fails the build
// if that stops being true, which is the machine-checkable form of H10
// acceptance criterion 1.
//
// The package is split so that the parts worth testing do not need a terminal:
//
//	timeline.go  events  → bounded, classified timeline rows   (pure)
//	palette.go   query   → ranked commands                     (pure)
//	styles.go    theme   → painted strings                     (no I/O)
//	session.go   daemon  → start / stream / cancel             (SDK only)
//	daemon.go    argv    → running daemon subprocess           (process I/O)
//	app.go       model   → glue over the above                 (Bubble Tea)
//
// Everything the vertical slice renders is decided by the pure layers; `app.go`
// only routes keys and messages into them. That is deliberate: a renderer that
// decides is a renderer that can only be tested by driving a terminal.
package tui

import (
	"fmt"
	"sort"
	"strings"

	prumo "github.com/raillen/prumo/sdk/prumo"
)

// Severity classifies a timeline row for painting. It is a closed vocabulary:
// an unknown event kind maps to SeverityInfo rather than to a new bucket, so a
// daemon that starts emitting a new kind renders as ordinary activity instead of
// breaking the view.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeveritySuccess Severity = "success"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Entry is one rendered timeline row. It is derived from an event and carries
// only what a reader needs: what happened, in what order, and how bad it is.
type Entry struct {
	Index    int
	EventID  string
	Kind     string
	Severity Severity
	Title    string
	Detail   string
}

// DefaultTimelineCapacity bounds how many rows the view keeps. The timeline is
// a window on the run, not the run's store — the daemon owns the full history
// and `Client.Events` can replay it. Keeping the view bounded is what stops a
// long run from turning the terminal into an unbounded scrollback.
const DefaultTimelineCapacity = 500

// Timeline is a bounded, append-only view of a run's events.
type Timeline struct {
	capacity int
	entries  []Entry
	dropped  int
}

// NewTimeline returns a timeline keeping at most capacity rows. A non-positive
// capacity falls back to DefaultTimelineCapacity.
func NewTimeline(capacity int) *Timeline {
	if capacity <= 0 {
		capacity = DefaultTimelineCapacity
	}
	return &Timeline{capacity: capacity}
}

// Append adds one event, dropping the oldest row when the cap is reached.
func (t *Timeline) Append(ev prumo.Event) {
	entry := Entry{
		EventID:  ev.ID,
		Kind:     ev.Kind,
		Severity: severityOf(ev.Kind),
		Title:    titleOf(ev),
		Detail:   detailOf(ev),
	}
	t.entries = append(t.entries, entry)
	if len(t.entries) > t.capacity {
		over := len(t.entries) - t.capacity
		t.entries = append([]Entry{}, t.entries[over:]...)
		t.dropped += over
	}
	t.reindex()
}

// Entries returns the visible rows, oldest first, with stable 1-based indices.
func (t *Timeline) Entries() []Entry {
	return append([]Entry{}, t.entries...)
}

// Len is the number of visible rows.
func (t *Timeline) Len() int { return len(t.entries) }

// Dropped is how many rows aged out of the window. It is reported in the
// statusline: a view that silently forgets is a view that lies about the run.
func (t *Timeline) Dropped() int { return t.dropped }

// Last returns the newest row, if any.
func (t *Timeline) Last() (Entry, bool) {
	if len(t.entries) == 0 {
		return Entry{}, false
	}
	return t.entries[len(t.entries)-1], true
}

func (t *Timeline) reindex() {
	for i := range t.entries {
		t.entries[i].Index = t.dropped + i + 1
	}
}

// severityOf maps an event kind to a severity. Only kinds whose meaning is a
// failure or a success are special-cased; everything else is ordinary activity.
func severityOf(kind string) Severity {
	switch {
	case kind == "run.finished", kind == "completed", kind == "tool_call_ready", kind == "permission_approved":
		return SeveritySuccess
	case kind == "warning", kind == "permission_wait":
		return SeverityWarning
	case kind == "error", kind == "failed", kind == "permission_denied", kind == "permission_rejected", kind == "cancelled":
		return SeverityError
	default:
		return SeverityInfo
	}
}

// titleOf is the one-line summary of an event. It prefers a human field the
// event actually carries over restating the kind, because a timeline that only
// repeats `text` is a timeline nobody reads.
func titleOf(ev prumo.Event) string {
	switch ev.Kind {
	case "run.started":
		return "run started"
	case "run.finished":
		return "run " + stringField(ev.Payload, "status", "finished")
	case "context.compiled":
		return fmt.Sprintf("context compiled (%s tokens)",
			numberField(ev.Payload, "tokens"))
	case "text_delta":
		if text := stringField(ev.Payload, "text", ""); text != "" {
			return firstLine(text)
		}
		return "text"
	case "tool_call", "tool_call_ready":
		if name := stringField(ev.Payload, "name", stringField(ev.Payload, "tool", "tool")); name != "" {
			return name
		}
		return "tool call"
	case "permission_denied":
		return "permission denied: " + stringField(ev.Payload, "tool", "tool")
	case "permission_wait":
		return "waiting for approval: " + stringField(ev.Payload, "tool", "tool")
	case "permission_approved":
		return "permission approved"
	case "permission_rejected":
		return "permission rejected"
	case "usage",
		"usage_updated":
		return "usage updated"
	}
	if ev.Kind == "" {
		return "event"
	}
	return ev.Kind
}

// detailOf renders the payload compactly. Keys are sorted so the same event
// always renders identically — an unstable detail line makes golden frames
// useless.
func detailOf(ev prumo.Event) string {
	if len(ev.Payload) == 0 {
		return ""
	}
	keys := make([]string, 0, len(ev.Payload))
	for k := range ev.Payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == "text" {
			continue // already the title
		}
		parts = append(parts, k+"="+compact(ev.Payload[k]))
	}
	return strings.Join(parts, "  ")
}

func compact(v any) string {
	switch value := v.(type) {
	case nil:
		return "-"
	case string:
		return strings.ReplaceAll(firstLine(value), "\n", " ")
	case float64:
		if value == float64(int64(value)) {
			return fmt.Sprintf("%d", int64(value))
		}
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", value), "0"), ".")
	case bool:
		return fmt.Sprintf("%v", value)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func stringField(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	if value, ok := payload[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

func numberField(payload map[string]any, key string) string {
	if payload == nil {
		return "?"
	}
	if value, ok := payload[key].(float64); ok {
		return fmt.Sprintf("%d", int64(value))
	}
	return "?"
}

// firstLine truncates a multi-line value to its first line so one event can
// never grow the timeline row to fill the screen.
func firstLine(s string) string {
	if idx := strings.IndexAny(s, "\r\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}
