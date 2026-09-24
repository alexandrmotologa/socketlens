package rules

import (
	"strings"
	"sync"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/google/uuid"
)

// Rule defines an automated event-driven trigger and action.
type Rule struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Condition string `json:"condition"` // "contains", "exact", "event_name", "opcode"
	Pattern   string `json:"pattern"`
	Action    string `json:"action"` // "reply", "drop", "alert"
	Response  string `json:"response,omitempty"`
}

// ActionResult describes what action should be taken when a rule triggers.
type ActionResult struct {
	RuleID   string
	Action   string
	Response string
}

// Engine evaluates streaming frames against user-defined rules.
type Engine struct {
	mu    sync.RWMutex
	rules map[string]*Rule
}

// NewEngine creates an automation rule engine.
func NewEngine() *Engine {
	return &Engine{
		rules: make(map[string]*Rule),
	}
}

// AddRule registers a new automation rule.
func (e *Engine) AddRule(r Rule) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if r.ID == "" {
		r.ID = "rule_" + uuid.NewString()[:8]
	}
	r.Enabled = true
	e.rules[r.ID] = &r
	return r.ID
}

// ListRules returns all configured rules.
func (e *Engine) ListRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	list := make([]Rule, 0, len(e.rules))
	for _, r := range e.rules {
		list = append(list, *r)
	}
	return list
}

// DeleteRule removes a rule by ID.
func (e *Engine) DeleteRule(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, id)
}

// Evaluate evaluates a frame and returns an action if a rule matches.
func (e *Engine) Evaluate(f *client.Frame) *ActionResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	payloadStr := string(f.Payload)

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		matched := false
		switch rule.Condition {
		case "contains":
			matched = strings.Contains(payloadStr, rule.Pattern)
		case "exact":
			matched = strings.TrimSpace(payloadStr) == strings.TrimSpace(rule.Pattern)
		case "event_name":
			matched = f.EventName == rule.Pattern
		case "opcode":
			matched = string(f.OpCode) == rule.Pattern
		}

		if matched {
			return &ActionResult{
				RuleID:   rule.ID,
				Action:   rule.Action,
				Response: rule.Response,
			}
		}
	}

	return nil
}
