package cliops

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/resolver"
	"github.com/raillen/prumo/internal/validation"
)

func (s *Service) ContextPlan(root, task string) map[string]any {
	profile, strategy, reason := pickContext(task)
	prumo, _ := readJSON(filepath.Join(root, "prumo.json"))
	contextValue, _ := prumo["context"].(map[string]any)
	profiles, _ := contextValue["profiles"].(map[string]any)
	budget := profiles[profile]
	if budget == nil {
		budget = profiles[fmt.Sprint(contextValue["budget_profile"])]
	}
	return map[string]any{"task": task, "profile": profile, "strategy": strategy, "budget": budget, "reasons": []string{reason}}
}

func pickContext(task string) (string, string, string) {
	tokens := map[string]bool{}
	normalized := strings.NewReplacer("/", " ", "-", " ").Replace(strings.ToLower(task))
	for _, token := range strings.Fields(normalized) {
		tokens[token] = true
	}
	for _, hint := range []string{"rename", "typo", "label", "copy", "text", "spelling"} {
		if tokens[hint] {
			return "small", "direct", "localized-task-hint"
		}
	}
	for _, hint := range []string{"architecture", "migration", "security", "refactor", "rewrite", "redesign"} {
		if tokens[hint] {
			return "large", "progressive-retrieval", "high-impact-task-hint"
		}
	}
	for _, hint := range []string{"bug", "regression", "unknown", "investigate", "debug", "failure", "crash"} {
		if tokens[hint] {
			return "medium", "progressive-retrieval", "uncertain-task-hint"
		}
	}
	return "medium", "context-pack-or-structural-retrieval", "default"
}

func (s *Service) ReportSummary(root string) (map[string]any, error) {
	data, err := readJSON(filepath.Join(root, ".prumo/history/project-intelligence.json"))
	if err != nil {
		return map[string]any{}, nil
	}
	summary, _ := data["summary"].(map[string]any)
	return summary, nil
}

func (s *Service) ReportAdd(root, file string) (map[string]any, error) {
	report, err := readJSON(file)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprint(report["id"])
	if id == "" || id == "<nil>" {
		return nil, errors.New("Task report requires id")
	}
	path := filepath.Join(root, ".prumo/history/project-intelligence.json")
	data, _ := readJSON(path)
	tasks, _ := data["tasks"].([]any)
	filtered := []any{}
	for _, task := range tasks {
		item, _ := task.(map[string]any)
		if fmt.Sprint(item["id"]) != id {
			filtered = append(filtered, task)
		}
	}
	filtered = append(filtered, report)
	data["tasks"] = filtered
	data["summary"] = summarize(filtered)
	data["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	return data, writeJSON(path, data)
}

func summarize(tasks []any) map[string]any {
	var input, output, cached, intermediate, cost float64
	for _, raw := range tasks {
		task, _ := raw.(map[string]any)
		tokens, _ := task["tokens"].(map[string]any)
		input += number(tokens["input"])
		output += number(tokens["output"])
		cached += number(tokens["cached"])
		intermediate += number(tokens["intermediate_output"])
		costMap, _ := task["cost"].(map[string]any)
		direct, _ := costMap["direct"].(map[string]any)
		cost += number(direct["amount"])
	}
	return map[string]any{
		"tasks": len(tasks), "input_tokens": int(input), "output_tokens": int(output),
		"cached_tokens": int(cached), "intermediate_output_tokens": int(intermediate),
		"direct_cost": float64(int(cost*1e6)) / 1e6,
	}
}

func number(value any) float64 {
	if v, ok := value.(float64); ok {
		return v
	}
	return 0
}

func (s *Service) Snapshot(root, output string) error {
	if output == "" {
		output = filepath.Join(root, ".prumo", filepath.Base(root)+"-prumo-snapshot.zip")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	defer archive.Close()
	for _, base := range []string{"ENTRYPOINT.md", "prumo.json", "PROJECT_STATE.md", "docs", ".ai", ".prumo/history"} {
		path := filepath.Join(root, base)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			err := filepath.Walk(path, func(child string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				rel, _ := filepath.Rel(root, child)
				if strings.Contains(rel, ".prumo/runtime") || strings.Contains(rel, ".prumo/cache") || strings.Contains(rel, "__pycache__") {
					return nil
				}
				return addZip(archive, child, rel)
			})
			if err != nil {
				return err
			}
			continue
		}
		if err := addZip(archive, path, base); err != nil {
			return err
		}
	}
	return nil
}

func addZip(archive *zip.Writer, path, rel string) error {
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	entry, err := archive.Create(rel)
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, input)
	return err
}

func (s *Service) Migrate(root string, dryRun bool) (map[string]any, error) {
	path := filepath.Join(root, "prumo.json")
	if _, err := readJSON(path); err != nil {
		if _, statErr := os.Stat(filepath.Join(root, ".prumo", "project-profile.yaml")); statErr == nil {
			return s.migrateV1(root, dryRun)
		}
		if _, statErr := os.Stat(filepath.Join(root, "PROJECT_MANIFEST.yaml")); statErr == nil {
			return s.migrateV1(root, dryRun)
		}
		return nil, fmt.Errorf("prumo.json not found; cannot migrate to v0.3")
	}
	return s.migrateV2(root, dryRun)
}

