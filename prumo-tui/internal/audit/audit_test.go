package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
)

func TestAuditRecordingAndSummary(t *testing.T) {
	tmpDir := t.TempDir()

	sess1 := session.Session{
		ID:               "sess-100",
		Title:            "feature/login",
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
		PromptTokens:     1000,
		CompletionTokens: 250,
		CacheReadTokens:  500,
		Cost:             0.0035,
		MessageCount:     4,
	}

	changes1 := []runtime.Change{
		{Path: "src/auth/login.go", Operation: "created"},
		{Path: "src/auth/login_test.go", Operation: "created"},
	}

	subs1 := []runtime.SubagentInfo{
		{ID: "sub-1", Role: "researcher", Status: "completed"},
	}

	audit, err := RecordSession(tmpDir, sess1, "gemini", "gemini-2.5-flash", "coder", "completed", subs1, changes1)
	if err != nil {
		t.Fatalf("failed to record session 1: %v", err)
	}

	if audit.Summary.TotalSessions != 1 {
		t.Fatalf("expected 1 total session, got %d", audit.Summary.TotalSessions)
	}
	if audit.Summary.TotalTokens != 1250 {
		t.Fatalf("expected 1250 total tokens, got %d", audit.Summary.TotalTokens)
	}
	if audit.Summary.TotalCostUSD != 0.0035 {
		t.Fatalf("expected cost 0.0035, got %f", audit.Summary.TotalCostUSD)
	}
	if audit.Summary.TotalFilesChanged != 2 {
		t.Fatalf("expected 2 files changed, got %d", audit.Summary.TotalFilesChanged)
	}

	// Verify file exists on disk
	expectedFile := filepath.Join(tmpDir, AuditRelPath)
	data, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("expected audit file to exist at %s: %v", expectedFile, err)
	}

	var loaded ProjectAudit
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse recorded audit file: %v", err)
	}

	if loaded.Sessions["sess-100"].Title != "feature/login" {
		t.Fatalf("expected title 'feature/login', got %q", loaded.Sessions["sess-100"].Title)
	}

	// Add second session
	sess2 := session.Session{
		ID:               "sess-200",
		Title:            "fix/race-condition",
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
		PromptTokens:     2000,
		CompletionTokens: 500,
		Cost:             0.0070,
	}

	changes2 := []runtime.Change{
		{Path: "src/auth/login.go", Operation: "modified"},
		{Path: "src/mutex.go", Operation: "created"},
	}

	audit2, err := RecordSession(tmpDir, sess2, "anthropic", "claude-3-7-sonnet", "coder", "completed", nil, changes2)
	if err != nil {
		t.Fatalf("failed to record session 2: %v", err)
	}

	if audit2.Summary.TotalSessions != 2 {
		t.Fatalf("expected 2 total sessions, got %d", audit2.Summary.TotalSessions)
	}
	if audit2.Summary.TotalTokens != 3750 {
		t.Fatalf("expected 3750 total tokens (1250 + 2500), got %d", audit2.Summary.TotalTokens)
	}
	if audit2.Summary.TotalFilesChanged != 3 { // login.go, login_test.go, mutex.go
		t.Fatalf("expected 3 distinct files changed, got %d", audit2.Summary.TotalFilesChanged)
	}

	// Check individual session file
	sess2File := filepath.Join(tmpDir, SessionsAuditDir, "sess-200.json")
	if _, err := os.Stat(sess2File); err != nil {
		t.Fatalf("expected session file %s to exist: %v", sess2File, err)
	}
}
