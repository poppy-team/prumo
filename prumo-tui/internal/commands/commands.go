// Package commands is the client's command surface: prompts a user writes once
// and reaches from the palette.
//
// It is deliberately the user's directory and not the repository's. A command
// is a prompt its author owns, and a project that has not decided what its
// command surface is should not have one imposed by its terminal client — so
// the client reads what the person running it wrote, and inventing nothing.
package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Command is one prompt the palette offers.
type Command struct {
	// ID is the file name without its extension, and is what identifies the
	// command in the palette and in the argument dialog.
	ID string
	// Title is what the palette shows.
	Title string
	// Description is the line under the title: what this command does.
	Description string
	// Body is the prompt, with `{{name}}` standing where an argument goes.
	Body string
	// Args are the names the dialog asks for, in order.
	Args []string
	// Path is where the file was read from, so a failure can name it.
	Path string
}

// ErrNoDirectory is returned when the client has nowhere to look, which is not
// an error a user needs to see: a client with no commands is the ordinary case.
var ErrNoDirectory = errors.New("no command directory is available")

// Dir returns the directory commands are read from.
//
// PRUMO_TUI_CONFIG_DIR moves it with the rest of the client's own settings, so
// a test can point at a directory it owns instead of the developer's home.
func Dir() string {
	if dir := os.Getenv("PRUMO_TUI_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "commands")
	}
	home, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "prumo-agent-tui", "commands")
}

// Load reads every command in the directory.
//
// A file that cannot be read is reported rather than skipped: a command that
// silently does not exist is indistinguishable from one that was never written,
// and the author would go looking for a mistake in the wrong place.
func Load() ([]Command, error) {
	dir := Dir()
	if dir == "" {
		return nil, ErrNoDirectory
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	loaded := make([]Command, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		command, err := parse(entry.Name(), path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		loaded = append(loaded, command)
	}
	sort.Slice(loaded, func(i, j int) bool { return loaded[i].ID < loaded[j].ID })
	return loaded, nil
}

// parse reads one command file.
//
// The format is markdown with an optional front block of `key: value` lines,
// because that is what a person writing a prompt reaches for: no schema to
// consult, and the body is prose either way.
func parse(name, path string) (Command, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Command{}, err
	}
	command := Command{
		ID:    strings.TrimSuffix(name, ".md"),
		Title: strings.TrimSuffix(name, ".md"),
		Path:  path,
	}

	body := string(data)
	if block, rest, ok := splitFrontBlock(body); ok {
		body = rest
		for _, line := range strings.Split(block, "\n") {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			switch key {
			case "title", "name":
				command.Title = value
			case "description":
				command.Description = value
			case "args":
				command.Args = parseList(value)
			}
		}
	}

	command.Body = strings.TrimSpace(body)
	if command.Body == "" {
		return Command{}, errors.New("the command has no prompt")
	}
	if command.Description == "" {
		// A palette entry with no description is an entry a user has to guess
		// at, which is the defect the palette's own rule refuses.
		command.Description = "A prompt from " + command.Path
	}
	return command, nil
}

// splitFrontBlock splits an optional leading block delimited by `---` lines.
func splitFrontBlock(body string) (block, rest string, ok bool) {
	trimmed := strings.TrimLeft(body, " \t\r\n")
	if !strings.HasPrefix(trimmed, "---") {
		return "", body, false
	}
	after := strings.TrimPrefix(trimmed, "---")
	end := strings.Index(after, "\n---")
	if end < 0 {
		return "", body, false
	}
	return strings.TrimSpace(after[:end]), strings.TrimSpace(after[end+4:]), true
}

// parseList reads `[a, b]` or `a, b` as a list of names.
func parseList(value string) []string {
	value = strings.Trim(value, "[]")
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// Expand fills the placeholders a prompt declares.
//
// A name with no value is left as it was written rather than replaced by an
// empty string: a prompt with a hole in it is one the author can see and fix,
// and a silently blank sentence is not.
func Expand(body string, args map[string]string) string {
	expanded := body
	for name, value := range args {
		if strings.TrimSpace(value) == "" {
			continue
		}
		expanded = strings.ReplaceAll(expanded, "{{"+name+"}}", value)
	}
	return expanded
}
