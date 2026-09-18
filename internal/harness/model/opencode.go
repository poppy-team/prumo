package model

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
)

// OpenCode delegates a whole turn to the opencode CLI.
//
// It is a ModelProvider with a difference that has to be said out loud rather
// than hidden behind the interface: the tools it calls are *its own*. Prumo
// hands it a goal, opencode runs its loop (its tools, its permission policy, its
// network), and what comes back is the answer and what it cost. So this provider
// never reports tool calls — forwarding them would make Prumo execute tools that
// were already executed, and claiming them would attribute a delegated decision
// to a policy that never saw it.
//
// The wire format is `opencode run --format json`, which emits one JSON object
// per line:
//
//	{"type":"text","timestamp":…,"sessionID":…,"part":{"type":"text","text":…}}
//	{"type":"reasoning",…,"part":{…}}
//	{"type":"tool_use",…,"part":{…}}
//	{"type":"step_finish",…,"part":{"tokens":{"input":…,"output":…,"cache":{"read":…,"write":…}},"cost":…}}
//	{"type":"error","timestamp":…,"sessionID":…,"error":{"name":…,"data":{"message":…}}}
//
// Shapes read from the installed binary (opencode 1.18.31) rather than guessed:
// the emitter is `{type, timestamp, sessionID, ...extra}`. An event this code
// does not recognise is ignored, never guessed at — the alternative is a
// transcript that says something the provider did not.
type OpenCode struct {
	// Binary is the executable; empty means `opencode`, or
	// $PRUMO_OPENCODE_BIN when that is set.
	Binary string
	// Model is a provider/model id, e.g. opencode/mimo-v2.5-free. Empty leaves
	// the choice to opencode's own configuration.
	Model string
	// Dir is the directory the delegated run works in.
	Dir string
}

const defaultOpenCodeBinary = "opencode"

// SetWorkspace tells the provider which directory a delegated run must work in.
//
// The provider factory has no workspace to give (a provider is built from a
// name, a URL and a key), so the daemon hands it over here instead of widening a
// signature that model discovery and health do not need.
func (o *OpenCode) SetWorkspace(dir string) { o.Dir = dir }

func NewOpenCode(mdl string) *OpenCode {
	bin := strings.TrimSpace(os.Getenv("PRUMO_OPENCODE_BIN"))
	if bin == "" {
		bin = defaultOpenCodeBinary
	}
	return &OpenCode{Binary: bin, Model: mdl, Dir: strings.TrimSpace(os.Getenv("PRUMO_OPENCODE_DIR"))}
}

func (o *OpenCode) Name() string { return "opencode" }

// Capabilities describes what this provider does, not what opencode could do:
// it streams text, reports usage, can be cancelled, and does **not** hand tool
// calls back — it runs them itself.
func (o *OpenCode) Capabilities() Capabilities {
	return Capabilities{Streaming: true, ToolCalls: false, Usage: true, Cancel: true, Health: true, ModelDiscovery: true}
}

// Models lists what this opencode reports it can run.
//
// The catalogue is opencode's, including models that need a subscription the
// user may not have: it is what the tool would offer, and a filtered list would
// hide the reason a choice fails later.
func (o *OpenCode) Models(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, o.binary(), "models").Output()
	if err != nil {
		return nil, fmt.Errorf("opencode models: %w", err)
	}
	var models []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, " ") {
			models = append(models, line)
		}
	}
	sort.Strings(models)
	return models, nil
}

func (o *OpenCode) Health(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, o.binary(), "--version").CombinedOutput()
	version := strings.TrimSpace(string(out))
	if err != nil {
		if version != "" {
			return "", fmt.Errorf("opencode unavailable: %s", version)
		}
		return "", fmt.Errorf("opencode unavailable: %w", err)
	}
	if version == "" {
		return "available (version unreported)", nil
	}
	return "available " + version, nil
}

