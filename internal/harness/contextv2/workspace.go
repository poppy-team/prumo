// Workspace-backed Context v2 compilation: turns a goal + workspace tree
// into a packed, replayable Manifest v2. No-LLM, deterministic, budget
// enforcing. Git/project metadata are best-effort: a run never fails
// because context enrichment failed — it degrades to the goal item.
package contextv2

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/knowledge"
	"github.com/raillen/prumo/internal/harness/model"
)

// CompileWorkspace builds candidates from the goal, entrypoints, git state
// and a bounded file listing, gates them, dedups and packs by
// marginal-utility-per-token.
func CompileWorkspace(runID, goal, root string, budget int, level string) Manifest {
	if budget <= 0 {
		budget = 8000
	}
	if level == "" {
		level = "L1"
	}
	goalItem := Item{
		Ref: "goal", Authority: "canonical", Trust: "high", Privacy: "internal",
		Freshness: "current", Score: 1.0, Method: "exact",
		TokenCost: estimateTokens(goal), Content: goal, Reason: "the run goal",
	}
	candidates := []Item{goalItem}
	abs := root
	if a, err := filepath.Abs(root); err == nil {
		abs = a
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return Compile(runID, candidates, budget, level)
	}
	for _, it := range entrypointItems(abs) {
		candidates = append(candidates, it)
	}
	modified := gitModified(abs)
	for _, it := range fileItems(abs, modified) {
		candidates = append(candidates, it)
	}
	eligible := make([]Item, 0, len(candidates))
	for _, it := range candidates {
		if Eligible(it, "reference", false) {
			eligible = append(eligible, it)
		}
	}
	// Memory Atlas as a structured source (never dumped wholesale).
	eligible = append(eligible, atlasItems(abs, goal)...)
	// Repository map + symbol hits (LSP when available, repomap fallback).
	eligible = append(eligible, codeIntelItems(abs, goal)...)
	// Lexical fusion: BM25 bonus over file candidates, then dedup+pack.
	refs := make([]string, 0, len(eligible))
	for _, it := range eligible {
		refs = append(refs, it.Ref)
	}
	for ref, bonus := range ftsBoost(abs, refs, Tokenize(goal)) {
		for i := range eligible {
			if eligible[i].Ref != ref {
				continue
			}
			eligible[i].Score += bonus
			// The lexical signal is part of the reason the item is here; naming
			// it is what makes the manifest explainable rather than merely
			// reproducible (W4.5).
			eligible[i].Reason = joinReason(eligible[i].Reason, "lexically matched the goal")
		}
	}
	manifest := CompileWithPolicy(runID, eligible, CompilePolicy{
		Budget: budget, Level: ParseLevel(level), Policy: "workspace-v2",
	})
	// Instruction overhead is measured over what actually shipped, not over the
	// candidates that were considered and dropped (W4.10).
	manifest.InstructionTokens = 0
	for _, it := range manifest.Included {
		if isInstructionSurface(it.Ref) {
			manifest.InstructionTokens += it.TokenCost
		}
	}
	return manifest
}

// instructionRefNames are the repository-root agent instruction surfaces. They
// are part of every compilation but their cost is reported separately, because
// instruction overhead is a different budget from retrieved content (W4.10).
var instructionRefNames = []string{"AGENTS.md", "CLAUDE.md", "ENTRYPOINT.md"}

// isInstructionSurface reports whether a candidate is an agent instruction
// surface rather than project content.
func isInstructionSurface(ref string) bool {
	slash := filepath.ToSlash(ref)
	for _, name := range instructionRefNames {
		if slash == name {
			return true
		}
	}
	return strings.Contains(slash, ".github/copilot-instructions") ||
		strings.Contains(slash, ".cursor/rules/")
}

func estimateTokens(s string) int { return model.EstimateTokens(s, "") }

var entrypoints = []string{"ENTRYPOINT.md", "AGENTS.md", "README.md", "prumo.json", "go.mod"}

