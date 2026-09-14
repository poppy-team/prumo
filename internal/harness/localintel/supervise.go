// Supervised workers (GAP-025 remainder): process supervision for local
// inference workers with health checks, restart backoff and clean stop.
// Workers implement WorkFunc (serve until ctx cancel); the supervisor owns
// lifecycle only — model selection stays with the Router. A llama-server
// detector grounds the primary generation runtime without requiring it.
package localintel

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// WorkFunc serves until ctx is cancelled, then returns.
type WorkFunc func(ctx context.Context) error

// Supervisor runs named workers with restart backoff.
type Supervisor struct {
	mu      sync.Mutex
	running map[string]context.CancelFunc
	status  map[string]string // running|stopped|failed
	Backoff time.Duration
}

// NewSupervisor creates a supervisor (default 1s restart backoff).
func NewSupervisor() *Supervisor {
	return &Supervisor{running: map[string]context.CancelFunc{}, status: map[string]string{}, Backoff: time.Second}
}

// Start launches (or relaunches) a named worker.
func (s *Supervisor) Start(name string, work WorkFunc) error {
	s.mu.Lock()
	if _, ok := s.running[name]; ok {
		s.mu.Unlock()
		return fmt.Errorf("worker %s already running", name)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.running[name] = cancel
	s.status[name] = "running"
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, name)
			if s.status[name] == "running" {
				s.status[name] = "stopped"
			}
			s.mu.Unlock()
		}()
		backoff := s.Backoff
		if backoff <= 0 {
			backoff = time.Second
		}
		for {
			err := work(ctx)
			if ctx.Err() != nil {
				return
			}
			if err == nil {
				s.mu.Lock()
				s.status[name] = "stopped"
				s.mu.Unlock()
				return
			}
			s.mu.Lock()
			s.status[name] = "failed"
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}()
	return nil
}

// Stop cancels a worker and waits briefly for exit.
func (s *Supervisor) Stop(name string) error {
	s.mu.Lock()
	cancel, ok := s.running[name]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("worker %s not running", name)
	}
	cancel()
	deadline := time.Now().Add(5 * time.Second)
	for {
		s.mu.Lock()
		_, still := s.running[name]
		s.mu.Unlock()
		if !still {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("worker %s did not stop", name)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Health reports running|stopped|failed|unknown.
func (s *Supervisor) Health(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.status[name]; ok {
		return st
	}
	return "unknown"
}

// DetectLlamaServer finds a llama.cpp server binary, if installed.
func DetectLlamaServer() string {
	for _, bin := range []string{"llama-server", "llama.cpp-server", "server"} {
		if _, err := exec.LookPath(bin); err == nil {
			return bin
		}
	}
	return ""
}
