package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

func TestAPIServerRoutes(t *testing.T) {
	mgr := client.NewManager()
	hub := NewTimelineHub()
	api := NewAPIServer(mgr, hub)

	router := api.Routes()

	// 1. Test Static Fallback Studio
	staticHandler := FileServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	staticHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for studio, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "SocketLens") {
		t.Errorf("expected SocketLens in HTML body")
	}

	// 2. Test Codec Decode API
	decodePayload, _ := json.Marshal(map[string]string{
		"payload": "{\"test\": 123}",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/codec/decode", bytes.NewReader(decodePayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for codec decode, got %d", rec.Code)
	}

	var decodeResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &decodeResp); err != nil {
		t.Fatalf("json parse error: %v", err)
	}

	if decodeResp["format"] != "json" {
		t.Errorf("expected format json, got %v", decodeResp["format"])
	}

	// 3. Test Mock Server Start/Status/Stop
	mockCfg, _ := json.Marshal(map[string]interface{}{
		"port":  0,
		"route": "/ws/feed",
		"mode":  "echo",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mock/start", bytes.NewReader(mockCfg))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for mock start, got %d", rec.Code)
	}

	// Verify status
	req = httptest.NewRequest(http.MethodGet, "/api/v1/mock/status", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for mock status, got %d", rec.Code)
	}

	// Stop mock
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mock/stop", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for mock stop, got %d", rec.Code)
	}
}
