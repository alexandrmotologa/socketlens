package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/coder/websocket"
)

// BreakpointFrame holds an intercepted frame awaiting developer decision.
type BreakpointFrame struct {
	ID        string           `json:"id"`
	Direction client.Direction `json:"direction"`
	OpCode    client.OpCode    `json:"opcode"`
	Payload   string           `json:"payload"`
	Timestamp time.Time        `json:"timestamp"`
	resumeCh  chan []byte
	dropCh    chan struct{}
}

// ProxyConfig configures the stream interception proxy.
type ProxyConfig struct {
	LocalPort        int               `json:"local_port"`
	TargetURL        string            `json:"target_url"`
	Headers          map[string]string `json:"headers,omitempty"`
	EnableBreakpoint bool              `json:"enable_breakpoint"`
}

// StreamProxy coordinates transparent stream forwarding and interception.
type StreamProxy struct {
	config      ProxyConfig
	listener    net.Listener
	httpServer  *http.Server
	onFrame     client.FrameHandler
	mu          sync.RWMutex
	running     bool
	frameSeq    uint64
	breakpoints map[string]*BreakpointFrame
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewStreamProxy creates a proxy instance.
func NewStreamProxy(cfg ProxyConfig, onFrame client.FrameHandler) *StreamProxy {
	return &StreamProxy{
		config:      cfg,
		onFrame:     onFrame,
		breakpoints: make(map[string]*BreakpointFrame),
	}
}

// Start opens the local listener and handles client connections.
func (p *StreamProxy) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return errors.New("proxy already running")
	}

	addr := fmt.Sprintf("127.0.0.1:%d", p.config.LocalPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("binding proxy listener to %s: %w", addr, err)
	}
	p.listener = listener

	p.ctx, p.cancel = context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleConnection)

	p.httpServer = &http.Server{Handler: mux}
	p.running = true

	go func() {
		_ = p.httpServer.Serve(p.listener)
	}()

	return nil
}

func (p *StreamProxy) handleConnection(w http.ResponseWriter, r *http.Request) {
	clientConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer clientConn.Close(websocket.StatusNormalClosure, "proxy client disconnected")

	// Connect upstream to target server
	dialCtx, dialCancel := context.WithTimeout(p.ctx, 10*time.Second)
	headers := http.Header{}
	for k, v := range p.config.Headers {
		headers.Set(k, v)
	}

	serverConn, _, err := websocket.Dial(dialCtx, p.config.TargetURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	dialCancel()

	if err != nil {
		_ = clientConn.Close(websocket.StatusInternalError, "failed to dial target server")
		return
	}
	defer serverConn.Close(websocket.StatusNormalClosure, "proxy server disconnected")

	connCtx, cancel := context.WithCancel(p.ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// Pump: Client -> Server
	go func() {
		defer wg.Done()
		defer cancel()
		p.forwardLoop(connCtx, clientConn, serverConn, client.DirectionOutbound)
	}()

	// Pump: Server -> Client
	go func() {
		defer wg.Done()
		defer cancel()
		p.forwardLoop(connCtx, serverConn, clientConn, client.DirectionInbound)
	}()

	wg.Wait()
}

func (p *StreamProxy) forwardLoop(ctx context.Context, src, dst *websocket.Conn, dir client.Direction) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgType, payload, err := src.Read(ctx)
		if err != nil {
			return
		}

		opcode := client.OpCodeText
		if msgType == websocket.MessageBinary {
			opcode = client.OpCodeBinary
		}

		seq := atomic.AddUint64(&p.frameSeq, 1)
		frame := &client.Frame{
			ID:        fmt.Sprintf("proxy_%d", seq),
			ConnID:    "proxy_session",
			Sequence:  seq,
			Timestamp: time.Now(),
			Direction: dir,
			Protocol:  client.ProtocolWS,
			OpCode:    opcode,
			Payload:   payload,
			Format:    client.FormatRaw,
			Length:    len(payload),
			Metadata:  map[string]string{"proxy": "true"},
		}

		if p.onFrame != nil {
			p.onFrame(frame)
		}

		// Check breakpoint tampering
		p.mu.RLock()
		bpEnabled := p.config.EnableBreakpoint
		p.mu.RUnlock()

		if bpEnabled {
			bp := &BreakpointFrame{
				ID:        frame.ID,
				Direction: dir,
				OpCode:    opcode,
				Payload:   string(payload),
				Timestamp: time.Now(),
				resumeCh:  make(chan []byte, 1),
				dropCh:    make(chan struct{}, 1),
			}

			p.mu.Lock()
			p.breakpoints[bp.ID] = bp
			p.mu.Unlock()

			// Wait for resume or drop decision
			select {
			case <-ctx.Done():
				return
			case modifiedPayload := <-bp.resumeCh:
				payload = modifiedPayload
			case <-bp.dropCh:
				continue // Drop frame without forwarding
			}
		}

		// Forward frame
		writeCtx, writeCancel := context.WithTimeout(ctx, 5*time.Second)
		err = dst.Write(writeCtx, msgType, payload)
		writeCancel()
		if err != nil {
			return
		}
	}
}

// ResumeBreakpoint releases an intercepted frame with optional modifications.
func (p *StreamProxy) ResumeBreakpoint(id string, modifiedPayload []byte, drop bool) error {
	p.mu.Lock()
	bp, exists := p.breakpoints[id]
	if exists {
		delete(p.breakpoints, id)
	}
	p.mu.Unlock()

	if !exists {
		return errors.New("breakpoint not found or already released")
	}

	if drop {
		close(bp.dropCh)
	} else {
		bp.resumeCh <- modifiedPayload
	}

	return nil
}

// Stop shuts down the proxy server.
func (p *StreamProxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.running = false
	if p.cancel != nil {
		p.cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if p.httpServer != nil {
		_ = p.httpServer.Shutdown(ctx)
	}
	if p.listener != nil {
		_ = p.listener.Close()
	}

	return nil
}

// Status returns proxy state information.
func (p *StreamProxy) Status() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pendingList := make([]map[string]interface{}, 0, len(p.breakpoints))
	for id, bp := range p.breakpoints {
		pendingList = append(pendingList, map[string]interface{}{
			"id":        id,
			"direction": bp.Direction,
			"opcode":    bp.OpCode,
			"payload":   bp.Payload,
			"timestamp": bp.Timestamp,
		})
	}

	return map[string]interface{}{
		"running":             p.running,
		"local_port":          p.config.LocalPort,
		"target_url":          p.config.TargetURL,
		"breakpoints_enabled": p.config.EnableBreakpoint,
		"pending_breakpoints": pendingList,
	}
}
