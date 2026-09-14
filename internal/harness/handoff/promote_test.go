package handoff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/planning"
)

func TestPromotePlanning(t *testing.T) {
	s := planning.PlanningSession{
		ID: "ps1", RunID: "R0", Scope: "harness", Goal: "ship it",
		Decisions: []planning.DecisionProposal{{ID: "d1"}},
		Open:      []planning.OpenQuestion{{ID: "q1"}},
	}
	p, err := PromotePlanning(s, "R-build")
	if err != nil {
		t.Fatal(err)
	}
	if p.RunID != "R-build" || len(p.DecisionIDs) != 1 || len(p.OpenIDs) != 1 {
		t.Fatalf("bad promotion: %+v", p)
	}
	if !strings.Contains(p.GoalText(), "ship it") || !strings.Contains(p.GoalText(), "d1") {
		t.Fatalf("goal text missing refs: %q", p.GoalText())
	}
	if _, err := PromotePlanning(s, ""); err == nil {
		t.Fatal("promotion requires run id")
	}
	bad := s
	bad.ID = ""
	if _, err := PromotePlanning(bad, "R"); err == nil {
		t.Fatal("invalid session must fail")
	}
}

func TestPromoteFile(t *testing.T) {
	cp := planning.SessionCheckpoint{Version: 1, SessionID: "ps2", RunID: "R0", Scope: "s", Goal: "g"}
	data, _ := json.Marshal(cp)
	path := filepath.Join(t.TempDir(), "ps2.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := PromoteFile(path, "R-b")
	if err != nil || p.SessionID != "ps2" {
		t.Fatalf("promote file failed: %+v %v", p, err)
	}
}

func TestPromoteBytes(t *testing.T) {
	cp := planning.SessionCheckpoint{Version: 1, SessionID: "ps3", RunID: "R0", Scope: "s", Goal: "g"}
	data, _ := json.Marshal(cp)
	p, err := PromoteBytes(data, "R-c")
	if err != nil || p.SessionID != "ps3" || p.RunID != "R-c" {
		t.Fatalf("promote bytes failed: %+v %v", p, err)
	}
	if _, err := PromoteBytes(nil, "R-c"); err == nil {
		t.Fatal("empty payload must fail")
	}
}
