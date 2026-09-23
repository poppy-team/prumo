// Package perm implements the deterministic Permission Engine.
// Policy decides; the LLM never enforces. Decisions are persistable.
package perm

import (
	"crypto/sha256"
	"encoding/hex"
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

// Fingerprint identifies the content a decision was made about.
//
// It is the action, the resource, the filesystem scope, the network
// destinations and the credential scopes, hashed. Every field that could change
// what the call does is in it. An approval is for one call: the id is
// predictable (it is derived from the tool call id), so without binding the
// decision to the content, a later call reusing that id would inherit an
// approval a human gave for something else (GAP-107).
func Fingerprint(req agent.PermissionRequest) string {
	parts := []string{
		"action=" + req.Action,
		"resource=" + req.Resource,
		"args=" + req.ArgumentsSummary,
		"fs=" + strings.Join(req.FilesystemScope, ","),
		"net=" + strings.Join(req.NetworkDests, ","),
		"cred=" + strings.Join(req.CredentialScopes, ","),
		"data=" + req.DataClass,
		"reversibility=" + req.Reversibility,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

// Engine evaluates PermissionRequests deterministically.
type Engine struct {
	Policy Policy
	Log    []agent.PermissionResolution
	// resolutions indexes decisions already taken, so re-evaluating the same
	// request does not ask again. Without it an approval would be recorded and
	// then ignored, which is what left the permission gate unanswerable.
	resolutions map[string]agent.PermissionResolution
}

func New(p Policy) *Engine {
	if p.DefaultAction == "" {
		p.DefaultAction = agent.PermissionAsk
	}
	return &Engine{Policy: p, resolutions: map[string]agent.PermissionResolution{}}
}

// Evaluate returns a persistable resolution.
func (e *Engine) Evaluate(req agent.PermissionRequest, toolKind string, actor string) agent.PermissionResolution {
	fingerprint := Fingerprint(req)
	req.Fingerprint = fingerprint
	// A decision already taken for this request outranks the policy: that is
	// what makes an approval durable across a re-evaluation. It only applies to
	// the same content, though. A request id reused for a different call is a
	// different decision, so the old one is not consulted.
	if res, ok := e.resolutions[req.ID]; ok && res.Fingerprint == fingerprint {
		return res
	}
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
		Fingerprint: fingerprint,
	}
	e.remember(res)
	return res
}

// Approve records a human/approver allow for one specific request content.
//
// The caller passes the fingerprint it was shown. Requiring it is what stops
// an approval meant for one call from being applied to another that happens to
// carry the same request id.
func (e *Engine) Approve(reqID, fingerprint, actor string) (agent.PermissionResolution, error) {
	if fingerprint == "" {
		return agent.PermissionResolution{}, fmt.Errorf("approval requires the fingerprint of the request being approved")
	}
	return e.record(reqID, fingerprint, agent.PermissionAllow, "approved by "+actor, actor), nil
}

// Deny records a deny for one specific request content.
func (e *Engine) Deny(reqID, fingerprint, actor, reason string) (agent.PermissionResolution, error) {
	if fingerprint == "" {
		return agent.PermissionResolution{}, fmt.Errorf("denial requires the fingerprint of the request being denied")
	}
	if reason == "" {
		reason = "denied by " + actor
	}
	return e.record(reqID, fingerprint, agent.PermissionDeny, reason, actor), nil
}

func (e *Engine) record(reqID, fingerprint string, d agent.PermissionDecision, reason, actor string) agent.PermissionResolution {
	res := agent.PermissionResolution{
		RequestID: reqID, Decision: d, Reason: reason, Scope: "once",
		Actor: actor, DecidedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Fingerprint: fingerprint,
	}
	e.remember(res)
	return res
}

func (e *Engine) remember(res agent.PermissionResolution) {
	if e.resolutions == nil {
		e.resolutions = map[string]agent.PermissionResolution{}
	}
	e.resolutions[res.RequestID] = res
	e.Log = append(e.Log, res)
}

// Resolution returns a decision already recorded for a request, if any. The
// fingerprint must be supplied so a caller can tell whether the decision it is
// about to use was made about the content it now holds.
func (e *Engine) Resolution(reqID, fingerprint string) (agent.PermissionResolution, bool) {
	res, ok := e.resolutions[reqID]
	if !ok {
		return agent.PermissionResolution{}, false
	}
	if fingerprint != "" && res.Fingerprint != fingerprint {
		return agent.PermissionResolution{}, false
	}
	return res, true
}