// Stream runs one delegated turn and translates opencode's events into the
// model events the runtime understands.
func (o *OpenCode) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	prompt := lastUserText(req.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("opencode needs a goal to run: no user message in the request")
	}

	args := []string{"run", "--format", "json"}
	if mdl := firstNonEmpty(req.Model, o.Model); mdl != "" {
		args = append(args, "--model", mdl)
	}
	if o.Dir != "" {
		args = append(args, "--dir", o.Dir)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, o.binary(), args...)
	if o.Dir != "" {
		cmd.Dir = o.Dir
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("opencode: %w", err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot start opencode: %w", err)
	}

	ch := make(chan agent.ModelEvent, 64)
	go func() {
		defer close(ch)

		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
		finish := ""
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			var ev struct {
				Type string `json:"type"`
				Part *struct {
					Text   string `json:"text"`
					Reason string `json:"reason"`
					Tokens *struct {
						Input  int `json:"input"`
						Output int `json:"output"`
						Cache  *struct {
							Read  int `json:"read"`
							Write int `json:"write"`
						} `json:"cache"`
					} `json:"tokens"`
					Cost float64 `json:"cost"`
				} `json:"part"`
				Error *struct {
					Name string `json:"name"`
					Data *struct {
						Message string `json:"message"`
						Ref     string `json:"ref"`
					} `json:"data"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				// A line that is not JSON is not an event this provider
				// understands; saying so beats inventing one.
				select {
				case ch <- agent.ModelEvent{Kind: agent.EventWarning, RequestID: req.RequestID,
					Error: "opencode emitted a line that is not an event"}:
				case <-ctx.Done():
					return
				}
				continue
			}
			switch ev.Type {
			case "text":
				if ev.Part != nil && ev.Part.Text != "" {
					if !emit(ctx, ch, agent.ModelEvent{Kind: agent.EventTextDelta, RequestID: req.RequestID, Text: ev.Part.Text}) {
						return
					}
				}
			case "reasoning":
				if ev.Part != nil && ev.Part.Text != "" {
					if !emit(ctx, ch, agent.ModelEvent{Kind: agent.EventReasoningDelta, RequestID: req.RequestID, Text: ev.Part.Text}) {
						return
					}
				}
			case "step_finish":
				if ev.Part != nil {
					if ev.Part.Reason != "" {
						finish = ev.Part.Reason
					}
					if ev.Part.Tokens != nil || ev.Part.Cost != 0 {
						usage := &agent.Usage{CostUSD: ev.Part.Cost}
						if t := ev.Part.Tokens; t != nil {
							usage.InputTokens, usage.OutputTokens = t.Input, t.Output
							if t.Cache != nil {
								usage.CacheReadTokens, usage.CacheWriteTokens = t.Cache.Read, t.Cache.Write
							}
						}
						if !emit(ctx, ch, agent.ModelEvent{Kind: agent.EventUsageUpdated, RequestID: req.RequestID, Usage: usage}) {
							return
						}
					}
				}
			case "tool_use":
				// Deliberately not translated: opencode ran that tool itself.
				// Forwarding it as a tool call would make Prumo run it a second
				// time, and forwarding it as text would put words in the model's
				// mouth.
			case "error":
				msg := "opencode reported an error"
				if ev.Error != nil {
					msg = "opencode: " + ev.Error.Name
					if ev.Error.Data != nil && ev.Error.Data.Message != "" {
						msg = "opencode: " + ev.Error.Data.Message
						if ev.Error.Data.Ref != "" {
							msg += " (" + ev.Error.Data.Ref + ")"
						}
					}
				}
				emit(ctx, ch, agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: msg})
				return
			}
		}
		scanErr := sc.Err()
		// The process state is only known once Wait returns, so it is called
		// here rather than deferred: a deferred Wait would be read as "the
		// process ended cleanly" every single time.
		waitErr := cmd.Wait()
		if scanErr != nil {
			emit(ctx, ch, agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
				Error: "opencode stream ended early: " + scanErr.Error()})
			return
		}
		// The process is the turn: an exit without an error event is a finished
		// turn, and one with a non-zero status is reported with what it said,
		// because "the run ended" and "the run worked" are different facts.
		if waitErr != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = "opencode: " + waitErr.Error()
			}
			emit(ctx, ch, agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: msg})
			return
		}
		// `finish` is opencode's own reason for stopping. It is read so a future
		// event can carry it; today the run's stop reason is the runtime's to
		// decide, and writing it into the completed event's Text would put a
		// reason where the contract says prose goes.
		_ = finish
		emit(ctx, ch, agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true})
	}()

	return ch, nil
}

func (o *OpenCode) binary() string {
	if strings.TrimSpace(o.Binary) != "" {
		return o.Binary
	}
	return defaultOpenCodeBinary
}

func emit(ctx context.Context, ch chan<- agent.ModelEvent, ev agent.ModelEvent) bool {
	select {
	case ch <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

// lastUserText is the goal this delegated turn runs: the newest thing the person
// asked. The conversation before it belongs to Prumo's history, and opencode
// keeps its own.
func lastUserText(messages []agent.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == agent.RoleUser && strings.TrimSpace(messages[i].Content) != "" {
			return messages[i].Content
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
