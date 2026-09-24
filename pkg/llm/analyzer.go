package llm

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// ToolCall represents an incremental or completed LLM function call.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// SessionReport compiles real-time LLM streaming metrics.
type SessionReport struct {
	Model             string        `json:"model"`
	StartedAt         time.Time     `json:"started_at"`
	FirstTokenAt      time.Time     `json:"first_token_at,omitempty"`
	FinishedAt        time.Time     `json:"finished_at,omitempty"`
	TTFTMs            float64       `json:"ttft_ms"`
	TotalTokens       int           `json:"total_tokens"`
	TokensPerSecond   float64       `json:"tokens_per_sec"`
	AvgJitterMs       float64       `json:"avg_jitter_ms"`
	ReconstructedText string        `json:"reconstructed_text"`
	ToolCalls         []ToolCall    `json:"tool_calls,omitempty"`
	IsFinished        bool          `json:"is_finished"`
}

// Analyzer processes SSE token events.
type Analyzer struct {
	mu           sync.RWMutex
	report       SessionReport
	lastTokenAt  time.Time
	jitters      []time.Duration
	activeTools  map[int]*ToolCall
}

// NewAnalyzer creates an LLM stream inspector.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		report: SessionReport{
			StartedAt: time.Now(),
		},
		activeTools: make(map[int]*ToolCall),
		jitters:     make([]time.Duration, 0, 128),
	}
}

// IngestChunk parses an SSE line or JSON payload.
func (a *Analyzer) IngestChunk(data []byte) {
	str := strings.TrimSpace(string(data))
	if str == "" || str == "[DONE]" || str == "data: [DONE]" {
		a.mu.Lock()
		a.report.IsFinished = true
		a.report.FinishedAt = time.Now()
		a.calculateTPS()
		a.mu.Unlock()
		return
	}

	// Remove leading "data: " if present
	if strings.HasPrefix(str, "data:") {
		str = strings.TrimSpace(strings.TrimPrefix(str, "data:"))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		return
	}

	now := time.Now()

	a.mu.Lock()
	defer a.mu.Unlock()

	// Extract model name if not yet recorded
	if a.report.Model == "" {
		if m, ok := parsed["model"].(string); ok {
			a.report.Model = m
		}
	}

	// 1. OpenAI format: choices[0].delta
	if choices, ok := parsed["choices"].([]interface{}); ok && len(choices) > 0 {
		if firstChoice, ok := choices[0].(map[string]interface{}); ok {
			if delta, ok := firstChoice["delta"].(map[string]interface{}); ok {
				// Content token
				if content, ok := delta["content"].(string); ok && content != "" {
					a.recordToken(content, now)
				}

				// Tool call deltas
				if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
					for _, tc := range toolCalls {
						if tcMap, ok := tc.(map[string]interface{}); ok {
							idx := 0
							if idxf, ok := tcMap["index"].(float64); ok {
								idx = int(idxf)
							}
							if _, exists := a.activeTools[idx]; !exists {
								a.activeTools[idx] = &ToolCall{}
							}
							tool := a.activeTools[idx]
							if id, ok := tcMap["id"].(string); ok && id != "" {
								tool.ID = id
							}
							if fn, ok := tcMap["function"].(map[string]interface{}); ok {
								if name, ok := fn["name"].(string); ok && name != "" {
									tool.Name = name
								}
								if args, ok := fn["arguments"].(string); ok {
									tool.Arguments += args
								}
							}
						}
					}
				}
			}

			// Finish reason
			if fr, ok := firstChoice["finish_reason"].(string); ok && fr != "" {
				a.report.IsFinished = true
				a.report.FinishedAt = now
				a.calculateTPS()
			}
		}
	}

	// 2. Anthropic format: type == "content_block_delta"
	if typ, ok := parsed["type"].(string); ok && typ == "content_block_delta" {
		if delta, ok := parsed["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok && text != "" {
				a.recordToken(text, now)
			}
		}
	}

	// 3. Ollama format: response string
	if resp, ok := parsed["response"].(string); ok && resp != "" {
		a.recordToken(resp, now)
		if done, ok := parsed["done"].(bool); ok && done {
			a.report.IsFinished = true
			a.report.FinishedAt = now
			a.calculateTPS()
		}
	}
}

func (a *Analyzer) recordToken(token string, now time.Time) {
	if a.report.TotalTokens == 0 {
		a.report.FirstTokenAt = now
		a.report.TTFTMs = float64(now.Sub(a.report.StartedAt).Microseconds()) / 1000.0
	} else if !a.lastTokenAt.IsZero() {
		jitter := now.Sub(a.lastTokenAt)
		a.jitters = append(a.jitters, jitter)
	}

	a.lastTokenAt = now
	a.report.TotalTokens++
	a.report.ReconstructedText += token
	a.calculateTPS()
}

func (a *Analyzer) calculateTPS() {
	if a.report.TotalTokens <= 1 || a.report.FirstTokenAt.IsZero() {
		return
	}

	endTime := time.Now()
	if !a.report.FinishedAt.IsZero() {
		endTime = a.report.FinishedAt
	}

	durationSec := endTime.Sub(a.report.FirstTokenAt).Seconds()
	if durationSec > 0.05 {
		a.report.TokensPerSecond = float64(a.report.TotalTokens) / durationSec
	}

	if len(a.jitters) > 0 {
		var totalJitter time.Duration
		for _, j := range a.jitters {
			totalJitter += j
		}
		a.report.AvgJitterMs = (float64(totalJitter.Microseconds()) / 1000.0) / float64(len(a.jitters))
	}

	// Rebuild tool calls list
	toolList := make([]ToolCall, 0, len(a.activeTools))
	for _, t := range a.activeTools {
		toolList = append(toolList, *t)
	}
	a.report.ToolCalls = toolList
}

// Report returns current immutable snapshot of the analysis.
func (a *Analyzer) Report() SessionReport {
	a.mu.RLock()
	defer a.mu.RUnlock()
	rep := a.report
	toolList := make([]ToolCall, 0, len(a.activeTools))
	for _, t := range a.activeTools {
		toolList = append(toolList, *t)
	}
	rep.ToolCalls = toolList
	return rep
}
