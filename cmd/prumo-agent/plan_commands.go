package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/app"
	"github.com/raillen/prumo/internal/contextcompiler"
	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/planning"
	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/protocol/goals"
)

// planSessionData is the harness-neutral payload of `prumo plan status`.
type planSessionData struct {
	Sessions []app.SessionStatus `json:"sessions"`
}

// planQuestionsData is the payload of `prumo plan questions`.
type planQuestionsData struct {
	Session   string                  `json:"session"`
	Questions []planning.OpenQuestion `json:"questions"`
}

// planResumeData is the payload of `prumo plan resume` (and `prumo plan
// --goal <id>` / `prumo plan --resume`). It is the interaction artifact a
// harness reads to drive the next turn: the reconstructed session, its status,
// the compiled context manifest with pressure, the open questions in impact
// order, and whether the run can still proceed.
type planResumeData struct {
	Session   planning.PlanningSession `json:"session"`
	Status    app.SessionStatus        `json:"status"`
	Budget    int                      `json:"budget"`
	Manifest  contextcompiler.Manifest `json:"manifest"`
	Questions []planning.OpenQuestion  `json:"questions"`
	Resumable bool                     `json:"resumable"`
}

// runPlan drives the `prumo plan` interaction protocol. The same application
// service serves both surfaces: `--json` emits the stable machine envelope,
// plain mode prints deterministic human text.
func runPlan(asJSON bool, args []string) int {
	path := "."
	sessionID := ""
	goal := ""
	budget := 0
	action := ""
	question := ""
	statement := ""
	classification := ""
	actor := ""
	source := ""
	var evidence []string
	apply := false
	planRequested := false
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		action = args[0]
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		value := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			v := args[i+1]
			i++
			return v, true
		}
		switch args[i] {
		case "--path":
			v, ok := value()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --path requires a value")
				return exitUsage
			}
			path = v
		case "--session":
			v, ok := value()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --session requires a value")
				return exitUsage
			}
			sessionID = v
		case "--goal":
			v, ok := value()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --goal requires a value")
				return exitUsage
			}
			goal = v
		case "--budget":
			v, ok := value()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --budget requires a value")
				return exitUsage
			}
			parsed, err := strconv.Atoi(v)
			if err != nil || parsed < 0 {
				fmt.Fprintln(os.Stderr, "error: --budget requires a non-negative integer")
				return exitUsage
			}
			budget = parsed
		case "--question", "--statement", "--classification", "--actor", "--source":
			name := args[i]
			v, ok := value()
			if !ok {
				fmt.Fprintf(os.Stderr, "error: %s requires a value\n", name)
				return exitUsage
			}
			switch name {
			case "--question":
				question = v
			case "--statement":
				statement = v
			case "--classification":
				classification = v
			case "--actor":
				actor = v
			case "--source":
				source = v
			}
		case "--evidence":
			v, ok := value()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --evidence requires a value")
				return exitUsage
			}
			evidence = append(evidence, v)
		case "--apply":
			apply = true
		case "--plan":
			planRequested = true
		case "--resume":
			if action == "" {
				action = "resume"
			}
		default:
			fmt.Fprintf(os.Stderr, "error: unsupported plan option: %s\n", args[i])
			return exitUsage
		}
	}
	if action == "" {
		switch {
		case goal != "":
			action = "resume"
		default:
			action = "status"
		}
	}
	switch action {
	case "status":
		return runPlanStatus(asJSON, path, sessionID)
	case "questions":
		return runPlanQuestions(asJSON, path, sessionID)
	case "resume":
		return runPlanResume(asJSON, path, sessionID, goal, budget)
	case "answer":
		return runPlanAnswer(asJSON, path, sessionID, question, statement, classification, actor, source, evidence)
	case "decisions":
		return runPlanDecisions(asJSON, path, sessionID)
	case "delta":
		return runPlanDelta(asJSON, path, sessionID, goal, apply)
	case "blueprint":
		return runPlanBlueprint(asJSON, path, sessionID, goal, planRequested)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown plan subcommand: %s\n", action)
		return exitUsage
	}
}

func planSession(path, sessionID, goal string) (planning.PlanningSession, error) {
	if goal != "" {
		return app.FocusSession(path, goal, sessionID)
	}
	if sessionID != "" {
		return app.LoadSession(path, sessionID)
	}
	return app.LatestSession(path)
}

func runPlanStatus(asJSON bool, path, sessionID string) int {
	if sessionID != "" {
		session, err := app.LoadSession(path, sessionID)
		if err != nil {
			return serviceError(asJSON, err)
		}
		return printPlanStatus(asJSON, []app.SessionStatus{app.StatusOf(session)})
	}
	sessions, err := app.ListSessions(path)
	if err != nil {
		return serviceError(asJSON, err)
	}
	statuses := make([]app.SessionStatus, 0, len(sessions))
	for _, s := range sessions {
		statuses = append(statuses, app.StatusOf(s))
	}
	return printPlanStatus(asJSON, statuses)
}

