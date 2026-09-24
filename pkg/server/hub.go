package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/alexandrmotologa/socketlens/pkg/codec"
	"github.com/coder/websocket"
)

// UIClient represents an open browser control WebSocket connection.
type UIClient struct {
	conn     *websocket.Conn
	sendChan chan []byte
}

// TimelineHub distributes timeline frames and metrics to web studio clients.
type TimelineHub struct {
	mu           sync.RWMutex
	clients      map[*UIClient]struct{}
	recentFrames []*client.Frame
	batchBuffer  []*client.Frame
	broadcast    chan []byte
	register     chan *UIClient
	unregister   chan *UIClient
}

// NewTimelineHub creates a timeline distribution hub.
func NewTimelineHub() *TimelineHub {
	h := &TimelineHub{
		clients:      make(map[*UIClient]struct{}),
		recentFrames: make([]*client.Frame, 0, 1000),
		batchBuffer:  make([]*client.Frame, 0, 64),
		broadcast:    make(chan []byte, 1024),
		register:     make(chan *UIClient, 32),
		unregister:   make(chan *UIClient, 32),
	}
	go h.run()
	go h.batchTicker()
	return h
}

func (h *TimelineHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			// Send recent history batch to newly connected browser client
			if len(h.recentFrames) > 0 {
				historyMsg, err := json.Marshal(map[string]interface{}{
					"type":   "history",
					"frames": h.recentFrames,
				})
				if err == nil {
					client.sendChan <- historyMsg
				}
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.sendChan)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.sendChan <- message:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *TimelineHub) batchTicker() {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		if len(h.batchBuffer) == 0 {
			h.mu.Unlock()
			continue
		}

		batch := h.batchBuffer
		h.batchBuffer = make([]*client.Frame, 0, 64)
		h.mu.Unlock()

		msg, err := json.Marshal(map[string]interface{}{
			"type":   "batch",
			"frames": batch,
		})
		if err == nil {
			h.broadcast <- msg
		}
	}
}

// IngestFrame receives a frame from the protocol client, decodes it, and queues for distribution.
func (h *TimelineHub) IngestFrame(f *client.Frame) {
	// Auto decode payload
	fmtName, dec, err := codec.DecodeFrame(f.Payload)
	if err == nil {
		f.Format = client.PayloadFormat(fmtName)
		f.Decoded = dec
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Maintain recent history window
	h.recentFrames = append(h.recentFrames, f)
	if len(h.recentFrames) > 1000 {
		h.recentFrames = h.recentFrames[1:]
	}

	h.batchBuffer = append(h.batchBuffer, f)
}

// BroadcastState publishes connection state transitions to browser clients.
func (h *TimelineHub) BroadcastState(connID string, state client.ConnectionState, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	msg, _ := json.Marshal(map[string]interface{}{
		"type":          "state",
		"connection_id": connID,
		"state":         state,
		"error":         errStr,
		"timestamp":     time.Now().UTC(),
	})

	h.broadcast <- msg
}

// HandleWS upgrades browser HTTP requests to control WebSockets.
func (h *TimelineHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	uiClient := &UIClient{
		conn:     conn,
		sendChan: make(chan []byte, 512),
	}

	h.register <- uiClient

	ctx, cancel := context.WithCancel(r.Context())
	defer func() {
		cancel()
		h.unregister <- uiClient
		_ = conn.Close(websocket.StatusNormalClosure, "disconnecting")
	}()

	// Write loop
	go func() {
		for msg := range uiClient.sendChan {
			writeCtx, writeCancel := context.WithTimeout(ctx, 2*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, msg)
			writeCancel()
			if err != nil {
				return
			}
		}
	}()

	// Read loop to detect disconnects
	for {
		_, _, readErr := conn.Read(ctx)
		if readErr != nil {
			break
		}
	}
}
