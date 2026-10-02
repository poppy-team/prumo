package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/raillen/prumo-tui/internal/message"
)

func TestRenderAssistantMessageThinkingAccordion(t *testing.T) {
	now := time.Now().UnixMilli()

	msg := message.Message{
		ID:        "msg-think-1",
		Role:      message.Assistant,
		Model:     "claude-3-7-sonnet",
		CreatedAt: now,
		Parts: []message.ContentPart{
			message.ReasoningContent{Thinking: "Analyzing architectural invariants and dependencies."},
			message.TextContent{Text: "Here is the proposed refactor plan."},
			message.Finish{Reason: message.FinishReasonEndTurn, Time: now + 2500},
		},
	}

	// 1. Collapsed by default
	renderedCollapsed := renderAssistantMessage(msg, 0, nil, nil, "", false, false, 80, 0)
	if len(renderedCollapsed) < 2 {
		t.Fatalf("expected at least 2 ui messages (thinking badge + content), got %d", len(renderedCollapsed))
	}
	thinkingBadge := ansi.Strip(renderedCollapsed[0].content)
	if !strings.Contains(thinkingBadge, "Thought process (collapsed") {
		t.Fatalf("expected collapsed thinking badge, got: %s", thinkingBadge)
	}
	if strings.Contains(thinkingBadge, "Analyzing architectural invariants") {
		t.Fatalf("collapsed thinking badge should not reveal thinking content: %s", thinkingBadge)
	}
	mainContent := ansi.Strip(renderedCollapsed[1].content)
	if !strings.Contains(mainContent, "Here is the proposed refactor plan") {
		t.Fatalf("expected main content to be rendered, got: %s", mainContent)
	}

	// 2. Expanded
	renderedExpanded := renderAssistantMessage(msg, 0, nil, nil, "", false, true, 80, 0)
	if len(renderedExpanded) < 2 {
		t.Fatalf("expected at least 2 ui messages, got %d", len(renderedExpanded))
	}
	expandedBlock := ansi.Strip(renderedExpanded[0].content)
	if !strings.Contains(expandedBlock, "Thought process (expanded") {
		t.Fatalf("expected expanded thinking header, got: %s", expandedBlock)
	}
	if !strings.Contains(expandedBlock, "Analyzing architectural invariants") {
		t.Fatalf("expanded thinking block should contain thinking content: %s", expandedBlock)
	}
	mainContentExpanded := ansi.Strip(renderedExpanded[1].content)
	if !strings.Contains(mainContentExpanded, "Here is the proposed refactor plan") {
		t.Fatalf("expected main content to be rendered, got: %s", mainContentExpanded)
	}
}

func TestRenderAssistantMessageLiveThinking(t *testing.T) {
	now := time.Now().UnixMilli()

	// Live streaming: thinking content exists, text content is empty, not finished
	msg := message.Message{
		ID:        "msg-live-1",
		Role:      message.Assistant,
		Model:     "o1-preview",
		CreatedAt: now,
		Parts: []message.ContentPart{
			message.ReasoningContent{Thinking: "Synthesizing deep reasoning tokens..."},
		},
	}

	renderedLive := renderAssistantMessage(msg, 0, nil, nil, "", false, false, 80, 0)
	if len(renderedLive) == 0 {
		t.Fatal("expected at least 1 ui message for live thinking")
	}
	liveBlock := ansi.Strip(renderedLive[0].content)
	if !strings.Contains(liveBlock, "Thinking...") {
		t.Fatalf("expected live thinking header, got: %s", liveBlock)
	}
	if !strings.Contains(liveBlock, "Synthesizing deep reasoning tokens") {
		t.Fatalf("expected streaming thinking content, got: %s", liveBlock)
	}
}
