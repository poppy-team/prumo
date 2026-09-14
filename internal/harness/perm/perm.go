// Package perm implements the deterministic Permission Engine.
// Policy decides; the LLM never enforces. Decisions are persistable.
package perm

import (
	"fmt"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// Policy is a deterministic allow/ask/deny rule set.
type Policy struct {
	// DefaultAction applies when no rule matches: allow|ask|deny.
	DefaultAction agent.PermissionDecision `json:"default_action"`
	// DenyPrefixes blocks resource prefixes (e.g. "/etc", "..").
	DenyPrefixes []string `json:"deny_prefixes,omitempty"`
	// AskKinds forces ask for tool kinds (e.g. side-effecting).
	AskKinds []string `json:"ask_kinds,omitempty"`
	// AllowActions exact-matches safe actions.
	AllowActions []string `json:"allow_actions,omitempty"`
}

// Engine evaluates PermissionRequests deterministically.
type Engine struct {
	Policy Policy
	Log    []agent.PermissionResolution
}

func New(p Policy) *Engine {
	if p.DefaultAction == "" {
		p.DefaultAction = agent.PermissionAsk
	}
	return &Engine{Policy: p}
}

// Evaluate returns a persistable resolution.
func (e *Engine) Evaluate(req agent.PermissionRequest, toolKind string, actor string) agent.PermissionResolution {
	decision := e.Policy.DefaultAction
	reason := "default policy"
	for _, a := range e.Policy.AllowActions {
		if a == req.Action {
			decision = agent.PermissionAllow
			reason = "explicit allow list"
		}
	}
	for _, prefix := range e.Policy.DenyPrefixes {
		if prefix != "" && strings.Contains(req.Resource, prefix) {
			decision = agent.PermissionDeny
			reason = fmt.Sprintf("resource matches deny prefix %q", prefix)
		}
	}
	for _, k := range e.Policy.AskKinds {
		if k == toolKind && decision == agent.PermissionAllow {
			decision = agent.PermissionAsk
			reason = fmt.Sprintf("kind %q requires approval", toolKind)
		}
	}
	// Destructive tool calls are never auto-allowed unless explicitly listed.
	if toolKind == "destructive" && decision == agent.PermissionAllow {
		listed := false
		for _, a := range e.Policy.AllowActions {
			if a == req.Action {
				listed = true
			}
		}
		if !listed {
			decision = agent.PermissionAsk
			reason = "destructive action requires approval"
		}
	}
	res := agent.PermissionResolution{
		RequestID: req.ID, Decision: decision, Reason: reason,
		Scope: "once", Actor: actor, DecidedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	e.Log = append(e.Log, res)
	return res
}

// Approve records a human/approver allow; Deny records a deny.
func (e *Engine) Approve(reqID, actor string) agent.PermissionResolution {
	return e.record(reqID, agent.PermissionAllow, "approved by "+actor, actor)
}

func (e *Engine) Deny(reqID, actor, reason string) agent.PermissionResolution {
	if reason == "" {
		reason = "denied by " + actor
	}
	return e.record(reqID, agent.PermissionDeny, reason, actor)
}

func (e *Engine) record(reqID string, d agent.PermissionDecision, reason, actor string) agent.PermissionResolution {
	res := agent.PermissionResolution{RequestID: reqID, Decision: d, Reason: reason, Scope: "once", Actor: actor, DecidedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	e.Log = append(e.Log, res)
	return res
}