var legacyFiles = []string{"PROJECT_MANIFEST.yaml", ".prumo/project-profile.yaml", ".ai/agents/manifest.yaml", ".ai/skills/manifest.yaml", ".ai/recipes/manifest.yaml", ".ai/orchestration/model-policy.yaml", ".ai/orchestration/orchestrator.yaml", ".ai/orchestration/fallbacks.yaml", ".ai/orchestration/model-scorecard.yaml"}

func (s *Service) migrateV1(root string, dryRun bool) (map[string]any, error) {
	if dryRun {
		return map[string]any{"from_version": 1, "to_version": 3, "dry_run": true, "changes": []string{"Migrate v1 YAML to v2 JSON, then v2 to v3"}, "snapshot": nil}, nil
	}
	profilePath := filepath.Join(root, ".prumo", "project-profile.yaml")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("Legacy .prumo/project-profile.yaml not found")
	}
	profile, err := parseLegacyYAML(string(profileData))
	if err != nil {
		return nil, err
	}
	profileFile := filepath.Join(root, ".prumo", "profile-v1.json")
	if err := writeJSON(profileFile, profile); err != nil {
		return nil, err
	}
	legacyGoals := []string{}
	goalsBase := filepath.Join(root, ".ai", "goals")
	_ = filepath.Walk(goalsBase, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".goal.yaml") || strings.HasSuffix(path, ".goal.yml") {
			legacyGoals = append(legacyGoals, path)
		}
		return nil
	})
	if _, err := s.Init(root, profileFile); err != nil {
		return nil, err
	}
	_ = os.Remove(profileFile)
	created := []string{filepath.Join(root, "prumo.json")}
	for _, old := range legacyGoals {
		data, err := os.ReadFile(old)
		if err != nil {
			continue
		}
		goal, err := parseLegacyYAML(string(data))
		if err != nil {
			continue
		}
		target := strings.TrimSuffix(strings.TrimSuffix(old, ".goal.yml"), ".goal.yaml") + ".goal.json"
		if err := writeJSON(target, goal); err != nil {
			return nil, err
		}
		created = append(created, target)
		_ = os.Remove(old)
	}
	for _, rel := range legacyFiles {
		_ = os.Remove(filepath.Join(root, rel))
	}
	report, err := s.migrateV2(root, false)
	if err != nil {
		return nil, err
	}
	report["from_version"] = 1
	report["created"] = created
	return report, nil
}

func (s *Service) migrateV2(root string, dryRun bool) (map[string]any, error) {
	path := filepath.Join(root, "prumo.json")
	data, err := readJSON(path)
	if err != nil {
		return nil, fmt.Errorf("prumo.json not found; cannot migrate to v0.3")
	}
	changes := []string{}
	version, _ := data["version"].(float64)
	needsMigration := version < 3 || data["protocol"] == nil
	report := map[string]any{"from_version": 2, "to_version": 3, "dry_run": dryRun, "changes": changes, "snapshot": nil}

	// The snapshot is taken before anything is written, because its whole purpose
	// is to be the state the migration started from. Taken afterwards it is a
	// snapshot of the migration's own result: rolling back restores the migrated
	// file, and the version that caused the migration is gone (GAP-145).
	var snapshot string
	if !dryRun && needsMigration {
		snapshot = filepath.Join(root, ".prumo", "snapshots", fmt.Sprintf("pre_migration_v03_%s.zip", time.Now().UTC().Format("20060102150405")))
		if err := s.Snapshot(root, snapshot); err != nil {
			// A migration that cannot be snapshotted is not started. Reporting
			// the error here, before the write, is the difference between a
			// reversible migration and an irreversible one.
			return nil, err
		}
		report["snapshot"] = snapshot
	}

	if needsMigration {
		data["version"] = 3
		data["protocol"] = map[string]any{"version": 3, "compatible": ">=3 <4"}
		changes = append(changes, "Updated prumo.json to version 3 with protocol metadata")
		report["changes"] = changes
		if !dryRun {
			if err := writeJSON(path, data); err != nil {
				return nil, err
			}
		}
	}
	return report, nil
}

