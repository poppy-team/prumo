// Protocol manifest: the versioned IDL baseline. Ops, schemas and error
// codes clients may rely on. The checked-in schemas/protocol-manifest.json
// must match Manifest(); manifest_test.go enforces it.
package protocol

// Ops lists every daemon/CLI protocol operation in stable order.
var Ops = []string{"start", "status", "list", "events", "cancel", "steer", "schedule", "unschedule", "jobs", "protocol", "approve", "deny", "models", "diff", "subscribe"}

// ErrorCodes lists stable machine-readable error codes.
var ErrorCodes = []string{
	"unknown op",
	"invalid json",
	"goal required",
	"run_id required",
	"unknown run",
	"corrupt record",
	"no events for run",
	"run already active",
	"run not active",
	"no tool executor configured",
	"client version required",
	"permission request required",
	"no pending permission",
	"path required",
	"cursor beyond event log",
}

// OpArgs documents required arguments per op.
var OpArgs = map[string][]string{
	"start":      {"goal"},
	"status":     {"run_id"},
	"list":       {},
	"events":     {"run_id"},
	"cancel":     {"run_id"},
	"steer":      {"run_id", "message"},
	"schedule":   {"goal"},
	"unschedule": {"job_id"},
	"jobs":       {},
	"protocol":   {},
	"approve":    {"run_id", "request_id"},
	"deny":       {"run_id", "request_id"},
	// Models has no required argument: asking the default provider is the
	// common case, and the rest identify a provider to ask instead.
	"models":    {},
	"diff":      {"run_id", "path"},
	"subscribe": {"run_id"},
}

// Manifest returns the full IDL document.
func Manifest() map[string]any {
	ops := make([]any, 0, len(Ops))
	for _, name := range Ops {
		ops = append(ops, map[string]any{"name": name, "args": OpArgs[name]})
	}
	return map[string]any{
		"version":        Version,
		"min_compatible": MinCompatible,
		"transport":      []string{"unix-socket+jsonl (local)"},
		"ops":            ops,
		"schemas":        Schemas,
		"errors":         ErrorCodes,
	}
}
