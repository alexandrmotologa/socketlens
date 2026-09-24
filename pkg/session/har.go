package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

// HARLog represents the top-level HTTP Archive 1.2 document.
type HARLog struct {
	Log HAREntryLog `json:"log"`
}

type HAREntryLog struct {
	Version string     `json:"version"`
	Creator HARCreator `json:"creator"`
	Entries []HAREntry `json:"entries"`
}

type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type HAREntry struct {
	StartedDateTime    string              `json:"startedDateTime"`
	Time               int64               `json:"time"`
	Request            HARRequest          `json:"request"`
	Response           HARResponse         `json:"response"`
	WebSocketMessages  []HARWSMessage      `json:"_webSocketMessages,omitempty"`
}

type HARRequest struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
}

type HARResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
}

type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type HARWSMessage struct {
	Type   string  `json:"type"` // "send" or "receive"
	Time   float64 `json:"time"` // Epoch timestamp in seconds with millisecond decimals
	Opcode int     `json:"opcode"`
	Data   string  `json:"data"`
}

// ExportToHAR generates a Chrome DevTools-compatible HAR 1.2 JSON document.
func ExportToHAR(cfg client.ConnectionConfig, frames []*client.Frame) ([]byte, error) {
	wsMessages := make([]HARWSMessage, 0, len(frames))

	for _, f := range frames {
		msgType := "receive"
		if f.Direction == client.DirectionOutbound {
			msgType = "send"
		}

		opcodeNum := 1 // Text
		if f.OpCode == client.OpCodeBinary {
			opcodeNum = 2
		} else if f.OpCode == client.OpCodePing {
			opcodeNum = 9
		} else if f.OpCode == client.OpCodePong {
			opcodeNum = 10
		}

		data := f.Decoded
		if data == "" {
			data = string(f.Payload)
		}

		wsMessages = append(wsMessages, HARWSMessage{
			Type:   msgType,
			Time:   float64(f.Timestamp.UnixNano()) / 1e9,
			Opcode: opcodeNum,
			Data:   data,
		})
	}

	headers := []HARHeader{
		{Name: "Upgrade", Value: "websocket"},
		{Name: "Connection", Value: "Upgrade"},
	}
	for k, v := range cfg.Headers {
		headers = append(headers, HARHeader{Name: k, Value: v})
	}

	startTime := time.Now().UTC().Format(time.RFC3339Nano)
	if len(frames) > 0 {
		startTime = frames[0].Timestamp.UTC().Format(time.RFC3339Nano)
	}

	har := HARLog{
		Log: HAREntryLog{
			Version: "1.2",
			Creator: HARCreator{
				Name:    "SocketLens",
				Version: "1.0.0",
			},
			Entries: []HAREntry{
				{
					StartedDateTime: startTime,
					Time:            0,
					Request: HARRequest{
						Method:      "GET",
						URL:         cfg.URL,
						HTTPVersion: "HTTP/1.1",
						Headers:     headers,
					},
					Response: HARResponse{
						Status:      101,
						StatusText:  "Switching Protocols",
						HTTPVersion: "HTTP/1.1",
						Headers: []HARHeader{
							{Name: "Upgrade", Value: "websocket"},
							{Name: "Connection", Value: "Upgrade"},
						},
					},
					WebSocketMessages: wsMessages,
				},
			},
		},
	}

	return json.MarshalIndent(har, "", "  ")
}

// GenerateWscatCommand creates a CLI invocation string.
func GenerateWscatCommand(cfg client.ConnectionConfig) string {
	var parts []string
	parts = append(parts, "wscat", "-c", fmt.Sprintf("'%s'", cfg.URL))

	for k, v := range cfg.Headers {
		parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, v))
	}
	for _, sub := range cfg.Subprotocols {
		parts = append(parts, "-s", fmt.Sprintf("'%s'", sub))
	}

	return strings.Join(parts, " ")
}

// GenerateCurlCommand creates a cURL invocation string for SSE or handshake testing.
func GenerateCurlCommand(cfg client.ConnectionConfig) string {
	var parts []string
	parts = append(parts, "curl", "-N")

	if cfg.Protocol == client.ProtocolSSE {
		parts = append(parts, "-H", "'Accept: text/event-stream'")
	}

	for k, v := range cfg.Headers {
		parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, v))
	}

	parts = append(parts, fmt.Sprintf("'%s'", cfg.URL))
	return strings.Join(parts, " ")
}
