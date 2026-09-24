package mock

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServerConfig configures the embedded mock server.
type ServerConfig struct {
	Port  int         `json:"port"`
	Route string      `json:"route"`
	Mode  string      `json:"mode"` // echo, broadcast, llm-tokens, ticks
	Rate  int         `json:"rate"` // events/second
	Chaos ChaosConfig `json:"chaos"`
}

// Server represents an active embedded mock HTTP/WS/SSE server.
type Server struct {
	config   ServerConfig
	listener net.Listener
	httpSrv  *http.Server
	hub      *Hub
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	running  bool
}

// NewServer initializes a mock server.
func NewServer(cfg ServerConfig) *Server {
	if cfg.Route == "" {
		cfg.Route = "/ws"
	}
	if !strings.HasPrefix(cfg.Route, "/") {
		cfg.Route = "/" + cfg.Route
	}

	return &Server{
		config: cfg,
		hub:    newHub(),
	}
}

// Start launches the mock server listener.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("mock server already running")
	}

	addr := fmt.Sprintf("127.0.0.1:%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("binding mock listener: %w", err)
	}
	s.listener = listener

	s.ctx, s.cancel = context.WithCancel(context.Background())

	mux := http.NewServeMux()

	// WebSocket handler
	mux.HandleFunc(s.config.Route, func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketConnection(s.ctx, w, r, s.config, s.hub)
	})

	// SSE handler
	sseRoute := strings.TrimSuffix(s.config.Route, "/") + "/sse"
	mux.HandleFunc(sseRoute, s.handleSSE)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"socketlens-mock"}`))
	})

	s.httpSrv = &http.Server{
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 0,
	}

	s.running = true

	go func() {
		_ = s.httpSrv.Serve(s.listener)
	}()

	return nil
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	rate := s.config.Rate
	if rate <= 0 {
		rate = 10
	}
	interval := time.Second / time.Duration(rate)

	reqCtx := r.Context()

	switch s.config.Mode {
	case "ticks":
		var seq uint64
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-reqCtx.Done():
				return
			case <-time.After(interval):
			}

			if s.config.Chaos.ShouldDrop() {
				continue
			}
			s.config.Chaos.ApplyLatency()

			seq++
			tick := GenerateFinancialTick(seq)
			ev := FormatSSEEvent("tick", fmt.Sprintf("%d", seq), string(tick))
			_, _ = fmt.Fprint(w, ev)
			flusher.Flush()
		}

	default: // llm-tokens or echo stream
		for i, token := range sampleLLMPromptTokens {
			select {
			case <-s.ctx.Done():
				return
			case <-reqCtx.Done():
				return
			case <-time.After(interval):
			}

			if s.config.Chaos.ShouldDrop() {
				continue
			}
			s.config.Chaos.ApplyLatency()

			chunk := GenerateLLMChunk(token, i, false)
			ev := FormatSSEEvent("message", fmt.Sprintf("%d", i), chunk)
			_, _ = fmt.Fprint(w, ev)
			flusher.Flush()
		}

		finalChunk := GenerateLLMChunk("", len(sampleLLMPromptTokens), true)
		ev := FormatSSEEvent("message", "done", finalChunk)
		_, _ = fmt.Fprint(w, ev)
		flusher.Flush()
	}
}

// Stop gracefully shuts down the mock server.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	if s.cancel != nil {
		s.cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if s.httpSrv != nil {
		_ = s.httpSrv.Shutdown(ctx)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}

	return nil
}

// Addr returns the network address the server is listening on.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// WSURL returns the full WebSocket URL.
func (s *Server) WSURL() string {
	return fmt.Sprintf("ws://%s%s", s.Addr(), s.config.Route)
}

// SSEURL returns the full SSE URL.
func (s *Server) SSEURL() string {
	return fmt.Sprintf("http://%s%s/sse", s.Addr(), s.config.Route)
}
