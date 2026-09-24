package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/alexandrmotologa/socketlens/pkg/codec"
	"github.com/alexandrmotologa/socketlens/pkg/llm"
	"github.com/alexandrmotologa/socketlens/pkg/mock"
	"github.com/alexandrmotologa/socketlens/pkg/proxy"
	"github.com/alexandrmotologa/socketlens/pkg/rules"
	"github.com/alexandrmotologa/socketlens/pkg/session"
	"github.com/alexandrmotologa/socketlens/pkg/stress"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

// APIServer manages all API routes and background operations.
type APIServer struct {
	manager    *client.Manager
	hub        *TimelineHub
	mockSrv    *mock.Server
	recorder   *session.Recorder
	benchRun   *stress.Runner
	lastBench  *stress.BenchmarkReport
	proxySrv   *proxy.StreamProxy
	mu         sync.Mutex
	benchMutex sync.Mutex
}

// NewAPIServer creates an API instance.
func NewAPIServer(mgr *client.Manager, hub *TimelineHub) *APIServer {
	return &APIServer{
		manager: mgr,
		hub:     hub,
	}
}

// Routes sets up the chi router with CORS and JSON endpoints.
func (s *APIServer) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "Content-Disposition"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		// Connection endpoints
		r.Post("/connections", s.handleCreateConnection)
		r.Get("/connections", s.handleListConnections)
		r.Delete("/connections/{id}", s.handleCloseConnection)
		r.Post("/connections/{id}/send", s.handleSendFrame)

		// Mock server endpoints
		r.Post("/mock/start", s.handleStartMock)
		r.Post("/mock/stop", s.handleStopMock)
		r.Get("/mock/status", s.handleMockStatus)

		// Benchmarking endpoints
		r.Post("/bench/start", s.handleStartBench)
		r.Get("/bench/status", s.handleBenchStatus)

		// Recording endpoints
		r.Post("/record/start", s.handleStartRecord)
		r.Post("/record/stop", s.handleStopRecord)

		// Codec playground & diff
		r.Post("/codec/decode", s.handleCodecDecode)
		r.Post("/codec/diff", s.handleCodecDiff)

		// Proxy endpoints
		r.Post("/proxy/start", s.handleStartProxy)
		r.Post("/proxy/stop", s.handleStopProxy)
		r.Get("/proxy/status", s.handleProxyStatus)
		r.Post("/proxy/breakpoints/{id}/resume", s.handleResumeBreakpoint)

		// Schema validation endpoints
		r.Post("/schema/set", s.handleSetSchema)
		r.Post("/schema/clear", s.handleClearSchema)
		r.Post("/schema/validate", s.handleValidateSchema)

		// Automation rules endpoints
		r.Get("/rules", s.handleListRules)
		r.Post("/rules", s.handleAddRule)
		r.Delete("/rules/{id}", s.handleDeleteRule)

		// LLM Stream Inspector endpoints
		r.Get("/llm/report", s.handleLLMReport)
		r.Post("/llm/reset", s.handleLLMReset)

		// Export endpoints
		r.Get("/export/har", s.handleExportHAR)
		r.Get("/export/commands", s.handleExportCommands)
	})

	// Internal control WebSocket
	r.HandleFunc("/ws/control", s.hub.HandleWS)

	return r
}


func (s *APIServer) handleCreateConnection(w http.ResponseWriter, r *http.Request) {
	var req client.ConnectionConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		req.ID = "conn_" + uuid.NewString()[:8]
	}

	onFrame := func(f *client.Frame) {
		s.hub.IngestFrame(f)
		s.mu.Lock()
		rec := s.recorder
		s.mu.Unlock()
		if rec != nil {
			rec.Record(f)
		}
	}

	onState := func(state client.ConnectionState, err error) {
		s.hub.BroadcastState(req.ID, state, err)
	}

	c, err := client.CreateClient(req, onFrame, onState)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.manager.Register(req.ID, c, req)

	go func() {
		_ = c.Connect(context.Background())
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "connecting",
		"connection_id": req.ID,
		"config":        req,
	})
}

