package cliops

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/protocol/evidence"
	"github.com/raillen/prumo/internal/protocol/goals"
	"github.com/raillen/prumo/internal/resolver"
	"github.com/raillen/prumo/internal/validation"
)

type Service struct{ repoRoot string }

func New(repoRoot string) *Service { return &Service{repoRoot: repoRoot} }
func readJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, nil
}
func writeJSON(path string, value any) error {
	data, err := marshalPythonCompatible(value)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func marshalPythonCompatible(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
func writeText(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimRight(value, "\n")+"\n"), 0644)
}
func (s *Service) Profile(path string) (resolver.Profile, error) { return resolver.LoadProfile(path) }
func (s *Service) Resolve(profilePath string) (resolver.Resolution, error) {
	profile, err := s.Profile(profilePath)
	if err != nil {
		return resolver.Resolution{}, err
	}
	catalog, err := resolver.LoadCatalog(s.repoRoot)
	if err != nil {
		return resolver.Resolution{}, err
	}
	return catalog.Resolve(profile), nil
}
func (s *Service) ModelPolicy(profile resolver.Profile) (map[string]any, error) {
	models := profile.PreferredModels()
	if len(models) == 0 {
		return nil, errors.New("No preferred LLM roster configured. Prumo requires model preferences to be explicitly selected for every new project.")
	}
	roster := []map[string]any{}
	ids := []string{}
	for _, m := range models {
		id := fmt.Sprint(m["id"])
		provider := fmt.Sprint(m["provider"])
		if provider == "<nil>" || provider == "" {
			provider = "unknown"
		}
		ids = append(ids, id)
		roster = append(roster, map[string]any{"id": id, "provider": provider, "enabled": true})
	}
	roles := map[string]any{}
	for _, role := range []string{"architect", "debugger", "documentation-maintainer", "implementer", "release-verifier", "reviewer", "security-reviewer", "tester", "ux-reviewer"} {
		roles[role] = map[string]any{"preferred": append([]string{}, ids...), "fallback": []string{}}
	}
	return map[string]any{"version": 2, "selection_rule": "cheapest-reliable-model-that-passes-gates", "cross_provider_review": true, "context_aware_routing": true, "roster": roster, "roles": roles}, nil
}
func contextPolicy() map[string]any {
	profiles := map[string]any{}
	for _, v := range []struct {
		name                                      string
		target, hard, out, outHard, rounds, depth int
	}{{"small", 3000, 6000, 500, 1000, 1, 0}, {"medium", 8000, 16000, 1500, 3000, 2, 1}, {"large", 16000, 32000, 3000, 6000, 3, 1}} {
		profiles[v.name] = map[string]any{"context_target_tokens": v.target, "context_hard_tokens": v.hard, "output_target_tokens": v.out, "output_hard_tokens": v.outHard, "max_expansion_rounds": v.rounds, "max_delegation_depth": v.depth}
	}
	return map[string]any{"methodology": "lean-progressive-context", "architecture": "progressive-context-architecture", "mode": "progressive", "budget_profile": "medium", "profiles": profiles, "deep_recursion": map[string]any{"enabled": false, "experimental": true}, "runtime": map[string]any{"database": ".prumo/runtime/prumo.db", "completed_context_ttl": "0d", "failed_context_ttl": "7d"}}
}
func (s *Service) Init(root, profilePath string) (resolver.Resolution, error) {
	profile, err := s.Profile(profilePath)
	if err != nil {
		return resolver.Resolution{}, err
	}
	if len(profile.PreferredModels()) == 0 {
		return resolver.Resolution{}, errors.New("Preferred LLMs/providers are required per project. Add ai.preferred_models to the profile.")
	}
	catalog, err := resolver.LoadCatalog(s.repoRoot)
	if err != nil {
		return resolver.Resolution{}, err
	}
	resolution := catalog.Resolve(profile)
	project, _ := profile.Raw["project"].(map[string]any)
	orchestrator := profile.Orchestrator()
	autonomy := profile.Autonomy()
	policy, err := s.ModelPolicy(profile)
	if err != nil {
		return resolver.Resolution{}, err
	}
	config := map[string]any{"version": 3, "protocol": map[string]any{"version": 3, "compatible": ">=3 <4"}, "framework": map[string]any{"name": "prumo", "version": protocol.CLIVersion}, "project": project, "stack": profile.Raw["stack"], "features": profile.Raw["features"], "risk": profile.Raw["risk"], "quality": profile.Raw["quality"], "documentation": map[string]any{"entrypoint": "docs/PRUMO.md", "canonical_format": "markdown", "site": map[string]any{"enabled": true, "source": "docs", "generated": true, "public_internal_views": true}, "audiences": []string{"user", "developer", "operations", "agent"}, "virtual_chunking": true}, "context": contextPolicy(), "intelligence": map[string]any{"enabled": true, "path": ".prumo/history/project-intelligence.json", "task_reports": true, "track_input_tokens": true, "track_output_tokens": true, "track_cost": true, "distinguish_observed_estimated": true}, "orchestration": map[string]any{"protocol": "POP", "orchestrator": orchestrator, "autonomy": autonomy}, "goals": map[string]any{"active_phase": "P00", "active_goal": nil}, "ai": profile.Raw["ai"]}
	if err := writeJSON(filepath.Join(root, "prumo.json"), config); err != nil {
		return resolver.Resolution{}, err
	}
	for _, kind := range []string{"agents", "skills", "recipes"} {
		values := []string{}
		if kind == "agents" {
			values = resolution.Agents
		}
		if kind == "skills" {
			values = resolution.Skills
		}
		if kind == "recipes" {
			values = resolution.Recipes
		}
		reasons := map[string]any{}
		for _, v := range values {
			reasons[v] = resolution.Reasons[v]
		}
		if err := writeManifestJSON(filepath.Join(root, ".ai", kind, "manifest.json"), kind, values, reasons); err != nil {
			return resolver.Resolution{}, err
		}
	}
	if err := writeJSON(filepath.Join(root, ".ai", "orchestration", "model-policy.json"), policy); err != nil {
		return resolver.Resolution{}, err
	}
	if err := writeOrderedJSON(filepath.Join(root, ".ai", "orchestration", "orchestrator.json"), []orderedField{{"version", 2}, {"primary", orchestrator}, {"protocol", "project-orchestration-protocol-v2"}, {"autonomy", autonomy}, {"source_of_truth", "repository"}, {"context_methodology", "LPC"}}); err != nil {
		return resolver.Resolution{}, err
	}
	if err := writeOrderedJSON(filepath.Join(root, ".ai", "orchestration", "fallbacks.json"), []orderedField{{"version", 2}, {"rules", []string{"availability fallback", "quality fallback after bounded failed attempts", "targeted context expansion before broad re-read", "cross-provider review for high-risk changes when possible", "human escalation after fallback/budget exhaustion"}}, {"unbounded_retry", false}, {"unbounded_recursion", false}}); err != nil {
		return resolver.Resolution{}, err
	}
	if err := writeOrderedJSON(filepath.Join(root, ".ai", "orchestration", "model-scorecard.json"), []orderedField{{"version", 2}, {"records", []any{}}, {"note", "Append measured project-local model performance only."}}); err != nil {
		return resolver.Resolution{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeOrderedJSON(filepath.Join(root, ".prumo", "history", "project-intelligence.json"), []orderedField{{"version", 1}, {"updated_at", now}, {"summary", map[string]any{"tasks": 0, "input_tokens": 0, "output_tokens": 0, "cached_tokens": 0, "direct_cost": 0.0}}, {"tasks", []any{}}, {"debt", []any{}}}); err != nil {
		return resolver.Resolution{}, err
	}
	projectName := filepath.Base(root)
	if name, ok := project["name"].(string); ok && name != "" {
		projectName = name
	}
	for _, dir := range []string{"docs", ".prumo/runtime", ".prumo/cache", ".ai/goals/P00"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			return resolver.Resolution{}, err
		}
	}
	if err := writeText(filepath.Join(root, "PROJECT_STATE.md"), "# Current Project State\n\n- Project: **"+projectName+"**\n- Prumo: **"+protocol.CLIVersion+"**\n- Current phase: **P00 — Foundation**\n- Current goal: **not selected**\n- Context methodology: **Lean Progressive Context (LPC)**\n- Last updated: `"+now+"`\n\n## Next action\n\nDefine and lock the first measurable Goal before implementation begins.\n\n## Recovery order\n\n1. `ENTRYPOINT.md` or the platform adapter.\n2. `prumo.json`.\n3. `PROJECT_STATE.md`.\n4. `docs/PRUMO.md`.\n5. Active Goal under `.ai/goals/`.\n6. Only relevant canonical docs/symbols/tests selected by the context strategy.\n\nDo not load the entire repository by default.\n"); err != nil {
		return resolver.Resolution{}, err
	}
	if err := writeText(filepath.Join(root, "docs", "PRUMO.md"), "# Prumo — "+projectName+"\n\nThis is the intent router for humans and agents. Add links as stable documentation is created; do not create empty documentation solely to populate this map.\n\n## Current state\n\n- [Project state](../PROJECT_STATE.md)\n- `prumo.json` — canonical project configuration\n\n## I want to use the product\n\nAdd user tutorials, how-to guides, reference and explanations under `docs/user/` as needed.\n\n## I want to develop/contribute\n\nAdd onboarding, codebase tour, build/test/debug and task-oriented development guides under `docs/developer/`.\n\n## I want to operate/support it\n\nAdd deployment, configuration, observability, runbooks, backup/recovery, troubleshooting and release guidance under `docs/operations/` / `docs/support/` as needed.\n\n## I am an AI agent\n\n1. Read the active Goal.\n2. Use the smallest sufficient context.\n3. Prefer structural/symbol/document-section pointers.\n4. Expand only when evidence is insufficient.\n5. Keep output bounded.\n6. Update only impacted canonical docs.\n7. Record evidence/intelligence and garbage-collect temporary context.\n\n## Architecture / decisions / specs\n\nAdd stable architecture, ADR/RFC and specifications as the project grows.\n\n## Goals\n\nGoals live under `.ai/goals/<phase>/` and define measurable completion.\n\n## Durable intelligence\n\nCompact project/task intelligence lives in `.prumo/history/project-intelligence.json`.\n"); err != nil {
		return resolver.Resolution{}, err
	}
	if err := writeText(filepath.Join(root, "ENTRYPOINT.md"), "# Prumo entrypoint\n\n1. Read `prumo.json`, `PROJECT_STATE.md` and `docs/PRUMO.md`.\n2. Read the active Goal and its dependencies.\n3. Start with the minimum sufficient context; do not read the entire repository.\n4. Prefer Context Packs/task maps, document sections, symbols and related tests.\n5. Expand context only when evidence is insufficient; delegation depth is bounded by `prumo.json`.\n6. Never weaken acceptance criteria silently.\n7. Keep code, tests and canonical docs synchronized through a Documentation Delta.\n8. Keep intermediate output compact and do not persist task-specific context files.\n9. Before completion, record evidence/project intelligence and remove temporary context.\n"); err != nil {
		return resolver.Resolution{}, err
	}
	if err := ensureGitignore(root); err != nil {
		return resolver.Resolution{}, err
	}
	if err := scaffoldBaselineDocuments(root, projectName); err != nil {
		return resolver.Resolution{}, err
	}
	return resolution, nil
}

func writeManifestJSON(path, kind string, values []string, reasons map[string]any) error {
	var out strings.Builder
	out.WriteString("{\n")
	out.WriteString("  \"generated_by\": {\n")
	out.WriteString("    \"prumo\": \"0.2.0\"\n")
	out.WriteString("  },\n")
	valuesJSON, _ := json.MarshalIndent(values, "  ", "  ")
	out.WriteString(fmt.Sprintf("  %q: %s,\n", kind, valuesJSON))
	reasonsJSON, _ := json.MarshalIndent(reasons, "  ", "  ")
	out.WriteString(fmt.Sprintf("  \"reasons\": %s\n", reasonsJSON))
	out.WriteString("}\n")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(out.String()), 0644)
}

