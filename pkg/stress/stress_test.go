package stress

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/mock"
)

func TestStressRunnerAgainstMockServer(t *testing.T) {
	srv := mock.NewServer(mock.ServerConfig{
		Port:  0,
		Route: "/ws/bench",
		Mode:  "echo",
	})

	err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer srv.Stop()

	time.Sleep(50 * time.Millisecond)

	runner := NewRunner(BenchConfig{
		URL:          srv.WSURL(),
		Clients:      10,
		Duration:     2 * time.Second,
		RampUp:       200 * time.Millisecond,
		SendInterval: 100 * time.Millisecond,
		Payload:      "{\"type\":\"bench_ping\"}",
	})

	report, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("runner execution failed: %v", err)
	}

	if report.SuccessfulHandshakes < 5 {
		t.Errorf("expected successful handshakes >= 5, got %d", report.SuccessfulHandshakes)
	}

	if report.P50LatencyMs < 0 || report.P99LatencyMs < 0 {
		t.Errorf("invalid latency percentiles: p50=%.2f p99=%.2f", report.P50LatencyMs, report.P99LatencyMs)
	}

	summary := report.FormatSummary()
	if !strings.Contains(summary, "SocketLens Benchmark Results") {
		t.Errorf("summary missing header: %s", summary)
	}

	jsonBytes, err := report.ToJSON()
	if err != nil || len(jsonBytes) == 0 {
		t.Fatalf("failed to format report as JSON: %v", err)
	}
}
