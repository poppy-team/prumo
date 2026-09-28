package cliops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectGo(t *testing.T) {
	tempDir := t.TempDir()
	goModContent := `module github.com/example/my-service

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/spf13/cobra v1.8.0
)
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\nfunc main(){}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "handler.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "service.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	det := DetectProject(tempDir)
	if det.Name != "my-service" {
		t.Errorf("expected project name 'my-service', got %q", det.Name)
	}
	foundGo := false
	for _, l := range det.Languages {
		if l == "go" {
			foundGo = true
		}
	}
	if !foundGo {
		t.Errorf("expected 'go' in languages, got %v", det.Languages)
	}
	if !det.IsExistingCodebase {
		t.Errorf("expected IsExistingCodebase to be true")
	}

	profile := BuildDefaultProfile(det, "standard")
	if len(profile.PreferredModels()) == 0 {
		t.Errorf("expected preferred models to be populated")
	}
}

func TestDetectProjectNode(t *testing.T) {
	tempDir := t.TempDir()
	pkgJSON := `{
  "name": "super-web-app",
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "tsconfig.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	det := DetectProject(tempDir)
	if det.Name != "super-web-app" {
		t.Errorf("expected 'super-web-app', got %q", det.Name)
	}
	foundTS := false
	for _, l := range det.Languages {
		if l == "typescript" {
			foundTS = true
		}
	}
	if !foundTS {
		t.Errorf("expected 'typescript' in languages, got %v", det.Languages)
	}
	foundWeb := false
	for _, typ := range det.AppTypes {
		if typ == "web" {
			foundWeb = true
		}
	}
	if !foundWeb {
		t.Errorf("expected 'web' in app types, got %v", det.AppTypes)
	}
}
