package completions

import (
	"fmt"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/raillen/prumo-tui/internal/commands"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
)

// SlashCommand describes a slash command available in the composer.
type SlashCommand struct {
	Name        string
	Description string
}

var defaultSlashCommands = []SlashCommand{
	{Name: "/help", Description: "Show help and keyboard shortcuts"},
	{Name: "/model", Description: "Select or switch active model"},
	{Name: "/provider", Description: "Configure or switch LLM provider"},
	{Name: "/session", Description: "Switch active session"},
	{Name: "/theme", Description: "Switch UI color theme"},
	{Name: "/files", Description: "Pick and add file reference"},
	{Name: "/diff", Description: "View changed files and diffs"},
	{Name: "/jobs", Description: "View scheduled daemon jobs"},
	{Name: "/sidebar", Description: "Toggle sidebar panel"},
	{Name: "/new", Description: "Start a new conversation session"},
	{Name: "/logs", Description: "View runtime logs"},
	{Name: "/init", Description: "Initialize AGENTS.md memory file"},
	{Name: "/export", Description: "Export session timeline"},
	{Name: "/compact", Description: "Session compaction information"},
	{Name: "/quit", Description: "Exit Prumo TUI"},
}

type slashCommandContextGroup struct {
	prefix string
}

func (cg *slashCommandContextGroup) GetId() string {
	return cg.prefix
}

func (cg *slashCommandContextGroup) GetEntry() dialog.CompletionItemI {
	return dialog.NewCompletionItem(dialog.CompletionItem{
		Title: "Slash Commands",
		Value: "/",
	})
}

func (cg *slashCommandContextGroup) GetChildEntries(query string) ([]dialog.CompletionItemI, error) {
	query = strings.TrimPrefix(strings.TrimSpace(query), "/")
	query = strings.ToLower(query)

	allCommands := append([]SlashCommand{}, defaultSlashCommands...)

	// Also load user commands from commands directory
	if userCmds, err := commands.Load(); err == nil {
		for _, u := range userCmds {
			allCommands = append(allCommands, SlashCommand{
				Name:        "/" + u.ID,
				Description: u.Title + " - " + u.Description,
			})
		}
	}

	var matched []SlashCommand
	if query == "" {
		matched = allCommands
	} else {
		for _, cmd := range allCommands {
			nameWithoutSlash := strings.TrimPrefix(cmd.Name, "/")
			nameLower := strings.ToLower(nameWithoutSlash)
			descLower := strings.ToLower(cmd.Description)

			if strings.HasPrefix(nameLower, query) ||
				fuzzy.Match(query, nameLower) ||
				fuzzy.Match(query, descLower) {
				matched = append(matched, cmd)
			}
		}
	}

	items := make([]dialog.CompletionItemI, 0, len(matched))
	for _, cmd := range matched {
		item := dialog.NewCompletionItem(dialog.CompletionItem{
			Title: fmt.Sprintf("%-12s  %s", cmd.Name, cmd.Description),
			Value: cmd.Name + " ",
		})
		items = append(items, item)
	}

	return items, nil
}

// NewSlashCommandContextGroup creates a CompletionProvider for slash commands.
func NewSlashCommandContextGroup() dialog.CompletionProvider {
	return &slashCommandContextGroup{
		prefix: "slash",
	}
}
