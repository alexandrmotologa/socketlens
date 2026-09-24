package proxy

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/alexandrmotologa/socketlens/pkg/mock"
)

func TestStreamProxyRelay(t *testing.T) {
	// 1. Mock server target
	targetSrv := mock.NewServer(mock.ServerConfig{
		Port:  0,
		Route: "/feed",
		Mode:  "echo",
	})
	if err := targetSrv.Start(); err != nil {
		t.Fatalf("start target server: %v", err)
	}
	defer targetSrv.Stop()

	time.Sleep(50 * time.Millisecond)

	// 2. Proxy instance
	var proxiedFrames []*client.Frame
	var mu sync.Mutex

	proxy := NewStreamProxy(ProxyConfig{
		LocalPort:        9898,
		TargetURL:        targetSrv.WSURL(),
		EnableBreakpoint: false,
	}, func(f *client.Frame) {
		mu.Lock()
		proxiedFrames = append(proxiedFrames, f)
		mu.Unlock()
	})

	if err := proxy.Start(); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer proxy.Stop()

	time.Sleep(50 * time.Millisecond)

	// 3. Client connecting to proxy
	echoChan := make(chan string, 5)
	c := client.NewWSClient(client.ConnectionConfig{
		URL:      "ws://127.0.0.1:9898/",
		Protocol: client.ProtocolWS,
	}, func(f *client.Frame) {
		if f.Direction == client.DirectionInbound {
			echoChan <- string(f.Payload)
		}
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.Connect(ctx); err != nil {
		t.Fatalf("client connect to proxy: %v", err)
	}
	defer c.Close()

	time.Sleep(50 * time.Millisecond)

	testMsg := "hello through proxy"
	_ = c.Send([]byte(testMsg), client.OpCodeText)

	select {
	case echo := <-echoChan:
		if echo != testMsg {
			t.Errorf("expected '%s', got '%s'", testMsg, echo)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for echo through proxy")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(proxiedFrames) < 2 {
		t.Errorf("expected at least 2 proxied frames (inbound + outbound), got %d", len(proxiedFrames))
	}
}

func TestStreamProxyTampering(t *testing.T) {
	targetSrv := mock.NewServer(mock.ServerConfig{
		Port:  0,
		Route: "/feed",
		Mode:  "echo",
	})
	if err := targetSrv.Start(); err != nil {
		t.Fatalf("start target server: %v", err)
	}
	defer targetSrv.Stop()

	time.Sleep(50 * time.Millisecond)

	proxy := NewStreamProxy(ProxyConfig{
		LocalPort:        9899,
		TargetURL:        targetSrv.WSURL(),
		EnableBreakpoint: true, // Enable breakpoint
	}, nil)

	if err := proxy.Start(); err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	defer proxy.Stop()

	time.Sleep(50 * time.Millisecond)

	echoChan := make(chan string, 5)
	c := client.NewWSClient(client.ConnectionConfig{
		URL:      "ws://127.0.0.1:9899/",
		Protocol: client.ProtocolWS,
	}, func(f *client.Frame) {
		if f.Direction == client.DirectionInbound {
			echoChan <- string(f.Payload)
		}
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.Connect(ctx); err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer c.Close()

	time.Sleep(50 * time.Millisecond)

	// Send message
	_ = c.Send([]byte("original"), client.OpCodeText)

	// Wait for breakpoint hit
	var bpID string
	for i := 0; i < 20; i++ {
		status := proxy.Status()
		pending := status["pending_breakpoints"].([]map[string]interface{})
		if len(pending) > 0 {
			bpID = pending[0]["id"].(string)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if bpID == "" {
		t.Fatal("expected breakpoint hit")
	}

	// Tamper payload on resume!
	err := proxy.ResumeBreakpoint(bpID, []byte("tampered_by_socketlens"), false)
	if err != nil {
		t.Fatalf("resume breakpoint error: %v", err)
	}

	// Wait for server echo breakpoint (server reflects tampered payload)
	for i := 0; i < 20; i++ {
		status := proxy.Status()
		pending := status["pending_breakpoints"].([]map[string]interface{})
		if len(pending) > 0 {
			_ = proxy.ResumeBreakpoint(pending[0]["id"].(string), []byte(pending[0]["payload"].(string)), false)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case echo := <-echoChan:
		if !strings.Contains(echo, "tampered_by_socketlens") {
			t.Errorf("expected echo to contain tampered payload, got: %s", echo)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tampered echo")
	}
}
