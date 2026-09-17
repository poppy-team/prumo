package dialog

import "strings"

// Tool previews are driven by the harness's tool vocabulary, not by the client's.
//
// The imported dialog branched on a fixed set of upstream tool names and
// type-asserted their argument structs. Prumo's tools come from the ACI catalog
// (`fs.read`, `edit.patch`, `process.exec`, …) and their arguments are open
// maps, so classification happens here and extraction is by key.
type previewKind int

const (
	previewDefault previewKind = iota
	previewCommand
	previewDiff
	previewFile
	previewURL
)

// classifyAction maps an action name to the preview that explains it best.
func classifyAction(action string) previewKind {
	switch action {
	case "process.exec", "test.run", "shell", "bash":
		return previewCommand
	case "edit.patch", "edit.create", "edit.move", "edit.delete", "patch", "edit", "write":
		return previewDiff
	case "fs.read", "fs.list", "fs.search", "view", "ls", "glob", "grep":
		return previewFile
	case "fetch", "web.fetch", "sourcegraph":
		return previewURL
	}
	return previewDefault
}

// paramString returns the first present string argument among keys.
//
// Tool arguments are an open map: a client that assumed a schema would break
// the first time the harness added or renamed a parameter.
func paramString(params any, keys ...string) string {
	m, ok := params.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range keys {
		if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
