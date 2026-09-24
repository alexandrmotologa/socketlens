package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// SSEClient manages an event-stream subscription.
type SSEClient struct {
	config       ConnectionConfig
	stats        ConnectionStats
	frameSeq     uint64
	frameHandler FrameHandler
	stateHandler StateHandler

	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	closed      atomic.Bool
	lastEventID string
	retryWait   time.Duration
}

// NewSSEClient initializes an SSE client.
func NewSSEClient(cfg ConnectionConfig, onFrame FrameHandler, onState StateHandler) *SSEClient {
	if cfg.ID == "" {
		cfg.ID = "sse_" + uuid.NewString()[:8]
	}
	if cfg.ReconnectInterval == 0 {
		cfg.ReconnectInterval = 3 * time.Second
	}

	return &SSEClient{
		config:       cfg,
		stats:        ConnectionStats{State: StateDisconnected},
		frameHandler: onFrame,
		stateHandler: onState,
		retryWait:    cfg.ReconnectInterval,
	}
}

// Connect starts the SSE streaming reader.
func (c *SSEClient) Connect(parentCtx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(parentCtx)
	c.closed.Store(false)

	go c.lifecycle()
	return nil
}

func (c *SSEClient) lifecycle() {
	for !c.closed.Load() {
		err := c.streamOnce()
		if err != nil && !c.closed.Load() {
			c.setState(StateError, err)
		}

		if c.closed.Load() || !c.config.AutoReconnect {
			c.setState(StateDisconnected, nil)
			return
		}

		c.setState(StateReconnecting, nil)
		c.mu.Lock()
		c.stats.ReconnectAttempts++
		wait := c.retryWait
		c.mu.Unlock()

		select {
		case <-c.ctx.Done():
			c.setState(StateDisconnected, nil)
			return
		case <-time.After(wait):
		}
	}
}

func (c *SSEClient) streamOnce() error {
	c.setState(StateConnecting, nil)

	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, c.config.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}

	c.mu.RLock()
	if c.lastEventID != "" {
		req.Header.Set("Last-Event-ID", c.lastEventID)
	}
	c.mu.RUnlock()

	httpClient := &http.Client{
		Timeout: 0, // Streaming requests do not use client-level timeout
	}

	if c.config.TLSInsecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec
			},
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, resp.Status)
	}

	c.mu.Lock()
	c.stats.ConnectedAt = time.Now()
	c.mu.Unlock()
	c.setState(StateConnected, nil)

	scanner := bufio.NewScanner(resp.Body)
	// Support up to 512KB lines for large JSON SSE payloads
	scanner.Buffer(make([]byte, 64*1024), 512*1024)

	var currentEvent string
	var currentData strings.Builder
	var currentID string

	flush := func() {
		if currentData.Len() == 0 && currentEvent == "" {
			return
		}

		payload := []byte(strings.TrimSuffix(currentData.String(), "\n"))
		seq := atomic.AddUint64(&c.frameSeq, 1)

		if currentEvent == "" {
			currentEvent = "message"
		}

		frame := &Frame{
			ID:        fmt.Sprintf("%s_%d", c.config.ID, seq),
			ConnID:    c.config.ID,
			Sequence:  seq,
			Timestamp: time.Now(),
			Direction: DirectionInbound,
			Protocol:  ProtocolSSE,
			OpCode:    OpCodeEvent,
			Payload:   payload,
			Format:    FormatRaw,
			Length:    len(payload),
			EventName: currentEvent,
			Metadata:  map[string]string{},
		}

		if currentID != "" {
			frame.Metadata["id"] = currentID
			c.mu.Lock()
			c.lastEventID = currentID
			c.mu.Unlock()
		}

		c.mu.Lock()
		c.stats.FramesReceived++
		c.stats.BytesReceived += uint64(len(payload))
		c.mu.Unlock()

		if c.frameHandler != nil {
			c.frameHandler(frame)
		}

		currentEvent = ""
		currentData.Reset()
		currentID = ""
	}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line marks end of event block
			flush()
			continue
		}

		// Comment or ping line
		if strings.HasPrefix(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		field := parts[0]
		value := ""
		if len(parts) > 1 {
			value = strings.TrimPrefix(parts[1], " ")
		}

		switch field {
		case "event":
			currentEvent = value
		case "data":
			currentData.WriteString(value)
			currentData.WriteString("\n")
		case "id":
			currentID = value
		case "retry":
			if ms, err := strconv.Atoi(value); err == nil && ms > 0 {
				c.mu.Lock()
				c.retryWait = time.Duration(ms) * time.Millisecond
				c.mu.Unlock()
			}
		}
	}

	flush()
	return scanner.Err()
}

// Send returns an error because Server-Sent Events is a unidirectional server-to-client stream.
func (c *SSEClient) Send(payload []byte, opcode OpCode) error {
	return errors.New("server-sent events does not support outbound client messages")
}

// Close disconnects the SSE reader.
func (c *SSEClient) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	c.setState(StateDisconnected, nil)
	return nil
}

// Stats returns connection metrics.
func (c *SSEClient) Stats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stats := c.stats
	if !stats.ConnectedAt.IsZero() && stats.State == StateConnected {
		stats.UptimeSeconds = time.Since(stats.ConnectedAt).Seconds()
	}
	return stats
}

func (c *SSEClient) setState(s ConnectionState, err error) {
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
