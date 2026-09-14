// Structured validation (GAP-044 first slice): JSON-Schema-subset checking
// for model-produced tool arguments and response formats. Covered subset:
// object/array/string/number/integer/boolean types, required, properties,
// items, enum, additionalProperties:false. Anything richer is rejected as
// unsupported (explicit, not silently skipped) so callers know the boundary.
package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
)

// ValidateValue checks value against a JSON-Schema-subset document.
func ValidateValue(schema map[string]any, value any) error {
	if len(schema) == 0 {
		return nil
	}
	if enum, ok := schema["enum"].([]any); ok {
		for _, e := range enum {
			if fmt.Sprint(e) == fmt.Sprint(value) {
				return nil
			}
		}
		return fmt.Errorf("value %v not in enum", value)
	}
	typ, _ := schema["type"].(string)
	switch typ {
	case "", "any":
		return nil
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("want object, got %T", value)
		}
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				name, _ := r.(string)
				if _, ok := obj[name]; !ok {
					return fmt.Errorf("missing required property %q", name)
				}
			}
		}
		props, _ := schema["properties"].(map[string]any)
		for k, v := range obj {
			sub, ok := props[k].(map[string]any)
			if !ok {
				if closed, _ := schema["additionalProperties"].(bool); !closed {
					continue
				}
				return fmt.Errorf("additional property %q not allowed", k)
			}
			if err := ValidateValue(sub, v); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
		}
		return nil
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("want array, got %T", value)
		}
		if items, ok := schema["items"].(map[string]any); ok {
			for i, v := range arr {
				if err := ValidateValue(items, v); err != nil {
					return fmt.Errorf("[%d]: %w", i, err)
				}
			}
		}
		return nil
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("want string, got %T", value)
		}
		return nil
	case "number":
		switch value.(type) {
		case float64, int, int64:
			return nil
		}
		return fmt.Errorf("want number, got %T", value)
	case "integer":
		switch v := value.(type) {
		case int, int64:
			return nil
		case float64:
			if v == float64(int(v)) {
				return nil
			}
		}
		return fmt.Errorf("want integer, got %v", value)
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("want boolean, got %T", value)
		}
		return nil
	default:
		return fmt.Errorf("unsupported schema type %q", typ)
	}
}

// ValidateArgs checks tool arguments against the tool's declared schema.
// Unknown tools (no spec) pass — the registry, not the adapter, owns the
// catalog; known tools with bad args fail loudly.
func ValidateArgs(specs []agent.ToolSpec, name string, args map[string]any) error {
	for _, s := range specs {
		if s.Name != name {
			continue
		}
		if len(s.Schema) == 0 {
			return nil
		}
		if args == nil {
			args = map[string]any{}
		}
		return ValidateValue(s.Schema, args)
	}
	return nil
}

// ToolNames lists known tools (for error context).
func ToolNames(specs []agent.ToolSpec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	sort.Strings(out)
	return out
}

// UnknownToolError reports an undeclared tool call.
func UnknownToolError(specs []agent.ToolSpec, name string) error {
	return fmt.Errorf("unknown tool %q (known: %s)", name, strings.Join(ToolNames(specs), ", "))
}
