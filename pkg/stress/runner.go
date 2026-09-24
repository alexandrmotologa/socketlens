package stress

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// BenchConfig configures a benchmark load run.
type BenchConfig struct {
	URL          string            `json:"url"`
	Clients      int               `json:"clients"`
	Duration     time.Duration     `json:"duration"`
	RampUp       time.Duration     `json:"ramp_up"`
	SendInterval time.Duration     `json:"send_interval"`
	Payload      string            `json:"payload"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// Runner coordinates concurrent load workers.
type Runner struct {
	config    BenchConfig
	collector *MetricsCollector
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewRunner creates a stress test runner.
func NewRunner(cfg BenchConfig) *Runner {
	if cfg.Clients <= 0 {
		cfg.Clients = 50
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}
	if cfg.Payload == "" {
		cfg.Payload = "{\"type\":\"ping\"}"
	}

	return &Runner{
		config:    cfg,
		collector: NewMetricsCollector(),
	}
}

// Run executes the load test and blocks until completion.
func (r *Runner) Run(parentCtx context.Context) (*BenchmarkReport, error) {
	r.ctx, r.cancel = context.WithTimeout(parentCtx, r.config.Duration)
	defer r.cancel()

	var wg sync.WaitGroup

	staggerInterval := time.Duration(0)
	if r.config.RampUp > 0 && r.config.Clients > 1 {
		staggerInterval = r.config.RampUp / time.Duration(r.config.Clients)
	}

	for i := 0; i < r.config.Clients; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			if staggerInterval > 0 {
				delay := time.Duration(workerID) * staggerInterval
				select {
				case <-r.ctx.Done():
					return
				case <-time.After(delay):
				}
			}

			r.runWorker(workerID)
		}(i)
	}

	// Wait for all workers to shut down on ctx timeout
	wg.Wait()

	report := r.collector.Finalize(r.config.URL, r.config.Clients, r.config.Duration)
	return report, nil
}

func (r *Runner) runWorker(id int) {
	dialStart := time.Now()
	dialCtx, dialCancel := context.WithTimeout(r.ctx, 5*time.Second)

	headers := http.Header{}
	for k, v := range r.config.Headers {
		headers.Set(k, v)
	}

	conn, _, err := websocket.Dial(dialCtx, r.config.URL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	dialCancel()

	if err != nil {
		r.collector.RecordFailure()
		return
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "bench complete")
		r.collector.ClientDisconnected()
	}()

	handshakeDuration := time.Since(dialStart)
	r.collector.RecordHandshake(handshakeDuration)

	// Sender loop if interval configured
	if r.config.SendInterval > 0 {
		go func() {
			ticker := time.NewTicker(r.config.SendInterval)
			defer ticker.Stop()

			payloadBytes := []byte(r.config.Payload)
			for {
				select {
				case <-r.ctx.Done():
					return
				case <-ticker.C:
					writeCtx, writeCancel := context.WithTimeout(r.ctx, 2*time.Second)
					err := conn.Write(writeCtx, websocket.MessageText, payloadBytes)
					writeCancel()
					if err != nil {
						return
					}
					r.collector.AddFrameSent(len(payloadBytes))
				}
			}
		}()
	}

	// Reader loop
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		_, payload, err := conn.Read(r.ctx)
		if err != nil {
			return
		}
		r.collector.AddFrameReceived(len(payload))
	}
}

// Stop terminates the running benchmark early.
func (r *Runner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}