type orderedField struct {
	key   string
	value any
}

func writeOrderedJSON(path string, fields []orderedField) error {
	var out strings.Builder
	out.WriteString("{\n")
	for i, field := range fields {
		value, _ := json.MarshalIndent(field.value, "  ", "  ")
		out.WriteString(fmt.Sprintf("  %q: %s", field.key, value))
		if i < len(fields)-1 {
			out.WriteString(",")
		}
		out.WriteString("\n")
	}
	out.WriteString("}\n")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(out.String()), 0644)
}

func ensureGitignore(root string) error {
	path := filepath.Join(root, ".gitignore")
	current := ""
	if data, err := os.ReadFile(path); err == nil {
		current = string(data)
	}
	required := []string{".prumo/runtime/", ".prumo/cache/"}
	lines := strings.Split(current, "\n")
	missing := []string{}
	for _, need := range required {
		found := false
		for _, line := range lines {
			if line == need {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, need)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	addition := "\n# Prumo derived/runtime state\n" + strings.Join(missing, "\n") + "\n"
	return os.WriteFile(path, []byte(strings.TrimRight(current, "\n")+addition), 0644)
}
func (s *Service) SchemaDir() string {
	return filepath.Join(s.repoRoot, "schemas")
}

func (s *Service) Validate(root string) []string {
	out := []string{}
	if _, err := os.Stat(filepath.Join(root, "prumo.json")); err != nil {
		return []string{"not a recognized Prumo project: missing prumo.json"}
	}
	for _, rel := range []string{"docs/PRUMO.md", "PROJECT_STATE.md", "prumo.json", ".ai/orchestration/model-policy.json", ".ai/agents/manifest.json", ".ai/skills/manifest.json", ".ai/recipes/manifest.json", ".prumo/history/project-intelligence.json"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			out = append(out, "missing required file: "+rel)
		}
	}
	for _, rel := range []string{"prumo.json", ".ai/orchestration/model-policy.json"} {
		if data, err := os.ReadFile(filepath.Join(root, rel)); err == nil && !json.Valid(data) {
			out = append(out, rel+": invalid JSON")
		}
	}
	registry, err := validation.LoadRegistry(s.SchemaDir())
	if err != nil {
		return append(out, "schema registry: "+err.Error())
	}
	check := func(instancePath, schemaName string) {
		instanceFile := filepath.Join(root, instancePath)
		if _, err := os.Stat(instanceFile); err != nil {
			return
		}
		schemaFile := filepath.Join(s.SchemaDir(), schemaName)
		if _, err := os.Stat(schemaFile); err != nil {
			return
		}
		for _, message := range validation.ValidateFile(instanceFile, schemaName, s.SchemaDir()) {
			rel := strings.TrimPrefix(strings.TrimPrefix(message, root+"/"), root+string(os.PathSeparator))
			out = append(out, rel)
		}
		_ = registry
	}
	check("prumo.json", "prumo.schema.json")
	check(".ai/orchestration/model-policy.json", "model-policy.schema.json")
	for _, path := range listGoals(root) {
		rel, _ := filepath.Rel(root, path)
		check(rel, "goal.schema.json")
	}
	legacy := []string{"PROJECT_MANIFEST.yaml", ".prumo/project-profile.yaml", ".ai/agents/manifest.yaml", ".ai/skills/manifest.yaml", ".ai/recipes/manifest.yaml", ".ai/orchestration/model-policy.yaml", ".ai/orchestration/orchestrator.yaml", ".ai/orchestration/fallbacks.yaml", ".ai/orchestration/model-scorecard.yaml"}
	generated := []string{}
	for _, rel := range legacy {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			generated = append(generated, rel)
		}
	}
	goalsBase := filepath.Join(root, ".ai", "goals")
	_ = filepath.Walk(goalsBase, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".goal.yaml") || strings.HasSuffix(path, ".goal.yml") {
			if rel, err := filepath.Rel(root, path); err == nil {
				generated = append(generated, rel)
			}
		}
		return nil
	})
	if len(generated) > 0 {
		limit := generated
		if len(limit) > 10 {
			limit = limit[:10]
		}
		out = append(out, "v0.2 Prumo canonical state contains legacy YAML; migrate/remove: "+strings.Join(limit, ", "))
	}
	return out
}
func (s *Service) NewGoal(root, id, title, phase, objective string) error {
	return writeJSON(filepath.Join(root, ".ai", "goals", id+".goal.json"), goals.NewGoal(id, title, phase, objective))
}
func findGoal(root, id string) (string, error) {
	var match string
	base := filepath.Join(root, ".ai", "goals")
	_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || match != "" {
			return nil
		}
		name := filepath.Base(path)
		if name == id+".goal.json" || name == id+".goal.yaml" || name == id+".goal.yml" {
			match = path
		}
		return nil
	})
	if match == "" {
		return "", fmt.Errorf("Goal %s not found in %s", id, filepath.Join(root, ".ai/goals"))
	}
	return match, nil
}

