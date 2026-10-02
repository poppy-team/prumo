package app

import (
	"fmt"
	"strings"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/planning"
)

// SeedQuestionsFromReadiness is the intake the Living Plan was missing. It
// turns the documentation engine's coverage gaps into the OpenQuestions an
// interview session works from, so the planner stops waiting for someone to
// hand-write `.ai/plan/sessions/*.json` and starts answering the project's
// actual unknowns.
//
// A question is one per unmet obligation, i.e. one per contract * blocking
// question that readiness flags as unresolved. That mirrors how the README of
// the Living Plan sample had to be seeded manually — the difference is that
// this derives them from `docs readiness` automatically instead of requiring
// the operator, or a test fixture, to know the gaps in advance.
//
// The function is idempotent over the session's current contents: a question
// already present (same contract, same text) is never duplicated, and existing
// decisions are never touched.
func SeedQuestionsFromReadiness(root, goal string, session planning.PlanningSession) (planning.PlanningSession, []planning.OpenQuestion, error) {
	report, err := docengine.Readiness(root, goal)
	if err != nil {
		return session, nil, err
	}
	registry, err := docengine.LoadRegistry(root)
	if err != nil {
		return session, nil, err
	}

	identity := map[string]bool{}
	for _, existing := range session.Open {
		identity[questionKey(existing.Contract, existing.Question)] = true
	}

	added := []planning.OpenQuestion{}
	sequence := 0
	for _, coverage := range report.Coverage {
		switch coverage.State {
		case docengine.Missing, docengine.Partial, docengine.Stale, docengine.Unverified:
		default:
			continue
		}
		contract, ok := registry.Contracts[coverage.ContractID]
		if !ok {
			continue
		}

		questions := append([]string{}, coverage.BlockingQuestions...)
		if len(questions) == 0 {
			questions = append(questions, contract.BlockingQuestions...)
		}
		for _, questionText := range questions {
			if strings.TrimSpace(questionText) == "" {
				continue
			}
			key := questionKey(coverage.ContractID, questionText)
			if identity[key] {
				continue
			}
			sequence++
			priority := coveragePriority(coverage.State)
			added = append(added, planning.OpenQuestion{
				ID:       fmt.Sprintf("Q-%s-%d", coverage.ContractID, sequence),
				Scope:    session.Scope,
				Contract: coverage.ContractID,
				Priority: priority,
				Blocking: priority == "blocker",
				Status:   "open",
				Question: questionText,
				Reason:   coverageReason(coverage),
			})
			identity[key] = true
		}

		// Coverage with missing knowledge but no explicit blocking questions in
		// the contract still needs resolution: synthesize one targeted question
		// per unfulfilled knowledge item rather than leaving the gap invisible.
		if len(questions) == 0 {
			for _, knowledge := range coverage.MissingKnowledge {
				if strings.TrimSpace(knowledge) == "" {
					continue
				}
				questionText := fmt.Sprintf("For the %q contract, what is the %s?", coverage.ContractID, knowledge)
				key := questionKey(coverage.ContractID, questionText)
				if identity[key] {
					continue
				}
				sequence++
				added = append(added, planning.OpenQuestion{
					ID:       fmt.Sprintf("Q-%s-%d", coverage.ContractID, sequence),
					Scope:    session.Scope,
					Contract: coverage.ContractID,
					Priority: "high-risk",
					Status:   "open",
					Question: questionText,
					Reason:   fmt.Sprintf("coverage reports missing knowledge for %s", coverage.ContractID),
				})
				identity[key] = true
			}
		}
	}

	if len(added) == 0 {
		return session, nil, nil
	}
	next := session
	next.Open = append(next.Open, added...)
	next.Open = planning.PriorityOrder(next.Open)
	return next, added, nil
}

func questionKey(contract, question string) string {
	return strings.TrimSpace(strings.ToLower(contract)) + "|" + strings.TrimSpace(strings.ToLower(question))
}

func coveragePriority(state docengine.CoverageState) string {
	switch state {
	case docengine.Missing, docengine.Stale, docengine.Partial:
		return "blocker"
	default:
		return "high-risk"
	}
}

func coverageReason(coverage docengine.Coverage) string {
	return fmt.Sprintf("coverage state %s for %s", coverage.State, coverage.ContractID)
}
