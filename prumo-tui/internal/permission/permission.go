// Package permission is the client side of the permission gate.
//
// The harness decides *when* a tool call needs approval and does the waiting;
// the client only presents the request and sends back an answer. That is the
// opposite of the imported design, where the client's tool layer blocked on a
// channel until its own UI replied — here nothing blocks, because the run that
// stopped lives in the daemon and is answered over the protocol.
package permission

import (
	"context"
	"errors"
	"sync"

	"github.com/raillen/prumo-tui/internal/pubsub"
)

// ErrNotPending is returned when a decision arrives for a request that is no
// longer open.
var ErrNotPending = errors.New("permission request is not pending")

// CreatePermissionRequest is what the runtime raises when a run stops on a
// permission gate.
type CreatePermissionRequest struct {
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

// PermissionRequest is one open gate.
type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

// Responder answers a request against the harness.
type Responder interface {
	Approve(ctx context.Context, runID, requestID string) error
	Deny(ctx context.Context, runID, requestID, reason string) error
}

// Service presents requests and forwards decisions.
type Service struct {
	*pubsub.Broker[PermissionRequest]
	responder Responder

	mu      sync.Mutex
	pending map[string]PermissionRequest
}

// NewService returns a permission service backed by a responder.
func NewService(responder Responder) *Service {
	return &Service{
		Broker:    pubsub.NewBroker[PermissionRequest](),
		responder: responder,
		pending:   map[string]PermissionRequest{},
	}
}

// Raise publishes a request so the view can present it.
func (s *Service) Raise(req PermissionRequest) {
	s.mu.Lock()
	s.pending[req.ID] = req
	s.mu.Unlock()
	s.Publish(pubsub.CreatedEvent, req)
}

// Pending returns the requests still awaiting an answer.
func (s *Service) Pending() []PermissionRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PermissionRequest, 0, len(s.pending))
	for _, req := range s.pending {
		out = append(out, req)
	}
	return out
}

// Grant allows the request once.
func (s *Service) Grant(ctx context.Context, req PermissionRequest) error {
	return s.resolve(ctx, req, true)
}

// GrantPersistant allows the request for the rest of the session.
//
// The protocol answers one request at a time, so this currently behaves like
// Grant. It is kept as a distinct call rather than aliased silently: the
// distinction is real, and the day the protocol grows a session-scoped decision
// this is where it lands.
func (s *Service) GrantPersistant(ctx context.Context, req PermissionRequest) error {
	return s.resolve(ctx, req, true)
}

// Deny refuses the request and fails the run.
func (s *Service) Deny(ctx context.Context, req PermissionRequest) error {
	return s.resolve(ctx, req, false)
}

func (s *Service) resolve(ctx context.Context, req PermissionRequest, allow bool) error {
	s.mu.Lock()
	_, open := s.pending[req.ID]
	if open {
		delete(s.pending, req.ID)
	}
	s.mu.Unlock()
	if !open {
		return ErrNotPending
	}
	if s.responder == nil {
		return errors.New("no harness attached to answer permissions")
	}
	if allow {
		return s.responder.Approve(ctx, req.SessionID, req.ID)
	}
	return s.responder.Deny(ctx, req.SessionID, req.ID, "denied in the client")
}
