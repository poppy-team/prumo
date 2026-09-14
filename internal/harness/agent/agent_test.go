package agent

import "testing"

func TestPhasesDistinct(t *testing.T) {
	phases := []Phase{PhasePrepare, PhaseCompileContext, PhaseRequestModel, PhaseConsumeModelEvent, PhasePlanToolCalls, PhasePermissionCheck, PhaseExecuteTool, PhaseRecordObservation, PhaseEvaluateStop, PhaseCheckpoint, PhaseYield, PhaseComplete}
	seen := map[Phase]bool{}
	for _, p := range phases {
		if seen[p] {
			t.Fatalf("duplicate phase %s", p)
		}
		seen[p] = true
	}
}

func TestPermissionDecisions(t *testing.T) {
	if PermissionAllow == PermissionDeny || PermissionAsk == PermissionDeny {
		t.Fatal("permission decisions must be distinct")
	}
}
