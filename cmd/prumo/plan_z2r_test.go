package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/app"
	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/planning"
)

// writeDogfoodProject scaffolds a self-contained zero-to-ready project: a
// mini registry (2 contracts), a profile, a ready product.vision binding and
// an intentionally unbound project.scope contract, so readiness starts
// blocked and the Living Plan loop has something to close.
func writeDogfoodProject(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"prumo.json": `{"project":{"name":"dogfood","type":["core"]}}`,
		"docs/contracts/builtin.json": `[
			{"id":"product.vision","version":1,"role":"product-vision","required_knowledge":["target users","primary outcome"],"blocking_questions":["q:v-users","q:v-outcome"]},
			{"id":"project.scope","version":1,"role":"project-scope","required_knowledge":["non-goals","in-scope capabilities"],"blocking_questions":["q:s-boundaries"]}
		]`,
		"docs/profiles/builtin.json": `[{"id":"core-software","version":1,"capabilities":["core"],"contracts":["product.vision","project.scope"]}]`,
		"docs/contracts/bindings.json": `[
			{"contract_id":"product.vision","sources":["docs/product/vision.md"],"ownership":"human","authority":"canonical","answered_questions":["q:v-users","q:v-outcome"],"requirement_map":{"target users":"REQ-V-USERS","primary outcome":"REQ-V-OUTCOME"},"claims":[{"id":"CLAIM-V-USERS","statement":"Target users are CLI operators and automation harnesses.","status":"accepted","authority":"canonical-documentation","revision":1,"requires":["REQ-V-USERS"],"evidence":[{"id":"EV-V-USERS","type":"canonical-document","target":"CLAIM-V-USERS","revision":"1","verified":true,"artifact":"docs/product/vision.md"}]},{"id":"CLAIM-V-OUTCOME","statement":"The primary outcome is a provider-neutral planning review state.","status":"accepted","authority":"canonical-documentation","revision":1,"requires":["REQ-V-OUTCOME"],"evidence":[{"id":"EV-V-OUTCOME","type":"canonical-document","target":"CLAIM-V-OUTCOME","revision":"1","verified":true,"artifact":"docs/product/vision.md"}]}]}
		]`,
		"docs/product/vision.md": `# Vision

Target users: CLI operators and automation harnesses.

The primary outcome is a provider-neutral planning review state exposed by the Live Plan protocol.
`,
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// authorDogfoodScope completes the repository governance step: it authors the
// canonical scope document implied by the accepted decision and binds
// project.scope to it, answering its blocking question.
func authorDogfoodScope(t *testing.T, root string) {
	t.Helper()
	scope := `# Scope

In-scope capabilities: provider-neutral planning review state, session checkpoints.

Non-goals: networking, observability and storage backends are out of scope for the first delivery.
`
	if err := os.MkdirAll(filepath.Join(root, "docs", "project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "project", "scope.md"), []byte(scope), 0644); err != nil {
		t.Fatal(err)
	}
	bindings := `[
		{"contract_id":"product.vision","sources":["docs/product/vision.md"],"ownership":"human","authority":"canonical","answered_questions":["q:v-users","q:v-outcome"],"requirement_map":{"target users":"REQ-V-USERS","primary outcome":"REQ-V-OUTCOME"},"claims":[{"id":"CLAIM-V-USERS","statement":"Target users are CLI operators and automation harnesses.","status":"accepted","authority":"canonical-documentation","revision":1,"requires":["REQ-V-USERS"],"evidence":[{"id":"EV-V-USERS","type":"canonical-document","target":"CLAIM-V-USERS","revision":"1","verified":true,"artifact":"docs/product/vision.md"}]},{"id":"CLAIM-V-OUTCOME","statement":"The primary outcome is a provider-neutral planning review state.","status":"accepted","authority":"canonical-documentation","revision":1,"requires":["REQ-V-OUTCOME"],"evidence":[{"id":"EV-V-OUTCOME","type":"canonical-document","target":"CLAIM-V-OUTCOME","revision":"1","verified":true,"artifact":"docs/product/vision.md"}]}]},
		{"contract_id":"project.scope","sources":["docs/project/scope.md"],"ownership":"human","authority":"canonical","answered_questions":["q:s-boundaries"],"requirement_map":{"non-goals":"REQ-S-NONGOALS","in-scope capabilities":"REQ-S-INCAP"},"claims":[{"id":"CLAIM-S-NONGOALS","statement":"Networking, observability and storage backends are out of scope for the first delivery.","status":"accepted","authority":"canonical-documentation","revision":1,"requires":["REQ-S-NONGOALS"],"evidence":[{"id":"EV-S-NONGOALS","type":"canonical-document","target":"CLAIM-S-NONGOALS","revision":"1","verified":true,"artifact":"docs/project/scope.md"}]},{"id":"CLAIM-S-INCAP","statement":"Provider-neutral planning review state and session checkpoints are in scope.","status":"accepted","authority":"canonical-documentation","revision":1,"requires":["REQ-S-INCAP"],"evidence":[{"id":"EV-S-INCAP","type":"canonical-document","target":"CLAIM-S-INCAP","revision":"1","verified":true,"artifact":"docs/project/scope.md"}]}]}
	]` + "\n"
	if err := os.WriteFile(filepath.Join(root, "docs", "contracts", "bindings.json"), []byte(bindings), 0644); err != nil {
		t.Fatal(err)
	}
}

func seedDogfoodSession(t *testing.T, root string) {
	t.Helper()
	session := planning.NewPlanningSession("PLAN-DOG", "PLAN-DOG", "goal:G-DOG", "G-DOG")
	session.Open = []planning.OpenQuestion{
		{ID: "Q2", Scope: "goal:G-DOG", Contract: "product.vision", Priority: "high-risk", Status: "open", Question: "Which target users drive the primary outcome?"},
		{ID: "Q1", Scope: "goal:G-DOG", Contract: "project.scope", Priority: "blocker", Blocking: true, Status: "open", Question: "Which capabilities are out of scope for the first delivery?"},
	}
	if err := app.SaveSession(root, session); err != nil {
		t.Fatal(err)
	}
}

func answerDogfood(t *testing.T, root, session, question, statement, classification string) planAnswerData {
	t.Helper()
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "answer", "--session", session, "--path", root,
		"--question", question, "--statement", statement, "--classification", classification, "--actor", "owner"})
	if !envelope.Ok {
		t.Fatalf("plan answer failed: %s", string(envelope.Data))
	}
	var data planAnswerData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPlanAnswerResolvesDecision(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)

	data := answerDogfood(t, root, "PLAN-DOG", "Q1", "Networking and observability are out of scope for the first delivery.", "explicit-decision")

	if !data.Resolved || data.Session != "PLAN-DOG" {
		t.Fatalf("expected resolved answer, got %#v", data)
	}
	if data.Decision.ID != "DP-Q1-dec" || data.Decision.Status != planning.StatusAccepted {
		t.Fatalf("unexpected decision: %#v", data.Decision)
	}
	if len(data.Decision.Affected) != 1 || data.Decision.Affected[0] != "project.scope" {
		t.Fatalf("expected project.scope impact, got %#v", data.Decision.Affected)
	}
	if data.OpenRemaining != 1 || data.BlockersResolved != 1 || !data.Resumable {
		t.Fatalf("unexpected session state: %#v", data)
	}
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "decisions", "--session", "PLAN-DOG", "--path", root})
	var decisions planDecisionsData
	if err := json.Unmarshal(envelope.Data, &decisions); err != nil {
		t.Fatal(err)
	}
	if !envelope.Ok || len(decisions.Decisions) != 1 || decisions.Decisions[0].Status != planning.StatusAccepted {
		t.Fatalf("expected one persisted accepted decision: %#v", decisions)
	}
}

