package adoption

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/validation"
)

// Adoption wrote a prumo.json the project's own schema rejects: six keys at
// version 1, where the schema requires ten fields and puts a minimum of 2 on
// the version. So the framework created a file that failed Prumo's own
// validation, and every gate downstream was reporting on a manifest the
// framework had promised was valid (GAP-150).
//
// The point of this test is the schema, not the shape: it reads the real
// schemas/prumo.schema.json, so a future change to either side that breaks the
// other fails here.

func schemaPath(t *testing.T, name string) string {
	t.Helper()
	// The test runs in internal/adoption.
	repo := filepath.Clean(filepath.Join("..", ".."))
	path := filepath.Join(repo, "schemas", name)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cannot find %s: %v", path, err)
	}
	return path
}

func generatedManifest(t *testing.T) map[string]any {
	t.Helper()
	report := AdoptionReport{
		Version: 1,
		Repository: RepoSummary{
			Root: "/tmp/example-project",
		},
		Classification: RepositoryClassification{
			AppTypes:   []string{"cli"},
			Languages:  []string{"go"},
			Frameworks: []string{},
		},
	}
	proposals := GenerateMigrationProposals(report)
	for _, p := range proposals {
		if p.ID != "amp-init-prumo" {
			continue
		}
		for _, action := range p.Actions {
			if action.TargetPath != "prumo.json" {
				continue
			}
			var manifest map[string]any
			if err := json.Unmarshal([]byte(action.Content), &manifest); err != nil {
				t.Fatalf("the generated prumo.json is not valid JSON: %v", err)
			}
			return manifest
		}
	}
	t.Fatal("adoption generated no prumo.json proposal for a repository without one")
	return nil
}

func TestTheGeneratedManifestSatisfiesItsOwnSchema(t *testing.T) {
	manifest := generatedManifest(t)
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	schemaRaw, err := os.ReadFile(schemaPath(t, "prumo.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaRaw, &schema); err != nil {
		t.Fatal(err)
	}
	registry, err := validation.LoadRegistry(filepath.Dir(schemaPath(t, "prumo.schema.json")))
	if err != nil {
		t.Fatal(err)
	}
	// The project's own validator, not a second opinion written for this test.
	if problems := validation.ValidateSchema(manifest, schema, registry); len(problems) > 0 {
		t.Fatalf("the prumo.json adoption generates fails the project's own schema:\n  %s",
			strings.Join(problems, "\n  "))
	}
	_ = raw
}

func TestTheGeneratedManifestUsesAVersionTheSchemaAccepts(t *testing.T) {
	manifest := generatedManifest(t)
	version, ok := manifest["version"].(float64)
	if !ok {
		t.Fatalf("version is %T, not a number: %+v", manifest["version"], manifest["version"])
	}
	if version < 2 {
		t.Fatalf("version %v is below the schema's minimum of 2", version)
	}
}

func TestTheGeneratedManifestCarriesEveryRequiredSection(t *testing.T) {
	manifest := generatedManifest(t)
	for _, section := range []string{
		"version", "framework", "project", "documentation",
		"context", "intelligence", "orchestration", "goals", "ai", "protocol",
	} {
		if _, ok := manifest[section]; !ok {
			t.Errorf("the generated manifest omits the required section %q", section)
		}
	}
}

func TestTheGeneratedManifestInventsNoModelRoster(t *testing.T) {
	// A framework that fills in a default model list is the thing the
	// provider-neutral rule exists to prevent. Adoption does not know which
	// models a project uses, so it must leave a visible placeholder rather than
	// a confident-looking preference nobody chose.
	manifest := generatedManifest(t)
	ai, _ := manifest["ai"].(map[string]any)
	models, _ := ai["preferred_models"].([]any)
	if len(models) == 0 {
		t.Fatal("the schema requires at least one model and the manifest has none")
	}
	for _, m := range models {
		name, _ := m.(string)
		if name == "" {
			t.Fatal("an empty string is not a model preference")
		}
		if !strings.Contains(name, "<configure:") {
			t.Fatalf("adoption filled in the model %q without knowing anything about the project", name)
		}
	}
}
