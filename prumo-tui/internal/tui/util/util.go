package util

import (
	"fmt"
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
