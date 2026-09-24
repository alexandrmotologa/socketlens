package stress

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// BenchmarkReport summarizes stress test results.
type BenchmarkReport struct {
	TargetURL            string        `json:"target_url"`
	TargetClients        int           `json:"target_clients"`
	ActiveClients        int           `json:"active_clients"`
	PeakClients          int           `json:"peak_clients"`
	SuccessfulHandshakes int           `json:"successful_handshakes"`
	FailedHandshake      int           `json:"failed_handshakes"`
	Duration             time.Duration `json:"duration"`

	P50LatencyMs float64 `json:"p50_latency_ms"`
	P90LatencyMs float64 `json:"p90_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	MinLatencyMs float64 `json:"min_latency_ms"`
	MaxLatencyMs float64 `json:"max_latency_ms"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`

	FramesSent     uint64  `json:"frames_sent"`
	FramesReceived uint64  `json:"frames_received"`
	BytesSent      uint64  `json:"bytes_sent"`
	BytesReceived  uint64  `json:"bytes_received"`
	MessagesPerSec float64 `json:"messages_per_sec"`
	BytesPerSec    float64 `json:"bytes_per_sec"`
}

// FormatSummary generates a readable report table.
func (r *BenchmarkReport) FormatSummary() string {
	return fmt.Sprintf(`=== SocketLens Benchmark Results ===
Target:             %s
Target Concurrency: %d clients
Active Connections: %d clients
Failed Handshakes:  %d
Test Duration:      %s

Handshake Latency:
  p50:              %.2f ms
  p90:              %.2f ms
  p95:              %.2f ms
  p99:              %.2f ms
  Min / Max:        %.2f ms / %.2f ms
  Average:          %.2f ms

Throughput:
  Frames Sent:      %d
  Frames Received:  %d
  Message Rate:     %.1f msgs/sec
  Bandwidth:        %.2f KB/sec
`,
		r.TargetURL,
		r.TargetClients,
		r.ActiveClients,
		r.FailedHandshake,
		r.Duration,
		r.P50LatencyMs,
		r.P90LatencyMs,
		r.P95LatencyMs,
		r.P99LatencyMs,
		r.MinLatencyMs,
		r.MaxLatencyMs,
		r.AvgLatencyMs,
		r.FramesSent,
		r.FramesReceived,
		r.MessagesPerSec,
		r.BytesPerSec/1024.0,
	)
}

// MetricsCollector compiles real-time telemetry.
type MetricsCollector struct {
	mu                   sync.Mutex
	latencies            []time.Duration
	activeClients        int
	peakClients          int
	successfulHandshakes int
	failedHandshake      int
	framesSent           uint64
	framesReceived       uint64
	bytesSent            uint64
	bytesReceived        uint64
	startTime            time.Time
}

// NewMetricsCollector creates a thread-safe collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies: make([]time.Duration, 0, 1024),
		startTime: time.Now(),
	}
}

// RecordHandshake logs a successful connection latency.
func (m *MetricsCollector) RecordHandshake(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencies = append(m.latencies, d)
	m.activeClients++
	m.successfulHandshakes++
	if m.activeClients > m.peakClients {
		m.peakClients = m.activeClients
	}
}

// RecordFailure logs a failed connection attempt.
func (m *MetricsCollector) RecordFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedHandshake++
}

// ClientDisconnected decrements active connection count.
func (m *MetricsCollector) ClientDisconnected() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeClients > 0 {
		m.activeClients--
	}
}

// AddFrameSent logs outbound message volume.
func (m *MetricsCollector) AddFrameSent(bytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.framesSent++
	m.bytesSent += uint64(bytes)
}

// AddFrameReceived logs inbound message volume.
func (m *MetricsCollector) AddFrameReceived(bytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.framesReceived++
	m.bytesReceived += uint64(bytes)
}

// Finalize compiles all metrics into an immutable report.
func (m *MetricsCollector) Finalize(targetURL string, targetClients int, duration time.Duration) *BenchmarkReport {
	m.mu.Lock()
	defer m.mu.Unlock()

	report := &BenchmarkReport{
		TargetURL:            targetURL,
		TargetClients:        targetClients,
		ActiveClients:        m.activeClients,
		PeakClients:          m.peakClients,
		SuccessfulHandshakes: m.successfulHandshakes,
		FailedHandshake:      m.failedHandshake,
		Duration:             duration,
		FramesSent:           m.framesSent,
		FramesReceived:       m.framesReceived,
		BytesSent:            m.bytesSent,
		BytesReceived:        m.bytesReceived,
	}

	durSec := duration.Seconds()
	if durSec > 0 {
		totalFrames := float64(m.framesSent + m.framesReceived)
		report.MessagesPerSec = totalFrames / durSec
		totalBytes := float64(m.bytesSent + m.bytesReceived)
		report.BytesPerSec = totalBytes / durSec
	}

	if len(m.latencies) == 0 {
		return report
	}

	// Sort latencies for percentiles
	sort.Slice(m.latencies, func(i, j int) bool {
		return m.latencies[i] < m.latencies[j]
	})

	n := len(m.latencies)
	report.MinLatencyMs = float64(m.latencies[0].Microseconds()) / 1000.0
	report.MaxLatencyMs = float64(m.latencies[n-1].Microseconds()) / 1000.0

	var sum time.Duration
	for _, l := range m.latencies {
		sum += l
	}
	report.AvgLatencyMs = (float64(sum.Microseconds()) / 1000.0) / float64(n)

	report.P50LatencyMs = float64(m.latencies[int(float64(n)*0.50)].Microseconds()) / 1000.0
	report.P90LatencyMs = float64(m.latencies[int(float64(n)*0.90)].Microseconds()) / 1000.0
	report.P95LatencyMs = float64(m.latencies[int(float64(n)*0.95)].Microseconds()) / 1000.0
	p99Idx := int(float64(n) * 0.99)
	if p99Idx >= n {
		p99Idx = n - 1
	}
	report.P99LatencyMs = float64(m.latencies[p99Idx].Microseconds()) / 1000.0

	return report
}

// ToJSON converts report to formatted JSON bytes.
func (r *BenchmarkReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
