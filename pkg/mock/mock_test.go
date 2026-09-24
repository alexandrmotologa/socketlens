package mock

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

func TestMockServerEcho(t *testing.T) {
	srv := NewServer(ServerConfig{
		Port:  0, // Random port
		Route: "/ws/test",
		Mode:  "echo",
	})

	err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer srv.Stop()

	// Wait for listener to be active
	time.Sleep(50 * time.Millisecond)

	wsURL := srv.WSURL()
	if !strings.Contains(wsURL, "/ws/test") {
		t.Fatalf("unexpected WS URL: %s", wsURL)
	}

	echoReceived := make(chan string, 5)
	cfg := client.ConnectionConfig{
		ID:       "mock_test_client",
		URL:      wsURL,
		Protocol: client.ProtocolWS,
	}

	wsClient := client.NewWSClient(cfg, func(f *client.Frame) {
		if f.Direction == client.DirectionInbound {
			echoReceived <- string(f.Payload)
		}
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to mock server: %v", err)
	}
	defer wsClient.Close()

	time.Sleep(50 * time.Millisecond)

	msg := "hello from socketlens"
	err = wsClient.Send([]byte(msg), client.OpCodeText)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	select {
	case received := <-echoReceived:
		if received != msg {
			t.Errorf("expected '%s', got '%s'", msg, received)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for echo from mock server")
	}
}

func TestMockServerSSEStreaming(t *testing.T) {
	srv := NewServer(ServerConfig{
		Port:  0,
		Route: "/feed",
		Mode:  "llm-tokens",
		Rate:  50, // Fast for tests
	})

	err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer srv.Stop()

	time.Sleep(50 * time.Millisecond)

	var frames []*client.Frame
	var mu sync.Mutex
	done := make(chan struct{})

	cfg := client.ConnectionConfig{
		ID:       "mock_sse_client",
		URL:      srv.SSEURL(),
		Protocol: client.ProtocolSSE,
	}

	sseClient := client.NewSSEClient(cfg, func(f *client.Frame) {
		mu.Lock()
		frames = append(frames, f)
		count := len(frames)
		mu.Unlock()

		if count >= 5 {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = sseClient.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect SSE to mock server: %v", err)
	}
	defer sseClient.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for SSE tokens from mock server")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(frames) < 5 {
		t.Errorf("expected at least 5 frames, got %d", len(frames))
	}
}

func TestHealthCheck(t *testing.T) {
	srv := NewServer(ServerConfig{
		Port:  0,
		Route: "/ws",
	})
	if err := srv.Start(); err != nil {
		t.Fatalf("start error: %v", err)
	}
	defer srv.Stop()

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://%s/health", srv.Addr()))
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}
