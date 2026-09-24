package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestWebSocketClientEcho(t *testing.T) {
	// Setup local echo WebSocket server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept error: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "done")

		for {
			typ, msg, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			err = conn.Write(r.Context(), typ, msg)
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	var receivedFrames []*Frame
	var mu sync.Mutex
	frameReceived := make(chan struct{}, 10)

	onFrame := func(f *Frame) {
		mu.Lock()
		receivedFrames = append(receivedFrames, f)
		mu.Unlock()
		if f.Direction == DirectionInbound {
			frameReceived <- struct{}{}
		}
	}

	cfg := ConnectionConfig{
		ID:                "test_ws",
		URL:               wsURL,
		Protocol:          ProtocolWS,
		HandshakeTimeout:  3 * time.Second,
		HeartbeatInterval: 50 * time.Millisecond,
	}

	wsClient := NewWSClient(cfg, onFrame, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer wsClient.Close()

	// Wait for connected state
	time.Sleep(100 * time.Millisecond)

	// Send message
	testMsg := []byte("{\"type\": \"ping\", \"seq\": 1}")
	err = wsClient.Send(testMsg, OpCodeText)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	select {
	case <-frameReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for echo response")
	}

	stats := wsClient.Stats()
	if stats.FramesSent == 0 || stats.FramesReceived == 0 {
		t.Errorf("expected frames recorded, got sent=%d recv=%d", stats.FramesSent, stats.FramesReceived)
	}
}

func TestSSEClientStreaming(t *testing.T) {
	// Setup local SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		events := []string{
			"id: 101\nevent: order_created\ndata: {\"order_id\": \"ord_1\"}\n\n",
			"id: 102\nevent: order_filled\ndata: {\"order_id\": \"ord_1\", \"status\": \"complete\"}\n\n",
		}

		for _, ev := range events {
			fmt.Fprint(w, ev)
			flusher.Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	defer server.Close()

	var receivedFrames []*Frame
	var mu sync.Mutex
	done := make(chan struct{})

	onFrame := func(f *Frame) {
		mu.Lock()
		receivedFrames = append(receivedFrames, f)
		count := len(receivedFrames)
		mu.Unlock()

		if count >= 2 {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	}

	cfg := ConnectionConfig{
		ID:       "test_sse",
		URL:      server.URL,
		Protocol: ProtocolSSE,
	}

	sseClient := NewSSEClient(cfg, onFrame, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := sseClient.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect SSE: %v", err)
	}
	defer sseClient.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE events")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(receivedFrames) < 2 {
		t.Fatalf("expected at least 2 frames, got %d", len(receivedFrames))
	}

	if receivedFrames[0].EventName != "order_created" {
		t.Errorf("expected event_name order_created, got %s", receivedFrames[0].EventName)
	}
	if receivedFrames[0].Metadata["id"] != "101" {
		t.Errorf("expected id 101, got %s", receivedFrames[0].Metadata["id"])
	}
	if receivedFrames[1].EventName != "order_filled" {
		t.Errorf("expected event_name order_filled, got %s", receivedFrames[1].EventName)
	}
}

func TestManagerRegistry(t *testing.T) {
	mgr := NewManager()

	cfg := ConnectionConfig{
		ID:       "sess_1",
		URL:      "ws://localhost:8080/feed",
		Protocol: ProtocolWS,
	}

	client, err := CreateClient(cfg, nil, nil)
	if err != nil {
		t.Fatalf("create client error: %v", err)
	}

	mgr.Register("sess_1", client, cfg)

	list := mgr.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(list))
	}

	retrieved, retrievedCfg, ok := mgr.Get("sess_1")
	if !ok || retrieved == nil || retrievedCfg.URL != cfg.URL {
		t.Fatal("failed to retrieve registered client")
	}

	err = mgr.Close("sess_1")
	if err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	if len(mgr.List()) != 0 {
		t.Errorf("expected empty list after close, got %d", len(mgr.List()))
	}
}