func TestPlanAnswerNeverPromotesAgentSuggestion(t *testing.T) {
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)

	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "answer", "--session", "PLAN-DOG", "--path", root,
		"--question", "Q1", "--statement", "Suggest keeping SQLite as the journal backend.", "--classification", "agent-suggestion",
		"--source", "model", "--actor", ""})
	if !envelope.Ok {
		t.Fatalf("plan answer failed: %s", string(envelope.Data))
	}
	var data planAnswerData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Resolved {
		t.Fatal("agent suggestion must not resolve a question")
	}
	if data.Decision.ID != "DP-Q1-sug" || data.Decision.Status != planning.StatusProposed {
		t.Fatalf("expected proposed suggestion, got %#v", data.Decision)
	}
	if data.OpenRemaining != 2 || !strings.Contains(data.Note, "not promoted") {
		t.Fatalf("expected question to stay open with promotion note: %#v", data)
	}
	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "decisions", "--session", "PLAN-DOG", "--path", root})
	var decisions planDecisionsData
	if err := json.Unmarshal(envelope.Data, &decisions); err != nil {
		t.Fatal(err)
	}
	if len(decisions.Decisions) != 0 {
		t.Fatalf("suggestion must not be persisted as a canonical decision: %#v", decisions.Decisions)
	}
}

