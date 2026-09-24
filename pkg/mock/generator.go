package mock

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var sampleLLMPromptTokens = []string{
	"The", " architecture", " of", " modern", " distributed", " systems", " relies",
	" on", " event-driven", " pipelines", " and", " low-latency", " message",
	" queues.", " When", " deploying", " streaming", " services,", " teams",
	" must", " balance", " throughput,", " memory", " allocation,", " and",
	" backpressure.", " SocketLens", " provides", " the", " tooling", " required",
	" to", " inspect", " these", " interactions", " with", " precision.",
}

// GenerateLLMChunk formats a mock OpenAI-style chat completion chunk.
func GenerateLLMChunk(token string, index int, isLast bool) string {
	finishReason := interface{}(nil)
	deltaContent := token
	if isLast {
		finishReason = "stop"
		deltaContent = ""
	}

	payload := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "gpt-4o-mock",
		"choices": []map[string]interface{}{
			{
				"index": index,
				"delta": map[string]interface{}{
					"content": deltaContent,
				},
				"finish_reason": finishReason,
			},
		},
	}

	bytes, _ := json.Marshal(payload)
	return string(bytes)
}

// GenerateFinancialTick creates a realistic crypto/equity quote payload.
func GenerateFinancialTick(seq uint64) []byte {
	symbols := []string{"BTC-USD", "ETH-USD", "SOL-USD", "NVDA", "AAPL"}
	sym := symbols[rand.Intn(len(symbols))]

	basePrices := map[string]float64{
		"BTC-USD": 64200.0,
		"ETH-USD": 3450.0,
		"SOL-USD": 145.0,
		"NVDA":    128.0,
		"AAPL":    225.0,
	}

	base := basePrices[sym]
	change := (rand.Float64() - 0.5) * (base * 0.002)
	price := base + change
	size := 0.1 + rand.Float64()*5.0

	tick := map[string]interface{}{
		"type":      "ticker",
		"sequence":  seq,
		"symbol":    sym,
		"price":     fmt.Sprintf("%.2f", price),
		"bid":       fmt.Sprintf("%.2f", price-0.05),
		"ask":       fmt.Sprintf("%.2f", price+0.05),
		"size":      fmt.Sprintf("%.4f", size),
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}

	bytes, _ := json.Marshal(tick)
	return bytes
}

// FormatSSEEvent wraps a data string in W3C SSE wire framing.
func FormatSSEEvent(event string, id string, data string) string {
	var b strings.Builder
	if id != "" {
		b.WriteString(fmt.Sprintf("id: %s\n", id))
	}
	if event != "" {
		b.WriteString(fmt.Sprintf("event: %s\n", event))
	}
	for _, line := range strings.Split(data, "\n") {
		b.WriteString(fmt.Sprintf("data: %s\n", line))
	}
	b.WriteString("\n")
	return b.String()
}
