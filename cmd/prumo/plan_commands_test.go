package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/app"
	"github.com/raillen/prumo/internal/planning"
	"github.com/raillen/prumo/internal/protocol"
)

type planEnvelope struct {
	ProtocolVersion string                `json:"protocol_version"`
	Ok              bool                  `json:"ok"`
	Data            json.RawMessage       `json:"data"`
	Diagnostics     []protocol.Diagnostic `json:"diagnostics"`
}

func runPlanJSON(t *testing.T, args []string) (planEnvelope, string) {
	t.Helper()
	code, output := captureOutput(func() int { return run(args) })
	var envelope planEnvelope
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("invalid plan JSON (exit %d): %v %q", code, err, output)
	}
	return envelope, output
}

func writePlanBindings(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "docs", "contracts", "bindings.json")
	content := `[
		{"contract_id": "architecture.system", "sources": ["docs/architecture/overview.md"], "ownership": "human", "authority": "canonical"}
	]` + "\n"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func seededSession(t *testing.T, root string) {
	t.Helper()
	session := planning.NewPlanningSession("PLAN-G042", "PLAN-G042", "goal:G042", "G042")
	session.Open = []planning.OpenQuestion{
		{ID: "Q2", Scope: "goal:G042", Contract: "architecture.system", Priority: "high-risk", Status: "open", Question: "Which subsystem owns audit state?"},
		{ID: "Q1", Scope: "goal:G042", Contract: "architecture.system", Priority: "blocker", Blocking: true, Status: "open", Question: "Where does the control plane persist its journal?"},
	}
	if err := app.SaveSession(root, session); err != nil {
		t.Fatal(err)
	}
}

func TestPlanGoalStartsSession(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "--goal", "G042", "--path", root, "--session", "S1"})
	if !envelope.Ok {
		t.Fatalf("expected ok envelope: %s", string(envelope.Data))
	}
	var data planResumeData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Session.ID != "S1" || data.Session.Scope != "goal:G042" || data.Session.RunID != "S1" {
		t.Fatalf("unexpected created session: %#v", data.Session)
	}
	if data.Budget != app.DefaultResumeBudget {
		t.Fatalf("expected default budget, got %d", data.Budget)
	}
	if envelope.ProtocolVersion != protocol.ProtocolVersion {
		t.Fatalf("unexpected protocol version %q", envelope.ProtocolVersion)
	}
}

func TestPlanQuestionsOrderedByPriority(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	seededSession(t, root)
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "questions", "--session", "PLAN-G042", "--path", root})
	if !envelope.Ok {
		t.Fatalf("expected ok envelope: %s", string(envelope.Data))
	}
	var data planQuestionsData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Questions) != 2 || data.Questions[0].ID != "Q1" || data.Questions[0].Priority != "blocker" {
		t.Fatalf("expected blocker first, got %#v", data.Questions)
	}
	if data.Questions[1].ID != "Q2" {
		t.Fatalf("expected Q2 second, got %#v", data.Questions)
	}
}

func TestPlanStatusAggregatesSessions(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	sessionA := planning.NewPlanningSession("PLAN-A", "PLAN-A", "goal:G001", "G001")
	if err := app.SaveSession(root, sessionA); err != nil {
		t.Fatal(err)
	}
	seededSession(t, root)
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "status", "--path", root})
	if !envelope.Ok {
		t.Fatalf("expected ok envelope: %s", string(envelope.Data))
	}
	var data planSessionData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Sessions) != 2 || data.Sessions[0].SessionID != "PLAN-A" || data.Sessions[1].SessionID != "PLAN-G042" {
		t.Fatalf("expected deterministic session list, got %#v", data.Sessions)
	}
	if !data.Sessions[1].Resumable || data.Sessions[1].OpenQuestions != 2 {
		t.Fatalf("expected seeded session resumable with 2 open questions: %#v", data.Sessions[1])
	}
}

func TestPlanResumeCompilesContext(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writePlanBindings(t, root)
	seededSession(t, root)
	envelope, output := runPlanJSON(t, []string{"--json", "plan", "resume", "--session", "PLAN-G042", "--path", root, "--budget", "4000"})
	if !envelope.Ok {
		t.Fatalf("expected ok envelope: %s", string(envelope.Data))
	}
	var data planResumeData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	if !data.Resumable || data.Manifest.Pressure != "healthy" {
		t.Fatalf("expected healthy resumable context: %#v", data.Manifest)
	}
	if data.Manifest.EstimatedTokens > data.Budget {
		t.Fatalf("context exceeded budget: %d > %d", data.Manifest.EstimatedTokens, data.Budget)
	}
	foundDoc := false
	for _, s := range data.Manifest.Sources {
		if strings.HasPrefix(s.Ref, "doc:") {
			foundDoc = true
		}
	}
	if !foundDoc {
		t.Fatalf("expected canonical doc source from bindings: %#v", data.Manifest.Sources)
	}
	envelopeAgain, outputAgain := runPlanJSON(t, []string{"--json", "plan", "resume", "--session", "PLAN-G042", "--path", root, "--budget", "4000"})
	if !envelopeAgain.Ok || output != outputAgain {
		t.Fatal("expected deterministic resume output")
	}
}

func TestPlanResumePressureStopsAtCritical(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	seededSession(t, root)
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "resume", "--session", "PLAN-G042", "--path", root, "--budget", "40"})
	if !envelope.Ok {
		t.Fatalf("expected ok envelope: %s", string(envelope.Data))
	}
	var data planResumeData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Manifest.Pressure != "critical" || data.Resumable {
		t.Fatalf("expected critical non-resumable context: %#v", data.Manifest)
	}
}

func TestPlanMissingSessionErrors(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	code, _ := captureOutput(func() int {
		return run([]string{"--json", "plan", "status", "--session", "NOPE", "--path", t.TempDir()})
	})
	if code != exitValidation {
		t.Fatalf("expected validation exit, got %d", code)
	}
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "status", "--session", "NOPE", "--path", t.TempDir()})
	if envelope.Ok || len(envelope.Diagnostics) == 0 {
		t.Fatalf("expected error diagnostics: %s", string(envelope.Data))
	}
}

func TestPlanUsage(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	if code := run([]string{"plan", "unknown-subcommand", "--path", t.TempDir()}); code != exitUsage {
		t.Fatalf("expected usage exit for unknown subcommand, got %d", code)
	}
	if code := run([]string{"plan", "resume", "--budget", "abc", "--path", t.TempDir()}); code != exitUsage {
		t.Fatalf("expected usage exit for bad budget, got %d", code)
	}
	if code := run([]string{"plan", "resume", "--unsupported", "x", "--path", t.TempDir()}); code != exitUsage {
		t.Fatalf("expected usage exit for unsupported option, got %d", code)
	}
}

func TestPlanTextOutput(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	seededSession(t, root)
	code, output := captureOutput(func() int {
		return run([]string{"plan", "questions", "--session", "PLAN-G042", "--path", root})
	})
	if code != exitOK || !strings.Contains(output, "[blocker] Q1") || !strings.Contains(output, "[high-risk] Q2") {
		t.Fatalf("unexpected text output: %d %q", code, output)
	}
	code, output = captureOutput(func() int {
		return run([]string{"plan", "status", "--path", t.TempDir()})
	})
	if code != exitOK || !strings.Contains(output, "no planning sessions found") {
		t.Fatalf("unexpected empty status output: %d %q", code, output)
	}
}
