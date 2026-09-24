package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

// ReplayConfig configures replay playback.
type ReplayConfig struct {
	FilePath string
	Target   client.StreamClient
	Speed    float64
	Loop     bool
	OnFrame  func(frame *RecordedFrame)
}

// Replayer executes time-accurate playback of recorded streaming sessions.
type Replayer struct {
	config ReplayConfig
}

// NewReplayer creates a session replayer.
func NewReplayer(cfg ReplayConfig) *Replayer {
	if cfg.Speed <= 0 {
		cfg.Speed = 1.0
	}
	return &Replayer{config: cfg}
}

// Play reads and dispatches the recorded frames.
func (r *Replayer) Play(ctx context.Context) error {
	for {
		err := r.playOnce(ctx)
		if err != nil {
			return err
		}

		if !r.config.Loop {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (r *Replayer) playOnce(ctx context.Context) error {
	file, err := os.Open(r.config.FilePath)
	if err != nil {
		return fmt.Errorf("opening replay file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 512*1024)

	var lastOffsetMs int64 = 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var rf RecordedFrame
		if err := json.Unmarshal(line, &rf); err != nil {
			continue
		}

		// Calculate sleep interval scaled by speed factor
		deltaMs := rf.OffsetMs - lastOffsetMs
		if deltaMs > 0 {
			scaledWait := time.Duration(float64(deltaMs)/r.config.Speed) * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(scaledWait):
			}
		}
		lastOffsetMs = rf.OffsetMs

		// Transmit to target client if available
		if r.config.Target != nil && rf.Direction == string(client.DirectionOutbound) {
			opcode := client.OpCodeText
			if rf.OpCode == string(client.OpCodeBinary) {
				opcode = client.OpCodeBinary
			}
			_ = r.config.Target.Send([]byte(rf.Payload), opcode)
		}

		if r.config.OnFrame != nil {
			r.config.OnFrame(&rf)
		}
	}

	return scanner.Err()
}
