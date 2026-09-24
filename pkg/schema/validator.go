package schema

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Violation describes a schema compliance failure.
type Violation struct {
	Property string `json:"property"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
}

// Validator checks payloads against structured rules.
type Validator struct {
	mu     sync.RWMutex
	schema map[string]interface{}
	active bool
}

// NewValidator initializes a schema validator.
func NewValidator() *Validator {
	return &Validator{}
}

// SetSchema parses and applies an active JSON Schema.
func (v *Validator) SetSchema(rawSchema []byte) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(rawSchema, &parsed); err != nil {
		return fmt.Errorf("parsing JSON schema: %w", err)
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.schema = parsed
	v.active = len(parsed) > 0
	return nil
}

// ClearSchema removes the active validation schema.
func (v *Validator) ClearSchema() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.schema = nil
	v.active = false
}

// IsActive returns whether a schema is currently loaded.
func (v *Validator) IsActive() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.active
}

// Validate checks payload bytes against the loaded schema.
func (v *Validator) Validate(payload []byte) ([]Violation, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.active || v.schema == nil {
		return nil, nil
	}

	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return []Violation{
			{
				Property: "$",
				Expected: "valid JSON",
				Actual:   "malformed syntax",
				Message:  err.Error(),
			},
		}, nil
	}

	return v.checkObject("", v.schema, data), nil
}

func (v *Validator) checkObject(path string, schemaDef map[string]interface{}, val interface{}) []Violation {
	var violations []Violation

	expectedType, _ := schemaDef["type"].(string)
	if expectedType != "" {
		actualType := getType(val)
		if expectedType == "integer" && actualType == "number" {
			// Accept integer as number float without fraction
		} else if actualType != expectedType {
			violations = append(violations, Violation{
				Property: pathOrRoot(path),
				Expected: expectedType,
				Actual:   actualType,
				Message:  fmt.Sprintf("expected type %s, got %s", expectedType, actualType),
			})
			return violations
		}
	}

	// Required fields validation
	valMap, isMap := val.(map[string]interface{})
	if isMap {
		if reqList, ok := schemaDef["required"].([]interface{}); ok {
			for _, r := range reqList {
				if fieldName, ok := r.(string); ok {
					if _, exists := valMap[fieldName]; !exists {
						propPath := fieldName
						if path != "" {
							propPath = fmt.Sprintf("%s.%s", path, fieldName)
						}
						violations = append(violations, Violation{
							Property: propPath,
							Expected: "present",
							Actual:   "missing",
							Message:  fmt.Sprintf("required field '%s' is missing", fieldName),
						})
					}
				}
			}
		}

		// Property schema recursion
		if props, ok := schemaDef["properties"].(map[string]interface{}); ok {
			for propKey, propSchema := range props {
				if subSchemaMap, ok := propSchema.(map[string]interface{}); ok {
					if subVal, exists := valMap[propKey]; exists {
						subPath := propKey
						if path != "" {
							subPath = fmt.Sprintf("%s.%s", path, propKey)
						}
						violations = append(violations, v.checkObject(subPath, subSchemaMap, subVal)...)
					}
				}
			}
		}
	}

	return violations
}

func getType(val interface{}) string {
	if val == nil {
		return "null"
	}
	switch val.(type) {
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

func pathOrRoot(p string) string {
	if p == "" {
		return "$"
	}
	return strings.TrimPrefix(p, ".")
}
