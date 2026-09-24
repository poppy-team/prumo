package cliops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/connectors/antigravity"
	"github.com/raillen/prumo/internal/connectors/gemini"
	"github.com/raillen/prumo/internal/connectors/opencode"
	"github.com/raillen/prumo/internal/modelregistry"
	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/protocol/goals"
	"github.com/raillen/prumo/internal/protocol/plans"
	"github.com/raillen/prumo/internal/resolver"
)

func stringsFromAny(value any) []string {
	out := []string{}
	if raw, ok := value.([]any); ok {
		for _, item := range raw {
			out = append(out, fmt.Sprint(item))
		}
	}
	return out
}

func mapByID(items []map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, item := range items {
		if id, ok := item["id"].(string); ok {
			out[id] = item
		}
	}
	return out
}

func copyWorkforcePackage(source, target string) []string {
	created := []string{}
	files := []string{}
	_ = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(path, "__pycache__") || strings.HasSuffix(path, ".pyc") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	for _, file := range files {
		rel, err := filepath.Rel(source, file)
		if err != nil {
			continue
		}
		dest := filepath.Join(target, rel)
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			continue
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			continue
		}
		created = append(created, dest)
	}
	return created
}

func renderItem(item map[string]any) string {
	if item == nil {
		return "# Prumo Resource\n"
	}
	name := fmt.Sprint(item["name"])
	if name == "<nil>" || name == "" {
		name = fmt.Sprint(item["id"])
	}
	return fmt.Sprintf("# %s\n\nID: `%s`\n\n%s\n", name, item["id"], item["purpose"])
}

