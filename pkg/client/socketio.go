package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// SocketIOClient coordinates a Socket.io v4 connection over WebSocket.
type SocketIOClient struct {
	config       ConnectionConfig
	wsClient     *WSClient
	stats        ConnectionStats
	frameSeq     uint64
	frameHandler FrameHandler
	stateHandler StateHandler

	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	closed    atomic.Bool
	sid       string
	namespace string
}

// NewSocketIOClient creates a new Socket.io v4 client.
func NewSocketIOClient(cfg ConnectionConfig, namespace string, onFrame FrameHandler, onState StateHandler) *SocketIOClient {
	if cfg.ID == "" {
		cfg.ID = "sio_" + uuid.NewString()[:8]
	}
	if namespace == "" {
		namespace = "/"
	}

	client := &SocketIOClient{
		config:       cfg,
		stats:        ConnectionStats{State: StateDisconnected},
		frameHandler: onFrame,
		stateHandler: onState,
		namespace:    namespace,
	}

	return client
}

// Connect establishes the underlying WebSocket connection and Engine.IO handshake.
func (c *SocketIOClient) Connect(parentCtx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(parentCtx)
	c.closed.Store(false)

	targetURL, err := url.Parse(c.config.URL)
	if err != nil {
		return fmt.Errorf("parsing socket.io url: %w", err)
	}

	// Normalize scheme to ws/wss
	if targetURL.Scheme == "http" {
		targetURL.Scheme = "ws"
	} else if targetURL.Scheme == "https" {
		targetURL.Scheme = "wss"
	}

	// Ensure Engine.IO v4 path and query params
	if !strings.HasSuffix(targetURL.Path, "/socket.io/") && !strings.HasSuffix(targetURL.Path, "/socket.io") {
		targetURL.Path = strings.TrimSuffix(targetURL.Path, "/") + "/socket.io/"
	}

	q := targetURL.Query()
	q.Set("EIO", "4")
	q.Set("transport", "websocket")
	targetURL.RawQuery = q.Encode()

	wsCfg := c.config
	wsCfg.URL = targetURL.String()
	wsCfg.Protocol = ProtocolSocketIO

	c.wsClient = NewWSClient(wsCfg, c.handleWSFrame, c.handleWSState)
	return c.wsClient.Connect(c.ctx)
}

func (c *SocketIOClient) handleWSState(state ConnectionState, err error) {
	c.setState(state, err)
}

func (c *SocketIOClient) handleWSFrame(frame *Frame) {
	if frame.Direction == DirectionOutbound {
		if c.frameHandler != nil {
			c.frameHandler(frame)
		}
		return
	}

	payloadStr := string(frame.Payload)
	if len(payloadStr) == 0 {
		return
	}

	// Engine.IO packet parsing
	packetType := payloadStr[0]
	packetBody := payloadStr[1:]

	switch packetType {
	case '0': // Engine.IO OPEN
		var handshake struct {
			SID          string   `json:"sid"`
			Upgrades     []string `json:"upgrades"`
			PingInterval int      `json:"pingInterval"`
			PingTimeout  int      `json:"pingTimeout"`
		}
		if err := json.Unmarshal([]byte(packetBody), &handshake); err == nil {
			c.mu.Lock()
			c.sid = handshake.SID
			c.mu.Unlock()
		}

		// Connect to namespace if not default root
		if c.namespace != "/" {
			_ = c.wsClient.Send([]byte("40"+c.namespace+","), OpCodeText)
		}

	case '2': // Engine.IO PING from server -> reply with PONG
		_ = c.wsClient.Send([]byte("3"), OpCodeText)
		return

	case '3': // Engine.IO PONG
		return

	case '4': // Engine.IO MESSAGE -> Socket.io packet
		c.parseSocketIOPacket(packetBody)
		return
	}

	if c.frameHandler != nil {
		c.frameHandler(frame)
	}
}