func (s *APIServer) handleListConnections(w http.ResponseWriter, r *http.Request) {
	configs := s.manager.List()
	res := make([]map[string]interface{}, 0, len(configs))

	for id, cfg := range configs {
		c, _, ok := s.manager.Get(id)
		var stats client.ConnectionStats
		if ok {
			stats = c.Stats()
		}

		res = append(res, map[string]interface{}{
			"id":     id,
			"config": cfg,
			"stats":  stats,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *APIServer) handleCloseConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := s.manager.Close(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.hub.BroadcastState(id, client.StateDisconnected, nil)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "disconnected"})
}

func (s *APIServer) handleSendFrame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, _, ok := s.manager.Get(id)
	if !ok {
		http.Error(w, "connection not found", http.StatusNotFound)
		return
	}

	var req struct {
		Payload string         `json:"payload"`
		OpCode  client.OpCode  `json:"opcode"`
		Format  client.PayloadFormat `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.OpCode == "" {
		req.OpCode = client.OpCodeText
	}

	payloadBytes := []byte(req.Payload)

	// Format transformation if requested
	if req.Format == client.FormatMsgPack {
		var obj interface{}
		if err := json.Unmarshal(payloadBytes, &obj); err == nil {
			if encoded, encErr := codec.EncodeMsgPack(obj); encErr == nil {
				payloadBytes = encoded
				req.OpCode = client.OpCodeBinary
			}
		}
	} else if req.Format == client.FormatCBOR {
		var obj interface{}
		if err := json.Unmarshal(payloadBytes, &obj); err == nil {
			if encoded, encErr := codec.EncodeCBOR(obj); encErr == nil {
				payloadBytes = encoded
				req.OpCode = client.OpCodeBinary
			}
		}
	}

	err := c.Send(payloadBytes, req.OpCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "sent",
		"length": len(payloadBytes),
	})
}

func (s *APIServer) handleStartMock(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mockSrv != nil {
		_ = s.mockSrv.Stop()
	}

	var cfg mock.ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	s.mockSrv = mock.NewServer(cfg)
	if err := s.mockSrv.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "running",
		"port":    cfg.Port,
		"ws_url":  s.mockSrv.WSURL(),
		"sse_url": s.mockSrv.SSEURL(),
	})
}

func (s *APIServer) handleStopMock(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mockSrv != nil {
		_ = s.mockSrv.Stop()
		s.mockSrv = nil
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (s *APIServer) handleMockStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mockSrv == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"running": false})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"running": true,
		"addr":    s.mockSrv.Addr(),
		"ws_url":  s.mockSrv.WSURL(),
		"sse_url": s.mockSrv.SSEURL(),
	})
}

func (s *APIServer) handleStartBench(w http.ResponseWriter, r *http.Request) {
	var cfg stress.BenchConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if cfg.Clients <= 0 {
		cfg.Clients = 100
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}

	s.benchMutex.Lock()
	s.benchRun = stress.NewRunner(cfg)
	runner := s.benchRun
	s.benchMutex.Unlock()

	go func() {
		report, _ := runner.Run(context.Background())
		s.benchMutex.Lock()
		s.lastBench = report
		s.benchMutex.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "started",
		"target":  cfg.URL,
		"clients": cfg.Clients,
	})
}

func (s *APIServer) handleBenchStatus(w http.ResponseWriter, r *http.Request) {
	s.benchMutex.Lock()
	defer s.benchMutex.Unlock()

	if s.lastBench != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "complete",
			"report": s.lastBench,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "idle",
	})
}

func (s *APIServer) handleStartRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.FilePath == "" {
		req.FilePath = "recordings/session_" + time.Now().Format("20060102_150405") + ".jsonl"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recorder != nil {
		_ = s.recorder.Close()
	}

	rec, err := session.NewRecorder(req.FilePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.recorder = rec

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "recording",
		"file_path": req.FilePath,
	})
}

func (s *APIServer) handleStopRecord(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recorder != nil {
		_ = s.recorder.Close()
		s.recorder = nil
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (s *APIServer) handleCodecDecode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Payload string `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmtName, dec, err := codec.DecodeFrame([]byte(req.Payload))
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"format":  fmtName,
		"decoded": dec,
		"error":   errStr,
	})
}