func TestPlanAnswerUnresolvedKeepsQuestionOpen(t *testing.T) {
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)

	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "answer", "--session", "PLAN-DOG", "--path", root,
		"--question", "Q1", "--statement", "Deferring.", "--classification", "unresolved", "--actor", "owner"})
	var data planAnswerData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	if !envelope.Ok || data.Resolved || data.Decision.ID != "" {
		t.Fatalf("expected non-resolving answer, got %#v", data)
	}
	if data.OpenRemaining != 2 || !strings.Contains(data.Note, "question remains open") {
		t.Fatalf("expected question to remain open: %#v", data)
	}
}

func TestPlanAnswerErrors(t *testing.T) {
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)

	if code := run([]string{"plan", "answer", "--session", "PLAN-DOG", "--path", root,
		"--statement", "x", "--classification", "explicit-decision"}); code != exitUsage {
		t.Fatalf("expected usage exit for missing --question, got %d", code)
	}
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "answer", "--session", "PLAN-DOG", "--path", root,
		"--question", "Q99", "--statement", "x", "--classification", "explicit-decision", "--actor", "owner"})
	if envelope.Ok || len(envelope.Diagnostics) == 0 {
		t.Fatalf("expected error for unknown open question: %s", string(envelope.Data))
	}
	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "answer", "--session", "PLAN-DOG", "--path", root,
		"--question", "Q1", "--statement", "x", "--classification", "explicit-decision"})
	if envelope.Ok || len(envelope.Diagnostics) == 0 {
		t.Fatalf("expected error when a user decision lacks an actor: %s", string(envelope.Data))
	}
}

