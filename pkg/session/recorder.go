package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

// RecordedFrame models a frame in a JSONL archive.
type RecordedFrame struct {
	Sequence  uint64 `json:"seq"`
	OffsetMs  int64  `json:"offset_ms"`
	Direction string `json:"direction"`
	Protocol  string `json:"protocol"`
	OpCode    string `json:"opcode"`
	Payload   string `json:"payload"`
	Format    string `json:"format"`
	EventName string `json:"event_name,omitempty"`
}

// Recorder captures live stream traffic to disk.
type Recorder struct {
	file      *os.File
	writer    *bufio.Writer
	queue     chan *RecordedFrame
	startTime time.Time
	mu        sync.Mutex
	closed    bool
	done      chan struct{}
}

// NewRecorder creates an active file stream recorder.
func NewRecorder(filePath string) (*Recorder, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("creating record file: %w", err)
	}

	r := &Recorder{
		file:      file,
		writer:    bufio.NewWriterSize(file, 64*1024),
		queue:     make(chan *RecordedFrame, 1024),
		startTime: time.Now(),
		done:      make(chan struct{}),
	}

	go r.flushLoop()
	return r, nil
}

func (r *Recorder) flushLoop() {
	defer close(r.done)

	for frame := range r.queue {
		bytes, err := json.Marshal(frame)
		if err == nil {
			_, _ = r.writer.Write(bytes)
			_ = r.writer.WriteByte('\n')
		}
	}
	_ = r.writer.Flush()
}

// Record queues a frame to be written.
func (r *Recorder) Record(f *client.Frame) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	offset := time.Since(r.startTime).Milliseconds()

	rf := &RecordedFrame{
		Sequence:  f.Sequence,
		OffsetMs:  offset,
		Direction: string(f.Direction),
		Protocol:  string(f.Protocol),
		OpCode:    string(f.OpCode),
		Payload:   string(f.Payload),
		Format:    string(f.Format),
		EventName: f.EventName,
	}

	select {
	case r.queue <- rf:
	default:
		// Drop frame if buffer saturated rather than blocking network pipeline
	}
}

// Close flushes all queued records and closes the underlying file.
func (r *Recorder) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()

	close(r.queue)
	<-r.done

	return r.file.Close()
}
