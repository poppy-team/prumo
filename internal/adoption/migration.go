package adoption

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/migrations"

	"github.com/raillen/prumo/internal/harness/safepath"
)

// MigrationActionType represents the kind of file operation in an adoption migration proposal.
type MigrationActionType string

const (
	ActionCreateFile    MigrationActionType = "create_file"
	ActionUpdateFile    MigrationActionType = "update_file"
	ActionRecordBinding MigrationActionType = "record_binding"
)

// MigrationAction models an individual non-destructive action within a migration proposal.
type MigrationAction struct {
	Type        MigrationActionType `json:"type"`
	TargetPath  string              `json:"target_path"`
	Content     string              `json:"content"`
	Description string              `json:"description"`
	Reversible  bool                `json:"reversible"`
}

// AdoptionProposalStatus models the review and lifecycle state of a migration proposal.
type AdoptionProposalStatus string

const (
	ProposalStatusPendingReview AdoptionProposalStatus = "pending-review"
	ProposalStatusApproved      AdoptionProposalStatus = "approved"
	ProposalStatusRejected      AdoptionProposalStatus = "rejected"
	ProposalStatusApplied       AdoptionProposalStatus = "applied"
)

// AdoptionMigrationProposal is a formal, non-destructive proposal produced by the Adoption Engine.
// In accordance with governance and safety invariants, no mutations occur without review approval.
type AdoptionMigrationProposal struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	SourceFinding  string                 `json:"source_finding"`
	Status         AdoptionProposalStatus `json:"status"`
	ReviewRequired bool                   `json:"review_required"`
	ReviewNotes    string                 `json:"review_notes,omitempty"`
	ApprovedBy     string                 `json:"approved_by,omitempty"`
	ApprovedAt     string                 `json:"approved_at,omitempty"`
	Actions        []MigrationAction      `json:"actions"`
	Contract       migrations.Contract    `json:"contract"`
	CreatedAt      string                 `json:"created_at"`
}

// ReviewQueueItem encapsulates a migration proposal queued for human or governance review.
type ReviewQueueItem struct {
	ID          string                    `json:"id"`
	ItemType    string                    `json:"item_type"`
	Title       string                    `json:"title"`
	Status      AdoptionProposalStatus    `json:"status"`
	Proposal    AdoptionMigrationProposal `json:"proposal"`
	SubmittedAt string                    `json:"submitted_at"`
}

// ReviewQueue holds pending and reviewed migration proposals.
type ReviewQueue struct {
	Items []ReviewQueueItem `json:"items"`
}

// NewReviewQueue initializes an empty ReviewQueue.
func NewReviewQueue() *ReviewQueue {
	return &ReviewQueue{
		Items: make([]ReviewQueueItem, 0),
	}
}

// Submit enqueues a proposal into the ReviewQueue.
func (q *ReviewQueue) Submit(proposal AdoptionMigrationProposal) *ReviewQueueItem {
	item := ReviewQueueItem{
		ID:          "rq-" + proposal.ID,
		ItemType:    "adoption_migration_proposal",
		Title:       proposal.Title,
		Status:      ProposalStatusPendingReview,
		Proposal:    proposal,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
	}
	q.Items = append(q.Items, item)
	return &item
}

// Approve records an approval for the queued proposal.
func (q *ReviewQueue) Approve(itemID string, actor string, notes string) error {
	for i := range q.Items {
		if q.Items[i].ID == itemID || q.Items[i].Proposal.ID == itemID {
			q.Items[i].Status = ProposalStatusApproved
			q.Items[i].Proposal.Status = ProposalStatusApproved
			q.Items[i].Proposal.ApprovedBy = actor
			q.Items[i].Proposal.ApprovedAt = time.Now().UTC().Format(time.RFC3339)
			q.Items[i].Proposal.ReviewNotes = notes
			return nil
		}
	}
	return fmt.Errorf("review item %q not found", itemID)
}

// Reject records a rejection for the queued proposal.
func (q *ReviewQueue) Reject(itemID string, actor string, notes string) error {
	for i := range q.Items {
		if q.Items[i].ID == itemID || q.Items[i].Proposal.ID == itemID {
			q.Items[i].Status = ProposalStatusRejected
			q.Items[i].Proposal.Status = ProposalStatusRejected
			q.Items[i].Proposal.ApprovedBy = actor
			q.Items[i].Proposal.ReviewNotes = notes
			return nil
		}
	}
	return fmt.Errorf("review item %q not found", itemID)
}

// ListPending returns all items awaiting review.
func (q *ReviewQueue) ListPending() []ReviewQueueItem {
	var pending []ReviewQueueItem
	for _, item := range q.Items {
		if item.Status == ProposalStatusPendingReview {
			pending = append(pending, item)
		}
	}
	return pending
}