func entrypointItems(root string) []Item {
	out := []Item{}
	for _, name := range entrypoints {
		p := filepath.Join(root, name)
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		// Entry points are small and always considered; carrying their content is
		// what makes progressive disclosure meaningful for them (W4.4).
		content := ""
		if st.Size() <= 1<<18 {
			if data, err := os.ReadFile(p); err == nil {
				content = string(data)
			}
		}
		out = append(out, Item{
			Ref: name, Authority: "canonical", Trust: "high", Privacy: "internal",
			Freshness: st.ModTime().UTC().Format(time.RFC3339),
			Score:     0.9, Method: "exact", TokenCost: cappedEstimate(int(st.Size())),
			Content: content, Reason: "repository entry point",
		})
	}
	return out
}

func cappedEstimate(size int) int {
	// Size-proportional estimate through the versioned table (GAP-027).
	n := size / 4
	if n < 1 {
		n = 1
	}
	if n > 2000 {
		n = 2000
	}
	return n
}

func gitModified(root string) map[string]bool {
	out := map[string]bool{}
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = root
	data, err := cmd.Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		if len(line) < 4 {
			continue
		}
		name := strings.TrimSpace(line[2:])
		if i := strings.Index(name, " -> "); i >= 0 {
			name = name[i+4:]
		}
		name = strings.Trim(name, `"`)
		if name != "" {
			out[name] = true
		}
	}
	return out
}

var skipDirs = map[string]bool{".git": true, "node_modules": true, ".prumo": true, "target": true, "dist": true, ".venv": true}

func fileItems(root string, modified map[string]bool) []Item {
	out := []Item{}
	top, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	names := []string{}
	for _, e := range top {
		if skipDirs[e.Name()] {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	count := 0
	add := func(rel string, st os.FileInfo) {
		if count >= 50 || st.IsDir() || st.Size() > 1<<20 {
			return
		}
		count++
		score := 0.5
		if modified[rel] {
			score = 0.8
		}
		reason := "workspace file"
		if modified[rel] {
			reason = "recently modified workspace file"
		}
		out = append(out, Item{
			Ref: "file:" + rel, Authority: "reference", Trust: "medium", Privacy: "internal",
			Freshness: st.ModTime().UTC().Format(time.RFC3339), Rev: "",
			Score: score, Method: "structured", TokenCost: cappedEstimate(int(st.Size())),
			Reason: reason,
		})
	}
	for _, name := range names {
		p := filepath.Join(root, name)
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.IsDir() {
			sub, err := os.ReadDir(p)
			if err != nil {
				continue
			}
			subNames := []string{}
			for _, e := range sub {
				subNames = append(subNames, e.Name())
			}
			sort.Strings(subNames)
			for _, sn := range subNames {
				sp := filepath.Join(p, sn)
				sst, err := os.Stat(sp)
				if err != nil || sst.IsDir() {
					continue
				}
				add(filepath.Join(name, sn), sst)
			}
			continue
		}
		add(name, st)
	}
	return out
}

// atlasItems recalls top memories as candidates (pointer-sized, not dumps).
func atlasItems(root, goal string) []Item {
	a, err := knowledge.LoadAtlas(knowledge.AtlasPath(root))
	if err != nil || len(a.Records) == 0 {
		return nil
	}
	s := knowledge.New()
	s.Restore(a.Records, nil)
	recalled := knowledge.Recall(s, goal, 5)
	out := make([]Item, 0, len(recalled))
	for i, r := range recalled {
		out = append(out, Item{
			Ref: "memory:" + r.ID, Authority: "reference", Trust: "medium",
			Privacy: "internal", Freshness: r.UpdatedAt,
			Score: 0.6 - 0.05*float64(i), Method: "memory",
			TokenCost: cappedEstimate(len(r.Title) + len(r.Body)),
			Content:   r.Title, Reason: "recalled project memory",
		})
	}
	return out
}
