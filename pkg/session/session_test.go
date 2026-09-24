package session

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
)

func TestRecorderAndReplayer(t *testing.T) {
	tmpDir := t.TempDir()
	recordPath := filepath.Join(tmpDir, "session_test.jsonl")

	rec, err := NewRecorder(recordPath)
	if err != nil {
		t.Fatalf("failed to create recorder: %v", err)
	}

	frames := []*client.Frame{
		{
			Sequence:  1,
			Direction: client.DirectionOutbound,
			Protocol:  client.ProtocolWS,
			OpCode:    client.OpCodeText,
			Payload:   []byte("{\"action\":\"login\"}"),
			Format:    client.FormatJSON,
		},
		{
			Sequence:  2,
			Direction: client.DirectionInbound,
			Protocol:  client.ProtocolWS,
			OpCode:    client.OpCodeText,
			Payload:   []byte("{\"status\":\"ok\"}"),
			Format:    client.FormatJSON,
		},
	}

	for _, f := range frames {
		rec.Record(f)
		time.Sleep(10 * time.Millisecond)
	}

	err = rec.Close()
	if err != nil {
		t.Fatalf("failed to close recorder: %v", err)
	}

	// Verify file exists and has content
	info, err := os.Stat(recordPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected record file to exist and not be empty: %v", err)
	}

	// Replay frames
	var replayed []*RecordedFrame
	var mu sync.Mutex

	replayer := NewReplayer(ReplayConfig{
		FilePath: recordPath,
		Speed:    10.0, // 10x faster
		Loop:     false,
		OnFrame: func(rf *RecordedFrame) {
			mu.Lock()
			replayed = append(replayed, rf)
			mu.Unlock()
		},
	})

	err = replayer.Play(context.Background())
	if err != nil {
		t.Fatalf("replay error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(replayed) != 2 {
		t.Fatalf("expected 2 replayed frames, got %d", len(replayed))
	}

	if replayed[0].Sequence != 1 || replayed[0].Payload != "{\"action\":\"login\"}" {
		t.Errorf("frame 1 mismatch: %+v", replayed[0])
	}
	if replayed[1].Sequence != 2 || replayed[1].Payload != "{\"status\":\"ok\"}" {
		t.Errorf("frame 2 mismatch: %+v", replayed[1])
	}
}
