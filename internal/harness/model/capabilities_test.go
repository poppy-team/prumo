package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDeclarations(t *testing.T, body string) string {
	t.Helper()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".prumo"), 0o755); err != nil {
		t.Fatalf("cannot create .prumo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, DefaultDeclarationsPath), []byte(body), 0o644); err != nil {
		t.Fatalf("cannot write the declarations: %v", err)
	}
	return workspace
}

// The harness serves models it was told about and claims nothing about the rest:
// a missing file is the ordinary case, not a failure.
func TestAWorkspaceWithNoDeclarationsIsOrdinary(t *testing.T) {
	declared, err := LoadDeclarations(t.TempDir())
	if err != nil {
		t.Fatalf("a workspace with no declarations failed: %v", err)
	}
	if len(declared.Models) != 0 {
		t.Fatalf("found declarations in an empty workspace: %+v", declared)
	}
}

func TestDeclarationsAreReadAndPairedWithTheServedModels(t *testing.T) {
	workspace := writeDeclarations(t, `{
  "version": 1,
  "models": {
    "vision-model": {"text": true, "vision": true, "tools": true, "context_tokens": 200000},
    "plain-model": {"text": true}
  }
}`)
	declared, err := LoadDeclarations(workspace)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	infos := Describe([]string{"vision-model", "undeclared-model", "plain-model"}, declared)
	if len(infos) != 3 {
		t.Fatalf("described %d model(s), want 3", len(infos))
	}
	// The order is the provider's, so the list a user sees does not reshuffle.
	if infos[0].ID != "vision-model" || infos[2].ID != "plain-model" {
		t.Fatalf("the served order was not kept: %+v", infos)
	}
	if !infos[0].Declared || !infos[0].Capabilities.Vision || infos[0].Capabilities.ContextTokens != 200000 {
		t.Fatalf("the declaration was lost: %+v", infos[0])
	}
	// A model nobody declared is undeclared, not a model without features.
	if infos[1].Declared {
		t.Fatalf("a model with no declaration claims one: %+v", infos[1])
	}
	if len(infos[1].Capabilities.Features()) != 0 {
		t.Fatalf("an undeclared model reports features: %+v", infos[1])
	}
}

// The features read in a fixed order, so the same model always reads the same
// way — and an undeclared one reads as nothing rather than as a list of denials.
func TestFeaturesReadTheSameWayEveryTime(t *testing.T) {
	all := CapabilitySet{Text: true, Reasoning: true, Vision: true, Tools: true, Audio: true}
	if got := strings.Join(all.Features(), " "); got != "text reasoning vision tools audio" {
		t.Fatalf("features = %q", got)
	}
	onlyVision := CapabilitySet{Vision: true}
	if got := strings.Join(onlyVision.Features(), " "); got != "vision" {
		t.Fatalf("features = %q", got)
	}
	if got := (CapabilitySet{}).Features(); len(got) != 0 {
		t.Fatalf("an empty declaration reported %v", got)
	}
}

// A file that cannot be parsed is reported rather than ignored: declarations
// that were silently skipped would let a model be sent an image it cannot read.
func TestUnreadableDeclarationsAreReported(t *testing.T) {
	workspace := writeDeclarations(t, "{not json")
	if _, err := LoadDeclarations(workspace); err == nil {
		t.Fatal("a declarations file that cannot be parsed was accepted")
	}
}
