package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/protocol"
)

// AskOptions encapsulates parsed options for `prumo ask`.
type AskOptions struct {
	Prompt       string
	Files        []string
	ContextType  string
	Model        string
	Provider     string
	BaseURL      string
	APIKey       string
	SystemPrompt string
	Raw          bool
	AsJSON       bool
	StdinContent string
}

func parseAskArgs(asJSON bool, args []string) (AskOptions, []string, error) {
	opts := AskOptions{
		AsJSON:   asJSON,
		Provider: "fake",
	}

	positional := make([]string, 0)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			opts.AsJSON = true
		case "--raw":
			opts.Raw = true
		case "--file":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--file requires a path")
			}
			opts.Files = append(opts.Files, args[i+1])
			i++
		case "--context":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--context requires a value")
			}
			opts.ContextType = args[i+1]
			i++
		case "--model":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--model requires a value")
			}
			opts.Model = args[i+1]
			i++
		case "--provider":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--provider requires a value")
			}
			opts.Provider = args[i+1]
			i++
		case "--base-url":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--base-url requires a value")
			}
			opts.BaseURL = args[i+1]
			i++
		case "--api-key":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--api-key requires a value")
			}
			opts.APIKey = args[i+1]
			i++
		case "--system":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("--system requires a value")
			}
			opts.SystemPrompt = args[i+1]
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return opts, nil, fmt.Errorf("unknown flag %s", arg)
			}
			positional = append(positional, arg)
		}
	}

	opts.Prompt = strings.Join(positional, " ")
	return opts, positional, nil
}

func runAsk(asJSON bool, args []string) int {
	opts, _, err := parseAskArgs(asJSON, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitUsage
	}

	// Check if stdin is a pipe
	if fi, statErr := os.Stdin.Stat(); statErr == nil && (fi.Mode()&os.ModeCharDevice == 0) {
		bytes, readErr := io.ReadAll(os.Stdin)
		if readErr == nil && len(bytes) > 0 {
			opts.StdinContent = string(bytes)
		}
	}

	if opts.Prompt == "" && opts.StdinContent == "" {
		fmt.Fprintf(os.Stderr, "error: prompt or stdin input required for prumo ask\n")
		return exitUsage
	}

	// Assemble file context
	var fileContext strings.Builder
	for _, fPath := range opts.Files {
		content, readErr := os.ReadFile(fPath)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "error: failed to read file %s: %v\n", fPath, readErr)
			return exitUnavailable
		}
		fileContext.WriteString(fmt.Sprintf("\n--- File: %s ---\n%s\n", filepath.Clean(fPath), string(content)))
	}

	// Combine into prompt
	var userPrompt strings.Builder
	if opts.StdinContent != "" {
		userPrompt.WriteString("--- Stdin Input ---\n")
		userPrompt.WriteString(opts.StdinContent)
		userPrompt.WriteString("\n")
	}
	if fileContext.Len() > 0 {
		userPrompt.WriteString(fileContext.String())
		userPrompt.WriteString("\n")
	}
	if opts.Prompt != "" {
		userPrompt.WriteString(opts.Prompt)
	}

	// Resolve provider
	provider, err := model.ForName(opts.Provider, opts.BaseURL, opts.APIKey, opts.Model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to initialize model provider %s: %v\n", opts.Provider, err)
		return exitValidation
	}

	messages := make([]agent.Message, 0, 2)
	if opts.SystemPrompt != "" {
		messages = append(messages, agent.Message{
			Role:    "system",
			Content: opts.SystemPrompt,
		})
	}
	messages = append(messages, agent.Message{
		Role:    "user",
		Content: userPrompt.String(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := provider.Stream(ctx, agent.ModelRequest{
		Model:    opts.Model,
		Messages: messages,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: model call failed: %v\n", err)
		return exitValidation
	}

	var responseBuilder strings.Builder
	for ev := range stream {
		switch ev.Kind {
		case agent.EventTextDelta:
			responseBuilder.WriteString(ev.Text)
		case agent.EventError:
			fmt.Fprintf(os.Stderr, "error: model error: %s\n", ev.Error)
			return exitValidation
		}
	}

	finalText := responseBuilder.String()

	if opts.AsJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"prompt":   opts.Prompt,
			"response": finalText,
			"model":    opts.Model,
			"provider": opts.Provider,
		}))
	}

	if opts.Raw {
		fmt.Print(finalText)
	} else {
		fmt.Println(finalText)
	}

	return exitOK
}
