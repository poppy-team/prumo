// Package handoff implements typed pointer-first Handoff v2: refs, not
// transcripts. Cross-provider continuation never depends on transcripts.
package handoff

import (
	"fmt"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// Bundle is the canonical handoff unit.
type Bundle struct {
	Handoff agent.Handoff `json:"handoff"`
	// Required refs: run, checkpoint, context manifest, workspace rev.
	Refs map[string]string `json:"refs"`
}

// RequiredRefs lists refs a valid bundle must carry.
var RequiredRefs = []string{"run", "checkpoint", "context_manifest", "workspace_rev"}

// Build constructs a bundle from canonical state.
func Build(from, to string, state agent.NativeAgentState, workspaceRev, summary string, extra map[string]string) (Bundle, error) {
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return Bundle{}, fmt.Errorf("handoff requires from and to")
	}
	if state.RunID == "" {
		return Bundle{}, fmt.Errorf("handoff requires run_id")
	}
	refs := map[string]string{
		"run":              state.RunID,
		"checkpoint":       state.RunID + "-latest",
		"context_manifest": state.ContextManifestID,
		"workspace_rev":    workspaceRev,
	}
	for k, v := range extra {
		refs[k] = v
	}
	h := agent.Handoff{
		ID:   fmt.Sprintf("handoff-%s-%d", state.RunID, time.Now().UTC().UnixNano()),
		From: from, To: to, RunID: state.RunID,
		CheckpointID: refs["checkpoint"], WorkspaceRev: workspaceRev,
		ContextManifestID: state.ContextManifestID, Refs: refs,
		Summary: summary, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	return Bundle{Handoff: h, Refs: refs}, nil
}

// Validate enforces pointer-first completeness.
func (b Bundle) Validate() error {
	for _, k := range RequiredRefs {
		if strings.TrimSpace(b.Refs[k]) == "" {
			return fmt.Errorf("handoff bundle missing ref %q", k)
		}
	}
	if strings.TrimSpace(b.Handoff.From) == "" || strings.TrimSpace(b.Handoff.To) == "" {
		return fmt.Errorf("handoff requires from/to")
	}
	return nil
}