func TestPlanDeltaProposesThenApplies(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)
	answerDogfood(t, root, "PLAN-DOG", "Q1", "Networking and observability are out of scope for the first delivery.", "explicit-decision")
	answerDogfood(t, root, "PLAN-DOG", "Q2", "CLI operators and automation harnesses define the primary outcome.", "constraint")

	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "delta", "--session", "PLAN-DOG", "--path", root})
	if !envelope.Ok {
		t.Fatalf("plan delta failed: %s", string(envelope.Data))
	}
	var proposal planDeltaData
	if err := json.Unmarshal(envelope.Data, &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal.Apply || proposal.Delta.State != "proposed" {
		t.Fatalf("expected proposed delta, got %#v", proposal)
	}
	if len(proposal.Impacts) != 2 || proposal.Impacts[0].ContractID != "project.scope" || proposal.Impacts[1].ContractID != "product.vision" {
		t.Fatalf("unexpected impacts: %#v", proposal.Impacts)
	}
	if proposal.Readiness.Ready || len(proposal.Readiness.BlockingContracts) == 0 {
		t.Fatalf("expected readiness still blocked: %#v", proposal.Readiness)
	}
	if _, err := os.Stat(filepath.Join(root, ".ai", "docs", "deltas", proposal.Delta.ID+".json")); err != nil {
		t.Fatalf("delta not persisted: %v", err)
	}

	authorDogfoodScope(t, root)
	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "delta", "--apply", "--session", "PLAN-DOG", "--path", root})
	if !envelope.Ok {
		t.Fatalf("plan delta --apply failed: %s", string(envelope.Data))
	}
	var applied planDeltaData
	if err := json.Unmarshal(envelope.Data, &applied); err != nil {
		t.Fatal(err)
	}
	if !applied.Apply || applied.Delta.State != "applied" {
		t.Fatalf("expected applied delta, got %#v", applied)
	}
	if len(applied.Delta.Evidence) < 2 || applied.Delta.Evidence[0] != "DP-Q1-dec" || applied.Delta.Evidence[1] != "DP-Q2-dec" {
		t.Fatalf("expected decision evidence in applied delta: %#v", applied.Delta)
	}
	if _, err := os.Stat(filepath.Join(root, ".ai", "docs", "deltas", applied.Delta.ID+".json")); err != nil {
		t.Fatalf("applied delta not persisted: %v", err)
	}
	if !applied.Readiness.Ready || len(applied.Readiness.BlockingContracts) > 0 {
		t.Fatalf("expected readiness to pass after governance: %#v", applied.Readiness)
	}
}

func TestPlanDeltaRequiresAcceptedDecisions(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)
	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "delta", "--session", "PLAN-DOG", "--path", root})
	if envelope.Ok || len(envelope.Diagnostics) == 0 {
		t.Fatalf("expected error without accepted decisions: %s", string(envelope.Data))
	}
}

func TestPlanBlueprintProposesGoal(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)
	answerDogfood(t, root, "PLAN-DOG", "Q1", "Networking and observability are out of scope for the first delivery.", "explicit-decision")
	answerDogfood(t, root, "PLAN-DOG", "Q2", "CLI operators and automation harnesses define the primary outcome.", "constraint")
	authorDogfoodScope(t, root)

	envelope, _ := runPlanJSON(t, []string{"--json", "plan", "blueprint", "--session", "PLAN-DOG", "--path", root})
	if !envelope.Ok {
		t.Fatalf("plan blueprint failed: %s", string(envelope.Data))
	}
	var simple planBlueprintData
	if err := json.Unmarshal(envelope.Data, &simple); err != nil {
		t.Fatal(err)
	}
	if !simple.ReadinessReady {
		t.Fatalf("expected readiness to be reached: %#v", simple)
	}
	if simple.PlanRequired || len(simple.Tasks) != 0 {
		t.Fatalf("expected simple goal without inflated task DAG: %#v", simple)
	}
	if state, _ := simple.Goal["state"].(string); state != "DRAFT" {
		t.Fatalf("expected proposed goal in DRAFT: %#v", simple.Goal)
	}
	acceptance, _ := simple.Goal["acceptance"].([]any)
	if len(acceptance) == 0 {
		t.Fatalf("expected acceptance criteria: %#v", simple.Goal)
	}

	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "blueprint", "--plan", "--session", "PLAN-DOG", "--path", root})
	if !envelope.Ok {
		t.Fatalf("plan blueprint --plan failed: %s", string(envelope.Data))
	}
	var planned planBlueprintData
	if err := json.Unmarshal(envelope.Data, &planned); err != nil {
		t.Fatal(err)
	}
	if !planned.PlanRequired || len(planned.Tasks) == 0 {
		t.Fatalf("expected task DAG when explicitly requested: %#v", planned)
	}
	if planned.Rationale != "plan explicitly requested" {
		t.Fatalf("unexpected rationale: %q", planned.Rationale)
	}
}

