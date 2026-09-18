// Package agent is the client's agent surface.
//
// It keeps the shape the imported view layer expects — the view calls Run and
// reads a stream of AgentEvent — while the run itself belongs to the harness.
// Nothing here talks to a provider.
package agent

import (
	"context"

	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/pubsub"
)

// AgentEventType classifies what happened.
type AgentEventType string

const (
	AgentEventTypeError      AgentEventType = "error"
	AgentEventTypeResponse   AgentEventType = "response"
	AgentEventTypeConnection AgentEventType = "connection"
)

// ConnectionState is the client's own link to the harness.
//
// It is separate from the run's state on purpose: a run that stopped because its
// daemon went away and a run that failed are different facts, and a client that
// folded them into one could not tell a user which happened.
type ConnectionState string

const (
	ConnectionLive         ConnectionState = "live"
	ConnectionReconnecting ConnectionState = "reconnecting"
	ConnectionOffline      ConnectionState = "offline"
)

// Connection is how the client is reaching the daemon, with the attempt count a
// user needs to tell a retry from a hang.
type Connection struct {
	State   ConnectionState
	Attempt int
	Of      int
	Reason  string
}

// AgentEvent is one step of a run as the view consumes it.
type AgentEvent struct {
	Type      AgentEventType
	Message   message.Message
	Error     error
	SessionID string
	Done      bool
	// Connection carries the link to the harness when Type is Connection.
	Connection Connection
}

// Service is the agent surface the view layer uses.
//
// There is no Summarize here: compaction belongs to the harness, which
// summarizes a run as its context budget requires, and the protocol exposes no
// operation to ask for one. An operation the client cannot perform is not part
// of its surface.
type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() models.Model
	Run(ctx context.Context, sessionID string, content string) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	Steer(ctx context.Context, sessionID, message string) error
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	Update(modelID models.ModelID) (models.Model, error)
}
