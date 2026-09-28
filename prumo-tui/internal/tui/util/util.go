package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func ReportError(err error) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeError,
		Msg:  err.Error(),
	})
}

// ReportFailure reports something the client could not do, in the shape the
// interaction specification asks of every user-visible failure: what failed, and
// what the person reading it can do next.
//
// The error itself is carried unchanged — it is the record of what happened —
// and the two halves around it are the client's: an operation a reader can name,
// and an action that is actually available. A failure that only says what broke
// leaves the reader to guess, which is the thing this shape exists to prevent.
func ReportFailure(operation, next string, err error) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeError,
		Msg:  fmt.Sprintf("%s: %s — %s", operation, err, next),
	})
}

type InfoType int

const (
	InfoTypeInfo InfoType = iota
	InfoTypeWarn
	InfoTypeError
)

func ReportInfo(info string) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeInfo,
		Msg:  info,
	})
}

func ReportWarn(warn string) tea.Cmd {
	return CmdHandler(InfoMsg{
		Type: InfoTypeWarn,
		Msg:  warn,
	})
}

type (
	InfoMsg struct {
		Type InfoType
		Msg  string
		TTL  time.Duration
	}
	ClearStatusMsg struct{}
)

func Clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

// GitBranch inspects the workspace directory for the active Git branch.
// It reads .git/HEAD directly without shelling out to git CLI.
func GitBranch(dir string) string {
	if dir == "" {
		dir = "."
	}
	headPath := filepath.Join(dir, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		parentHead := filepath.Join(dir, "..", ".git", "HEAD")
		if d, err := os.ReadFile(parentHead); err == nil {
			data = d
		} else {
			return ""
		}
	}
	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}
	if len(content) >= 7 {
		return content[:7]
	}
	return content
}

// DefaultSessionTitle generates the title for an initial session:
// - Named after the active Git branch if present.
// - If no branch exists, named `<YYYY-MM-DD> - <project_name>`.
func DefaultSessionTitle(workspace string) string {
	if branch := GitBranch(workspace); branch != "" {
		return branch
	}
	projName := "project"
	if workspace != "" {
		abs, err := filepath.Abs(workspace)
		if err == nil {
			projName = filepath.Base(abs)
		} else {
			projName = filepath.Base(workspace)
		}
	} else if cwd, err := os.Getwd(); err == nil {
		projName = filepath.Base(cwd)
	}
	if projName == "" || projName == "." || projName == "/" {
		projName = "project"
	}
	dateStr := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s - %s", dateStr, projName)
}
