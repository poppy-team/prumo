// Package agent defines HA0 canonical contracts for the Prumo Harness.
//
// Design authority: notebook pages 03/04/05/19/35 (ACCEPTED) reconciled
// with repository reality (internal/runtime, internal/toolgateway,
// internal/budget, internal/modelregistry). Vendor types must not escape
// adapters; see internal/harness/model and internal/harness/extagent.
package agent

import (
	"context"
	"time"
)

// Phase is a reentrant NativeAgent state-machine step. See 05.
type Phase string

const (
	PhasePrepare           Phase = "prepare"
	PhaseCompileContext    Phase = "compile_context"
	PhaseRequestModel      Phase = "request_model"
	PhaseConsumeModelEvent Phase = "consume_model_event"
	PhasePlanToolCalls     Phase = "plan_tool_calls"
	PhasePermissionCheck   Phase = "permission_check"
	PhaseExecuteTool       Phase = "execute_tool"
	PhaseRecordObservation Phase = "record_observation"
	PhaseEvaluateStop      Phase = "evaluate_stop"
	PhaseCheckpoint        Phase = "checkpoint"
	PhaseYield             Phase = "yield"
	PhaseComplete          Phase = "complete"
	PhaseFailed            Phase = "failed"
)

// Role distinguishes message authors without leaking vendor roles.
type Role string

const (
	RoleSystem   Role = "system"
	RoleUser     Role = "user"
	RoleAgent    Role = "agent"
	RoleTool     Role = "tool"
	RoleApprover Role = "approver"
)

// Message is a normalized conversation unit.
type Message struct {
	ID        string         `json:"id"`
	TurnID    string         `json:"turn_id,omitempty"`
	Role      Role           `json:"role"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// Turn groups one model call + its tool calls + observations.
type Turn struct {
	ID          string   `json:"id"`
	RunID       string   `json:"run_id"`
	SessionID   string   `json:"session_id"`
	Index       int      `json:"index"`
	MessageIDs  []string `json:"message_ids,omitempty"`
	ToolCallIDs []string `json:"tool_call_ids,omitempty"`
	Status      string   `json:"status"`
	StartedAt   string   `json:"started_at"`
	EndedAt     string   `json:"ended_at,omitempty"`
}

// ToolCall is a vendor-neutral tool invocation request.
type ToolCall struct {
	ID        string         `json:"id"`
	TurnID    string         `json:"turn_id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	// IdempotencyKey is required before any observable side effect.
	IdempotencyKey string `json:"idempotency_key"`
}

// ToolResult is the bounded, normalized outcome of a ToolCall.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output"`
	Truncated  bool   `json:"truncated,omitempty"`
	ArtifactID string `json:"artifact_id,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Observation records a ToolResult into conversation history.
type Observation struct {
	ID         string `json:"id"`
	TurnID     string `json:"turn_id"`
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
}

// PermissionDecision is the deterministic policy outcome.
type PermissionDecision string

const (
	PermissionAllow PermissionDecision = "allow"
	PermissionAsk   PermissionDecision = "ask"
	PermissionDeny  PermissionDecision = "deny"
)

// PermissionRequest carries enough context for policy without LLM enforcement.
type PermissionRequest struct {
	ID               string   `json:"id"`
	RunID            string   `json:"run_id"`
	TurnID           string   `json:"turn_id"`
	Agent            string   `json:"agent,omitempty"`
	Action           string   `json:"action"`
	Resource         string   `json:"resource,omitempty"`
	ArgumentsSummary string   `json:"arguments_summary,omitempty"`
	FilesystemScope  []string `json:"filesystem_scope,omitempty"`
	NetworkDests     []string `json:"network_dests,omitempty"`
	CredentialScopes []string `json:"credential_scopes,omitempty"`
	DataClass        string   `json:"data_class,omitempty"`
	Reversibility    string   `json:"reversibility,omitempty"`
	Risk             string   `json:"risk,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

// PermissionResolution is the persisted approval event.
type PermissionResolution struct {
	RequestID   string             `json:"request_id"`
	Decision    PermissionDecision `json:"decision"`
	Reason      string             `json:"reason,omitempty"`
	Scope       string             `json:"scope,omitempty"`
	Constraints []string           `json:"constraints,omitempty"`
	Actor       string             `json:"actor,omitempty"`
	DecidedAt   string             `json:"decided_at"`
}

// ModelRequest is what the NativeAgent asks a ModelProvider to do.
type ModelRequest struct {
	RequestID string     `json:"request_id"`
	RunID     string     `json:"run_id"`
	TurnID    string     `json:"turn_id"`
	Model     string     `json:"model"`
	Messages  []Message  `json:"messages"`
	Tools     []ToolSpec `json:"tools,omitempty"`
	// ResponseFormat, when set, requests JSON-schema-shaped output
	// (OpenAI json_schema; other providers best-effort, documented).
	ResponseFormat map[string]any `json:"response_format,omitempty"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	Temperature    float64        `json:"temperature,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// ToolSpec advertises one callable tool to the model.
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema,omitempty"`
}

// ModelEventKind normalizes streaming events across vendors.
type ModelEventKind string

