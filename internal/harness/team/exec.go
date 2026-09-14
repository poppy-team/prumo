// Team execution: concurrent roles in isolated workspaces, per-role hard
// budgets, least-context review gates and bounded automatic delegation.
// Roles never share a workspace without policy (Validate); reviewers get
// diff+requirements+evidence, never the implementer trajectory.
package team

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Work is one role's unit of work inside its workspace. Usage reports
// consumed budget units (e.g. {"tool_calls": 3}).
type Work func(ctx context.Context, r Role) (evidence []string, usage map[string]float64, err error)

// Reviewer judges least-context review input.
type Reviewer func(ctx context.Context, input ReviewInput) (ReviewVerdict, error)

// Suggester proposes follow-up roles for suggested/bounded-auto modes.
// It sees prior results only — never full trajectories.
type Suggester func(ctx context.Context, t Team, results []RoleResult) ([]Role, error)

// RoleResult is one role's outcome.
type RoleResult struct {
	Role     string             `json:"role"`
	Evidence []string           `json:"evidence,omitempty"`
	Usage    map[string]float64 `json:"usage,omitempty"`
	Error    string             `json:"error,omitempty"`
}

// Summary is the team's terminal outcome.
type Summary struct {
	TeamID  string         `json:"team_id"`
	Status  string         `json:"status"` // complete|changes_requested|failed
	Results []RoleResult   `json:"results"`
	Review  *ReviewVerdict `json:"review,omitempty"`
}

// Runner executes a team.
type Runner struct {
	Team Team
	Work Work
	// ReviewerRole names the review-only role (excluded from Work). Empty
	// disables the review gate.
	ReviewerRole string
	Review       Reviewer
	Requirements []string
	// Suggester enables suggested/bounded-auto modes.
	Suggester Suggester
	// MaxDelegations caps spawned roles in bounded-auto mode (default 3).
	MaxDelegations int
}

// Run validates, executes, budgets, reviews and summarizes.
func (r Runner) Run(ctx context.Context) (Summary, error) {
	if r.Work == nil {
		return Summary{}, fmt.Errorf("team runner requires work")
	}
	switch r.Team.Delegation {
	case DelegationSolo:
		if len(r.Team.Roles) != 1 {
			return Summary{}, fmt.Errorf("solo requires exactly 1 role, got %d", len(r.Team.Roles))
		}
	case DelegationSuggested, DelegationBoundedAuto:
		if r.Suggester == nil {
			return Summary{}, fmt.Errorf("mode %s requires a suggester", r.Team.Delegation)
		}
	}
	if err := r.Team.Validate(); err != nil {
		return Summary{}, err
	}
	maxDel := r.MaxDelegations
	if maxDel <= 0 {
		maxDel = 3
	}
	roles := append([]Role{}, r.Team.Roles...)
	var results []RoleResult
	before := map[string]map[string]string{}
	delegations := 0
	for {
		batch := pendingRoles(roles, r.ReviewerRole, results)
		for _, role := range batch {
			if role.Workspace != "" {
				before[role.Name] = SnapshotWorkspace(role.Workspace)
			}
		}
		got, err := runBatch(ctx, batch, r.Work)
		if err != nil {
			return Summary{}, err
		}
		results = append(results, got...)
		if r.Team.Delegation != DelegationSuggested && r.Team.Delegation != DelegationBoundedAuto {
			roles = nil
			break
		}
		if r.Team.Delegation == DelegationBoundedAuto && delegations >= maxDel {
			break
		}
		extra, err := r.Suggester(ctx, r.Team, results)
		if err != nil {
			return Summary{}, err
		}
		if len(extra) == 0 {
			break
		}
		delegations += len(extra)
		roles = append(roles, extra...)
		grown := Team{ID: r.Team.ID, Roles: roles}
		if err := grown.Validate(); err != nil {
			return Summary{}, err
		}
	}
	for _, res := range results {
		if res.Error != "" {
			return Summary{TeamID: r.Team.ID, Status: "failed", Results: results}, nil
		}
	}
	if r.ReviewerRole == "" || r.Review == nil {
		return Summary{TeamID: r.Team.ID, Status: "complete", Results: results}, nil
	}
	input := r.reviewInput(before, results)
	verdict, err := r.Review(ctx, input)
	if err != nil {
		return Summary{}, err
	}
	status := "complete"
	if !verdict.Approve {
		status = "changes_requested"
	}
	return Summary{TeamID: r.Team.ID, Status: status, Results: results, Review: &verdict}, nil
}