// GenerateMigrationProposals creates formal migration proposals from an AdoptionReport.
// Non-destructive: proposals declare actions, affected artifacts, and preconditions without mutating disk.
func GenerateMigrationProposals(report AdoptionReport) []AdoptionMigrationProposal {
	var proposals []AdoptionMigrationProposal
	now := time.Now().UTC().Format(time.RFC3339)

	// Check if repository already has prumo.json
	hasPrumoJSON := false
	for _, art := range report.PrumoArtifacts {
		if strings.Contains(art, "prumo.json") {
			hasPrumoJSON = true
			break
		}
	}

	if !hasPrumoJSON {
		profileName := "standard"
		if len(report.Profiles) > 0 {
			profileName = report.Profiles[0].Profile
		}

		capList := make([]string, 0, len(report.Capabilities))
		for _, c := range report.Capabilities {
			capList = append(capList, c.Capability)
		}

		manifestData := map[string]interface{}{
			"version":      1,
			"profile":      profileName,
			"app_types":    report.Classification.AppTypes,
			"languages":    report.Classification.Languages,
			"frameworks":   report.Classification.Frameworks,
			"capabilities": capList,
		}
		contentBytes, _ := json.MarshalIndent(manifestData, "", "  ")

		p := AdoptionMigrationProposal{
			ID:             "amp-init-prumo",
			Title:          "Initialize Prumo manifest (prumo.json)",
			Description:    fmt.Sprintf("Creates prumo.json with detected profile %q and %d capabilities.", profileName, len(capList)),
			SourceFinding:  "classification.primary_profile",
			Status:         ProposalStatusPendingReview,
			ReviewRequired: true,
			Actions: []MigrationAction{
				{
					Type:        ActionCreateFile,
					TargetPath:  "prumo.json",
					Content:     string(contentBytes) + "\n",
					Description: "Create root prumo.json manifest with detected repository classification.",
					Reversible:  true,
				},
			},
			Contract: migrations.Contract{
				ID:                "contract-amp-init-prumo",
				FromVersion:       "0.0.0",
				ToVersion:         "0.5.0",
				Preconditions:     []string{"manifest_absent:prumo.json"},
				BackupStrategy:    "none",
				AffectedArtifacts: []string{"prumo.json"},
				Reversible:        true,
				Rollback:          "rm prumo.json",
			},
			CreatedAt: now,
		}
		proposals = append(proposals, p)
	}

	// Doc bindings proposal
	if len(report.DocBindings) > 0 {
		bindingsData := map[string]interface{}{
			"version":  1,
			"bindings": report.DocBindings,
		}
		bBytes, _ := json.MarshalIndent(bindingsData, "", "  ")

		p := AdoptionMigrationProposal{
			ID:             "amp-doc-bindings",
			Title:          "Record semantic documentation bindings",
			Description:    fmt.Sprintf("Records %d candidate documentation contract bindings to docs/contracts/bindings.json.", len(report.DocBindings)),
			SourceFinding:  "doc_bindings.candidates",
			Status:         ProposalStatusPendingReview,
			ReviewRequired: true,
			Actions: []MigrationAction{
				{
					Type:        ActionRecordBinding,
					TargetPath:  filepath.Join("docs", "contracts", "bindings.json"),
					Content:     string(bBytes) + "\n",
					Description: "Persist documentation bindings for M5 documentation contract enforcement.",
					Reversible:  true,
				},
			},
			Contract: migrations.Contract{
				ID:                "contract-amp-doc-bindings",
				FromVersion:       "0.0.0",
				ToVersion:         "0.5.0",
				Preconditions:     []string{"documentation_directory_exists"},
				BackupStrategy:    "backup_file",
				AffectedArtifacts: []string{"docs/contracts/bindings.json"},
				Reversible:        true,
				Rollback:          "restore_or_remove:docs/contracts/bindings.json",
			},
			CreatedAt: now,
		}
		proposals = append(proposals, p)
	}

	return proposals
}

// PlannedChange describes the anticipated effect of a migration action during dry-run.
type PlannedChange struct {
	Action      MigrationActionType `json:"action"`
	TargetPath  string              `json:"target_path"`
	Exists      bool                `json:"exists"`
	DiffPreview string              `json:"diff_preview"`
}

// DryRunResult reports the evaluation of preconditions and anticipated changes without disk mutations.
type DryRunResult struct {
	ProposalID          string          `json:"proposal_id"`
	PreconditionsPassed bool            `json:"preconditions_passed"`
	Violations          []string        `json:"violations,omitempty"`
	Plan                []PlannedChange `json:"plan"`
}

