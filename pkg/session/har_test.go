package session

import (
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

func TestHARExport(t *testing.T) {
	cfg := client.ConnectionConfig{
		URL:      "wss://api.example.com/stream",
		Protocol: client.ProtocolWS,
		Headers:  map[string]string{"Authorization": "Bearer token123"},
	}

	frames := []*client.Frame{
		{
			Sequence:  1,
			Timestamp: time.Now(),
			Direction: client.DirectionOutbound,
			OpCode:    client.OpCodeText,
			Payload:   []byte("{\"action\":\"subscribe\"}"),
		},
		{
			Sequence:  2,
			Timestamp: time.Now(),
			Direction: client.DirectionInbound,
			OpCode:    client.OpCodeText,
			Payload:   []byte("{\"status\":\"ok\"}"),
		},
	}

	harBytes, err := ExportToHAR(cfg, frames)
	if err != nil {
		t.Fatalf("HAR export failed: %v", err)
	}

	harStr := string(harBytes)
	if !strings.Contains(harStr, "_webSocketMessages") || !strings.Contains(harStr, "SocketLens") {
		t.Errorf("HAR missing key properties: %s", harStr)
	}

	wscatCmd := GenerateWscatCommand(cfg)
	if !strings.Contains(wscatCmd, "wscat -c") || !strings.Contains(wscatCmd, "token123") {
		t.Errorf("wscat command malformed: %s", wscatCmd)
	}

	curlCmd := GenerateCurlCommand(cfg)
	if !strings.Contains(curlCmd, "curl -N") {
		t.Errorf("curl command malformed: %s", curlCmd)
	}
}