func TestDogfoodZeroToReady(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)

	// 1. Readiness is blocked: project.scope is unbound.
	envelope, _ := runPlanJSON(t, []string{"--json", "docs", "readiness", "--path", root, "--goal", "G-DOG"})
	if !envelope.Ok {
		t.Fatalf("readiness failed: %s", string(envelope.Data))
	}
	var report docengine.ReadinessReport
	if err := json.Unmarshal(envelope.Data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Ready || len(report.BlockingContracts) != 1 || report.BlockingContracts[0] != "project.scope" {
		t.Fatalf("expected project.scope to block readiness: %#v", report)
	}

	// 2. The planner surfaces minimal questions, blocker first.
	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "questions", "--session", "PLAN-DOG", "--path", root})
	var questions planQuestionsData
	if err := json.Unmarshal(envelope.Data, &questions); err != nil {
		t.Fatal(err)
	}
	if len(questions.Questions) != 2 || questions.Questions[0].ID != "Q1" || questions.Questions[0].Priority != "blocker" {
		t.Fatalf("expected blocker first, got %#v", questions.Questions)
	}

	// 3. Answers are classified and previewed.
	first := answerDogfood(t, root, "PLAN-DOG", "Q1", "Networking and observability are out of scope for the first delivery.", "explicit-decision")
	if !first.Resolved || first.OpenRemaining != 1 {
		t.Fatalf("expected first blocker resolved: %#v", first)
	}
	second := answerDogfood(t, root, "PLAN-DOG", "Q2", "CLI operators and automation harnesses define the primary outcome.", "constraint")
	if !second.Resolved || second.OpenRemaining != 0 || second.Resumable {
		t.Fatalf("expected all questions closed: %#v", second)
	}

	// 4. Accepted decisions produce a proposed documentation delta.
	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "delta", "--session", "PLAN-DOG", "--path", root})
	var delta planDeltaData
	if err := json.Unmarshal(envelope.Data, &delta); err != nil {
		t.Fatal(err)
	}
	if delta.Delta.State != "proposed" || len(delta.Impacts) != 2 {
		t.Fatalf("expected proposed delta covering both contracts: %#v", delta)
	}

	// 5. Repository governance integrates the patch: authoring the scope doc.
	authorDogfoodScope(t, root)

	// 6. Applying the delta records governance and readiness re-passes.
	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "delta", "--apply", "--session", "PLAN-DOG", "--path", root})
	var applied planDeltaData
	if err := json.Unmarshal(envelope.Data, &applied); err != nil {
		t.Fatal(err)
	}
	if applied.Delta.State != "applied" || !applied.Readiness.Ready {
		t.Fatalf("expected applied delta with passing readiness: %#v", applied)
	}

	// 7. The goal receives an acceptance/test plan.
	envelope, _ = runPlanJSON(t, []string{"--json", "plan", "blueprint", "--plan", "--session", "PLAN-DOG", "--path", root})
	var blueprint planBlueprintData
	if err := json.Unmarshal(envelope.Data, &blueprint); err != nil {
		t.Fatal(err)
	}
	if !envelope.Ok || !blueprint.ReadinessReady || blueprint.PlanRequired != true || len(blueprint.Tasks) == 0 {
		t.Fatalf("expected ready scope with acceptance/test plan: %#v", blueprint)
	}
}

func TestPlanAnswerTextOutput(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writeDogfoodProject(t, root)
	seedDogfoodSession(t, root)
	code, output := captureOutput(func() int {
		return run([]string{"plan", "answer", "--session", "PLAN-DOG", "--path", root,
			"--question", "Q1", "--statement", "Out of scope.", "--classification", "explicit-decision", "--actor", "owner"})
	})
	if code != exitOK || !strings.Contains(output, "resolved Q1") {
		t.Fatalf("unexpected text output: %d %q", code, output)
	}
	code, output = captureOutput(func() int {
		return run([]string{"plan", "decisions", "--session", "PLAN-DOG", "--path", root})
	})
	if code != exitOK || !strings.Contains(output, "DP-Q1-dec") {
		t.Fatalf("unexpected decisions text output: %d %q", code, output)
	}
}
