package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCommand(t *testing.T, name, body string) {
	t.Helper()
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("cannot create the command directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("cannot write the command: %v", err)
	}
}

// The surface is the user's own directory: a command is a prompt its author
// owns, and the client invents nothing.
func TestCommandsAreReadFromTheUsersDirectory(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	writeCommand(t, "review.md", `---
title: Review the working tree
description: Look for problems before committing
args: [area]
---
Look at the {{area}} area and report what is wrong.
`)
	writeCommand(t, "notes.md", "Summarise the open questions in this repository.\n")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded %d command(s), want 2", len(loaded))
	}
	// Sorted by id, so the palette is the same on every run.
	if loaded[0].ID != "notes" || loaded[1].ID != "review" {
		t.Fatalf("commands are not in a stable order: %+v", loaded)
	}

	review := loaded[1]
	if review.Title != "Review the working tree" || review.Description != "Look for problems before committing" {
		t.Fatalf("the front block was not read: %+v", review)
	}
	if len(review.Args) != 1 || review.Args[0] != "area" {
		t.Fatalf("the declared arguments were not read: %+v", review.Args)
	}
	if !strings.HasPrefix(review.Body, "Look at the {{area}} area") {
		t.Fatalf("the body kept its front block: %q", review.Body)
	}
}

// A client with no commands is the ordinary case, not a broken one.
func TestAnEmptyDirectoryIsNotAFailure(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	loaded, err := Load()
	if err != nil {
		t.Fatalf("an empty directory failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("found %d command(s) in an empty directory", len(loaded))
	}
}

// A command nobody can explain is a command a user has to guess at, which is the
// defect the palette's own rule refuses.
func TestACommandWithoutADescriptionSaysWhereItCameFrom(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	writeCommand(t, "bare.md", "Do the thing.\n")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Description == "" {
		t.Fatalf("a command with no description reached the palette: %+v", loaded)
	}
	if !strings.Contains(loaded[0].Description, "bare.md") {
		t.Fatalf("the description does not say where it came from: %q", loaded[0].Description)
	}
}

// A file that cannot be parsed is reported rather than skipped: a command the
// author wrote and cannot find is a mistake they would look for in the wrong
// place.
func TestACommandWithNoPromptIsRefused(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	writeCommand(t, "empty.md", "---\ntitle: Nothing\n---\n")

	if _, err := Load(); err == nil {
		t.Fatal("a command with no prompt was accepted")
	}
}

// An unfilled placeholder stays visible: a prompt with a hole in it is one the
// author can see and fix, and a silently blank sentence is not.
func TestExpansionLeavesUnfilledPlaceholdersAlone(t *testing.T) {
	got := Expand("Look at {{area}} and {{what}}.", map[string]string{"area": "the parser", "what": "   "})

	if !strings.Contains(got, "Look at the parser and") {
		t.Fatalf("the filled placeholder was not replaced: %q", got)
	}
	if !strings.Contains(got, "{{what}}") {
		t.Fatalf("an unfilled placeholder was silently emptied: %q", got)
	}
}
