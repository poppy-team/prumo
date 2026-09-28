// Package audit provides deterministic project and session audit telemetry persistence.
//
// Telemetry is saved in .prumo/runtime/audit/telemetry.json as canonical JSON,
// allowing offline post-run verification, cost tracking, token budget analysis,
// and workforce delegation auditing separated by project, session, and tasks.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

const (
	AuditSchemaVersion = "prumo.audit.v1"
	AuditRelPath       = ".prumo/runtime/audit/telemetry.json"
	SessionsAuditDir   = ".prumo/runtime/audit/sessions"
)

var mu sync.Mutex

// ProjectAudit captures the full auditable telemetry of a project across all sessions and tasks.
type ProjectAudit struct {
	SchemaVersion string                  `json:"schema_version"`
	ProjectName   string                  `json:"project_name"`
	WorkspacePath string                  `json:"workspace_path"`
	GitBranch     string                  `json:"git_branch"`
	LastUpdatedAt string                  `json:"last_updated_at"`
	Summary       AuditSummary            `json:"summary"`
	Sessions      map[string]SessionAudit `json:"sessions"`
}

// AuditSummary contains aggregated metrics for the entire project.
type AuditSummary struct {
	TotalSessions        int     `json:"total_sessions"`
	TotalTokens          int64   `json:"total_tokens"`
	TotalPromptTokens    int64   `json:"total_prompt_tokens"`
	TotalOutputTokens    int64   `json:"total_output_tokens"`
	TotalCacheReadTokens int64   `json:"total_cache_read_tokens"`
	TotalCostUSD         float64 `json:"total_cost_usd"`
	TotalFilesChanged    int     `json:"total_files_changed"`
}

// SessionAudit contains granular telemetry, workforce tree, and task history for one session.
type SessionAudit struct {
	SessionID string           `json:"session_id"`
	Title     string           `json:"title"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
	Provider  string           `json:"provider"`
	Model     string           `json:"model"`
	Status    string           `json:"status"` // "active", "completed"
	Telemetry SessionTelemetry `json:"telemetry"`
	Workforce WorkforceAudit   `json:"workforce"`
	Changes   []runtime.Change `json:"changes"`
	Tasks     []TaskAudit      `json:"tasks,omitempty"`
}

// SessionTelemetry breaks down tokens, cost, and harness usage reports.
type SessionTelemetry struct {
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CacheReadTokens  int64   `json:"cache_read_tokens"`
	CacheWriteTokens int64   `json:"cache_write_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CostUSD          float64 `json:"cost_usd"`
	MessageCount     int64   `json:"message_count"`
	UsageReports     int64   `json:"usage_reports"`
}

// WorkforceAudit documents the primary agent role and delegated subagents.
type WorkforceAudit struct {
	PrimaryAgent string                 `json:"primary_agent"`
	Subagents    []runtime.SubagentInfo `json:"subagents"`
}

// TaskAudit records individual goals/prompts executed in the session.
type TaskAudit struct {
	ID        string `json:"id"`
	Goal      string `json:"goal"`
	Status    string `json:"status"`
	AgentRole string `json:"agent_role"`
	Timestamp string `json:"timestamp"`
}

// AuditPath returns the absolute path to the project telemetry JSON file.
func AuditPath(workspace string) string {
	if workspace == "" {
		workspace = "."
	}
	return filepath.Join(workspace, AuditRelPath)
}

// Load reads existing project audit telemetry from disk, or initializes a new record if missing.
func Load(workspace string) (*ProjectAudit, error) {
	path := AuditPath(workspace)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			absWs, _ := filepath.Abs(workspace)
			projName := filepath.Base(absWs)
			if projName == "" || projName == "." || projName == "/" {
				projName = "project"
			}
			branch := util.GitBranch(workspace)
			return &ProjectAudit{
				SchemaVersion: AuditSchemaVersion,
				ProjectName:   projName,
				WorkspacePath: absWs,
				GitBranch:     branch,
				LastUpdatedAt: time.Now().UTC().Format(time.RFC3339),
				Summary:       AuditSummary{},
				Sessions:      make(map[string]SessionAudit),
			}, nil
		}
		return nil, err
	}

	var audit ProjectAudit
	if err := json.Unmarshal(data, &audit); err != nil {
		return nil, fmt.Errorf("audit telemetry corrupt: %w", err)
	}
	if audit.Sessions == nil {
		audit.Sessions = make(map[string]SessionAudit)
	}
	return &audit, nil
}

