package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// WSClient manages a WebSocket connection lifecycle.
type WSClient struct {
	config       ConnectionConfig
	conn         *websocket.Conn
	stats        ConnectionStats
	frameSeq     uint64
	frameHandler FrameHandler
	stateHandler StateHandler

	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	sendChan  chan *Frame
	closed    atomic.Bool
	reconWait time.Duration
}

// NewWSClient creates an unstarted WebSocket client.
func NewWSClient(cfg ConnectionConfig, onFrame FrameHandler, onState StateHandler) *WSClient {
	if cfg.ID == "" {
		cfg.ID = "ws_" + uuid.NewString()[:8]
	}
	if cfg.HandshakeTimeout == 0 {
		cfg.HandshakeTimeout = 10 * time.Second
	}
	if cfg.ReconnectInterval == 0 {
		cfg.ReconnectInterval = 2 * time.Second
	}

	return &WSClient{
		config:       cfg,
		stats:        ConnectionStats{State: StateDisconnected},
		frameHandler: onFrame,
		stateHandler: onState,
		sendChan:     make(chan *Frame, 256),
		reconWait:    cfg.ReconnectInterval,
	}
}

// Connect establishes the WebSocket connection and starts read/write loops.
func (c *WSClient) Connect(parentCtx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(parentCtx)
	c.closed.Store(false)

	return c.dialAndRun()
}

func (c *WSClient) dialAndRun() error {
	c.setState(StateConnecting, nil)

	ctx, dialCancel := context.WithTimeout(c.ctx, c.config.HandshakeTimeout)
	defer dialCancel()

	headers := http.Header{}
	for k, v := range c.config.Headers {
		headers.Set(k, v)
	}

	opts := &websocket.DialOptions{
		HTTPHeader:   headers,
		Subprotocols: c.config.Subprotocols,
	}

	if c.config.TLSInsecure {
		opts.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			},
		}
	}

	conn, _, err := websocket.Dial(ctx, c.config.URL, opts)
	if err != nil {
		c.setState(StateError, err)
		if c.config.AutoReconnect && !c.closed.Load() {
			go c.reconnectLoop()
			return nil
		}
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.stats.ConnectedAt = time.Now()
	c.reconWait = c.config.ReconnectInterval
	c.mu.Unlock()

	c.setState(StateConnected, nil)

	go c.writeLoop()
	go c.readLoop()
	if c.config.HeartbeatInterval > 0 {
		go c.heartbeatLoop()
	}

	return nil
}

func (c *WSClient) readLoop() {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			_ = c.conn.Close(websocket.StatusNormalClosure, "closing read loop")
		}
		c.mu.Unlock()

		if !c.closed.Load() && c.config.AutoReconnect {
			c.reconnectLoop()
		} else {
			c.setState(StateDisconnected, nil)
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		msgType, payload, err := conn.Read(c.ctx)
		if err != nil {
			if !c.closed.Load() && !errors.Is(err, context.Canceled) {
				c.setState(StateError, err)
			}
			return
		}

		opcode := OpCodeText
		if msgType == websocket.MessageBinary {
			opcode = OpCodeBinary
		}

		seq := atomic.AddUint64(&c.frameSeq, 1)
		frame := &Frame{
			ID:        fmt.Sprintf("%s_%d", c.config.ID, seq),
			ConnID:    c.config.ID,
			Sequence:  seq,
			Timestamp: time.Now(),
			Direction: DirectionInbound,
			Protocol:  c.config.Protocol,
			OpCode:    opcode,
			Payload:   payload,
			Format:    FormatRaw,
			Length:    len(payload),
		}

		c.mu.Lock()
		c.stats.FramesReceived++
		c.stats.BytesReceived += uint64(len(payload))
		c.mu.Unlock()

		if c.frameHandler != nil {
			c.frameHandler(frame)
		}
	}
}

