package completions

import (
	"strings"
	"testing"
)

func TestSlashCommandCompletionAllEntries(t *testing.T) {
	provider := NewSlashCommandContextGroup()
	entries, err := provider.GetChildEntries("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) < len(defaultSlashCommands) {
		t.Fatalf("expected at least %d entries, got %d", len(defaultSlashCommands), len(entries))
	}

	foundModel := false
	foundHelp := false
	foundProvider := false
	for _, entry := range entries {
		val := entry.GetValue()
		if strings.HasPrefix(val, "/model") {
			foundModel = true
		}
		if strings.HasPrefix(val, "/help") {
			foundHelp = true
		}
		if strings.HasPrefix(val, "/provider") {
			foundProvider = true
		}
	}

	if !foundModel || !foundHelp || !foundProvider {
		t.Fatalf("missing expected slash commands: model=%v, help=%v, provider=%v", foundModel, foundHelp, foundProvider)
	}
}

func TestSlashCommandCompletionFiltering(t *testing.T) {
	provider := NewSlashCommandContextGroup()
	entries, err := provider.GetChildEntries("mod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected matches for 'mod', got 0")
	}

	matched := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.GetValue(), "/model") {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("expected /model to match 'mod', got: %v", entries)
	}
}