// pendingRoles returns roles that have not completed yet, excluding the
// review-only role (reviewers judge, they do not implement).
func pendingRoles(roles []Role, reviewer string, done []RoleResult) (batch []Role) {
	finished := map[string]bool{}
	for _, d := range done {
		finished[d.Role] = true
	}
	for _, role := range roles {
		if role.Name == reviewer || finished[role.Name] {
			continue
		}
		batch = append(batch, role)
	}
	return batch
}

// runBatch executes roles concurrently with fail-fast cancel. Budgets are
// hard caps: exceeding a declared limit fails the role, never silently.
func runBatch(ctx context.Context, roles []Role, work Work) ([]RoleResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]RoleResult, len(roles))
	var wg sync.WaitGroup
	for i, role := range roles {
		wg.Add(1)
		go func(i int, role Role) {
			defer wg.Done()
			ev, usage, err := work(ctx, role)
			res := RoleResult{Role: role.Name, Evidence: ev, Usage: usage}
			if err != nil {
				res.Error = err.Error()
				cancel()
			} else if over := overBudget(role, usage); over != "" {
				res.Error = "budget exhausted: " + over
				cancel()
			}
			results[i] = res
		}(i, role)
	}
	wg.Wait()
	if ctx.Err() != nil && len(roles) > 0 {
		// Surface cancellation only when a role did not already report it.
		failed := false
		for _, res := range results {
			if res.Error != "" {
				failed = true
			}
		}
		if !failed {
			return nil, ctx.Err()
		}
	}
	return results, nil
}

func overBudget(role Role, usage map[string]float64) string {
	for k, used := range usage {
		if limit, ok := role.Budget[k]; ok && used > limit {
			return fmt.Sprintf("%s used %.0f > limit %.0f", k, used, limit)
		}
	}
	return ""
}

// reviewInput builds the least-context payload: before/after workspace
// diffs plus declared requirements and evidence. Trajectories never enter.
func (r Runner) reviewInput(before map[string]map[string]string, results []RoleResult) ReviewInput {
	var diffs []string
	var evidence []string
	for _, res := range results {
		evidence = append(evidence, res.Evidence...)
	}
	for _, role := range r.Team.Roles {
		if role.Name == r.ReviewerRole || role.Workspace == "" {
			continue
		}
		if d := DiffSnapshots(role.Workspace, before[role.Name], SnapshotWorkspace(role.Workspace)); d != "" {
			diffs = append(diffs, "### "+role.Name+"\n"+d)
		}
	}
	return ReviewInput{Diff: strings.Join(diffs, "\n"), Requirements: r.Requirements, Evidence: evidence}
}

var snapshotSkip = map[string]bool{".git": true, ".prumo": true, "node_modules": true}

// SnapshotWorkspace hashes workspace files (bounded) for later diffing.
func SnapshotWorkspace(root string) map[string]string {
	out := map[string]string{}
	count := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || count >= 200 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if snapshotSkip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > 1<<18 {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:8])
		count++
		return nil
	})
	return out
}

// DiffSnapshots renders added/modified files between two snapshots with
// bounded content snippets (40 lines / 8KB total).
func DiffSnapshots(root string, before, after map[string]string) string {
	changed := []string{}
	for rel, sum := range after {
		if before[rel] != sum {
			changed = append(changed, rel)
		}
	}
	sort.Strings(changed)
	var b strings.Builder
	used := 0
	for _, rel := range changed {
		if used >= 8000 {
			b.WriteString("…[truncated]\n")
			break
		}
		status := "M"
		if _, ok := before[rel]; !ok {
			status = "A"
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || len(data) > 1<<14 || !isText(data) {
			b.WriteString(status + " " + rel + " (binary/large)\n")
			continue
		}
		b.WriteString(status + " " + rel + "\n")
		lines := strings.Split(string(data), "\n")
		if len(lines) > 40 {
			lines = lines[:40]
		}
		for _, ln := range lines {
			b.WriteString("  " + ln + "\n")
			used += len(ln) + 3
		}
	}
	return b.String()
}

func isText(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return true
}
