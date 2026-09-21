// Package decision implements the Decision Intelligence Runtime (L1) per
// Constitución 83 (docs/runtime/decision-intelligence-runtime.md).
//
// Separates decision intelligence (closed answer spaces, classification, scoring,
// deterministic rules, heuristics) from generative intelligence (L2 LLMs) and
// human authority (L3).
//
// Core principle: "Hard-code invariants; configure policies; evaluate heuristics."
// Deterministic -> Decision Model -> Generative Model -> Human Authority.
package decision

import (
	"context"
	"fmt"
	"time"
)

// QuestionKind categorizes the decision type.
type QuestionKind string

const (
	KindChoice      QuestionKind = "choice"
	KindScore       QuestionKind = "score"
	KindProbability QuestionKind = "probability"
	KindBoolean     QuestionKind = "boolean"
)

// DecisionQuestion represents a single closed-space question to be decided.
type DecisionQuestion struct {
	ID          string       `json:"id"`
	Kind        QuestionKind `json:"kind"`
	Prompt      string       `json:"prompt"`
	Choices     []string     `json:"choices,omitempty"`
	MinScore    float64      `json:"min_score,omitempty"`
	MaxScore    float64      `json:"max_score,omitempty"`
}

// DecisionRequest bundles the state, questions, constraints, and confidence thresholds.
type DecisionRequest struct {
	ContextID           string             `json:"context_id,omitempty"`
	State               map[string]any     `json:"state"`
	Questions           []DecisionQuestion `json:"questions"`
	Constraints         map[string]any     `json:"constraints,omitempty"`
	ConfidenceThreshold float64            `json:"confidence_threshold"` // default 0.70
}

// DecisionAnswer holds the outcome for a single question.
type DecisionAnswer struct {
	QuestionID    string  `json:"question_id"`
	Choice        string  `json:"choice,omitempty"`
	Score         float64 `json:"score,omitempty"`
	Probability   float64 `json:"probability,omitempty"`
	Confidence    float64 `json:"confidence"`
	ReasonCode    string  `json:"reason_code"`
	Abstained     bool    `json:"abstained"`
	AbstainReason string  `json:"abstain_reason,omitempty"`
}

// DecisionResult aggregates answers from a DecisionProvider.
type DecisionResult struct {
	Answers       []DecisionAnswer  `json:"answers"`
	Provider      string            `json:"provider"`
	ModelRevision string            `json:"model_revision,omitempty"`
	LatencyMs     int64             `json:"latency_ms"`
	CostUSD       float64           `json:"cost_usd"`
	TraceMetadata map[string]any    `json:"trace_metadata,omitempty"`
}

// DecisionProvider defines the canonical interface for decision-making components.
type DecisionProvider interface {
	Name() string
	Capabilities() []string
	Decide(ctx context.Context, req DecisionRequest) (DecisionResult, error)
}

// Rule evaluates a condition against state and produces an answer.
type Rule struct {
	QuestionID  string
	Condition   func(state map[string]any) bool
	Outcome     string
	Confidence  float64
	ReasonCode  string
}

// RuleDecisionProvider is the deterministic L0 provider evaluating explicit rules.
type RuleDecisionProvider struct {
	rules []Rule
}

// NewRuleDecisionProvider creates a RuleDecisionProvider with a set of rules.
func NewRuleDecisionProvider(rules ...Rule) *RuleDecisionProvider {
	return &RuleDecisionProvider{rules: rules}
}

func (p *RuleDecisionProvider) Name() string {
	return "rule-provider-l0"
}

func (p *RuleDecisionProvider) Capabilities() []string {
	return []string{"choice", "boolean", "deterministic"}
}

