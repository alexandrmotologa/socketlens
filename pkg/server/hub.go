package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/alexandrmotologa/socketlens/pkg/codec"
	"github.com/alexandrmotologa/socketlens/pkg/llm"
	"github.com/alexandrmotologa/socketlens/pkg/rules"
	"github.com/alexandrmotologa/socketlens/pkg/schema"
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
	validator    *schema.Validator
	rulesEngine  *rules.Engine
	llmAnalyzer  *llm.Analyzer
	clientMgr    *client.Manager
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
		validator:    schema.NewValidator(),
		rulesEngine:  rules.NewEngine(),
		llmAnalyzer:  llm.NewAnalyzer(),
	}
	go h.run()
	go h.batchTicker()
	return h
}

// SetClientManager binds the client manager to enable auto-responders.
func (h *TimelineHub) SetClientManager(mgr *client.Manager) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clientMgr = mgr
}

// Validator returns the active schema validator.
func (h *TimelineHub) Validator() *schema.Validator {
	return h.validator
}

// RulesEngine returns the automation rules engine.
func (h *TimelineHub) RulesEngine() *rules.Engine {
	return h.rulesEngine
}

// LLMAnalyzer returns the real-time LLM stream analyzer.
func (h *TimelineHub) LLMAnalyzer() *llm.Analyzer {
	return h.llmAnalyzer
}

// RecentFrames returns a snapshot slice of recent timeline frames.
func (h *TimelineHub) RecentFrames() []*client.Frame {
	h.mu.RLock()
	defer h.mu.RUnlock()
	copied := make([]*client.Frame, len(h.recentFrames))
	copy(copied, h.recentFrames)
	return copied
}

// Clear clears the in-memory timeline history.
func (h *TimelineHub) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recentFrames = make([]*client.Frame, 0, 1000)
	h.batchBuffer = make([]*client.Frame, 0, 64)
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
	// 1. Auto decode payload
	fmtName, dec, err := codec.DecodeFrame(f.Payload)
	if err == nil {
		f.Format = client.PayloadFormat(fmtName)
		f.Decoded = dec
	}

	// 2. Real-time Schema Validation
	if h.validator != nil && h.validator.IsActive() {
		violations, _ := h.validator.Validate(f.Payload)
		if len(violations) > 0 {
			for _, v := range violations {
				f.SchemaErrors = append(f.SchemaErrors, v.Message)
			}
		}
	}

	// 3. LLM Stream Ingestion
	if f.Protocol == client.ProtocolSSE || strings.Contains(string(f.Payload), "chat.completion") {
		if h.llmAnalyzer != nil {
			h.llmAnalyzer.IngestChunk(f.Payload)
		}
	}

	// 4. Automation Rules Evaluation
	if h.rulesEngine != nil && f.Direction == client.DirectionInbound {
		action := h.rulesEngine.Evaluate(f)
		if action != nil && action.Action == "reply" && action.Response != "" {
			h.mu.RLock()
			mgr := h.clientMgr
			h.mu.RUnlock()
			if mgr != nil {
				c, _, ok := mgr.Get(f.ConnID)
				if ok && c != nil {
					_ = c.Send([]byte(action.Response), client.OpCodeText)
				}
			}
		}
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
