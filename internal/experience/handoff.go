package experience

import (
	"fmt"
	"strings"
	"time"
)

// HandoffStatus models the lifecycle state of a structured agent/session handoff.
//
// # What exists today
//
// A handoff can be created, persisted, listed, acknowledged and rejected. Those
// five are reachable from the CLI and from a real file provider.
//
// # What does not
//
// A handoff is an export artifact, not a live protocol. There is no dispatch —
// nothing resolves `to` to a running agent and delivers the bundle — and no
// continuation — nothing on the receiving side resumes the run. HandoffTransferred
// is the state a dispatch would set, and no code assigns it, so the transition
// pending -> transferred -> acknowledged is documented but not walkable.
//
// That is a deliberate state of affairs, not an oversight, and it is recorded
// here because the type reads like a protocol and a reader should not have to
// grep for this to find out (GAP-135). Implementing dispatch and continuation is
// a feature decision, not a correction, and it is better made alongside whatever
// becomes the primary agent interface than guessed at now.
type HandoffStatus string

const (
	HandoffPending HandoffStatus = "pending"
	// HandoffTransferred is unreachable today: no code assigns it, because no
	// code dispatches a handoff. It is kept because a dispatch would need it and
	// deleting it would make that implementation change the vocabulary.
	HandoffTransferred  HandoffStatus = "transferred"
	HandoffAcknowledged HandoffStatus = "acknowledged"
	HandoffRejected     HandoffStatus = "rejected"
)

func (s HandoffStatus) Valid() bool {
	switch s {
	case HandoffPending, HandoffTransferred, HandoffAcknowledged, HandoffRejected:
		return true
	}
	return false
}

// CreateHandoff constructs and validates an agent-to-agent state handoff.
func CreateHandoff(id, from, to, runID, goalID string, summary Summary, decisions, questions, evidence []string) (Handoff, error) {
	if strings.TrimSpace(id) == "" {
		return Handoff{}, fmt.Errorf("handoff requires an ID")
	}
	if strings.TrimSpace(from) == "" {
		return Handoff{}, fmt.Errorf("handoff requires a sender ('from')")
	}
	if strings.TrimSpace(to) == "" {
		return Handoff{}, fmt.Errorf("handoff requires a recipient ('to')")
	}

	h := Handoff{
		ID:              id,
		From:            from,
		To:              to,
		RunID:           runID,
		GoalID:          goalID,
		Status:          HandoffPending,
		Summary:         summary,
		ActiveDecisions: decisions,
		OpenQuestions:   questions,
		Evidence:        evidence,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	return h, nil
}

// Acknowledge marks the handoff as accepted by the receiving agent.
func (h *Handoff) Acknowledge(actor string) error {
	if h.Status != HandoffPending && h.Status != HandoffTransferred {
		return fmt.Errorf("cannot acknowledge handoff in status %q", h.Status)
	}
	h.Status = HandoffAcknowledged
	h.AcknowledgedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}

// Reject marks the handoff as rejected by the recipient.
func (h *Handoff) Reject(actor, reason string) error {
	if h.Status != HandoffPending && h.Status != HandoffTransferred {
		return fmt.Errorf("cannot reject handoff in status %q", h.Status)
	}
	h.Status = HandoffRejected
	return nil
}