func printPlanStatus(asJSON bool, statuses []app.SessionStatus) int {
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(planSessionData{Sessions: statuses}))
	}
	if len(statuses) == 0 {
		fmt.Println("no planning sessions found")
		return exitOK
	}
	fmt.Println("SESSION ID    SCOPE          GOAL      DEC  OPEN  RESUMABLE")
	for _, s := range statuses {
		fmt.Printf("%-13s %-13s %-9s %3d %4d  %v\n",
			s.SessionID, s.Scope, s.Goal, s.Decisions, s.OpenQuestions, s.Resumable)
	}
	return exitOK
}

func runPlanQuestions(asJSON bool, path, sessionID string) int {
	session, err := planSession(path, sessionID, "")
	if err != nil {
		return serviceError(asJSON, err)
	}
	questions := app.SessionOpenQuestions(session)
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(planQuestionsData{Session: session.ID, Questions: questions}))
	}
	if len(questions) == 0 {
		fmt.Println("no open questions")
		return exitOK
	}
	for _, q := range questions {
		fmt.Printf("- [%s] %s: %s\n", q.Priority, q.ID, questionText(q))
	}
	return exitOK
}

func runPlanResume(asJSON bool, path, sessionID, goal string, budget int) int {
	session, err := planSession(path, sessionID, goal)
	if err != nil {
		return serviceError(asJSON, err)
	}
	bindings, err := app.LoadSessionBindings(path)
	if err != nil {
		return serviceError(asJSON, err)
	}
	report := app.CompileResumeContext(session, bindings, budget)
	effective := budget
	if effective <= 0 {
		effective = app.DefaultResumeBudget
	}
	questions := app.SessionOpenQuestions(session)
	data := planResumeData{
		Session:   report.Session,
		Status:    app.StatusOf(session),
		Budget:    effective,
		Manifest:  report.Manifest,
		Questions: questions,
		Resumable: app.Resumable(report),
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(data))
	}
	fmt.Printf("Session %s | scope %s | decisions %d | open questions %d\n",
		session.ID, session.Scope, len(session.Decisions), len(questions))
	fmt.Printf("Context: %d/%d tokens (%s)\n",
		report.Manifest.EstimatedTokens, effective, report.Manifest.Pressure)
	fmt.Printf("Resumable: %v\n", data.Resumable)
	if q, found := planning.NextOpen(session.Open); found {
		fmt.Printf("Next question [%s %s]: %s\n", q.Priority, q.ID, questionText(q))
	}
	return exitOK
}

func questionText(q planning.OpenQuestion) string {
	if q.Question != "" {
		return q.Question
	}
	if q.Topic != "" {
		return q.Topic
	}
	if q.Reason != "" {
		return q.Reason
	}
	return "see question record"
}

// planAnswerData is the payload of `prumo plan answer`: the decision proposal
// derived from a classified answer, the question it closed, and the remaining
// session state. It is the record a harness uses to advance the interaction.
type planAnswerData struct {
	Session          string                    `json:"session"`
	Decision         planning.DecisionProposal `json:"decision,omitempty"`
	Resolved         bool                      `json:"resolved"`
	Question         planning.OpenQuestion     `json:"question"`
	OpenRemaining    int                       `json:"open_remaining"`
	BlockersResolved int                       `json:"blockers_resolved"`
	MandatoryPreview bool                      `json:"preview_mandatory"`
	Resumable        bool                      `json:"resumable"`
	Note             string                    `json:"note,omitempty"`
}

// planDecisionsData is the payload of `prumo plan decisions`: the canonical
// decision records recorded so far in a planning session.
type planDecisionsData struct {
	Session   string                      `json:"session"`
	Decisions []planning.DecisionProposal `json:"decisions"`
}

// planDeltaData is the payload of `prumo plan delta [--apply]`: the
// documentation impacts implied by the accepted decisions, the Documentation
// Delta produced (proposed without --apply, applied with it), and the
// recomputed readiness.
type planDeltaData struct {
	Session   string                    `json:"session"`
	Goal      string                    `json:"goal"`
	Apply     bool                      `json:"apply"`
	Impacts   []docengine.Impact        `json:"impacts"`
	Delta     docengine.Delta           `json:"delta"`
	Readiness docengine.ReadinessReport `json:"readiness"`
}

