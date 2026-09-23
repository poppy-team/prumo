package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// slowTools models a tool call that takes real time, which is what a drain has
// to wait for and a cancel has to interrupt.
type slowTools struct {
	delay time.Duration
	mu    sync.Mutex
	n     int
}

func (s *slowTools) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	s.mu.Lock()
	s.n++
	s.mu.Unlock()
	select {
	case <-time.After(s.delay):
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "ok"}, nil
	case <-ctx.Done():
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: "cancelled"}, ctx.Err()
	}
}
func (s *slowTools) KindOf(string) string      { return "side-effecting" }
func (s *slowTools) OperationOf(string) string { return "modified" }
func (s *slowTools) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}
