// Managed regions + JSON Patch (GAP-007 first slice): surgical updates
// without whole-file rewrites. Markdown regions are delimited by
// <!-- prumo:begin <id> --> ... <!-- prumo:end <id> -->; JSON Patch
// implements RFC 6902 add/remove/replace on decoded documents. Full
// AST-aware Markdown editing remains future work.
package doccompile

import (
	"fmt"
	"strconv"
	"strings"
)

// Region markers delimit generated zones inside curated files.
func beginMarker(id string) string { return "<!-- prumo:begin " + id + " -->" }
func endMarker(id string) string   { return "<!-- prumo:end " + id + " -->" }

// ReplaceRegion swaps a managed region's body, preserving markers and all
// surrounding curated content. Missing region is an error, never an append.
func ReplaceRegion(doc, id, body string) (string, error) {
	begin, end := beginMarker(id), endMarker(id)
	bi := strings.Index(doc, begin)
	if bi < 0 {
		return "", fmt.Errorf("managed region %q not found", id)
	}
	rest := doc[bi+len(begin):]
	ei := strings.Index(rest, end)
	if ei < 0 {
		return "", fmt.Errorf("managed region %q missing end marker", id)
	}
	var b strings.Builder
	b.WriteString(doc[:bi+len(begin)])
	if !strings.HasSuffix(doc[:bi+len(begin)], "\n") {
		b.WriteString("\n")
	}
	b.WriteString(strings.Trim(body, "\n"))
	b.WriteString("\n")
	b.WriteString(rest[ei:])
	return b.String(), nil
}

// ExtractRegion returns a managed region's body.
func ExtractRegion(doc, id string) (string, error) {
	begin, end := beginMarker(id), endMarker(id)
	bi := strings.Index(doc, begin)
	if bi < 0 {
		return "", fmt.Errorf("managed region %q not found", id)
	}
	rest := doc[bi+len(begin):]
	ei := strings.Index(rest, end)
	if ei < 0 {
		return "", fmt.Errorf("managed region %q missing end marker", id)
	}
	return strings.Trim(rest[:ei], "\n"), nil
}

// PatchOp is one RFC 6902 operation (add/remove/replace subset).
type PatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// ApplyPatch applies ops to a decoded JSON document (maps/slices/strings).
// Paths use JSON Pointer syntax (/a/0/b, with ~0/~1 escapes).
func ApplyPatch(doc any, ops []PatchOp) (any, error) {
	for i, op := range ops {
		var err error
		doc, err = applyOne(doc, op)
		if err != nil {
			return nil, fmt.Errorf("op %d (%s %s): %w", i, op.Op, op.Path, err)
		}
	}
	return doc, nil
}

func unescape(seg string) string {
	seg = strings.ReplaceAll(seg, "~1", "/")
	return strings.ReplaceAll(seg, "~0", "~")
}

func splitPointer(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid pointer %q", path)
	}
	parts := strings.Split(path[1:], "/")
	for i := range parts {
		parts[i] = unescape(parts[i])
	}
	return parts, nil
}

func applyOne(doc any, op PatchOp) (any, error) {
	parts, err := splitPointer(op.Path)
	if err != nil {
		return nil, err
	}
	switch op.Op {
	case "add", "replace", "remove":
		return applyEdit(doc, parts, op)
	default:
		return nil, fmt.Errorf("unsupported op %q (add|remove|replace)", op.Op)
	}
}

func applyEdit(node any, parts []string, op PatchOp) (any, error) {
	if len(parts) == 0 {
		if op.Op == "remove" {
			return nil, fmt.Errorf("cannot remove document root")
		}
		return op.Value, nil
	}
	switch n := node.(type) {
	case map[string]any:
		key := parts[0]
		if len(parts) == 1 {
			if op.Op == "remove" {
				if _, ok := n[key]; !ok {
					return nil, fmt.Errorf("missing key %q", key)
				}
				delete(n, key)
				return n, nil
			}
			if op.Op == "replace" {
				if _, ok := n[key]; !ok {
					return nil, fmt.Errorf("missing key %q", key)
				}
			}
			n[key] = op.Value
			return n, nil
		}
		child, ok := n[key]
		if !ok {
			return nil, fmt.Errorf("missing key %q", key)
		}
		next, err := applyEdit(child, parts[1:], op)
		if err != nil {
			return nil, err
		}
		n[key] = next
		return n, nil
	case []any:
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx < 0 {
			return nil, fmt.Errorf("bad index %q", parts[0])
		}
		if len(parts) == 1 {
			switch op.Op {
			case "remove":
				if idx >= len(n) {
					return nil, fmt.Errorf("index %d out of range", idx)
				}
				return append(n[:idx], n[idx+1:]...), nil
			case "replace":
				if idx >= len(n) {
					return nil, fmt.Errorf("index %d out of range", idx)
				}
				n[idx] = op.Value
				return n, nil
			default: // add
				if idx > len(n) {
					return nil, fmt.Errorf("index %d out of range", idx)
				}
				return append(n[:idx], append([]any{op.Value}, n[idx:]...)...), nil
			}
		}
		if idx >= len(n) {
			return nil, fmt.Errorf("index %d out of range", idx)
		}
		next, err := applyEdit(n[idx], parts[1:], op)
		if err != nil {
			return nil, err
		}
		n[idx] = next
		return n, nil
	default:
		return nil, fmt.Errorf("cannot descend into %T", node)
	}
}