// planBlueprintData is the payload of `prumo plan blueprint [--plan]`: the
// proposed Goal intent/acceptance criteria for the session scope plus, when
// required or explicitly requested, the task DAG.
type planBlueprintData struct {
	Scope            string             `json:"scope"`
	Session          string             `json:"session"`
	Goal             goals.Goal         `json:"goal"`
	ReadinessReady   bool               `json:"readiness_ready"`
	PlanRequired     bool               `json:"plan_required"`
	Rationale        string             `json:"rationale"`
	Tasks            []map[string]any   `json:"tasks,omitempty"`
	DocumentationGap []docengine.Impact `json:"documentation_gap,omitempty"`
	// DocumentationPlan is the Goal preflight (W17.7): the documentation
	// obligations an implementation would owe, with a reason per obligation.
	DocumentationPlan *docengine.DocumentationPlan `json:"documentation_plan,omitempty"`
	PlanFindings      []docengine.SemanticFinding  `json:"plan_findings,omitempty"`
}

// runPlanAnswer ingests a classified answer to an open question, resolves it
// through the authority model, folds the result into the session checkpoint
// and persists it. It never promotes an agent suggestion: inferences stay
// proposed and only explicit sources accept a project decision.
func runPlanAnswer(asJSON bool, path, sessionID, question, statement, classification, actor, source string, evidence []string) int {
	if strings.TrimSpace(question) == "" || strings.TrimSpace(statement) == "" || strings.TrimSpace(classification) == "" {
		fmt.Fprintln(os.Stderr, "error: plan answer requires --question, --statement and --classification")
		return exitUsage
	}
	session, err := planSession(path, sessionID, "")
	if err != nil {
		return serviceError(asJSON, err)
	}
	var target *planning.OpenQuestion
	for i := range session.Open {
		if session.Open[i].ID == question {
			target = &session.Open[i]
			break
		}
	}
	if target == nil {
		return serviceError(asJSON, fmt.Errorf("question %s is not open in session %s", question, session.ID))
	}
	src := planning.Source(strings.TrimSpace(source))
	if src == "" {
		src = planning.SourceUser
	}
	result, err := planning.Resolve(planning.ResolveRequest{
		Question:       *target,
		Statement:      strings.TrimSpace(statement),
		Classification: planning.AnswerClassification(strings.TrimSpace(classification)),
		Source:         src,
		Actor:          strings.TrimSpace(actor),
		Evidence:       evidence,
	})
	if err != nil {
		return serviceError(asJSON, err)
	}
	next, err := session.WithResults([]planning.ResolveResult{result}, nil)
	if err != nil {
		return serviceError(asJSON, err)
	}
	if err := app.SaveSession(path, next); err != nil {
		return serviceError(asJSON, err)
	}
	openRemaining := len(next.Open)
	data := planAnswerData{
		Session:          next.ID,
		Decision:         result.Proposal,
		Resolved:         result.Resolved,
		Question:         result.Question,
		OpenRemaining:    openRemaining,
		BlockersResolved: next.LastPreview.BlockersResolved,
		MandatoryPreview: next.LastPreview.Mandatory,
		Resumable:        openRemaining > 0,
	}
	switch planning.AnswerClassification(strings.TrimSpace(classification)) {
	case planning.ClassificationAgentSuggestion:
		data.Note = "recorded as a proposed agent suggestion; not promoted to an accepted decision"
	case planning.ClassificationHypothesis:
		data.Note = "recorded as a proposed hypothesis with inferred confidence"
	case planning.ClassificationUnresolved:
		data.Note = "recorded as unresolved; question remains open"
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(data))
	}
	if data.Resolved {
		fmt.Printf("resolved %s (%s): open questions now %d\n", result.Question.ID, classification, openRemaining)
	} else {
		fmt.Printf("recorded %s answer (%s): open questions %d\n", result.Question.ID, classification, openRemaining)
	}
	if data.Note != "" {
		fmt.Println(data.Note)
	}
	return exitOK
}

func runPlanDecisions(asJSON bool, path, sessionID string) int {
	session, err := planSession(path, sessionID, "")
	if err != nil {
		return serviceError(asJSON, err)
	}
	data := planDecisionsData{Session: session.ID, Decisions: session.Decisions}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(data))
	}
	if len(data.Decisions) == 0 {
		fmt.Printf("no decisions recorded for session %s\n", session.ID)
		return exitOK
	}
	fmt.Printf("DECISIONS  %s\n", session.ID)
	for _, d := range data.Decisions {
		fmt.Printf("- %s [%s] %s\n", d.ID, d.Status, d.Statement)
		if len(d.Affected) > 0 {
			fmt.Printf("    affects: %s\n", strings.Join(d.Affected, ", "))
		}
	}
	return exitOK
}

