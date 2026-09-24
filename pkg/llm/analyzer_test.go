package llm

import (
	"strings"
	"testing"
	"time"
)

func TestLLMAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()

	chunks := []string{
		`data: {"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`data: {"choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
		`data: {"choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	for _, c := range chunks {
		analyzer.IngestChunk([]byte(c))
		time.Sleep(10 * time.Millisecond)
	}

	report := analyzer.Report()
	if report.TotalTokens != 3 {
		t.Errorf("expected 3 tokens, got %d", report.TotalTokens)
	}

	if report.ReconstructedText != "Hello world!" {
		t.Errorf("expected 'Hello world!', got '%s'", report.ReconstructedText)
	}

	if !report.IsFinished {
		t.Error("expected report.IsFinished to be true")
	}

	if report.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", report.Model)
	}

	if report.TTFTMs < 0 {
		t.Errorf("invalid TTFT: %.2f", report.TTFTMs)
	}
}

func TestLLMToolCalls(t *testing.T) {
	analyzer := NewAnalyzer()

	chunk1 := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"get_weather","arguments":"{\"loc"}}]},"finish_reason":null}]}`
	chunk2 := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":\"Bucharest\"}"}}]},"finish_reason":"tool_calls"}]}`

	analyzer.IngestChunk([]byte(chunk1))
	analyzer.IngestChunk([]byte(chunk2))

	report := analyzer.Report()
	if len(report.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(report.ToolCalls))
	}

	tc := report.ToolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("expected name get_weather, got %s", tc.Name)
	}
	if !strings.Contains(tc.Arguments, "Bucharest") {
		t.Errorf("expected arguments containing Bucharest, got %s", tc.Arguments)
	}
}
