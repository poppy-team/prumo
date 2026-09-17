// Package session is the client's view of a run.
//
// The daemon owns runs; this is the client's projection of them. A session id
// is a run id: the client passes it when it starts a run, so re-attaching to a
// session is exactly re-attaching to the run the protocol already knows.
package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raillen/prumo-tui/internal/pubsub"
)

// Session is one run as the client presents it.
type Session struct {
	ID               string
	ParentSessionID  string
	Title            string
	MessageCount     int64
	PromptTokens     int64
	CompletionTokens int64
	SummaryMessageID string
	Cost             float64
	CreatedAt        int64
	UpdatedAt        int64
}

// ErrNotFound is returned for an unknown session.
var ErrNotFound = errors.New("session not found")

// Service is the session surface the view layer uses.
type Service interface {
	pubsub.Suscriber[Session]
	Create(ctx context.Context, title string) (Session, error)
	Get(ctx context.Context, id string) (Session, error)
	List(ctx context.Context) ([]Session, error)
	Save(ctx context.Context, session Session) (Session, error)
	Delete(ctx context.Context, id string) error
}

// Store is an in-memory session index.
//
// It caches what the protocol said; it is never a source of truth. The daemon
// keeps run records on disk, and folding its list back in is what makes a
// restarted client agree with it again.
type Store struct {
	*pubsub.Broker[Session]

	mu    sync.RWMutex
	items map[string]Session
}

// NewStore returns an empty session index.
func NewStore() *Store {
	return &Store{Broker: pubsub.NewBroker[Session](), items: map[string]Session{}}
}

// Create makes a local session id. The run itself starts when the first message
// is sent, so an abandoned session never becomes a harness run.
func (s *Store) Create(_ context.Context, title string) (Session, error) {
	now := time.Now().Unix()
	session := Session{ID: uuid.New().String(), Title: title, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.items[session.ID] = session
	s.mu.Unlock()
	s.Publish(pubsub.CreatedEvent, session)
	return session, nil
}

// Get returns one session.
func (s *Store) Get(_ context.Context, id string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	return session, nil
}

// List returns every known session.
func (s *Store) List(_ context.Context) ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Session, 0, len(s.items))
	for _, session := range s.items {
		out = append(out, session)
	}
	return out, nil
}

// Save records a session, creating it if it is new.
func (s *Store) Save(_ context.Context, session Session) (Session, error) {
	if session.UpdatedAt == 0 {
		session.UpdatedAt = time.Now().Unix()
	}
	s.mu.Lock()
	_, existed := s.items[session.ID]
	s.items[session.ID] = session
	s.mu.Unlock()
	if existed {
		s.Publish(pubsub.UpdatedEvent, session)
	} else {
		s.Publish(pubsub.CreatedEvent, session)
	}
	return session, nil
}

// Delete forgets a session locally. It does not cancel the run: stopping work
// is Cancel, and conflating the two would make dismissing a row kill a job.
func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	session, ok := s.items[id]
	delete(s.items, id)
	s.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	s.Publish(pubsub.DeletedEvent, session)
	return nil
}
