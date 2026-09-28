package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/adoption"
	"github.com/raillen/prumo/internal/cliops"
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
	subcommand := ""

	// Check for leading positional subcommand (scan, facts, classify, scaffold, apply)
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		candidate := strings.ToLower(args[0])
		switch candidate {
		case "scan", "facts", "classify", "scaffold", "apply":
			subcommand = candidate
			args = args[1:]
		}
	}

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

	if subcommand == "apply" {
		apply = true
	} else if subcommand == "scaffold" {
		dryRun = true
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

	// Handle specific subcommands
	if subcommand == "scan" {
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{
				"root":            report.Repository.Root,
				"files_scanned":   report.FactsSummary.FilesScanned,
				"total_facts":     report.FactsSummary.TotalFacts,
				"prumo_artifacts": report.PrumoArtifacts,
			}))
		}
		fmt.Printf("=== ADOPTION SCAN: %s ===\n", report.Repository.Root)
		fmt.Printf("Files scanned: %d | Total facts: %d\n\n",
			report.FactsSummary.FilesScanned, report.FactsSummary.TotalFacts)
		if len(report.PrumoArtifacts) > 0 {
			fmt.Println("Existing Prumo Artifacts:")
			for _, art := range report.PrumoArtifacts {
				fmt.Printf("  • %s\n", art)
			}
		} else {
			fmt.Println("No existing Prumo artifacts detected (fresh adoption target).")
		}
		return exitOK
	}

	if subcommand == "facts" {
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{
				"summary": report.Ledger.Summary,
				"entries": report.Ledger.Entries,
			}))
		}
		fmt.Printf("=== OBSERVED TECHNICAL FACTS (%d entries) ===\n\n", len(report.Ledger.Entries))
		for _, entry := range report.Ledger.Entries {
			conf := string(entry.Confidence)
			fmt.Printf("  • [%s] %s (confidence: %s, score: %.2f)\n", entry.Category, entry.Claim, conf, entry.Score)
		}
		return exitOK
	}

	if subcommand == "classify" {
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(report.Classification))
		}
		fmt.Printf("=== REPOSITORY CLASSIFICATION ===\n\n")
		fmt.Printf("App Types:    %s\n", strings.Join(report.Classification.AppTypes, ", "))
		fmt.Printf("Languages:    %s\n", strings.Join(report.Classification.Languages, ", "))
		fmt.Printf("Frameworks:   %s\n", strings.Join(report.Classification.Frameworks, ", "))
		fmt.Printf("Toolchains:   %s\n", strings.Join(report.Classification.Toolchains, ", "))
		fmt.Printf("Capabilities: %s\n", strings.Join(report.Classification.Capabilities, ", "))
		return exitOK
	}

	// Uncertainty resolution
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

		// Ensure complete Protocol v3 workspace (.ai manifests, docs/PRUMO.md, PROJECT_STATE.md, etc.)
		svc := cliops.New(repoRoot())
		det := cliops.DetectProject(path)
		profile := cliops.BuildDefaultProfile(det, "standard")
		_, _ = svc.InitWithProfile(path, profile)

		if asJSON {
			return printEnvelope(protocol.OkEnvelope(ApplySummary{Applied: applied}))
		}
		fmt.Printf("=== ADOPTION MIGRATION APPLIED (%d proposal(s)) ===\n\n", len(applied))
		for _, app := range applied {
			fmt.Printf("✓ Proposal %s applied successfully (Journal hash: %s)\n", app.ProposalID, app.JournalEntry.Hash)
		}
		fmt.Printf("\n✓ Project adopted into Prumo Protocol v3 in %s\n", path)
		fmt.Println("\nNext steps:")
		fmt.Println("  prumo validate        # Verify workspace conformance")
		fmt.Println("  prumo doctor          # Verify harness and connector health")
		fmt.Println("  prumo compile --all   # Compile agent adapters")
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
	fmt.Println("\nTo apply this adoption and configure Prumo Protocol v3, run:")
	fmt.Printf("  prumo adopt --apply %s   # or: prumo adopt apply %s\n", path, path)
	return exitOK
}