func (c *SocketIOClient) parseSocketIOPacket(body string) {
	if len(body) == 0 {
		return
	}

	sioType := body[0]
	rest := body[1:]

	// Skip namespace prefix if present (e.g. "/admin,")
	if strings.HasPrefix(rest, "/") {
		commaIdx := strings.Index(rest, ",")
		if commaIdx != -1 {
			rest = rest[commaIdx+1:]
		}
	}

	seq := atomic.AddUint64(&c.frameSeq, 1)

	switch sioType {
	case '0': // CONNECT
		frame := &Frame{
			ID:        fmt.Sprintf("%s_%d", c.config.ID, seq),
			ConnID:    c.config.ID,
			Sequence:  seq,
			Timestamp: time.Now(),
			Direction: DirectionInbound,
			Protocol:  ProtocolSocketIO,
			OpCode:    OpCodeEvent,
			Payload:   []byte(rest),
			Length:    len(rest),
			EventName: "connect",
		}
		c.dispatchFrame(frame)

	case '2': // EVENT: ["event_name", payload]
		var parts []interface{}
		eventName := "event"
		if err := json.Unmarshal([]byte(rest), &parts); err == nil && len(parts) > 0 {
			if str, ok := parts[0].(string); ok {
				eventName = str
			}
		}

		frame := &Frame{
			ID:        fmt.Sprintf("%s_%d", c.config.ID, seq),
			ConnID:    c.config.ID,
			Sequence:  seq,
			Timestamp: time.Now(),
			Direction: DirectionInbound,
			Protocol:  ProtocolSocketIO,
			OpCode:    OpCodeEvent,
			Payload:   []byte(rest),
			Length:    len(rest),
			EventName: eventName,
		}
		c.dispatchFrame(frame)

	case '1': // DISCONNECT
		c.setState(StateDisconnected, nil)
	}
}

func (c *SocketIOClient) dispatchFrame(frame *Frame) {
	c.mu.Lock()
	c.stats.FramesReceived++
	c.stats.BytesReceived += uint64(len(frame.Payload))
	c.mu.Unlock()

	if c.frameHandler != nil {
		c.frameHandler(frame)
	}
}

// Emit sends a Socket.io v4 event.
func (c *SocketIOClient) Emit(eventName string, data interface{}) error {
	if c.wsClient == nil {
		return errors.New("client not connected")
	}

	payloadSlice := []interface{}{eventName, data}
	payloadBytes, err := json.Marshal(payloadSlice)
	if err != nil {
		return fmt.Errorf("marshalling event: %w", err)
	}

	var packet string
	if c.namespace == "/" {
		packet = fmt.Sprintf("42%s", string(payloadBytes))
	} else {
		packet = fmt.Sprintf("42%s,%s", c.namespace, string(payloadBytes))
	}

	return c.wsClient.Send([]byte(packet), OpCodeText)
}

// Send transmits a raw or JSON event packet over the Socket.io connection.
func (c *SocketIOClient) Send(payload []byte, opcode OpCode) error {
	if c.wsClient == nil {
		return errors.New("client not connected")
	}

	payloadStr := strings.TrimSpace(string(payload))
	var packet string

	// If already an Engine.IO/Socket.io packet (e.g. starts with "42")
	if strings.HasPrefix(payloadStr, "4") {
		packet = payloadStr
	} else if strings.HasPrefix(payloadStr, "[") {
		// Valid JSON event array: ["event", data]
		if c.namespace == "/" {
			packet = fmt.Sprintf("42%s", payloadStr)
		} else {
			packet = fmt.Sprintf("42%s,%s", c.namespace, payloadStr)
		}
	} else {
		// Default to emitting as "message" event
		encoded, _ := json.Marshal([]interface{}{"message", payloadStr})
		if c.namespace == "/" {
			packet = fmt.Sprintf("42%s", string(encoded))
		} else {
			packet = fmt.Sprintf("42%s,%s", c.namespace, string(encoded))
		}
	}

	return c.wsClient.Send([]byte(packet), OpCodeText)
}

// Close closes the connection.
func (c *SocketIOClient) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	if c.wsClient != nil {
		return c.wsClient.Close()
	}
	return nil
}

// Stats returns connection metrics.
func (c *SocketIOClient) Stats() ConnectionStats {
	if c.wsClient != nil {
		return c.wsClient.Stats()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *SocketIOClient) setState(s ConnectionState, err error) {
	c.mu.Lock()
	c.stats.State = s
	if err != nil {
		c.stats.LastError = err.Error()
	}
	c.mu.Unlock()

	if c.stateHandler != nil {
		c.stateHandler(s, err)
	}
}
