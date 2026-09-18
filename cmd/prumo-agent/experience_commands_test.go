package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/experience"
	"github.com/raillen/prumo/internal/protocol"
)

func TestRunExperienceStatus(t *testing.T) {
	tmpDir := t.TempDir()

	code, out := captureOutput(func() int {
		return run([]string{"experience", "status", "--path", tmpDir})
	})
	if code != exitOK {
		t.Fatalf("expected exitOK, got %d; out: %s", code, out)
	}
	if !strings.Contains(out, "PRUMO EXPERIENCE LAYER STATUS") {
		t.Errorf("expected status header, got:\n%s", out)
	}

	// JSON mode
	codeJSON, outJSON := captureOutput(func() int {
		return run([]string{"--json", "experience", "status", "--path", tmpDir})
	})
	if codeJSON != exitOK {
		t.Fatalf("expected exitOK, got %d; out: %s", codeJSON, outJSON)
	}
	var env protocol.Envelope
	if err := json.Unmarshal([]byte(outJSON), &env); err != nil {
		t.Fatalf("failed unmarshaling status JSON: %v", err)
	}
	if !env.Ok {
		t.Errorf("expected env.Ok == true")
	}
}

func TestRunExperienceHandoffLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create handoff
	codeCreate, outCreate := captureOutput(func() int {
		return run([]string{
			"experience", "handoff", "create",
			"--path", tmpDir,
			"--id", "ho-lifecycle",
			"--from", "agent-planner",
			"--to", "agent-coder",
			"--goal", "P00-G01",
		})
	})
	if codeCreate != exitOK {
		t.Fatalf("handoff create failed: %d, out: %s", codeCreate, outCreate)
	}
	if !strings.Contains(outCreate, "Created handoff ho-lifecycle") {
		t.Errorf("unexpected create output: %s", outCreate)
	}

	// 2. Show handoff
	codeShow, outShow := captureOutput(func() int {
		return run([]string{"experience", "handoff", "show", "ho-lifecycle", "--path", tmpDir})
	})
	if codeShow != exitOK {
		t.Fatalf("handoff show failed: %d, out: %s", codeShow, outShow)
	}
	if !strings.Contains(outShow, "HANDOFF: ho-lifecycle (pending)") {
		t.Errorf("unexpected show output: %s", outShow)
	}

	// 3. Acknowledge handoff
	codeAck, outAck := captureOutput(func() int {
		return run([]string{"experience", "handoff", "ack", "ho-lifecycle", "--actor", "agent-coder", "--path", tmpDir})
	})
	if codeAck != exitOK {
		t.Fatalf("handoff ack failed: %d, out: %s", codeAck, outAck)
	}
	if !strings.Contains(outAck, "Acknowledged handoff ho-lifecycle by agent-coder") {
		t.Errorf("unexpected ack output: %s", outAck)
	}

	// 4. Verify status is now acknowledged
	codeShow2, outShow2 := captureOutput(func() int {
		return run([]string{"experience", "handoff", "show", "ho-lifecycle", "--path", tmpDir})
	})
	if codeShow2 != exitOK {
		t.Fatalf("handoff show 2 failed: %d, out: %s", codeShow2, outShow2)
	}
	if !strings.Contains(outShow2, "(acknowledged)") {
		t.Errorf("expected acknowledged status in show output, got: %s", outShow2)
	}
}

func TestRunExperienceEvents(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, ".prumo", "experience")

	prov, err := experience.NewFileProvider(storeDir)
	if err != nil {
		t.Fatalf("NewFileProvider failed: %v", err)
	}
	_ = prov.RecordEvent(experience.NewEvent("e1", "sess-test", experience.EventGoalSelected, "Selected P00-G01", nil))

	code, out := captureOutput(func() int {
		return run([]string{"experience", "events", "--session", "sess-test", "--path", tmpDir})
	})
	if code != exitOK {
		t.Fatalf("expected exitOK, got %d; out: %s", code, out)
	}
	if !strings.Contains(out, "SESSION EVENTS") || !strings.Contains(out, "Selected P00-G01") {
		t.Errorf("unexpected events output: %s", out)
	}
}