func (s *APIServer) handleCodecDiff(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPayload string `json:"old_payload"`
		NewPayload string `json:"new_payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := codec.CompareJSON([]byte(req.OldPayload), []byte(req.NewPayload))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *APIServer) handleStartProxy(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.proxySrv != nil {
		_ = s.proxySrv.Stop()
	}

	var cfg proxy.ProxyConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if cfg.LocalPort == 0 {
		cfg.LocalPort = 8081
	}

	onFrame := func(f *client.Frame) {
		s.hub.IngestFrame(f)
		s.mu.Lock()
		rec := s.recorder
		s.mu.Unlock()
		if rec != nil {
			rec.Record(f)
		}
	}

	p := proxy.NewStreamProxy(cfg, onFrame)
	if err := p.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.proxySrv = p

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "running",
		"local_port": cfg.LocalPort,
		"target_url": cfg.TargetURL,
	})
}

func (s *APIServer) handleStopProxy(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.proxySrv != nil {
		_ = s.proxySrv.Stop()
		s.proxySrv = nil
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (s *APIServer) handleProxyStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	p := s.proxySrv
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if p == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"running": false})
		return
	}

	_ = json.NewEncoder(w).Encode(p.Status())
}

func (s *APIServer) handleResumeBreakpoint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.mu.Lock()
	p := s.proxySrv
	s.mu.Unlock()

	if p == nil {
		http.Error(w, "proxy not running", http.StatusBadRequest)
		return
	}

	var req struct {
		Payload string `json:"payload"`
		Drop    bool   `json:"drop"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := p.ResumeBreakpoint(id, []byte(req.Payload), req.Drop)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
}

func (s *APIServer) handleSetSchema(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.hub.Validator().SetSchema(body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "applied", "active": true})
}

func (s *APIServer) handleClearSchema(w http.ResponseWriter, r *http.Request) {
	s.hub.Validator().ClearSchema()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "cleared", "active": false})
}

func (s *APIServer) handleValidateSchema(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Payload string `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	violations, err := s.hub.Validator().Validate([]byte(req.Payload))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":      len(violations) == 0,
		"violations": violations,
	})
}

func (s *APIServer) handleListRules(w http.ResponseWriter, r *http.Request) {
	rulesList := s.hub.RulesEngine().ListRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rulesList)
}

func (s *APIServer) handleAddRule(w http.ResponseWriter, r *http.Request) {
	var rule rules.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := s.hub.RulesEngine().AddRule(rule)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "added"})
}

func (s *APIServer) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.hub.RulesEngine().DeleteRule(id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *APIServer) handleLLMReport(w http.ResponseWriter, r *http.Request) {
	report := s.hub.LLMAnalyzer().Report()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

func (s *APIServer) handleLLMReset(w http.ResponseWriter, r *http.Request) {
	s.hub.llmAnalyzer = llm.NewAnalyzer()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
}

func (s *APIServer) handleExportHAR(w http.ResponseWriter, r *http.Request) {
	connID := r.URL.Query().Get("connection_id")
	frames := s.hub.RecentFrames()

	var cfg client.ConnectionConfig
	if connID != "" {
		_, cCfg, ok := s.manager.Get(connID)
		if ok {
			cfg = cCfg
		}
	}
	if cfg.URL == "" {
		cfg.URL = "ws://socketlens.local/timeline"
		cfg.Protocol = client.ProtocolWS
	}

	harBytes, err := session.ExportToHAR(cfg, frames)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"socketlens_export.har\"")
	_, _ = w.Write(harBytes)
}

func (s *APIServer) handleExportCommands(w http.ResponseWriter, r *http.Request) {
	connID := r.URL.Query().Get("connection_id")
	var cfg client.ConnectionConfig
	if connID != "" {
		_, cCfg, ok := s.manager.Get(connID)
		if ok {
			cfg = cCfg
		}
	}
	if cfg.URL == "" {
		cfg.URL = "ws://localhost:8080/ws"
		cfg.Protocol = client.ProtocolWS
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"wscat": session.GenerateWscatCommand(cfg),
		"curl":  session.GenerateCurlCommand(cfg),
	})
}

