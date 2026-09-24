package rules

import (
	"testing"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

func TestRulesEngine(t *testing.T) {
	engine := NewEngine()

	ruleID := engine.AddRule(Rule{
		Name:      "Challenge Responder",
		Condition: "contains",
		Pattern:   "auth_challenge",
		Action:    "reply",
		Response:  "{\"type\":\"auth_response\",\"token\":\"secret123\"}",
	})

	if ruleID == "" {
		t.Fatal("expected non-empty rule ID")
	}

	// 1. Matching frame
	matchingFrame := &client.Frame{
		Payload: []byte(`{"action": "auth_challenge", "nonce": "999"}`),
	}
	action := engine.Evaluate(matchingFrame)
	if action == nil {
		t.Fatal("expected action triggered for matching frame")
	}
	if action.Action != "reply" || action.Response != "{\"type\":\"auth_response\",\"token\":\"secret123\"}" {
		t.Errorf("unexpected action result: %+v", action)
	}

	// 2. Non-matching frame
	nonMatchingFrame := &client.Frame{
		Payload: []byte(`{"action": "ticker", "price": 100}`),
	}
	action = engine.Evaluate(nonMatchingFrame)
	if action != nil {
		t.Errorf("expected nil action, got %+v", action)
	}
}
