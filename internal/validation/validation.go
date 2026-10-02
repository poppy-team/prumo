package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	prumo "github.com/raillen/prumo"
	"github.com/raillen/prumo/internal/protocol"
)

type Registry struct {
	byName map[string]map[string]any
	byID   map[string]map[string]any
}

// LoadRegistry builds the schema registry for dir. The binary embeds every
// canonical schema, so a missing or unreadable directory is not a failure: the
// embedded set is authoritative and a local directory only ever overrides it.
// That is what makes the released binary usable from any working directory
// instead of only from inside the framework source tree.
func LoadRegistry(dir string) (Registry, error) {
	registry := Registry{byName: map[string]map[string]any{}, byID: map[string]map[string]any{}}
	if err := registry.loadDir(dir); err != nil {
		// A project-local directory that does not exist, or that this process
		// cannot read, falls back to the embedded schemas instead of failing.
		// An override that exists but is broken still has to surface, otherwise
		// a typo in a schema would be silently masked by the embedded copy.
		if _, statErr := os.Stat(dir); statErr != nil {
			return registry, registry.loadEmbedded()
		}
		return registry, err
	}
	// Embedded schemas fill any gap the local directory does not provide.
	embedded, embeddedErr := loadRegistryFromFS(prumo.EmbeddedSchemas())
	if embeddedErr != nil {
		return registry, embeddedErr
	}
	registry.merge(embedded)
	return registry, nil
}

func (r Registry) loadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		schema, err := decodeSchema(entry.Name(), data)
		if err != nil {
			return err
		}
		r.byName[entry.Name()] = schema
		if id, ok := schema["$id"].(string); ok {
			r.byID[id] = schema
		}
	}
	return nil
}

func (r Registry) loadEmbedded() error {
	embedded, err := loadRegistryFromFS(prumo.EmbeddedSchemas())
	if err != nil {
		return err
	}
	r.merge(embedded)
	return nil
}

func loadRegistryFromFS(fsys fs.FS) (Registry, error) {
	registry := Registry{byName: map[string]map[string]any{}, byID: map[string]map[string]any{}}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return registry, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := fs.ReadFile(fsys, entry.Name())
		if err != nil {
			return registry, err
		}
		schema, err := decodeSchema(entry.Name(), data)
		if err != nil {
			return registry, err
		}
		registry.byName[entry.Name()] = schema
		if id, ok := schema["$id"].(string); ok {
			registry.byID[id] = schema
		}
	}
	return registry, nil
}

func decodeSchema(name string, data []byte) (map[string]any, error) {
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return schema, nil
}

// merge copies entries the receiver does not already have, so a local schema
// always wins over the embedded one of the same name.
func (r Registry) merge(other Registry) {
	for name, schema := range other.byName {
		if _, ok := r.byName[name]; !ok {
			r.byName[name] = schema
		}
	}
	for id, schema := range other.byID {
		if _, ok := r.byID[id]; !ok {
			r.byID[id] = schema
		}
	}
}

func (r Registry) Schema(name string) (map[string]any, bool) {
	if schema, ok := r.byName[name]; ok {
		return schema, true
	}
	schema, ok := r.byID[name]
	return schema, ok
}

func CheckJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w: %v", protocol.ErrValidation, err)
	}
	if value == nil {
		return fmt.Errorf("%w: JSON value is null", protocol.ErrValidation)
	}
	return nil
}

func CheckJSONObject(data []byte) error {
	if err := CheckJSON(data); err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("%w: expected JSON object", protocol.ErrValidation)
	}
	return nil
}

func ValidateSchema(instance any, schema map[string]any, registry Registry) []string {
	return validateNode(instance, schema, registry, "<root>")
}

func ValidateFile(instancePath, schemaName, schemaDir string) []string {
	instanceData, err := os.ReadFile(instancePath)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", instancePath, err)}
	}
	var instance any
	if err := json.Unmarshal(instanceData, &instance); err != nil {
		return []string{fmt.Sprintf("%s: invalid JSON: %v", instancePath, err)}
	}
	registry, err := LoadRegistry(schemaDir)
	if err != nil {
		return []string{err.Error()}
	}
	schema, ok := registry.Schema(schemaName)
	if !ok {
		return []string{fmt.Sprintf("schema not found: %s", schemaName)}
	}
	errors := ValidateSchema(instance, schema, registry)
	for i := range errors {
		errors[i] = instancePath + ": " + errors[i]
	}
	return errors
}

