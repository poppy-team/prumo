// Planning promotion (GAP-017): a PlanningSession becomes build input
// without becoming a Run. Promotion emits goal text plus provenance refs;
// the typed Handoff bundle is built later from the first checkpoint, so
// cross-phase lineage never depends on transcripts.
package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/planning"
)

// Promotion is the build-ready projection of a planning session.
type Promotion struct {
	SessionID   string   `json:"session_id"`
	RunID       string   `json:"run_id"`
	Goal        string   `json:"goal"`
	DecisionIDs []string `json:"decision_ids"`
	OpenIDs     []string `json:"open_ids"`
	Sources     []string `json:"sources"`
	Summary     string   `json:"summary"`
}

// PromotePlanning validates the session and projects it.
func PromotePlanning(s planning.PlanningSession, runID string) (Promotion, error) {
	if err := s.Validate(); err != nil {
		return Promotion{}, err
	}
	if strings.TrimSpace(runID) == "" {
		return Promotion{}, fmt.Errorf("promotion requires a run id")
	}
	p := Promotion{SessionID: s.ID, RunID: runID, Goal: s.Goal}
	for _, d := range s.Decisions {
		if d.ID != "" {
			p.DecisionIDs = append(p.DecisionIDs, d.ID)
		}
	}
	for _, q := range s.Open {
		if q.ID != "" {
			p.OpenIDs = append(p.OpenIDs, q.ID)
		}
	}
	for _, src := range s.ContextSources() {
		if src.Ref != "" {
			p.Sources = append(p.Sources, src.Ref)
		}
	}
	p.Summary = fmt.Sprintf("session %s → run %s (%d decisions, %d open, %d sources)",
		s.ID, runID, len(p.DecisionIDs), len(p.OpenIDs), len(p.Sources))
	return p, nil
}

// GoalText renders the run input: goal plus accepted-decision pointers.
func (p Promotion) GoalText() string {
	var b strings.Builder
	b.WriteString(p.Goal)
	if len(p.DecisionIDs) > 0 {
		b.WriteString("\n\nAccepted decisions: " + strings.Join(p.DecisionIDs, ", "))
	}
	if len(p.OpenIDs) > 0 {
		b.WriteString("\nOpen questions: " + strings.Join(p.OpenIDs, ", "))
	}
	return b.String()
}

// PromoteFile loads a persisted planning checkpoint and promotes it.
func PromoteFile(path, runID string) (Promotion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Promotion{}, err
	}
	return PromoteBytes(data, runID)
}

// PromoteBytes validates a persisted planning checkpoint payload and
// promotes it. It exists so callers can promote in-memory payloads without
// round-tripping through a temporary file.
func PromoteBytes(data []byte, runID string) (Promotion, error) {
	var cp planning.SessionCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return Promotion{}, err
	}
	s := planning.PlanningSession{
		ID: cp.SessionID, RunID: cp.RunID, Scope: cp.Scope, Goal: cp.Goal,
		Decisions: cp.Decisions, Open: cp.Open, LastPreview: cp.Preview, UpdatedAt: cp.UpdatedAt,
	}
	return PromotePlanning(s, runID)
}
