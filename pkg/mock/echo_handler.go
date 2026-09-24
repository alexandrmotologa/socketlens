package mock

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

type clientConn struct {
	id   string
	conn *websocket.Conn
}

// Hub coordinates active mock connections for broadcasting.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*clientConn
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[string]*clientConn),
	}
}

func (h *Hub) register(c *clientConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.id] = c
}

func (h *Hub) unregister(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, id)
}

func (h *Hub) broadcast(ctx context.Context, msgType websocket.MessageType, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		_ = client.conn.Write(writeCtx, msgType, payload)
		cancel()
	}
}

func handleWebSocketConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, cfg ServerConfig, hub *Hub) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "mock server shutdown")

	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &clientConn{id: clientID, conn: conn}
	hub.register(client)
	defer hub.unregister(clientID)

	// Chaos reset timer if configured
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if cfg.Chaos.Reset > 0 {
		time.AfterFunc(cfg.Chaos.Reset, func() {
			cancel()
			_ = conn.Close(websocket.StatusGoingAway, "chaos simulation disconnect")
		})
	}

	rate := cfg.Rate
	if rate <= 0 {
		rate = 10
	}
	interval := time.Second / time.Duration(rate)

	switch cfg.Mode {
	case "llm-tokens":
		go func() {
			for i, token := range sampleLLMPromptTokens {
				select {
				case <-connCtx.Done():
					return
				case <-time.After(interval):
				}

				if cfg.Chaos.ShouldDrop() {
					continue
				}
				cfg.Chaos.ApplyLatency()

				chunk := GenerateLLMChunk(token, i, false)
				writeCtx, writeCancel := context.WithTimeout(connCtx, 2*time.Second)
				_ = conn.Write(writeCtx, websocket.MessageText, []byte(chunk))
				writeCancel()
			}

			// Final chunk
			finalChunk := GenerateLLMChunk("", len(sampleLLMPromptTokens), true)
			writeCtx, writeCancel := context.WithTimeout(connCtx, 2*time.Second)
			_ = conn.Write(writeCtx, websocket.MessageText, []byte(finalChunk))
			writeCancel()
		}()

	case "ticks":
		go func() {
			var seq uint64
			for {
				select {
				case <-connCtx.Done():
					return
				case <-time.After(interval):
				}

				if cfg.Chaos.ShouldDrop() {
					continue
				}
				cfg.Chaos.ApplyLatency()

				atomic.AddUint64(&seq, 1)
				tick := GenerateFinancialTick(seq)
				writeCtx, writeCancel := context.WithTimeout(connCtx, 2*time.Second)
				_ = conn.Write(writeCtx, websocket.MessageText, tick)
				writeCancel()
			}
		}()
	}

	// Echo & read loop
	for {
		msgType, payload, readErr := conn.Read(connCtx)
		if readErr != nil {
			return
		}

		if cfg.Chaos.ShouldDrop() {
			continue
		}
		cfg.Chaos.ApplyLatency()

		if cfg.Mode == "broadcast" {
			hub.broadcast(connCtx, msgType, payload)
		} else if cfg.Mode == "echo" || cfg.Mode == "" {
			writeCtx, writeCancel := context.WithTimeout(connCtx, 2*time.Second)
			_ = conn.Write(writeCtx, msgType, payload)
			writeCancel()
		}
	}
}
