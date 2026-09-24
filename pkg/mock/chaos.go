package mock

import (
	"math/rand"
	"time"
)

// ChaosConfig sets fault injection parameters.
type ChaosConfig struct {
	Latency  time.Duration `json:"latency"`
	DropRate int           `json:"drop_rate"` // 0 to 100%
	Reset    time.Duration `json:"reset"`     // Force close after duration
}

// ShouldDrop returns true if a frame should be dropped based on DropRate.
func (c *ChaosConfig) ShouldDrop() bool {
	if c.DropRate <= 0 {
		return false
	}
	if c.DropRate >= 100 {
		return true
	}
	return rand.Intn(100) < c.DropRate
}

// ApplyLatency pauses execution if Latency is configured.
func (c *ChaosConfig) ApplyLatency() {
	if c.Latency <= 0 {
		return
	}
	// Introduce jitter +/- 20%
	jitter := float64(c.Latency) * (0.8 + 0.4*rand.Float64())
	time.Sleep(time.Duration(jitter))
}