func ValidateWorkforce(repoRoot string) []string {
	out := []string{}
	base := filepath.Join(repoRoot, "src", "prumo", "resources", "workforce")
	checks := []struct{ kind, manifest, document, schema string }{
		{"skills", "manifest.json", "SKILL.md", "skill.schema.json"},
		{"agents", "manifest.json", "AGENT.md", "agent.schema.json"},
		{"recipes", "recipe.json", "RECIPE.md", "recipe.schema.json"},
	}
	for _, check := range checks {
		dir := filepath.Join(base, check.kind)
		entries, err := os.ReadDir(dir)
		if err != nil {
			out = append(out, fmt.Sprintf("missing workforce directory: %s", check.kind))
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			packageDir := filepath.Join(dir, entry.Name())
			manifest := filepath.Join(packageDir, check.manifest)
			if check.kind == "agents" {
				if _, err := os.Stat(manifest); err != nil {
					manifest = filepath.Join(packageDir, "agent.json")
				}
			}
			if check.kind == "recipes" {
				if _, err := os.Stat(manifest); err != nil {
					manifest = filepath.Join(packageDir, "manifest.json")
				}
			}
			if _, err := os.Stat(manifest); err != nil {
				out = append(out, fmt.Sprintf("%s package %s is missing %s", strings.Title(strings.TrimSuffix(check.kind, "s")), entry.Name(), check.manifest))
				continue
			}
			if _, err := os.Stat(filepath.Join(packageDir, check.document)); err != nil && check.kind == "skills" {
				out = append(out, fmt.Sprintf("Skill package %s is missing SKILL.md", entry.Name()))
			}
			for _, message := range validation.ValidateFile(manifest, check.schema, filepath.Join(repoRoot, "schemas")) {
				out = append(out, fmt.Sprintf("%s %s: %s", strings.Title(strings.TrimSuffix(check.kind, "s")), entry.Name(), message))
			}
		}
	}
	return out
}

func (s *Service) FrameworkCheck() []string {
	out := []string{}
	catalog, err := resolver.LoadCatalog(s.repoRoot)
	if err != nil {
		out = append(out, "catalog: "+err.Error())
	} else {
		skills := map[string]map[string]any{}
		for _, item := range catalog.Sections["skills"] {
			if id, ok := item["id"].(string); ok {
				skills[id] = item
			}
		}
		agents := map[string]map[string]any{}
		for _, item := range catalog.Sections["agents"] {
			if id, ok := item["id"].(string); ok {
				agents[id] = item
			}
		}
		recipes := map[string]map[string]any{}
		for _, item := range catalog.Sections["recipes"] {
			if id, ok := item["id"].(string); ok {
				recipes[id] = item
			}
		}
		for _, key := range []string{"agents", "skills", "recipes", "bundles"} {
			ids := map[string]bool{}
			duplicate := false
			for _, item := range catalog.Sections[key] {
				id := fmt.Sprint(item["id"])
				if ids[id] {
					duplicate = true
					break
				}
				ids[id] = true
			}
			if duplicate {
				out = append(out, "duplicate IDs in catalog section: "+key)
			}
		}
		for id, agent := range agents {
			for _, skill := range stringsFromAny(agent["requires_skills_any"]) {
				if _, ok := skills[skill]; !ok {
					out = append(out, fmt.Sprintf("agent %s references unknown skill %s", id, skill))
				}
			}
		}
		for id, recipe := range recipes {
			for _, skill := range stringsFromAny(recipe["requires_skills_any"]) {
				if _, ok := skills[skill]; !ok {
					out = append(out, fmt.Sprintf("recipe %s references unknown skill %s", id, skill))
				}
			}
		}
		for id, skill := range skills {
			for _, dep := range stringsFromAny(skill["requires"]) {
				if _, ok := skills[dep]; !ok {
					out = append(out, fmt.Sprintf("skill %s references unknown dependency %s", id, dep))
				}
			}
		}
		for _, bundle := range catalog.Sections["bundles"] {
			bundleID := fmt.Sprint(bundle["id"])
			for _, agent := range stringsFromAny(bundle["agents"]) {
				if _, ok := agents[agent]; !ok {
					out = append(out, fmt.Sprintf("bundle %s references unknown agent %s", bundleID, agent))
				}
			}
			for _, skill := range stringsFromAny(bundle["skills"]) {
				if _, ok := skills[skill]; !ok {
					out = append(out, fmt.Sprintf("bundle %s references unknown skill %s", bundleID, skill))
				}
			}
			for _, recipe := range stringsFromAny(bundle["recipes"]) {
				if _, ok := recipes[recipe]; !ok {
					out = append(out, fmt.Sprintf("bundle %s references unknown recipe %s", bundleID, recipe))
				}
			}
		}
		out = append(out, ValidateWorkforce(s.repoRoot)...)
	}
	for _, name := range []string{"generic", "chatgpt", "claude", "kimi", "codex", "claude-code", "traycer"} {
		if _, err := os.Stat(filepath.Join(s.repoRoot, "src", "prumo", "resources", "adapters", name+".md")); err != nil {
			out = append(out, "missing adapter: "+name)
		}
	}
	for _, name := range []string{"prumo.schema.json", "project-profile.schema.json", "goal.schema.json", "model-policy.schema.json", "task-report.schema.json", "repository-policy.schema.json", "documentation-contract.schema.json", "documentation-profile.schema.json", "documentation-binding.schema.json", "documentation-coverage.schema.json", "documentation-readiness.schema.json", "documentation-delta.schema.json", "documentation-finding.schema.json", "executor-session.schema.json", "checkpoint.schema.json", "continuation-record.schema.json", "budget-envelope.schema.json", "context-manifest.schema.json", "side-effect-journal.schema.json", "tool-descriptor.schema.json", "model-descriptor.schema.json", "execution-environment.schema.json"} {
		if _, err := os.Stat(filepath.Join(s.repoRoot, "schemas", name)); err != nil {
			out = append(out, "missing schema: "+name)
		}
	}
	return out
}
