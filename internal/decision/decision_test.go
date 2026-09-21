package decision

import (
	"context"
	"testing"
)

func TestRuleDecisionProvider_MatchAndAbstain(t *testing.T) {
	rule1 := Rule{
		QuestionID: "q-routing",
		Condition: func(state map[string]any) bool {
			return state["task_type"] == "trivial_read"
		},
		Outcome:    "ROUTE-CHEAP",
		Confidence: 0.95,
		ReasonCode: "REASON-MATCHED-TRIVIAL-TASK",
	}
	rule2 := Rule{
		QuestionID: "q-routing",
		Condition: func(state map[string]any) bool {
			return state["task_type"] == "uncertain_task"
		},
		Outcome:    "ROUTE-FALLBACK",
		Confidence: 0.50, // below default 0.70 threshold
		ReasonCode: "REASON-LOW-CONFIDENCE-FALLBACK",
	}

	provider := NewRuleDecisionProvider(rule1, rule2)

	ctx := context.Background()

	// 1. Successful high-confidence match
	req1 := DecisionRequest{
		State: map[string]any{"task_type": "trivial_read"},
		Questions: []DecisionQuestion{
			{ID: "q-routing", Kind: KindChoice, Choices: []string{"ROUTE-CHEAP", "ROUTE-PRIMARY", "ROUTE-FALLBACK"}},
		},
		ConfidenceThreshold: 0.70,
	}
	res1, err := provider.Decide(ctx, req1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res1.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(res1.Answers))
	}
	if res1.Answers[0].Abstained {
		t.Fatalf("expected unabstained answer")
	}
	if res1.Answers[0].Choice != "ROUTE-CHEAP" {
		t.Fatalf("expected ROUTE-CHEAP, got %q", res1.Answers[0].Choice)
	}

	// 2. Below confidence threshold -> abstention
	req2 := DecisionRequest{
		State: map[string]any{"task_type": "uncertain_task"},
		Questions: []DecisionQuestion{
			{ID: "q-routing", Kind: KindChoice},
		},
		ConfidenceThreshold: 0.70,
	}
	res2, err := provider.Decide(ctx, req2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res2.Answers[0].Abstained {
		t.Fatalf("expected abstention for confidence below threshold")
	}
	if res2.Answers[0].ReasonCode != "REASON-DECISION-BELOW-THRESHOLD" {
		t.Fatalf("expected REASON-DECISION-BELOW-THRESHOLD, got %q", res2.Answers[0].ReasonCode)
	}

	// 3. No match -> abstention
	req3 := DecisionRequest{
		State: map[string]any{"task_type": "completely_unknown"},
		Questions: []DecisionQuestion{
			{ID: "q-routing", Kind: KindChoice},
		},
	}
	res3, err := provider.Decide(ctx, req3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res3.Answers[0].Abstained {
		t.Fatalf("expected abstention for no match")
	}
	if res3.Answers[0].ReasonCode != "REASON-DECISION-NO-MATCH" {
		t.Fatalf("expected REASON-DECISION-NO-MATCH, got %q", res3.Answers[0].ReasonCode)
	}
}

func TestDecisionEngine_Escalation(t *testing.T) {
	// Provider with no matching rules
	emptyProvider := NewRuleDecisionProvider()
	engine := NewDecisionEngine(emptyProvider)

	ctx := context.Background()
	req := DecisionRequest{
		State: map[string]any{"key": "val"},
		Questions: []DecisionQuestion{
			{ID: "q-critical", Kind: KindBoolean},
		},
	}

	res, err := engine.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(res.Answers))
	}
	if !res.Answers[0].Abstained {
		t.Fatalf("expected escalation abstention")
	}
	if res.Answers[0].ReasonCode != "REASON-DECISION-ESCALATE" && res.Answers[0].ReasonCode != "REASON-DECISION-NO-MATCH" {
		t.Fatalf("expected escalation or no-match reason code, got %q", res.Answers[0].ReasonCode)
	}
}
