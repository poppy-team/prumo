// Package experience implements session, project, and cross-project global learning.
//
// In accordance with Constitución 81 (docs/runtime/global-learning-layer.md):
// "Project experience teaches. Global experience generalizes. Canonical project state still wins."
//
// Global patterns are reusable hypotheses with provenance and confidence.
// They never override canonical specifications, ADRs, or explicit project requirements.
package experience

import (
	"fmt"
	"strings"
	"time"
)

// PatternType classifies the nature of the learned pattern.
type PatternType string

const (
	PatternPreference         PatternType = "preference"
	PatternPhilosophy         PatternType = "philosophy"
	PatternHeuristic          PatternType = "heuristic"
	PatternWorkflow           PatternType = "workflow"
	PatternArchitecture       PatternType = "architecture-pattern"
	PatternAntiPattern        PatternType = "anti-pattern"
	PatternQualityPractice    PatternType = "quality-practice"
	PatternTechnologyAffinity PatternType = "technology-affinity"
	PatternCandidateSkill     PatternType = "candidate-skill"
	PatternCandidateRecipe    PatternType = "candidate-recipe"
	PatternCandidatePolicy    PatternType = "candidate-policy"
)

// PatternScope defines the boundary of applicability.
type PatternScope string

const (
	ScopeSession   PatternScope = "session"
	ScopeProject   PatternScope = "project"
	ScopeWorkspace PatternScope = "workspace"
	ScopeGlobal    PatternScope = "global"
)

// ReviewStatus tracks the review lifecycle of a candidate pattern.
type ReviewStatus string

const (
	ReviewProposed ReviewStatus = "proposed"
	ReviewReviewed ReviewStatus = "reviewed"
	ReviewAccepted ReviewStatus = "accepted"
	ReviewRejected ReviewStatus = "rejected"
	ReviewPromoted ReviewStatus = "promoted"
)

// GlobalPattern represents an aggregated cross-project or project-level pattern.
type GlobalPattern struct {
	ID             string       `json:"id"`
	Type           PatternType  `json:"type"`
	Scope          PatternScope `json:"scope"`
	Title          string       `json:"title"`
	Description    string       `json:"description"`
	Confidence     float64      `json:"confidence"`
	ProjectCount   int          `json:"project_count"`
	Occurrences    int          `json:"occurrences"`
	Sources        []string     `json:"sources"` // Project IDs or Session IDs
	ReviewStatus   ReviewStatus `json:"review_status"`
	PromotionTarget string      `json:"promotion_target,omitempty"` // e.g. "skill:code-audit", "recipe:ci-setup"
	CreatedAt      string       `json:"created_at"`
	UpdatedAt      string       `json:"updated_at"`
}

// GlobalLearningRegistry manages observations and aggregates candidate patterns.
type GlobalLearningRegistry struct {
	patterns map[string]*GlobalPattern
}

// NewGlobalLearningRegistry constructs a new in-memory registry.
func NewGlobalLearningRegistry() *GlobalLearningRegistry {
	return &GlobalLearningRegistry{
		patterns: make(map[string]*GlobalPattern),
	}
}

// RecordObservation incorporates a new observation into candidate patterns.
// If a pattern with the given title and type exists, occurrences and confidence are updated.
func (r *GlobalLearningRegistry) RecordObservation(projectID string, pType PatternType, title, desc string, initialConfidence float64) *GlobalPattern {
	key := fmt.Sprintf("%s:%s", pType, strings.ToLower(strings.TrimSpace(title)))
	now := time.Now().UTC().Format(time.RFC3339)

	pat, exists := r.patterns[key]
	if !exists {
		pat = &GlobalPattern{
			ID:           fmt.Sprintf("pat-%d", len(r.patterns)+1),
			Type:         pType,
			Scope:        ScopeProject,
			Title:        title,
			Description:  desc,
			Confidence:   initialConfidence,
			ProjectCount: 1,
			Occurrences:  1,
			Sources:      []string{projectID},
			ReviewStatus: ReviewProposed,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		r.patterns[key] = pat
		return pat
	}

	pat.Occurrences++
	pat.UpdatedAt = now

	// Check if this is a distinct project
	foundProj := false
	for _, s := range pat.Sources {
		if s == projectID {
			foundProj = true
			break
		}
	}
	if !foundProj {
		pat.Sources = append(pat.Sources, projectID)
		pat.ProjectCount++
		// If observed across multiple projects, escalate scope to global
		if pat.ProjectCount >= 2 {
			pat.Scope = ScopeGlobal
		}
	}

	// Bayesian-like confidence reinforcement bounded at 0.99
	pat.Confidence = pat.Confidence + (1.0-pat.Confidence)*0.2
	if pat.Confidence > 0.99 {
		pat.Confidence = 0.99
	}

	return pat
}

// ReviewCandidate transitions a pattern through human or formal review.
func (r *GlobalLearningRegistry) ReviewCandidate(patternID string, status ReviewStatus, promotionTarget string) error {
	for _, pat := range r.patterns {
		if pat.ID == patternID {
			pat.ReviewStatus = status
			if promotionTarget != "" {
				pat.PromotionTarget = promotionTarget
			}
			pat.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return nil
		}
	}
	return fmt.Errorf("pattern with id %q not found", patternID)
}

// ListPatterns returns all patterns, optionally filtered by scope and review status.
func (r *GlobalLearningRegistry) ListPatterns(scope PatternScope, status ReviewStatus) []*GlobalPattern {
	res := make([]*GlobalPattern, 0)
	for _, p := range r.patterns {
		if scope != "" && p.Scope != scope {
			continue
		}
		if status != "" && p.ReviewStatus != status {
			continue
		}
		res = append(res, p)
	}
	return res
}