func (s *Service) Compile(root, target string) ([]string, error) {
	supported := map[string]bool{"generic": true, "chatgpt": true, "claude": true, "kimi": true, "codex": true, "claude-code": true, "traycer": true, "opencode": true, "gemini": true, "antigravity": true}
	if !supported[target] {
		return nil, fmt.Errorf("Unsupported target: %s", target)
	}
	if target == "opencode" {
		c := opencode.NewConnector()
		res, err := c.Compile(root, connectors.CompileOptions{RepoRoot: s.repoRoot})
		if err != nil {
			return nil, err
		}
		return res.CreatedPaths, nil
	}
	if target == "gemini" {
		c := gemini.NewConnector()
		res, err := c.Compile(root, connectors.CompileOptions{RepoRoot: s.repoRoot})
		if err != nil {
			return nil, err
		}
		return res.CreatedPaths, nil
	}
	if target == "antigravity" {
		c := antigravity.NewConnector()
		res, err := c.Compile(root, connectors.CompileOptions{RepoRoot: s.repoRoot})
		if err != nil {
			return nil, err
		}
		return res.CreatedPaths, nil
	}
	catalog, err := resolver.LoadCatalog(s.repoRoot)
	if err != nil {
		return nil, err
	}
	agentsManifest, _ := readJSON(filepath.Join(root, ".ai", "agents", "manifest.json"))
	skillsManifest, _ := readJSON(filepath.Join(root, ".ai", "skills", "manifest.json"))
	selectedAgents := stringsFromAny(agentsManifest["agents"])
	selectedSkills := stringsFromAny(skillsManifest["skills"])
	adapter, err := os.ReadFile(filepath.Join(s.repoRoot, "src", "prumo", "resources", "adapters", target+".md"))
	if err != nil {
		return nil, err
	}
	agentMap := mapByID(catalog.Sections["agents"])
	skillMap := mapByID(catalog.Sections["skills"])
	created := []string{}
	if target == "codex" || target == "claude-code" {
		prefix := ".codex"
		name := "AGENTS.md"
		if target == "claude-code" {
			prefix = ".claude"
			name = "CLAUDE.md"
		}
		path := filepath.Join(root, name)
		if err := writeText(path, string(adapter)); err != nil {
			return nil, err
		}
		created = append(created, path)
		for _, id := range selectedAgents {
			agentPackage := filepath.Join(s.repoRoot, "src", "prumo", "resources", "workforce", "agents", id, "AGENT.md")
			path := filepath.Join(root, prefix, "agents", id+".md")
			if data, err := os.ReadFile(agentPackage); err == nil {
				if err := writeText(path, string(data)); err != nil {
					return nil, err
				}
			} else if err := writeText(path, renderItem(agentMap[id])); err != nil {
				return nil, err
			}
			created = append(created, path)
		}
		for _, id := range selectedSkills {
			skillPackage := filepath.Join(s.repoRoot, "src", "prumo", "resources", "workforce", "skills", id)
			targetDir := filepath.Join(root, prefix, "skills", id)
			if info, err := os.Stat(skillPackage); err == nil && info.IsDir() {
				created = append(created, copyWorkforcePackage(skillPackage, targetDir)...)
			} else {
				path := filepath.Join(targetDir, "SKILL.md")
				if err := writeText(path, renderItem(skillMap[id])); err != nil {
					return nil, err
				}
				created = append(created, path)
			}
		}
		return created, nil
	}
	if target == "traycer" {
		path := filepath.Join(root, ".traycer", "PROJECT_PRUMO.md")
		content := strings.TrimRight(string(adapter), "\n") + "\n\n## LPC/PCA\nUse `ENTRYPOINT.md` + active Goal + progressive context. Generated context is not canonical.\n\n## Selected agents\n- " + strings.Join(selectedAgents, "\n- ") + "\n\n## Selected skills\n- " + strings.Join(selectedSkills, "\n- ")
		if err := writeText(path, content); err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	path := filepath.Join(root, ".prumo", "runtime", "compiled", target, "ENTRYPOINT.md")
	content := strings.TrimRight(string(adapter), "\n") + "\n\n## Lean Progressive Context\nStart at `ENTRYPOINT.md`, `prumo.json`, the active Goal and `docs/PRUMO.md`. Do not preload the repository. Expand only relevant document sections/symbols/tests.\n\nSelected agents: " + strings.Join(selectedAgents, ", ") + "\nSelected skills: " + strings.Join(selectedSkills, ", ") + "\n"
	if err := writeText(path, content); err != nil {
		return nil, err
	}
	ownership := filepath.Join(filepath.Dir(path), ".prumo-generated.json")
	if err := writeJSON(ownership, map[string]any{"prumo_generated": true, "prumo_version": protocol.CLIVersion, "generator": "compiler", "managed": true, "target": target}); err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func (s *Service) ExplainWorkforce(profilePath string) (map[string]any, error) {
	resolution, err := s.Resolve(profilePath)
	if err != nil {
		return nil, err
	}
	section := func(ids []string) map[string]any {
		out := map[string]any{}
		for _, id := range ids {
			out[id] = map[string]any{"reasons": resolution.Reasons[id], "traces": resolution.Traces[id]}
		}
		return out
	}
	return map[string]any{
		"summary": map[string]any{"agents_count": len(resolution.Agents), "skills_count": len(resolution.Skills), "recipes_count": len(resolution.Recipes)},
		"agents":  section(resolution.Agents),
		"skills":  section(resolution.Skills),
		"recipes": section(resolution.Recipes),
	}, nil
}

func (s *Service) workforceItem(kind, id string) map[string]any {
	base := filepath.Join(s.repoRoot, "src", "prumo", "resources", "workforce")
	var dir, manifestName, docName string
	switch kind {
	case "agents":
		dir, manifestName, docName = "agents", "manifest.json", "AGENT.md"
	case "skills":
		dir, manifestName, docName = "skills", "manifest.json", "SKILL.md"
	case "recipes":
		dir, manifestName, docName = "recipes", "recipe.json", "RECIPE.md"
	default:
		return nil
	}
	packageDir := filepath.Join(base, dir, id)
	manifestPath := filepath.Join(packageDir, manifestName)
	if manifestName == "manifest.json" && kind == "agents" {
		if _, err := os.Stat(manifestPath); err != nil {
			manifestPath = filepath.Join(packageDir, "agent.json")
		}
	}
	if manifestName == "recipe.json" {
		if _, err := os.Stat(manifestPath); err != nil {
			manifestPath = filepath.Join(packageDir, "manifest.json")
		}
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	var item map[string]any
	if err := json.Unmarshal(data, &item); err != nil {
		return nil
	}
	docPath := filepath.Join(packageDir, docName)
	if doc, err := os.ReadFile(docPath); err == nil {
		if _, exists := item["instructions"]; !exists {
			item["instructions"] = string(doc)
		}
	}
	return item
}

func (s *Service) ExplainCatalog(kind, id string) (map[string]any, bool, error) {
	singular := strings.TrimSuffix(kind, "s")
	if item := s.workforceItem(singular, id); item != nil {
		return explainShape(singular, item), true, nil
	}
	catalog, err := resolver.LoadCatalog(s.repoRoot)
	if err != nil {
		return nil, false, err
	}
	for _, item := range catalog.Sections[kind] {
		if fmt.Sprint(item["id"]) == id {
			return explainShape(singular, item), true, nil
		}
	}
	return nil, false, nil
}

func explainShape(kind string, item map[string]any) map[string]any {
	instructions, _ := item["instructions"].(string)
	switch kind {
	case "skill":
		return map[string]any{"id": item["id"], "name": item["name"], "version": intValue(item["version"], 1), "purpose": item["purpose"], "risk_level": stringValue(item["risk_level"], "low"), "modes": listValue(item["modes"]), "requires": listValue(item["requires"]), "select": mapValue(item["select"]), "instructions": strings.TrimSpace(instructions), "provenance": provenanceValue(item["provenance"])}
	case "agent":
		return map[string]any{"id": item["id"], "name": item["name"], "version": intValue(item["version"], 2), "purpose": item["purpose"], "risk_level": stringValue(item["risk_level"], "low"), "required_skills": listValue(item["required_skills"]), "requires_skills_any": listValue(item["requires_skills_any"]), "inputs": listValue(item["inputs"]), "outputs": listValue(item["outputs"]), "permissions": mapValue(item["permissions"]), "instructions": strings.TrimSpace(instructions)}
	default:
		return map[string]any{"id": item["id"], "name": item["name"], "purpose": item["purpose"], "steps": listValue(item["steps"]), "instructions": strings.TrimSpace(instructions)}
	}
}

func intValue(value any, fallback int) int {
	if number, ok := value.(float64); ok {
		return int(number)
	}
	return fallback
}

func stringValue(value any, fallback string) string {
	if text, ok := value.(string); ok && text != "" {
		return text
	}
	return fallback
}

func listValue(value any) []any {
	if values, ok := value.([]any); ok {
		return values
	}
	return []any{}
}

func mapValue(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func provenanceValue(value any) any {
	if provenance, ok := value.(map[string]any); ok {
		return provenance
	}
	return map[string]any{"origin": "framework", "license": "MIT"}
}

func (s *Service) ExplainContext(root, taskID string) (map[string]any, error) {
	budget := map[string]any{"max_input_tokens": 8000, "max_retrieved_tokens": 4000}
	path := filepath.Join(root, ".prumo", "runtime", "context", taskID+".cpack.json")
	if data, err := os.ReadFile(path); err == nil {
		var pack map[string]any
		if err := json.Unmarshal(data, &pack); err == nil {
			items := []any{}
			if raw, ok := pack["items"].([]any); ok {
				for _, entry := range raw {
					item, _ := entry.(map[string]any)
					if item == nil {
						items = append(items, entry)
						continue
					}
					items = append(items, map[string]any{"id": item["id"], "why_loaded": item["reason"], "source": item["source"], "estimated_token_cost": item["estimated_tokens"], "trust_level": item["trust"]})
				}
			}
			return map[string]any{"task_id": taskID, "strategy": "progressive-retrieval", "estimated_tokens": pack["estimated_tokens"], "items": items, "budget": budget}, nil
		}
	}
	return map[string]any{"task_id": taskID, "strategy": "progressive-retrieval", "budget": budget, "notes": "Context is lazily expanded following Lean Progressive Context (LPC/PCA)."}, nil
}

func (s *Service) ExplainModel(root, role string) (map[string]any, error) {
	if policy, err := readJSON(filepath.Join(root, ".ai", "orchestration", "model-policy.json")); err == nil {
		roles, _ := policy["roles"].(map[string]any)
		target := roles[role]
		if target == nil {
			target = "default"
		}
		var profile any
		if name, ok := target.(string); ok {
			if profiles, ok := policy["profiles"].(map[string]any); ok {
				profile = profiles[name]
			}
		}
		selection, _ := policy["selection_rule"].(string)
		if selection == "" {
			selection = "least_workforce"
		}
		review, hasReview := policy["cross_provider_review"].(bool)
		if !hasReview {
			review = true
		}
		return map[string]any{"role": role, "target": target, "profile": profile, "selection_rule": selection, "cross_provider_review": review}, nil
	}
	return map[string]any{"role": role, "target": "default", "selection_rule": "least_workforce", "cross_provider_review": true}, nil
}

func (s *Service) ExplainExecution(root, profileID string) (map[string]any, error) {
	if policy, err := readJSON(filepath.Join(root, ".ai", "orchestration", "execution-policy.json")); err == nil {
		backends, _ := policy["backends"].(map[string]any)
		if backend, ok := backends[profileID]; ok && backend != nil {
			return map[string]any{"profile_id": profileID, "backend": backend, "default_fallback": policy["default_fallback"]}, nil
		}
	}
	return map[string]any{"profile_id": profileID, "backend": nil, "error": "Execution policy not found or profile missing."}, nil
}

func (s *Service) ExplainRun(root, runID string) (map[string]any, error) {
	if runID == "" {
		return map[string]any{
			"topic":       "run",
			"description": "Explains harness agent run lifecycle, checkpoints, and derived completion.",
			"states":      []string{"planned", "running", "checkpointed", "evidenced", "verified", "completed"},
			"invariants":  "Completion is strictly derived from verified evidence; self-declared model text is rejected.",
		}, nil
	}
	path := filepath.Join(root, ".prumo", "runtime", "harness", "runs", runID, "record.json")
	if data, err := os.ReadFile(path); err == nil {
		var rec map[string]any
		if err := json.Unmarshal(data, &rec); err == nil {
			return map[string]any{"run_id": runID, "record": rec}, nil
		}
	}
	return map[string]any{
		"run_id":         runID,
		"status":         "active_or_ephemeral",
		"checkpoint_dir": filepath.Join(root, ".prumo", "runtime", "harness", "runs", runID),
	}, nil
}

func (s *Service) ExplainRoute(root, routeClass string) (map[string]any, error) {
	classes := map[string]any{
		"ROUTE-PRIMARY":  "Default primary route for high-capability task execution.",
		"ROUTE-FALLBACK": "Fallback route engaged upon primary exhaustion, cooldown, or network failure.",
		"ROUTE-CHEAP":    "Cost-optimized route for trivial, read-only, or high-volume sub-tasks.",
	}
	quotaStatuses := []string{"known", "estimated", "unknown", "cooldown", "exhausted"}
	reasonCodes := []string{"REASON-ROUTING-PRIMARY", "REASON-FALLBACK-COOLDOWN", "REASON-EXHAUSTED-FAILOVER"}
	if desc, ok := classes[routeClass]; ok {
		return map[string]any{
			"class":          routeClass,
			"description":    desc,
			"quota_statuses": quotaStatuses,
			"reason_codes":   reasonCodes,
		}, nil
	}
	return map[string]any{
		"topic":          "route",
		"classes":        classes,
		"quota_statuses": quotaStatuses,
		"reason_codes":   reasonCodes,
	}, nil
}

func (s *Service) ExplainBudget(root, scope string) (map[string]any, error) {
	return map[string]any{
		"topic":          "budget",
		"scope":          scope,
		"hierarchy":      "Project -> Goal -> Task -> Run",
		"review_reserve": "10-20% reserved capacity strictly protected from implementation overreach",
		"hard_stop":      "ErrBlockedBudget triggers immediate safe stop and durable blocked_budget checkpoint",
		"dimensions":     []string{"tokens", "cost_usd", "tool_calls"},
	}, nil
}

func (s *Service) ExplainDecision(root, decisionID string) (map[string]any, error) {
	return map[string]any{
		"topic": "decision",
		"id":    decisionID,
		"levels": map[string]string{
			"L0 - Rule":      "Deterministic Go code and policy engines",
			"L1 - Decision":  "Closed answer space DecisionProvider (scoring, classification, routing)",
			"L2 - Reasoning": "Generative LLM for open synthesis and code generation",
			"L3 - Authority": "Human approval for high-risk or irreversible mutations",
		},
		"pipeline":   "DETERMINISTIC -> DECISION MODEL -> GENERATIVE MODEL -> HUMAN",
		"abstention": "DecisionProviders abstain when uncertainty exceeds threshold, escalating up the hierarchy",
	}, nil
}

func (s *Service) Doctor(root string) ([]map[string]any, error) {
	findings := []map[string]any{}
	rel := func(path string) string {
		if value, err := filepath.Rel(root, path); err == nil {
			return value
		}
		return path
	}
	for _, message := range s.Validate(root) {
		findings = append(findings, map[string]any{"category": "schema/structure", "severity": "ERROR", "message": message, "target": root})
	}
	// Pricing ages. The table is versioned and dated, but a rate nobody has
	// revisited produces a wrong number in a cost report, and nothing said so
	// (GAP-155). It is a WARNING rather than an ERROR: a stale rate is worth
	// knowing about and is not a broken project.
	for _, entry := range modelregistry.DefaultPricing().Stale(time.Now()) {
		age := fmt.Sprintf("%d days", entry.AgeDays)
		if entry.AgeDays < 0 {
			age = "an unreadable date"
		}
		findings = append(findings, map[string]any{
			"category": "pricing", "severity": "WARNING",
			"message": fmt.Sprintf(
				"Pricing for %s/%s is effective from %s (%s old); verify it against the provider",
				entry.Provider, entry.Model, entry.EffectiveFrom, age),
			"target": "modelregistry/pricing",
		})
	}
	prumo, err := readJSON(filepath.Join(root, "prumo.json"))
	if err == nil {
		version, _ := prumo["version"].(float64)
		if version < 2 {
			findings = append(findings, map[string]any{"category": "version", "severity": "ERROR", "message": fmt.Sprintf("Project version %v is deprecated; run 'prumo migrate'", prumo["version"]), "target": "prumo.json"})
		}
		if protocolValue, ok := prumo["protocol"].(map[string]any); ok {
			if protocolVersion, ok := protocolValue["version"].(float64); ok && protocolVersion > 3 {
				findings = append(findings, map[string]any{"category": "protocol", "severity": "WARNING", "message": fmt.Sprintf("Project protocol version %v is newer than framework runtime v0.3", protocolVersion), "target": "prumo.json"})
			}
		}
	}
	goalIDs := map[string]bool{}
	for _, path := range listGoals(root) {
		goal, err := readJSON(path)
		if err != nil {
			findings = append(findings, map[string]any{"category": "goal", "severity": "ERROR", "message": fmt.Sprintf("Failed to read goal file %s: %v", filepath.Base(path), err), "target": rel(path)})
			continue
		}
		id := fmt.Sprint(goal["id"])
		if id == "" || id == "<nil>" {
			id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		goalIDs[id] = true
		state := strings.ToUpper(fmt.Sprint(goal["state"]))
		if state == "LOCKED" || state == "EXECUTING" || state == "VERIFYING" || state == "REVIEWING" || state == "DONE" {
			if valid, msg := goals.VerifyLock(goals.Goal(goal)); !valid {
				findings = append(findings, map[string]any{"category": "goal-lock", "severity": "ERROR", "message": fmt.Sprintf("Goal %s lock integrity violation: %s", id, msg), "target": rel(path)})
			}
		}
	}
	for _, path := range listGoals(root) {
		goal, err := readJSON(path)
		if err != nil {
			continue
		}
		id := fmt.Sprint(goal["id"])
		for _, dep := range stringsFromAny(goal["dependencies"]) {
			if !goalIDs[dep] {
				findings = append(findings, map[string]any{"category": "goal-dependencies", "severity": "ERROR", "message": fmt.Sprintf("Goal %s references non-existent dependency goal '%s'", id, dep), "target": rel(path)})
			}
		}
	}
	planFiles := []string{}
	_ = filepath.Walk(filepath.Join(root, ".ai", "plans"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".plan.json") {
			planFiles = append(planFiles, path)
		}
		return nil
	})
	for _, path := range planFiles {
		plan, err := readJSON(path)
		if err != nil {
			findings = append(findings, map[string]any{"category": "plan", "severity": "ERROR", "message": fmt.Sprintf("Failed to validate plan %s: %v", filepath.Base(path), err), "target": rel(path)})
			continue
		}
		pid := fmt.Sprint(plan["id"])
		if pid == "" || pid == "<nil>" {
			pid = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		if goalID := fmt.Sprint(plan["goal_id"]); goalID != "" && goalID != "<nil>" && !goalIDs[goalID] {
			findings = append(findings, map[string]any{"category": "plan", "severity": "ERROR", "message": fmt.Sprintf("Plan %s references unknown goal '%s'", pid, goalID), "target": rel(path)})
		}
		tasks := []map[string]any{}
		if raw, ok := plan["tasks"].([]any); ok {
			for _, item := range raw {
				if task, ok := item.(map[string]any); ok {
					tasks = append(tasks, task)
				}
			}
		}
		if len(tasks) > 0 {
			for _, message := range plans.CheckDAGCycles(tasks) {
				findings = append(findings, map[string]any{"category": "dag", "severity": "ERROR", "message": fmt.Sprintf("Plan %s %s", pid, message), "target": rel(path)})
			}
		}
	}
	catalog, catalogErr := resolver.LoadCatalog(s.repoRoot)
	knownSkills, knownAgents, knownRecipes := map[string]bool{}, map[string]bool{}, map[string]bool{}
	if catalogErr == nil {
		for _, item := range catalog.Sections["skills"] {
			knownSkills[fmt.Sprint(item["id"])] = true
		}
		for _, item := range catalog.Sections["agents"] {
			knownAgents[fmt.Sprint(item["id"])] = true
		}
		for _, item := range catalog.Sections["recipes"] {
			knownRecipes[fmt.Sprint(item["id"])] = true
		}
	}
	if manifest, err := readJSON(filepath.Join(root, ".ai", "skills", "manifest.json")); err == nil {
		for _, skill := range stringsFromAny(manifest["skills"]) {
			if !knownSkills[skill] {
				if _, err := os.Stat(filepath.Join(root, ".ai", "skills", skill)); err != nil {
					findings = append(findings, map[string]any{"category": "workforce", "severity": "WARNING", "message": fmt.Sprintf("Selected skill '%s' is not registered in framework catalog or local skills", skill), "target": ".ai/skills/manifest.json"})
				}
			}
		}
	}
	if manifest, err := readJSON(filepath.Join(root, ".ai", "agents", "manifest.json")); err == nil {
		for _, agent := range stringsFromAny(manifest["agents"]) {
			if !knownAgents[agent] {
				if _, err := os.Stat(filepath.Join(root, ".ai", "agents", agent)); err != nil {
					findings = append(findings, map[string]any{"category": "workforce", "severity": "WARNING", "message": fmt.Sprintf("Selected agent '%s' is not registered in framework catalog or local agents", agent), "target": ".ai/agents/manifest.json"})
				}
			}
		}
	}
	if manifest, err := readJSON(filepath.Join(root, ".ai", "recipes", "manifest.json")); err == nil {
		for _, recipe := range stringsFromAny(manifest["recipes"]) {
			if !knownRecipes[recipe] {
				if _, err := os.Stat(filepath.Join(root, ".ai", "recipes", recipe)); err != nil {
					findings = append(findings, map[string]any{"category": "workforce", "severity": "WARNING", "message": fmt.Sprintf("Selected recipe '%s' is not registered", recipe), "target": ".ai/recipes/manifest.json"})
				}
			}
		}
	} else {
		findings = append(findings, map[string]any{"category": "workforce", "severity": "WARNING", "message": "Missing recipes manifest", "target": ".ai/recipes/manifest.json"})
	}
	_ = filepath.Walk(filepath.Join(root, ".ai", "skills"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".skill.json") {
			return nil
		}
		if skill, err := readJSON(path); err == nil {
			provenance, ok := skill["provenance"].(map[string]any)
			if !ok {
				findings = append(findings, map[string]any{"category": "skill-provenance", "severity": "ERROR", "message": "Skill missing provenance.origin", "target": rel(path)})
				return nil
			}
			if _, ok := provenance["origin"]; !ok {
				findings = append(findings, map[string]any{"category": "skill-provenance", "severity": "ERROR", "message": "Skill missing provenance.origin", "target": rel(path)})
			}
		}
		return nil
	})
	if policy, err := readJSON(filepath.Join(root, ".ai", "orchestration", "model-policy.json")); err == nil {
		// The role check used to expect each role to name a profile and skip
		// anything else, which is every role the generator writes — so it never
		// checked a single file the framework produced. Both accepted forms are
		// now parsed, and a role the reader cannot understand is an error rather
		// than a role it passes over (GAP-132).
		findings = append(findings, modelPolicyFindings(policy)...)
		if profiles, ok := policy["profiles"].(map[string]any); ok {
			for name, raw := range profiles {
				profile, _ := raw.(map[string]any)
				if _, ok := profile["provider"]; !ok {
					findings = append(findings, map[string]any{"category": "model-policy", "severity": "ERROR", "message": fmt.Sprintf("Profile '%s' is incomplete (missing provider or model)", name), "target": ".ai/orchestration/model-policy.json"})
					continue
				}
				if _, ok := profile["model"]; !ok {
					findings = append(findings, map[string]any{"category": "model-policy", "severity": "ERROR", "message": fmt.Sprintf("Profile '%s' is incomplete (missing provider or model)", name), "target": ".ai/orchestration/model-policy.json"})
				}
			}
		}
	}
	if policy, err := readJSON(filepath.Join(root, ".ai", "orchestration", "execution-policy.json")); err == nil {
		if backends, ok := policy["backends"].(map[string]any); ok {
			for name, raw := range backends {
				backend, _ := raw.(map[string]any)
				if _, ok := backend["type"]; !ok {
					findings = append(findings, map[string]any{"category": "execution-policy", "severity": "ERROR", "message": fmt.Sprintf("Backend '%s' missing type", name), "target": ".ai/orchestration/execution-policy.json"})
				}
			}
		}
	}
	evidenceIDs := map[string]bool{}
	_ = filepath.Walk(filepath.Join(root, ".ai", "evidence"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".evidence.json") {
			return nil
		}
		if evidence, err := readJSON(path); err == nil {
			if id := fmt.Sprint(evidence["id"]); id != "" && id != "<nil>" {
				evidenceIDs[id] = true
			}
		}
		return nil
	})
	for _, path := range planFiles {
		plan, err := readJSON(path)
		if err != nil {
			continue
		}
		tasks, _ := plan["tasks"].([]any)
		for _, raw := range tasks {
			task, _ := raw.(map[string]any)
			if task == nil {
				continue
			}
			if gates, ok := task["gates"].([]any); ok {
				for _, rawGate := range gates {
					gate, _ := rawGate.(map[string]any)
					if gate == nil {
						continue
					}
					if _, ok := gate["evidence_type"]; !ok {
						findings = append(findings, map[string]any{"category": "invalid-gates", "severity": "ERROR", "message": fmt.Sprintf("Gate in task '%v' missing evidence_type", task["id"]), "target": rel(path)})
					}
				}
			}
			for _, evidenceID := range stringsFromAny(task["evidence"]) {
				if !evidenceIDs[evidenceID] {
					findings = append(findings, map[string]any{"category": "missing-evidence", "severity": "WARNING", "message": fmt.Sprintf("Task '%v' references missing evidence '%s'", task["id"], evidenceID), "target": rel(path)})
				}
			}
		}
	}
	return findings, nil
}
