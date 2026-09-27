package antigravity_test

import (
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/connectors/antigravity"
	"github.com/raillen/prumo/internal/connectors/testkit"
)

func TestAntigravityConnector(t *testing.T) {
	c := antigravity.NewConnector()
	testkit.RunAll(t, c)
}

func TestAntigravitySubagents(t *testing.T) {
	c := antigravity.NewConnector()
	tmp := t.TempDir()

	res, err := c.Compile(tmp, connectors.CompileOptions{})
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	expectedSubagents := []string{
		"architect.md",
		"executor.md",
		"verifier.md",
		"systems-architect.md",
		"isolation-auditor.md",
		"brand-designer.md",
		"creative-director.md",
		"svg-artist.md",
		"advertising-designer.md",
		"motion-designer.md",
		"ui-component-engineer.md",
		"visual-identity-auditor.md",
		"prototyper.md",
	}

	createdMap := make(map[string]bool)
	for _, p := range res.CreatedPaths {
		createdMap[p] = true
	}

	for _, sub := range expectedSubagents {
		target := filepath.Join(tmp, ".agents", "subagents", sub)
		if !createdMap[target] {
			t.Errorf("missing expected subagent in compile output: %s", sub)
		}
	}
}

