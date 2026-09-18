package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/adoption"
	"github.com/raillen/prumo/internal/protocol"
)

func runAdopt(asJSON bool, args []string) int {
	path := "."
	strict := false
	auditOnly := false
	interactive := false
	nonInteractive := false
	proposeMigration := false
	dryRun := false
	apply := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "error: --path requires a directory path\n")
				return exitUsage
			}
		case "--audit-only":
			auditOnly = true
		case "--strict":
			strict = true
		case "--interactive":
			interactive = true
		case "--non-interactive":
			nonInteractive = true
		case "--propose-migration":
			proposeMigration = true
		case "--dry-run":
			dryRun = true
		case "--apply":
			apply = true
		default:
			if !strings.HasPrefix(args[i], "-") && path == "." {
				path = args[i]
			}
		}
	}

	_ = auditOnly

	opts := adoption.ScanOptions{
		Budget:         adoption.DefaultBudget(),
		ParseManifests: true,
	}

	report, err := adoption.RunAdoptionAudit(path, opts)
	if err != nil {
		if asJSON {
			return envelopeError("adoption_error", err.Error())
		}
		fmt.Fprintf(os.Stderr, "error running adoption audit: %s\n", err)
		return exitInternal
	}

	// F-G07: Handle uncertainty resolution
	if nonInteractive {
		session := adoption.ResolveNonInteractive(&report.Ledger)
		if !asJSON && session.ResolvedCount > 0 {
			fmt.Printf("Non-interactive pass: resolved %d inference(s) (%d unconfirmed remain)\n\n",
				session.ResolvedCount, session.UnresolvedCount)
		}
	} else if interactive {
		questions := adoption.GenerateQuestions(report.Ledger)
		if len(questions) > 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for _, q := range questions {
				if !asJSON {
					fmt.Printf("\n[Adoption Interview] %s\n", q.Question)
					if q.Context != "" {
						fmt.Printf("  Context: %s\n", q.Context)
					}
					fmt.Printf("  Options: [%s] (default: %s)\n", strings.Join(q.Options, ", "), q.DefaultChoice)
					fmt.Print("  Choice > ")
				}

				var choiceStr string
				if scanner.Scan() {
					choiceStr = strings.TrimSpace(scanner.Text())
				}
				if choiceStr == "" {
					choiceStr = q.DefaultChoice
				}

				choice := adoption.ResolutionChoice{
					QuestionID:     q.ID,
					SelectedOption: choiceStr,
					Actor:          "human-operator",
					Rationale:      "Confirmed via interactive adoption interview",
				}
				_ = adoption.ApplyChoice(&report.Ledger, q, choice)
			}
		}
	}

	if strict {
		if len(report.Contradictions) > 0 || report.Ledger.Summary.RequiresConfirmation > 0 {
			msg := fmt.Sprintf("strict adoption check failed: %d contradiction(s), %d unconfirmed inference(s)",
				len(report.Contradictions), report.Ledger.Summary.RequiresConfirmation)
			if asJSON {
				return printEnvelope(protocol.ErrEnvelope(protocol.Diagnostic{
					Code:    "adoption_strict_failure",
					Message: msg,
				}))
			}
			fmt.Fprintf(os.Stderr, "error: %s\n", msg)
			return exitValidation
		}
	}

	if dryRun {
		proposals := adoption.GenerateMigrationProposals(report)
		type DryRunSummary struct {
			Proposals []adoption.AdoptionMigrationProposal `json:"proposals"`
			Results   []adoption.DryRunResult              `json:"results"`
		}
		var results []adoption.DryRunResult
		for _, p := range proposals {
			res, err := adoption.DryRun(path, p)
			if err != nil {
				if asJSON {
					return envelopeError("dry_run_error", err.Error())
				}
				fmt.Fprintf(os.Stderr, "dry run error for proposal %s: %s\n", p.ID, err)
				return exitInternal
			}
			results = append(results, res)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(DryRunSummary{
				Proposals: proposals,
				Results:   results,
			}))
		}
		fmt.Printf("=== ADOPTION MIGRATION DRY-RUN (%d proposal(s)) ===\n\n", len(proposals))
		for _, res := range results {
			fmt.Printf("Proposal %s (preconditions passed: %v)\n", res.ProposalID, res.PreconditionsPassed)
			for _, chg := range res.Plan {
				fmt.Printf("  Action: %s %s (exists: %v)\n", chg.Action, chg.TargetPath, chg.Exists)
				fmt.Printf("  Diff:\n%s\n", chg.DiffPreview)
			}
			if len(res.Violations) > 0 {
				fmt.Printf("  Violations: %s\n", strings.Join(res.Violations, ", "))
			}
			fmt.Println()
		}
		return exitOK
	}

	if apply {
		proposals := adoption.GenerateMigrationProposals(report)
		type ApplySummary struct {
			Applied []adoption.ApplyResult `json:"applied"`
		}
		var applied []adoption.ApplyResult
		for _, p := range proposals {
			p.Status = adoption.ProposalStatusApproved
			p.ApprovedBy = "operator"
			p.ApprovedAt = "now"
			res, err := adoption.Apply(path, p)
			if err != nil {
				if asJSON {
					return envelopeError("apply_error", err.Error())
				}
				fmt.Fprintf(os.Stderr, "error applying proposal %s: %s\n", p.ID, err)
				return exitInternal
			}
			applied = append(applied, res)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(ApplySummary{Applied: applied}))
		}
		fmt.Printf("=== ADOPTION MIGRATION APPLIED (%d proposal(s)) ===\n\n", len(applied))
		for _, app := range applied {
			fmt.Printf("✓ Proposal %s applied successfully (Journal hash: %s)\n", app.ProposalID, app.JournalEntry.Hash)
		}
		return exitOK
	}

	if proposeMigration {
		proposals := adoption.GenerateMigrationProposals(report)
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(proposals))
		}
		fmt.Printf("=== ADOPTION MIGRATION PROPOSALS (%d) ===\n\n", len(proposals))
		for _, p := range proposals {
			fmt.Printf("• [%s] %s\n", p.ID, p.Title)
			fmt.Printf("  Description: %s\n", p.Description)
			fmt.Printf("  Contract ID: %s (Reversible: %v)\n", p.Contract.ID, p.Contract.Reversible)
			for _, act := range p.Actions {
				fmt.Printf("    - %s -> %s\n", act.Type, act.TargetPath)
			}
			fmt.Println()
		}
		return exitOK
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(report))
	}

	fmt.Print(adoption.RenderHumanReport(report))
	return exitOK
}