func (c *WSClient) writeLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case frame, ok := <-c.sendChan:
			if !ok {
				return
			}

			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				continue
			}

			msgType := websocket.MessageText
			if frame.OpCode == OpCodeBinary {
				msgType = websocket.MessageBinary
			}

			writeCtx, writeCancel := context.WithTimeout(c.ctx, 5*time.Second)
			err := conn.Write(writeCtx, msgType, frame.Payload)
			writeCancel()

			if err != nil {
				c.setState(StateError, err)
				return
			}

			c.mu.Lock()
			c.stats.FramesSent++
			c.stats.BytesSent += uint64(len(frame.Payload))
			c.mu.Unlock()

			if c.frameHandler != nil {
				c.frameHandler(frame)
			}
		}
	}
}

func (c *WSClient) heartbeatLoop() {
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				continue
			}

			pingStart := time.Now()
			pingCtx, pingCancel := context.WithTimeout(c.ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			pingCancel()

			if err != nil {
				c.setState(StateError, fmt.Errorf("ping failed: %w", err))
				return
			}

			rtt := time.Since(pingStart)

			c.mu.Lock()
			c.stats.LastHeartbeatAt = time.Now()
			c.stats.HeartbeatLatencyMs = float64(rtt.Microseconds()) / 1000.0
			c.mu.Unlock()

			seq := atomic.AddUint64(&c.frameSeq, 1)
			frame := &Frame{
				ID:        fmt.Sprintf("%s_ping_%d", c.config.ID, seq),
				ConnID:    c.config.ID,
				Sequence:  seq,
				Timestamp: time.Now(),
				Direction: DirectionOutbound,
				Protocol:  c.config.Protocol,
				OpCode:    OpCodePing,
				Payload:   []byte("ping"),
				Length:    4,
				Latency:   rtt,
			}

			if c.frameHandler != nil {
				c.frameHandler(frame)
			}
		}
	}
}

func (c *WSClient) reconnectLoop() {
	c.setState(StateReconnecting, nil)

	for !c.closed.Load() {
		c.mu.Lock()
		c.stats.ReconnectAttempts++
		wait := c.reconWait
		// Exponential backoff capped at 30 seconds
		if c.reconWait < 30*time.Second {
			c.reconWait = time.Duration(float64(c.reconWait) * 1.5)
		}
		c.mu.Unlock()

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(wait):
		}

		if err := c.dialAndRun(); err == nil {
			return
		}
	}
}

// Send queues a frame for transmission.
func (c *WSClient) Send(payload []byte, opcode OpCode) error {
	if c.closed.Load() {
		return errors.New("client is closed")
	}

	seq := atomic.AddUint64(&c.frameSeq, 1)
	frame := &Frame{
		ID:        fmt.Sprintf("%s_%d", c.config.ID, seq),
		ConnID:    c.config.ID,
		Sequence:  seq,
		Timestamp: time.Now(),
		Direction: DirectionOutbound,
		Protocol:  c.config.Protocol,
		OpCode:    opcode,
		Payload:   payload,
		Format:    FormatRaw,
		Length:    len(payload),
	}

	select {
	case c.sendChan <- frame:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

// Close gracefully terminates the connection.
func (c *WSClient) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.State = StateDisconnected
	if c.conn != nil {
		err := c.conn.Close(websocket.StatusNormalClosure, "disconnect requested")
		c.conn = nil
		return err
	}
	return nil
}

// Stats returns current statistics.
func (c *WSClient) Stats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stats := c.stats
	if !stats.ConnectedAt.IsZero() && stats.State == StateConnected {
		stats.UptimeSeconds = time.Since(stats.ConnectedAt).Seconds()
	}
	return stats
}

func (c *WSClient) setState(s ConnectionState, err error) {
	c.mu.Lock()
	c.stats.State = s
	if err != nil {
		c.stats.LastError = err.Error()
	} else if s == StateConnected {
		c.stats.LastError = ""
	}
	c.mu.Unlock()

	if c.stateHandler != nil {
		c.stateHandler(s, err)
	}
}
