package schemaruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	docengine "github.com/raillen/prumo/internal/documentation"
)

type stateMatrix struct {
	ID      string `json:"id"`
	Surface string `json:"surface"`
	States  []struct {
		State          string   `json:"state"`
		Applicable     bool     `json:"applicable"`
		Reason         string   `json:"reason"`
		UserVisibleTxt string   `json:"user_visible_text"`
		Recovery       string   `json:"recovery"`
		Evidence       []string `json:"evidence"`
		MediaRef       string   `json:"media_ref"`
	} `json:"states"`
	Transitions []struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Trigger string `json:"trigger"`
	} `json:"transitions"`
}

// TestStateMatrixIsCompleteAndExplicit is the artifact-level half of the
// ui.state-model contract. The schema keeps the vocabulary closed; this keeps
// the repository's own matrix honest: every state declared exactly once, every
// non-applicable state carrying a reason, and every applicable state carrying
// what the user sees and how they recover.
func TestStateMatrixIsCompleteAndExplicit(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "ui-ux", "state-matrix.json"))
	if err != nil {
		t.Fatalf("the ui.state-model contract requires this artifact: %v", err)
	}
	var matrix stateMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatalf("state matrix is not valid JSON: %v", err)
	}
	if !strings.HasPrefix(matrix.ID, "ui:") || matrix.ID == "ui:" {
		t.Errorf("matrix id must match the ui: pattern, got %q", matrix.ID)
	}
	if strings.TrimSpace(matrix.Surface) == "" {
		t.Error("matrix must name the surface it describes")
	}

	seen := map[string]int{}
	for _, state := range matrix.States {
		seen[state.State]++
		switch {
		case !state.Applicable && strings.TrimSpace(state.Reason) == "":
			t.Errorf("state %q is not applicable but states no reason", state.State)
		case state.Applicable && strings.TrimSpace(state.UserVisibleTxt) == "":
			t.Errorf("state %q is applicable but declares no user-visible text", state.State)
		case state.Applicable && strings.TrimSpace(state.Recovery) == "":
			t.Errorf("state %q is applicable but declares no recovery behaviour", state.State)
		}
	}
	for _, state := range docengine.UIStates() {
		switch seen[state] {
		case 1:
		case 0:
			t.Errorf("the state vocabulary declares %q but the matrix does not", state)
		default:
			t.Errorf("state %q is declared %d times in the matrix", state, seen[state])
		}
	}
	if len(matrix.States) != len(docengine.UIStates()) {
		t.Errorf("matrix declares %d states, vocabulary has %d",
			len(matrix.States), len(docengine.UIStates()))
	}

	// A transition to a state nobody declared would render as a dead end.
	for _, transition := range matrix.Transitions {
		if strings.TrimSpace(transition.Trigger) == "" {
			t.Errorf("transition %s → %s has no trigger", transition.From, transition.To)
		}
		if transition.From != "final" && seen[transition.From] == 0 {
			t.Errorf("transition starts at undeclared state %q", transition.From)
		}
		if seen[transition.To] == 0 {
			t.Errorf("transition ends at undeclared state %q", transition.To)
		}
	}
}
