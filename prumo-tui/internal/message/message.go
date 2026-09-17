package message

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/pubsub"
)

// ErrNotFound is returned for an unknown message.
var ErrNotFound = errors.New("message not found")

// CreateMessageParams is what a producer supplies to start a message.
type CreateMessageParams struct {
	Role  MessageRole
	Parts []ContentPart
	Model models.ModelID
}

// Service is the message surface the view layer uses.
type Service interface {
	pubsub.Suscriber[Message]
	Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error)
	Update(ctx context.Context, message Message) error
	Get(ctx context.Context, id string) (Message, error)
	List(ctx context.Context, sessionID string) ([]Message, error)
	Delete(ctx context.Context, id string) error
}

// Store keeps the conversation the client is drawing.
//
// It is derived state by construction: a run's durable record is the daemon's
// event log, and this is what the view folds those events into. Losing it
// costs a re-read, never a fact.
type Store struct {
	*pubsub.Broker[Message]

	mu    sync.RWMutex
	items map[string]Message
	order []string
}

// NewStore returns an empty message store.
func NewStore() *Store {
	return &Store{Broker: pubsub.NewBroker[Message](), items: map[string]Message{}}
}

// Create stores a new message.
func (s *Store) Create(_ context.Context, sessionID string, params CreateMessageParams) (Message, error) {
	now := nowUnix()
	msg := Message{
		ID:        newID(),
		SessionID: sessionID,
		Role:      params.Role,
		Parts:     params.Parts,
		Model:     params.Model,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if msg.Role != Assistant {
		msg.Parts = append(msg.Parts, Finish{Reason: FinishReasonEndTurn, Time: now})
	}
	s.put(msg)
	s.Publish(pubsub.CreatedEvent, msg)
	return msg, nil
}

// Update replaces a stored message.
func (s *Store) Update(_ context.Context, msg Message) error {
	s.mu.Lock()
	if _, ok := s.items[msg.ID]; !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	msg.UpdatedAt = nowUnix()
	s.items[msg.ID] = msg
	s.mu.Unlock()
	s.Publish(pubsub.UpdatedEvent, msg)
	return nil
}

// Get returns one message.
func (s *Store) Get(_ context.Context, id string) (Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msg, ok := s.items[id]
	if !ok {
		return Message{}, ErrNotFound
	}
	return msg, nil
}

// List returns a session's messages in creation order.
func (s *Store) List(_ context.Context, sessionID string) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.order))
	for _, id := range s.order {
		if s.items[id].SessionID == sessionID {
			ids = append(ids, id)
		}
	}
	sort.SliceStable(ids, func(i, j int) bool {
		return s.items[ids[i]].CreatedAt < s.items[ids[j]].CreatedAt
	})
	out := make([]Message, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.items[id])
	}
	return out, nil
}

// Delete removes one message.
func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	msg, ok := s.items[id]
	delete(s.items, id)
	for i, existing := range s.order {
		if existing == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	s.Publish(pubsub.DeletedEvent, msg)
	return nil
}

func (s *Store) put(msg Message) {
	s.mu.Lock()
	if _, exists := s.items[msg.ID]; !exists {
		s.order = append(s.order, msg.ID)
	}
	s.items[msg.ID] = msg
	s.mu.Unlock()
}

func newID() string { return uuid.New().String() }

func nowUnix() int64 { return time.Now().Unix() }
