package schema

import (
	"testing"
)

func TestJSONSchemaValidator(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"required": ["id", "type", "price"],
		"properties": {
			"id": { "type": "string" },
			"type": { "type": "string" },
			"price": { "type": "number" },
			"active": { "type": "boolean" }
		}
	}`)

	v := NewValidator()
	err := v.SetSchema(schemaJSON)
	if err != nil {
		t.Fatalf("failed to set schema: %v", err)
	}

	// 1. Valid payload
	validPayload := []byte(`{"id": "ord_1", "type": "limit", "price": 100.5, "active": true}`)
	violations, err := v.Validate(validPayload)
	if err != nil || len(violations) > 0 {
		t.Errorf("expected 0 violations for valid payload, got %d: %+v", len(violations), violations)
	}

	// 2. Missing required field "price" and wrong type for "id"
	invalidPayload := []byte(`{"id": 12345, "type": "limit"}`)
	violations, err = v.Validate(invalidPayload)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}

	if len(violations) < 2 {
		t.Fatalf("expected at least 2 violations, got %d: %+v", len(violations), violations)
	}
}