func validateNode(instance any, schema map[string]any, registry Registry, path string) []string {
	if ref, ok := schema["$ref"].(string); ok {
		resolved, found := registry.Schema(ref)
		if !found {
			return []string{fmt.Sprintf("%s: unresolved $ref %q", path, ref)}
		}
		return validateNode(instance, resolved, registry, path)
	}
	errors := []string{}
	if constValue, ok := schema["const"]; ok && !jsonEqual(instance, constValue) {
		errors = append(errors, fmt.Sprintf("%s: must equal %v", path, constValue))
	}
	if enumValues, ok := schema["enum"].([]any); ok && !containsJSON(enumValues, instance) {
		errors = append(errors, fmt.Sprintf("%s: must be one of enum values", path))
	}
	if types, ok := schema["type"].(string); ok && !matchesType(instance, types) {
		return append(errors, fmt.Sprintf("%s: expected type %s", path, types))
	}
	if minLength, ok := numberValue(schema["minLength"]); ok {
		if value, ok := instance.(string); ok && float64(len([]rune(value))) < minLength {
			errors = append(errors, fmt.Sprintf("%s: length must be at least %v", path, minLength))
		}
	}
	if minimum, ok := numberValue(schema["minimum"]); ok {
		if value, ok := numberValue(instance); ok && value < minimum {
			errors = append(errors, fmt.Sprintf("%s: must be >= %v", path, minimum))
		}
	}
	if pattern, ok := schema["pattern"].(string); ok {
		if value, ok := instance.(string); ok {
			matched, err := regexp.MatchString(pattern, value)
			if err != nil || !matched {
				errors = append(errors, fmt.Sprintf("%s: does not match pattern %q", path, pattern))
			}
		}
	}
	if required, ok := schema["required"].([]any); ok {
		if object, ok := instance.(map[string]any); ok {
			for _, raw := range required {
				name := fmt.Sprint(raw)
				if _, exists := object[name]; !exists {
					errors = append(errors, fmt.Sprintf("%s: missing required property %q", path, name))
				}
			}
		}
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		if object, ok := instance.(map[string]any); ok {
			for name, rawSchema := range properties {
				value, exists := object[name]
				if !exists {
					continue
				}
				childSchema, ok := rawSchema.(map[string]any)
				if !ok {
					continue
				}
				errors = append(errors, validateNode(value, childSchema, registry, path+"."+name)...)
			}
			if additional, exists := schema["additionalProperties"]; exists {
				if allowed, isBool := additional.(bool); isBool && !allowed {
					for name := range object {
						if _, defined := properties[name]; !defined {
							errors = append(errors, fmt.Sprintf("%s: additional property %q is not allowed", path, name))
						}
					}
				}
			}
		}
	}
	if items, ok := schema["items"].(map[string]any); ok {
		if values, ok := instance.([]any); ok {
			for index, value := range values {
				errors = append(errors, validateNode(value, items, registry, fmt.Sprintf("%s[%d]", path, index))...)
			}
		}
	}
	for keyword, mode := range map[string]string{"allOf": "all", "anyOf": "any", "oneOf": "one"} {
		branches, ok := schema[keyword].([]any)
		if !ok {
			continue
		}
		matches := 0
		branchErrors := [][]string{}
		for _, rawBranch := range branches {
			branch, ok := rawBranch.(map[string]any)
			if !ok {
				continue
			}
			branchError := validateNode(instance, branch, registry, path)
			branchErrors = append(branchErrors, branchError)
			if len(branchError) == 0 {
				matches++
			}
		}
		switch mode {
		case "all":
			for _, branchError := range branchErrors {
				errors = append(errors, branchError...)
			}
		case "any":
			if matches == 0 {
				errors = append(errors, fmt.Sprintf("%s: must match at least one schema", path))
			}
		case "one":
			if matches != 1 {
				errors = append(errors, fmt.Sprintf("%s: must match exactly one schema", path))
			}
		}
	}
	return errors
}

func matchesType(value any, expected string) bool {
	switch expected {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	case "number":
		_, ok := numberValue(value)
		return ok
	case "integer":
		number, ok := numberValue(value)
		return ok && math.Trunc(number) == number
	default:
		return true
	}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func jsonEqual(left, right any) bool {
	leftData, _ := json.Marshal(left)
	rightData, _ := json.Marshal(right)
	return bytes.Equal(leftData, rightData)
}

func containsJSON(values []any, value any) bool {
	for _, candidate := range values {
		if jsonEqual(candidate, value) {
			return true
		}
	}
	return false
}

func isJSONDocument(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".json")
}
