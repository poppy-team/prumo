// Protocol manifest: the versioned IDL baseline. Ops, schemas and error
// codes clients may rely on. The checked-in schemas/protocol-manifest.json
// must match Manifest(); manifest_test.go enforces it.
package protocol

// Ops lists every daemon/CLI protocol operation in stable order.
var Ops = []string{"start", "status", "list", "events", "cancel", "steer", "schedule", "unschedule", "jobs", "protocol"}

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
