package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/protocol"
)

func TestDocsResourcesListAndRead(t *testing.T) {
	root := testRepoRoot(t)
	code, out := captureOutput(func() int {
		return runDocumentation(true, []string{"docs", "resources", "list", "--path", root})
	})
	if code != exitOK {
		t.Fatalf("resources list failed: %d %s", code, out)
	}
	var env protocol.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output must be a JSON envelope: %v\n%s", err, out)
	}
	data, _ := env.Data.(map[string]any)
	if count, _ := data["count"].(float64); count == 0 {
		t.Fatalf("expected documentation resources: %s", out)
	}
	readCode, readOut := captureOutput(func() int {
		return runDocumentation(false, []string{"docs", "resources", "read", "prumo://docs/product/vision", "--path", root})
	})
	if readCode != exitOK {
		t.Fatalf("resources read failed: %d %s", readCode, readOut)
	}
	if !strings.Contains(readOut, "Vision") {
		t.Fatalf("read must return Markdown: %s", readOut)
	}
	missingCode, _ := captureOutput(func() int {
		return runDocumentation(false, []string{"docs", "resources", "read", "prumo://docs/nope", "--path", root})
	})
	if missingCode == exitOK {
		t.Fatal("reading an unknown resource must fail")
	}
	usageCode, _ := captureOutput(func() int {
		return runDocumentation(false, []string{"docs", "resources", "read", "--path", root})
	})
	if usageCode != exitUsage {
		t.Fatalf("read without a uri must be a usage error, got %d", usageCode)
	}
}

func TestDocsResourcesToolsAreReadOnly(t *testing.T) {
	code, out := captureOutput(func() int {
		return runDocumentation(true, []string{"docs", "resources", "tools", "--path", testRepoRoot(t)})
	})
	if code != exitOK {
		t.Fatalf("resources tools failed: %d %s", code, out)
	}
	var env protocol.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	data, _ := env.Data.(map[string]any)
	tools, _ := data["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("expected the documentation surface to advertise tools")
	}
	for _, raw := range tools {
		tool, _ := raw.(map[string]any)
		if tool["kind"] != "read-only" {
			t.Fatalf("documentation tool must be read-only: %#v", tool)
		}
	}
}

func TestDocsMutationsDescribesTheGate(t *testing.T) {
	code, out := captureOutput(func() int {
		return runDocumentation(false, []string{"docs", "mutations", "--path", testRepoRoot(t)})
	})
	if code != exitOK {
		t.Fatalf("docs mutations failed: %d %s", code, out)
	}
	if !strings.Contains(out, "all_verbs") || !strings.Contains(out, "enabled") {
		t.Fatalf("the gate report must name the verb vocabulary and what is enabled: %s", out)
	}
	// The repository ships the writer disabled.
	if !strings.Contains(out, "\"require_approval\": true") {
		t.Fatalf("the repository must require approval before documentation changes: %s", out)
	}
}
