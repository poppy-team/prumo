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
	AgentEventTypeError     AgentEventType = "error"
	AgentEventTypeResponse  AgentEventType = "response"
	AgentEventTypeSummarize AgentEventType = "summarize"
)

// AgentEvent is one step of a run as the view consumes it.
type AgentEvent struct {
	Type    AgentEventType
	Message message.Message
	Error   error

	// Set while summarizing.
	SessionID string
	Progress  string
	Done      bool
}

// Service is the agent surface the view layer uses.
type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() models.Model
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	Update(modelID models.ModelID) (models.Model, error)
	Summarize(ctx context.Context, sessionID string) error
}