// RecordSession updates the project telemetry JSON and writes an individual session audit record.
// All writes are deterministic and format-indented with sorted keys for git-friendly review.
func RecordSession(workspace string, sess session.Session, provider, model, primaryAgent, status string, subagents []runtime.SubagentInfo, changes []runtime.Change) (*ProjectAudit, error) {
	mu.Lock()
	defer mu.Unlock()

	audit, err := Load(workspace)
	if err != nil {
		return nil, err
	}

	absWs, err := filepath.Abs(workspace)
	if err == nil {
		audit.WorkspacePath = absWs
	}
	audit.GitBranch = util.GitBranch(workspace)
	audit.LastUpdatedAt = time.Now().UTC().Format(time.RFC3339)

	sessID := sess.ID
	if sessID == "" {
		sessID = "default"
	}
	title := sess.Title
	if title == "" {
		title = util.DefaultSessionTitle(workspace)
	}
	if primaryAgent == "" {
		primaryAgent = "coder"
	}
	if status == "" {
		status = "completed"
	}

	createdTime := time.Unix(sess.CreatedAt, 0).UTC().Format(time.RFC3339)
	if sess.CreatedAt == 0 {
		createdTime = audit.LastUpdatedAt
	}
	updatedTime := time.Unix(sess.UpdatedAt, 0).UTC().Format(time.RFC3339)
	if sess.UpdatedAt == 0 {
		updatedTime = audit.LastUpdatedAt
	}

	totalTokens := sess.PromptTokens + sess.CompletionTokens

	// Copy and sort changes for determinism
	changesCopy := make([]runtime.Change, len(changes))
	copy(changesCopy, changes)
	sort.Slice(changesCopy, func(i, j int) bool {
		if changesCopy[i].Path == changesCopy[j].Path {
			return changesCopy[i].Operation < changesCopy[j].Operation
		}
		return changesCopy[i].Path < changesCopy[j].Path
	})

	// Copy and sort subagents for determinism
	subsCopy := make([]runtime.SubagentInfo, len(subagents))
	copy(subsCopy, subagents)
	sort.Slice(subsCopy, func(i, j int) bool {
		return subsCopy[i].ID < subsCopy[j].ID
	})

	sessAudit := SessionAudit{
		SessionID: sessID,
		Title:     title,
		CreatedAt: createdTime,
		UpdatedAt: updatedTime,
		Provider:  provider,
		Model:     model,
		Status:    status,
		Telemetry: SessionTelemetry{
			PromptTokens:     sess.PromptTokens,
			CompletionTokens: sess.CompletionTokens,
			CacheReadTokens:  sess.CacheReadTokens,
			CacheWriteTokens: sess.CacheWriteTokens,
			TotalTokens:      totalTokens,
			CostUSD:          sess.Cost,
			MessageCount:     sess.MessageCount,
			UsageReports:     sess.UsageReports,
		},
		Workforce: WorkforceAudit{
			PrimaryAgent: primaryAgent,
			Subagents:    subsCopy,
		},
		Changes: changesCopy,
	}

	audit.Sessions[sessID] = sessAudit

	// Recompute project summary across all recorded sessions
	var (
		totalSessions int
		totalPrompt   int64
		totalOutput   int64
		totalCache    int64
		totalCost     float64
		filesMap      = make(map[string]bool)
	)

	for _, s := range audit.Sessions {
		totalSessions++
		totalPrompt += s.Telemetry.PromptTokens
		totalOutput += s.Telemetry.CompletionTokens
		totalCache += s.Telemetry.CacheReadTokens
		totalCost += s.Telemetry.CostUSD
		for _, c := range s.Changes {
			if c.Path != "" {
				filesMap[c.Path] = true
			}
		}
	}

	audit.Summary = AuditSummary{
		TotalSessions:        totalSessions,
		TotalTokens:          totalPrompt + totalOutput,
		TotalPromptTokens:    totalPrompt,
		TotalOutputTokens:    totalOutput,
		TotalCacheReadTokens: totalCache,
		TotalCostUSD:         totalCost,
		TotalFilesChanged:    len(filesMap),
	}

	// Deterministic serialization with 2-space indent and trailing newline
	data, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling audit telemetry: %w", err)
	}
	data = append(data, '\n')

	mainPath := AuditPath(workspace)
	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(mainPath, data, 0o644); err != nil {
		return nil, err
	}

	// Write individual session audit
	sessFileDir := filepath.Join(workspace, SessionsAuditDir)
	if err := os.MkdirAll(sessFileDir, 0o755); err == nil {
		sessData, err := json.MarshalIndent(sessAudit, "", "  ")
		if err == nil {
			sessData = append(sessData, '\n')
			_ = os.WriteFile(filepath.Join(sessFileDir, sessID+".json"), sessData, 0o644)
		}
	}

	return audit, nil
}
