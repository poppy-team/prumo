package cliops

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/resolver"
)

// DetectionResult holds the technical facts inferred from inspecting a workspace.
type DetectionResult struct {
	Name               string   `json:"name"`
	AppTypes           []string `json:"app_types"`
	Languages          []string `json:"languages"`
	Frameworks         []string `json:"frameworks"`
	BuildTools         []string `json:"build_tools"`
	Features           []string `json:"features"`
	Risks              []string `json:"risks"`
	IsExistingCodebase bool     `json:"is_existing_codebase"`
	SourceFileCount    int      `json:"source_file_count"`
}

// DetectProject scans the target root directory to infer project name, stack, and architecture.
func DetectProject(root string) DetectionResult {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}

	name := filepath.Base(absRoot)
	if name == "." || name == "/" || name == "" {
		name = "my-project"
	}

	res := DetectionResult{
		Name:       name,
		AppTypes:   []string{"cli"},
		Languages:  []string{},
		Frameworks: []string{},
		BuildTools: []string{},
		Features:   []string{"ai-assisted-development", "documentation"},
		Risks:      []string{},
	}

	// 1. Check Go
	goModPath := filepath.Join(absRoot, "go.mod")
	if fileExists(goModPath) {
		res.Languages = appendUnique(res.Languages, "go")
		res.BuildTools = appendUnique(res.BuildTools, "go")
		modName, imports := parseGoMod(goModPath)
		if modName != "" {
			res.Name = filepath.Base(modName)
		}
		for _, imp := range imports {
			if strings.Contains(imp, "cobra") || strings.Contains(imp, "bubbletea") {
				res.AppTypes = appendUnique(res.AppTypes, "cli")
			}
			if strings.Contains(imp, "gin-gonic") || strings.Contains(imp, "fiber") || strings.Contains(imp, "echo") || strings.Contains(imp, "chi") {
				res.AppTypes = appendUnique(res.AppTypes, "service")
				res.Frameworks = appendUnique(res.Frameworks, "http-service")
			}
			if strings.Contains(imp, "grpc") {
				res.Features = appendUnique(res.Features, "grpc")
				res.Frameworks = appendUnique(res.Frameworks, "grpc")
			}
		}
	}

	// 2. Check Node / TypeScript / JavaScript
	pkgJSONPath := filepath.Join(absRoot, "package.json")
	if fileExists(pkgJSONPath) {
		pkgName, deps := parsePackageJSON(pkgJSONPath)
		if pkgName != "" {
			res.Name = pkgName
		}
		if fileExists(filepath.Join(absRoot, "tsconfig.json")) {
			res.Languages = appendUnique(res.Languages, "typescript")
		}
		res.Languages = appendUnique(res.Languages, "javascript")

		if fileExists(filepath.Join(absRoot, "pnpm-lock.yaml")) {
			res.BuildTools = appendUnique(res.BuildTools, "pnpm")
		} else if fileExists(filepath.Join(absRoot, "yarn.lock")) {
			res.BuildTools = appendUnique(res.BuildTools, "yarn")
		} else if fileExists(filepath.Join(absRoot, "bun.lockb")) || fileExists(filepath.Join(absRoot, "bun.lock")) {
			res.BuildTools = appendUnique(res.BuildTools, "bun")
		} else {
			res.BuildTools = appendUnique(res.BuildTools, "npm")
		}

		for dep := range deps {
			switch dep {
			case "next", "nuxt", "astro", "remix":
				res.AppTypes = appendUnique(res.AppTypes, "web")
				res.Frameworks = appendUnique(res.Frameworks, dep)
			case "react", "vue", "svelte":
				res.AppTypes = appendUnique(res.AppTypes, "web")
				res.Frameworks = appendUnique(res.Frameworks, dep)
			case "express", "fastify", "@nestjs/core", "koa", "hono":
				res.AppTypes = appendUnique(res.AppTypes, "service")
				res.Frameworks = appendUnique(res.Frameworks, dep)
			case "commander", "yargs", "meow", "cac":
				res.AppTypes = appendUnique(res.AppTypes, "cli")
			}
		}
	}

	// 3. Check Rust
	cargoPath := filepath.Join(absRoot, "Cargo.toml")
	if fileExists(cargoPath) {
		res.Languages = appendUnique(res.Languages, "rust")
		res.BuildTools = appendUnique(res.BuildTools, "cargo")
		cargoName := parseCargoToml(cargoPath)
		if cargoName != "" {
			res.Name = cargoName
		}
	}

	// 4. Check Python
	pyProject := filepath.Join(absRoot, "pyproject.toml")
	reqTxt := filepath.Join(absRoot, "requirements.txt")
	pipfile := filepath.Join(absRoot, "Pipfile")
	if fileExists(pyProject) || fileExists(reqTxt) || fileExists(pipfile) {
		res.Languages = appendUnique(res.Languages, "python")
		if fileExists(pyProject) {
			res.BuildTools = appendUnique(res.BuildTools, "poetry")
		} else {
			res.BuildTools = appendUnique(res.BuildTools, "pip")
		}
	}

	// 5. Walk directory to count source files
	sourceExtensions := map[string]bool{
		".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".rs": true, ".py": true, ".java": true, ".c": true, ".cpp": true,
		".cs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true,
	}
	fileCount := 0
	_ = filepath.Walk(absRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := info.Name()
			if base == ".git" || base == "node_modules" || base == "vendor" || base == ".prumo" || base == "target" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if sourceExtensions[ext] {
			fileCount++
		}
		return nil
	})
	res.SourceFileCount = fileCount
	if fileCount >= 3 {
		res.IsExistingCodebase = true
	}

	if len(res.Languages) == 0 {
		res.Languages = []string{"general"}
	}
	if len(res.BuildTools) == 0 {
		res.BuildTools = []string{"general"}
	}

	return res
}