const (
	EventTextDelta      ModelEventKind = "text_delta"
	EventReasoningDelta ModelEventKind = "reasoning_delta"
	EventToolCallDelta  ModelEventKind = "tool_call_delta"
	EventToolCallReady  ModelEventKind = "tool_call_ready"
	EventUsageUpdated   ModelEventKind = "usage_updated"
	EventWarning        ModelEventKind = "warning"
	EventError          ModelEventKind = "error"
	EventCompleted      ModelEventKind = "completed"
	EventCancelled      ModelEventKind = "cancelled"
)

// ModelEvent is a single normalized stream unit.
type ModelEvent struct {
	Kind      ModelEventKind `json:"kind"`
	RequestID string         `json:"request_id"`
	Text      string         `json:"text,omitempty"`
	ToolCall  *ToolCall      `json:"tool_call,omitempty"`
	Usage     *Usage         `json:"usage,omitempty"`
	Error     string         `json:"error,omitempty"`
	Retryable bool           `json:"retryable,omitempty"`
	Finished  bool           `json:"finished,omitempty"`
}

// Usage normalizes token/cost telemetry.
type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

// NativeAgentState is the canonical resumable state. Provider-private
// reasoning is never required here.
type NativeAgentState struct {
	RunID             string         `json:"run_id"`
	SessionID         string         `json:"session_id"`
	TurnID            string         `json:"turn_id"`
	Phase             Phase          `json:"phase"`
	Revision          int            `json:"revision"`
	ContextManifestID string         `json:"context_manifest_id,omitempty"`
	ModelRoute        string         `json:"model_route,omitempty"`
	PendingRequest    *ModelRequest  `json:"pending_request,omitempty"`
	PendingTools      []ToolCall     `json:"pending_tools,omitempty"`
	PendingPerms      []string       `json:"pending_permissions,omitempty"`
	StopReason        string         `json:"stop_reason,omitempty"`
	Budget            map[string]any `json:"budget,omitempty"`
	UpdatedAt         string         `json:"updated_at"`
}

// Checkpoint persists a safe-point snapshot.
type Checkpoint struct {
	ID           string           `json:"id"`
	RunID        string           `json:"run_id"`
	State        NativeAgentState `json:"state"`
	WorkspaceRev string           `json:"workspace_rev,omitempty"`
	CreatedAt    string           `json:"created_at"`
}

// Continuation is the pointer-first resume bundle (no full transcript).
type Continuation struct {
	RunID             string            `json:"run_id"`
	CheckpointID      string            `json:"checkpoint_id"`
	ContextManifestID string            `json:"context_manifest_id,omitempty"`
	Summary           string            `json:"summary,omitempty"`
	Refs              map[string]string `json:"refs,omitempty"`
}

// Handoff carries typed refs across agents/providers.
type Handoff struct {
	ID                string            `json:"id"`
	From              string            `json:"from"`
	To                string            `json:"to"`
	RunID             string            `json:"run_id"`
	GoalID            string            `json:"goal_id,omitempty"`
	CheckpointID      string            `json:"checkpoint_id,omitempty"`
	WorkspaceRev      string            `json:"workspace_rev,omitempty"`
	ContextManifestID string            `json:"context_manifest_id,omitempty"`
	Refs              map[string]string `json:"refs,omitempty"`
	Summary           string            `json:"summary,omitempty"`
	CreatedAt         string            `json:"created_at"`
}

// AgentEvent is the normalized timeline unit for observability.
type AgentEvent struct {
	ID        string         `json:"id"`
	RunID     string         `json:"run_id"`
	TurnID    string         `json:"turn_id,omitempty"`
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// DomainEvent is the external (protocol) projection of AgentEvent.
type DomainEvent struct {
	Type      string         `json:"type"`
	RunID     string         `json:"run_id"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// EffectStatus tracks pending observable side effects.
type EffectStatus string

const (
	EffectPending    EffectStatus = "pending"
	EffectApplied    EffectStatus = "applied"
	EffectFailed     EffectStatus = "failed"
	EffectRolledBack EffectStatus = "rolled_back"
)

// PendingEffect has idempotency + recovery policy; duplicates must be zero.
type PendingEffect struct {
	ID               string       `json:"id"`
	Kind             string       `json:"kind"`
	Status           EffectStatus `json:"status"`
	IdempotencyKey   string       `json:"idempotency_key"`
	Target           string       `json:"target,omitempty"`
	ObservableEffect string       `json:"observable_effect,omitempty"`
	RecoveryPolicy   string       `json:"recovery_policy,omitempty"`
	CreatedAt        string       `json:"created_at"`
}

// NewID helpers avoid hidden globals; callers inject ids in production.
func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// AgentRuntime is the port surface the NativeAgent depends on. Each
// method takes ctx for propagation; implementations live in adapters.
type AgentRuntime interface {
	CompileContext(ctx context.Context, state NativeAgentState) (string, error)
	RequestModel(ctx context.Context, req ModelRequest) (<-chan ModelEvent, error)
	ExecuteTool(ctx context.Context, call ToolCall) (ToolResult, error)
	CheckPermission(ctx context.Context, req PermissionRequest) (PermissionResolution, error)
	SaveCheckpoint(ctx context.Context, cp Checkpoint) error
	EmitEvent(ctx context.Context, ev AgentEvent) error
}
