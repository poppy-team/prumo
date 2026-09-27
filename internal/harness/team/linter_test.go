package team

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintAgentContract(t *testing.T) {
	// 1. Valid agent
	validAgent := []byte(`{
		"id": "code-implementer",
		"name": "Code Implementer",
		"version": 1,
		"purpose": "Implement tested code changes within scope",
		"inputs": ["DirectiveIR", "Dossier"],
		"outputs": ["PatchDiff"],
		"risk_level": "medium"
	}`)
	findings := LintAgentContract(validAgent)
	if len(findings) > 0 {
		t.Fatalf("expected 0 findings for valid agent, got %d: %v", len(findings), findings)
	}

	// 2. Missing purpose and invalid allow_all
	badAgent := []byte(`{
		"id": "unbounded-agent",
		"name": "Unbounded Agent",
		"permissions": {"allow_all": true}
	}`)
	findingsBad := LintAgentContract(badAgent)
	foundPurpose := false
	foundPermissions := false
	for _, f := range findingsBad {
		if f.Rule == "required_purpose" {
			foundPurpose = true
		}
		if f.Rule == "bounded_permissions" {
			foundPermissions = true
		}
	}
	if !foundPurpose || !foundPermissions {
		t.Fatalf("expected required_purpose and bounded_permissions findings, got: %v", findingsBad)
	}
}

func TestLintSkillContract(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Missing manifest
	findings := LintSkillContract(tmpDir)
	if len(findings) == 0 || findings[0].Rule != "manifest_exists" {
		t.Fatalf("expected manifest_exists finding, got %v", findings)
	}

	// 2. Valid skill directory
	manifestContent := []byte(`{
		"id": "test-skill",
		"name": "Test Skill",
		"version": 1,
		"purpose": "Perform unit test analysis",
		"modes": ["testing", "review"]
	}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifestContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "instructions.md"), []byte("# Instructions\nDo testing."), 0644); err != nil {
		t.Fatal(err)
	}

	findingsValid := LintSkillContract(tmpDir)
	if len(findingsValid) > 0 {
		t.Fatalf("expected 0 findings for valid skill, got: %v", findingsValid)
	}

	// 3. Invalid mode
	badManifest := []byte(`{
		"id": "bad-skill",
		"name": "Bad Skill",
		"version": 1,
		"purpose": "Perform bad mode test",
		"modes": ["invalid_mode_name"]
	}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), badManifest, 0644); err != nil {
		t.Fatal(err)
	}
	findingsBad := LintSkillContract(tmpDir)
	foundModeErr := false
	for _, f := range findingsBad {
		if f.Rule == "valid_modes" {
			foundModeErr = true
		}
	}
	if !foundModeErr {
		t.Fatalf("expected valid_modes error, got: %v", findingsBad)
	}
}