// runPlanDelta maps the session's accepted decisions onto documentation
// impacts and produces a Documentation Delta. Without --apply the delta stays
// proposed; with --apply it is walked through governance (proposed → reviewed
// → accepted → applied) and readiness is recomputed. The loop never writes
// canonical documents: it only records the delta under .ai/docs.
func runPlanDelta(asJSON bool, path, sessionID, goal string, apply bool) int {
	session, err := planSession(path, sessionID, goal)
	if err != nil {
		return serviceError(asJSON, err)
	}
	accepted := acceptedAndBound(session.Decisions)
	if len(accepted) == 0 {
		return serviceError(asJSON, fmt.Errorf("no accepted decisions to document in session %s", session.ID))
	}
	source := strings.TrimSpace(session.Goal)
	if source == "" {
		source = strings.TrimSpace(session.Scope)
	}
	bindings, err := app.LoadSessionBindings(path)
	if err != nil {
		return serviceError(asJSON, err)
	}
	impacts := app.ImpactsFromDecisions(bindings, accepted)
	loop := app.LivingPlanLoop{Root: path}
	delta := docengine.Delta{}
	if apply {
		report, err := loop.Run(source, accepted, time.Now())
		if err != nil {
			return serviceError(asJSON, err)
		}
		if err := app.ValidateFeedback(report, accepted); err != nil {
			return serviceError(asJSON, err)
		}
		delta = report.Delta
	} else {
		delta, err = loop.PlanDelta(source, accepted, time.Now())
		if err != nil {
			return serviceError(asJSON, err)
		}
	}
	if err := docengine.SaveDelta(path, delta); err != nil {
		return serviceError(asJSON, err)
	}
	readiness, err := docengine.Readiness(path, source)
	if err != nil {
		return serviceError(asJSON, err)
	}
	data := planDeltaData{Session: session.ID, Goal: source, Apply: apply, Impacts: impacts, Delta: delta, Readiness: readiness}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(data))
	}
	fmt.Printf("Session %s | goal %s\n", session.ID, source)
	if len(impacts) == 0 {
		fmt.Println("no documentation impacts from accepted decisions")
	} else {
		contracts := []string{}
		for _, impact := range impacts {
			contracts = append(contracts, impact.ContractID)
		}
		fmt.Printf("documentation impacts: %s\n", strings.Join(contracts, ", "))
	}
	fmt.Printf("docs delta %s: %s\n", delta.ID, delta.State)
	fmt.Printf("readiness ready: %v\n", readiness.Ready)
	return exitOK
}

// runPlanBlueprint proposes the Goal intent/acceptance criteria for the
// session scope once the interaction has enough structure, plus (when required
// or explicitly requested with --plan) a task DAG. It is the acceptance/test
// plan hand-off of the zero-to-ready loop.
func runPlanBlueprint(asJSON bool, path, sessionID, goal string, planRequested bool) int {
	session, err := planSession(path, sessionID, goal)
	if err != nil {
		return serviceError(asJSON, err)
	}
	readinessGoal := strings.TrimSpace(session.Goal)
	if readinessGoal == "" {
		readinessGoal = strings.TrimSpace(session.Scope)
	}
	readiness, err := docengine.Readiness(path, readinessGoal)
	if err != nil {
		return serviceError(asJSON, err)
	}
	// Preflight (W17.7): the Goal carries its documentation obligations from
	// the moment it is proposed, derived from the contracts the session's
	// accepted decisions touch. The plan is derived runtime state.
	plan, err := docengine.BuildPlanForContracts(path, readinessGoal, app.AffectedContracts(session.LastPreview), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return serviceError(asJSON, err)
	}
	if err := docengine.SavePlan(path, plan); err != nil {
		return serviceError(asJSON, err)
	}
	output, err := app.ProposeGoalPlan(app.GoalPlanInput{Scope: session.Scope, Preview: session.LastPreview, Readiness: readiness, Plan: plan}, planRequested)
	if err != nil {
		return serviceError(asJSON, err)
	}
	data := planBlueprintData{
		Scope:             session.Scope,
		Session:           session.ID,
		Goal:              output.Goal,
		ReadinessReady:    readiness.Ready,
		PlanRequired:      output.PlanRequired,
		Rationale:         output.GammaRationale,
		Tasks:             output.Tasks,
		DocumentationGap:  output.DocumentationGap,
		DocumentationPlan: output.DocumentationPlan,
		PlanFindings:      output.PlanFindings,
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(data))
	}
	title, _ := output.Goal["title"].(string)
	fmt.Printf("Session %s\n", session.ID)
	fmt.Printf("Goal: %s\n", title)
	fmt.Printf("Readiness ready: %v\n", readiness.Ready)
	fmt.Printf("Plan required: %v (%s)\n", output.PlanRequired, output.GammaRationale)
	return exitOK
}

// acceptedAndBound keeps only accepted decision records, in original order.
func acceptedAndBound(decisions []planning.DecisionProposal) []planning.DecisionProposal {
	accepted := []planning.DecisionProposal{}
	for _, d := range decisions {
		if d.Status == planning.StatusAccepted {
			accepted = append(accepted, d)
		}
	}
	return accepted
}