func listGoals(root string) []string {
	out := []string{}
	base := filepath.Join(root, ".ai", "goals")
	_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(filepath.Base(path), ".goal.") {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func (s *Service) GoalState(root, id, state, reason string) (goals.Goal, error) {
	match, err := findGoal(root, id)
	if err != nil {
		return nil, err
	}
	// DONE requires that the goal's evidence resolves to real records, so the
	// project's evidence store is read before the transition rather than after.
	// Without this the transition could only ask whether the evidence array was
	// non-empty, which any non-empty value satisfies (GAP-124).
	records, err := loadEvidenceRecords(root)
	if err != nil {
		return nil, err
	}
	return goals.TransitionGoalWithEvidence(match, state, reason, records)
}

// loadEvidenceRecords reads every evidence record the project has produced.
// A project that has produced none yields an empty, non-nil slice so the DONE
// check reports "no such record" per reference instead of "no store".
func loadEvidenceRecords(root string) ([]evidence.Record, error) {
	dir := filepath.Join(root, ".prumo", "runtime", "harness")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []evidence.Record{}, nil
		}
		return nil, fmt.Errorf("read evidence store: %w", err)
	}
	records := []evidence.Record{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "evidence-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", name, readErr)
		}
		var record evidence.Record
		if unmarshalErr := json.Unmarshal(data, &record); unmarshalErr != nil {
			// A corrupt evidence file must not silently become "no evidence" for
			// the whole store; name the file instead.
			return nil, fmt.Errorf("parse %s: %w", name, unmarshalErr)
		}
		records = append(records, record)
	}
	return records, nil
}
func (s *Service) GoalAmend(root, id, file, reason, approvedBy string) (goals.Goal, error) {
	match, err := findGoal(root, id)
	if err != nil {
		return nil, err
	}
	amend := map[string]any{"reason": reason, "approved_by": approvedBy, "changes": map[string]any{}}
	if file != "" {
		var err error
		amend, err = readJSON(file)
		if err != nil {
			return nil, err
		}
	}
	return goals.AmendGoal(match, amend)
}
func (s *Service) GoalList(root string) ([]map[string]any, error) {
	out := []map[string]any{}
	for _, p := range listGoals(root) {
		v, e := readJSON(p)
		if e != nil {
			continue
		}
		out = append(out, map[string]any{"id": v["id"], "title": v["title"], "phase": v["phase"], "state": v["state"], "revision": v["revision"], "path": strings.TrimPrefix(p, root+string(os.PathSeparator))})
	}
	sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]["id"]) < fmt.Sprint(out[j]["id"]) })
	return out, nil
}
