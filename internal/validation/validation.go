package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
)

type Registry struct {
	byName map[string]map[string]any
	byID   map[string]map[string]any
}

func LoadRegistry(dir string) (Registry, error) {
	registry := Registry{byName: map[string]map[string]any{}, byID: map[string]map[string]any{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return registry, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return registry, err
		}
		var schema map[string]any
		if err := json.Unmarshal(data, &schema); err != nil {
			return registry, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		registry.byName[entry.Name()] = schema
		if id, ok := schema["$id"].(string); ok {
			registry.byID[id] = schema
		}
	}
	return registry, nil
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
	if enumValues, ok := enumList(schema["enum"]); ok && !containsJSON(enumValues, instance) {
		errors = append(errors, fmt.Sprintf("%s: must be one of enum values", path))
	}
	// `type` is a string or an array of strings. Reading only the string form
	// meant "type": ["boolean", "string"] — used all over the schemas to say "a
	// flag may be a bool or a path to a value that overrides it" — was silently
	// not checked at all (GAP-151).
	switch types := schema["type"].(type) {
	case string:
		if !matchesType(instance, types) {
			return append(errors, fmt.Sprintf("%s: expected type %s", path, types))
		}
	case []any:
		for _, candidate := range types {
			name, isString := candidate.(string)
			if isString && matchesType(instance, name) {
				return validateRemaining(instance, schema, registry, path, errors)
			}
		}
		return append(errors, fmt.Sprintf("%s: expected one of the declared types", path))
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
		}
	}
	errors = append(errors, validateAdditionalProperties(instance, schema, registry, path)...)
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

// validateAdditionalProperties applies the additionalProperties keyword to an
// object instance.
//
// It is read outside the named-properties block on purpose: a schema that
// constrains a map's values frequently declares no named properties at all, and
// handling it inside that block meant such a schema was never reached — the
// constraint looked written and did nothing.
//
// A schema-valued additionalProperties is how JSON Schema constrains every value
// of a map — the only way to say "every role must be one of these two shapes", or
// "every entry of profiles must name a provider and a model". It used to be read
// as a bool, found not to be one, and skipped, so a contract written that way
// passed every instance it was given. A contract that looks enforced and enforces
// nothing is worse than one never written (GAP-151).
func validateAdditionalProperties(instance any, schema map[string]any, registry Registry, path string) []string {
	additional, exists := schema["additionalProperties"]
	if !exists {
		return nil
	}
	object, isObject := instance.(map[string]any)
	if !isObject {
		return nil
	}
	named, _ := schema["properties"].(map[string]any)
	switch typed := additional.(type) {
	case bool:
		if typed {
			return nil
		}
		var errors []string
		for name := range object {
			if _, defined := named[name]; !defined {
				errors = append(errors, fmt.Sprintf("%s: additional property %q is not allowed", path, name))
			}
		}
		return errors
	case map[string]any:
		var errors []string
		for name, value := range object {
			if _, defined := named[name]; defined {
				continue
			}
			errors = append(errors, validateNode(value, typed, registry, path+"."+name)...)
		}
		return errors
	}
	return nil
}

// matchesType reports whether value has the JSON type expected.
//
// Arrays and objects are matched by reflection rather than by a type assertion.
// The assertion version accepted only []any and map[string]any, so validating a
// Go value that had not been through a JSON round trip — a struct built in
// memory, a []string literal — failed on every array and object in it. The
// tempting response to that failure is to loosen the schema, which converts a
// validator bug into a contract that no longer means anything (GAP-151).
func matchesType(value any, expected string) bool {
	switch expected {
	case "object":
		if _, ok := value.(map[string]any); ok {
			return true
		}
		rv := reflect.ValueOf(value)
		return rv.IsValid() && rv.Kind() == reflect.Map &&
			rv.Type().Key().Kind() == reflect.String
	case "array":
		if _, ok := value.([]any); ok {
			return true
		}
		rv := reflect.ValueOf(value)
		return rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array)
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

// enumList reads an enum as a list whether it arrived as []any (from JSON) or as
// a Go slice built in memory. Only the first form was recognised, so a schema
// checked against a value that had not been through JSON enforced nothing.
func enumList(value any) ([]any, bool) {
	switch values := value.(type) {
	case []any:
		return values, true
	case []string:
		out := make([]any, 0, len(values))
		for _, v := range values {
			out = append(out, v)
		}
		return out, true
	default:
		return nil, false
	}
}

// validateRemaining continues a node after the type check accepted a union. The
// union short-circuits the scalar checks below, so they are re-run here rather
// than skipped.
func validateRemaining(instance any, schema map[string]any, registry Registry, path string, errors []string) []string {
	rest := map[string]any{}
	for k, v := range schema {
		switch k {
		case "type", "enum", "const":
			continue
		default:
			rest[k] = v
		}
	}
	return append(errors, validateNode(instance, rest, registry, path)...)
}