// DryRun performs validation of preconditions and generates diff previews without modifying files.
func DryRun(repoRoot string, proposal AdoptionMigrationProposal) (DryRunResult, error) {
	result := DryRunResult{
		ProposalID:          proposal.ID,
		PreconditionsPassed: true,
		Plan:                make([]PlannedChange, 0, len(proposal.Actions)),
	}

	for _, cond := range proposal.Contract.Preconditions {
		if strings.HasPrefix(cond, "manifest_absent:") {
			rel := strings.TrimPrefix(cond, "manifest_absent:")
			target, pathErr := safepath.Resolve(repoRoot, rel, false)
			if pathErr != nil {
				result.PreconditionsPassed = false
				result.Violations = append(result.Violations, pathErr.Error())
				continue
			}
			if _, err := os.Stat(target); err == nil {
				result.PreconditionsPassed = false
				result.Violations = append(result.Violations, fmt.Sprintf("precondition failed: %s already exists", rel))
			}
		}
	}

	for _, action := range proposal.Actions {
		// A proposal is a document that can arrive from outside the process, so
		// its target paths are untrusted input. Joining them onto the repo root
		// without a containment test let "../.." write anywhere the process could
		// reach (GAP-110).
		fullPath, pathErr := safepath.Resolve(repoRoot, action.TargetPath, false)
		if pathErr != nil {
			result.PreconditionsPassed = false
			result.Violations = append(result.Violations, pathErr.Error())
			continue
		}
		exists := false
		var diffPreview string

		if existingBytes, err := os.ReadFile(fullPath); err == nil {
			exists = true
			diffPreview = fmt.Sprintf("--- %s (existing, %d bytes)\n+++ %s (proposed, %d bytes)",
				action.TargetPath, len(existingBytes), action.TargetPath, len(action.Content))
		} else {
			diffPreview = fmt.Sprintf("+++ %s (new file, %d bytes)\n%s",
				action.TargetPath, len(action.Content), action.Content)
		}

		result.Plan = append(result.Plan, PlannedChange{
			Action:      action.Type,
			TargetPath:  action.TargetPath,
			Exists:      exists,
			DiffPreview: diffPreview,
		})
	}

	return result, nil
}

// ApplyResult captures the outcome of applying an approved migration proposal.
type ApplyResult struct {
	ProposalID   string                  `json:"proposal_id"`
	Success      bool                    `json:"success"`
	JournalEntry migrations.JournalEntry `json:"journal_entry"`
	Errors       []string                `json:"errors,omitempty"`
}

// Apply executes an approved migration proposal against the target repository root.
// Invariant: Unapproved proposals cannot be applied.
func Apply(repoRoot string, proposal AdoptionMigrationProposal) (ApplyResult, error) {
	if proposal.Status != ProposalStatusApproved {
		return ApplyResult{
			ProposalID: proposal.ID,
			Success:    false,
			Errors:     []string{fmt.Sprintf("proposal %s is %s; requires Review Queue approval before applying", proposal.ID, proposal.Status)},
		}, fmt.Errorf("cannot apply unapproved proposal: status is %q", proposal.Status)
	}

	dryRun, err := DryRun(repoRoot, proposal)
	if err != nil {
		return ApplyResult{ProposalID: proposal.ID, Success: false, Errors: []string{err.Error()}}, err
	}
	if !dryRun.PreconditionsPassed {
		return ApplyResult{ProposalID: proposal.ID, Success: false, Errors: dryRun.Violations},
			fmt.Errorf("preconditions not satisfied: %s", strings.Join(dryRun.Violations, "; "))
	}

	hasher := sha256.New()
	for _, action := range proposal.Actions {
		targetFile, pathErr := safepath.Resolve(repoRoot, action.TargetPath, false)
		if pathErr != nil {
			return ApplyResult{
				ProposalID: proposal.ID,
				Success:    false,
				Errors:     []string{pathErr.Error()},
			}, pathErr
		}
		dir := filepath.Dir(targetFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return ApplyResult{
				ProposalID: proposal.ID,
				Success:    false,
				Errors:     []string{fmt.Sprintf("failed creating directory %s: %s", dir, err)},
			}, err
		}

		// Backup if needed
		if proposal.Contract.BackupStrategy == "backup_file" {
			if _, err := os.Stat(targetFile); err == nil {
				_ = os.Rename(targetFile, targetFile+".bak")
			}
		}

		if err := os.WriteFile(targetFile, []byte(action.Content), 0644); err != nil {
			return ApplyResult{
				ProposalID: proposal.ID,
				Success:    false,
				Errors:     []string{fmt.Sprintf("failed writing %s: %s", action.TargetPath, err)},
			}, err
		}

		hasher.Write([]byte(action.TargetPath))
		hasher.Write([]byte(action.Content))
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))
	journal := migrations.JournalEntry{
		MigrationID: proposal.Contract.ID,
		AppliedAt:   time.Now().UTC().Format(time.RFC3339),
		Result:      "success",
		Hash:        hashStr,
	}

	return ApplyResult{
		ProposalID:   proposal.ID,
		Success:      true,
		JournalEntry: journal,
	}, nil
}