func (p *RuleDecisionProvider) Decide(ctx context.Context, req DecisionRequest) (DecisionResult, error) {
	start := time.Now()
	threshold := req.ConfidenceThreshold
	if threshold <= 0 {
		threshold = 0.70
	}

	answers := make([]DecisionAnswer, 0, len(req.Questions))

	for _, q := range req.Questions {
		matched := false
		for _, r := range p.rules {
			if r.QuestionID == q.ID && r.Condition != nil && r.Condition(req.State) {
				matched = true
				if r.Confidence < threshold {
					// Abstain due to low confidence
					answers = append(answers, DecisionAnswer{
						QuestionID:    q.ID,
						Confidence:    r.Confidence,
						Abstained:     true,
						AbstainReason: fmt.Sprintf("confidence %.2f below threshold %.2f", r.Confidence, threshold),
						ReasonCode:    "REASON-DECISION-BELOW-THRESHOLD",
					})
				} else {
					answers = append(answers, DecisionAnswer{
						QuestionID: q.ID,
						Choice:     r.Outcome,
						Confidence: r.Confidence,
						ReasonCode: r.ReasonCode,
						Abstained:  false,
					})
				}
				break
			}
		}

		if !matched {
			// Abstain because no rule matched
			answers = append(answers, DecisionAnswer{
				QuestionID:    q.ID,
				Confidence:    0.0,
				Abstained:     true,
				AbstainReason: "no deterministic rule matched current state",
				ReasonCode:    "REASON-DECISION-NO-MATCH",
			})
		}
	}

	return DecisionResult{
		Answers:       answers,
		Provider:      p.Name(),
		LatencyMs:     time.Since(start).Milliseconds(),
		CostUSD:       0.0,
		TraceMetadata: map[string]any{"rules_evaluated": len(p.rules)},
	}, nil
}

// DecisionEngine coordinates deterministic rules, decision providers, and fallback escalation.
type DecisionEngine struct {
	providers []DecisionProvider
}

// NewDecisionEngine constructs an engine with an ordered chain of providers.
func NewDecisionEngine(providers ...DecisionProvider) *DecisionEngine {
	return &DecisionEngine{providers: providers}
}

// Evaluate runs the decision pipeline. For each question, it evaluates providers
// in priority order until an unabstained answer meeting the confidence threshold is found,
// or returns an explicit escalation indicator.
func (e *DecisionEngine) Evaluate(ctx context.Context, req DecisionRequest) (DecisionResult, error) {
	start := time.Now()
	if len(e.providers) == 0 {
		return DecisionResult{}, fmt.Errorf("decision engine has no registered providers")
	}

	finalAnswers := make([]DecisionAnswer, len(req.Questions))
	answered := make([]bool, len(req.Questions))
	var lastProviderName string

	for _, p := range e.providers {
		lastProviderName = p.Name()
		// Only ask questions not yet answered
		subQuestions := make([]DecisionQuestion, 0)
		idxMap := make([]int, 0)
		for i, q := range req.Questions {
			if !answered[i] {
				subQuestions = append(subQuestions, q)
				idxMap = append(idxMap, i)
			}
		}
		if len(subQuestions) == 0 {
			break
		}

		subReq := req
		subReq.Questions = subQuestions
		res, err := p.Decide(ctx, subReq)
		if err != nil {
			continue // try next provider on provider error
		}

		for subIdx, ans := range res.Answers {
			origIdx := idxMap[subIdx]
			if !ans.Abstained {
				finalAnswers[origIdx] = ans
				answered[origIdx] = true
			} else if finalAnswers[origIdx].QuestionID == "" {
				// Record the abstention if we don't have an answer yet
				finalAnswers[origIdx] = ans
			}
		}
	}

	// For any question still unanswered or abstained, mark escalation to L2/L3
	for i, q := range req.Questions {
		if !answered[i] {
			if finalAnswers[i].QuestionID == "" {
				finalAnswers[i] = DecisionAnswer{
					QuestionID:    q.ID,
					Confidence:    0.0,
					Abstained:     true,
					AbstainReason: "all decision providers abstained; escalate to L2 reasoning or L3 human",
					ReasonCode:    "REASON-DECISION-ESCALATE",
				}
			}
		}
	}

	return DecisionResult{
		Answers:   finalAnswers,
		Provider:  lastProviderName,
		LatencyMs: time.Since(start).Milliseconds(),
		CostUSD:   0.0,
		TraceMetadata: map[string]any{
			"total_providers": len(e.providers),
		},
	}, nil
}