// BuildDefaultProfile creates a valid resolver.Profile conforming to project-profile.schema.json.
func BuildDefaultProfile(det DetectionResult, preset string) resolver.Profile {
	selectedPreset := strings.ToLower(preset)
	if selectedPreset == "" {
		selectedPreset = "standard"
	}

	appTypes := det.AppTypes
	features := append([]string{}, det.Features...)
	risks := append([]string{}, det.Risks...)
	quality := map[string]any{"security": "standard", "documentation": "canonical"}

	switch selectedPreset {
	case "cli":
		appTypes = []string{"cli"}
		features = appendUnique(features, "cli-ux", "terminal-output")
	case "web":
		appTypes = []string{"web", "frontend"}
		features = appendUnique(features, "responsive-design", "ui-components")
	case "service":
		appTypes = []string{"service", "backend"}
		features = appendUnique(features, "api-design", "resilience")
		quality["security"] = "high"
	case "library":
		appTypes = []string{"library"}
		features = appendUnique(features, "clean-api", "zero-dependencies")
	case "minimal":
		features = []string{"documentation"}
	default: // standard
		features = appendUnique(features, "clean-code", "deterministic-testing")
	}

	preferredModels := []any{}

	raw := map[string]any{
		"version": 2,
		"project": map[string]any{
			"name": det.Name,
			"type": appTypes,
		},
		"stack": map[string]any{
			"languages":  det.Languages,
			"frameworks": det.Frameworks,
			"build":      det.BuildTools,
		},
		"features": features,
		"risk":     risks,
		"quality":  quality,
		"ai": map[string]any{
			"orchestrator":     "native",
			"autonomy":         "agentic",
			"preferred_models": preferredModels,
		},
	}

	return resolver.Profile{Raw: raw}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func appendUnique(slice []string, items ...string) []string {
	seen := map[string]bool{}
	for _, s := range slice {
		seen[s] = true
	}
	out := append([]string{}, slice...)
	for _, item := range items {
		if !seen[item] && item != "" {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func parseGoMod(path string) (string, []string) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	moduleName := ""
	imports := []string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			moduleName = strings.TrimSpace(strings.TrimPrefix(line, "module"))
		} else if strings.HasPrefix(line, "require (") || strings.HasPrefix(line, "require ") {
			if strings.Contains(line, " ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 && parts[0] != "require" {
					imports = append(imports, parts[0])
				}
			}
		} else if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "  ") {
			parts := strings.Fields(line)
			if len(parts) >= 1 && strings.Contains(parts[0], "/") {
				imports = append(imports, parts[0])
			}
		}
	}
	return moduleName, imports
}

func parsePackageJSON(path string) (string, map[string]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	var raw struct {
		Name         string            `json:"name"`
		Dependencies map[string]string `json:"dependencies"`
		DevDeps      map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", nil
	}
	deps := map[string]bool{}
	for k := range raw.Dependencies {
		deps[k] = true
	}
	for k := range raw.DevDeps {
		deps[k] = true
	}
	return raw.Name, deps
}

func parseCargoToml(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inPackage := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[package]" {
			inPackage = true
			continue
		}
		if strings.HasPrefix(line, "[") && line != "[package]" {
			inPackage = false
		}
		if inPackage && strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		}
	}
	return ""
}
