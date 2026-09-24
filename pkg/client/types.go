package client

import (
	"encoding/json"
	"time"
)

// Direction represents frame flow relative to the client.
type Direction string

const (
	DirectionInbound  Direction = "inbound"  // Server to client
	DirectionOutbound Direction = "outbound" // Client to server
)

// Protocol identifies the streaming protocol.
type Protocol string

const (
	ProtocolWS       Protocol = "ws"
	ProtocolWSS      Protocol = "wss"
	ProtocolSSE      Protocol = "sse"
	ProtocolSocketIO Protocol = "socketio"
)

// OpCode describes the frame type.
type OpCode string

const (
	OpCodeText   OpCode = "text"
	OpCodeBinary OpCode = "binary"
	OpCodePing   OpCode = "ping"
	OpCodePong   OpCode = "pong"
	OpCodeClose  OpCode = "close"
	OpCodeEvent  OpCode = "event" // For SSE / Socket.io named events
)

// PayloadFormat identifies the decoded or raw format of a payload.
type PayloadFormat string

const (
	FormatJSON     PayloadFormat = "json"
	FormatText     PayloadFormat = "text"
	FormatMsgPack  PayloadFormat = "msgpack"
	FormatCBOR     PayloadFormat = "cbor"
	FormatProtobuf PayloadFormat = "protobuf"
	FormatGzip     PayloadFormat = "gzip"
	FormatRaw      PayloadFormat = "raw"
)

// ConnectionState represents the current status of a stream session.
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateConnected    ConnectionState = "connected"
	StateReconnecting ConnectionState = "reconnecting"
	StateError        ConnectionState = "error"
)

// Frame represents a single transmitted or received streaming unit.
type Frame struct {
	ID        string            `json:"id"`
	ConnID    string            `json:"connection_id"`
	Sequence  uint64            `json:"sequence"`
	Timestamp time.Time         `json:"timestamp"`
	Direction Direction         `json:"direction"`
	Protocol  Protocol          `json:"protocol"`
	OpCode    OpCode            `json:"opcode"`
	Payload   []byte            `json:"payload"`
	Decoded   string            `json:"decoded,omitempty"`
	Format    PayloadFormat     `json:"format"`
	Length    int               `json:"length"`
	Latency   time.Duration     `json:"latency,omitempty"`
	EventName    string            `json:"event_name,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	SchemaErrors []string          `json:"schema_errors,omitempty"`
}

// Summary returns a brief human-readable string for logging and TUI.
func (f *Frame) Summary() string {
	if f.Decoded != "" {
		if len(f.Decoded) > 80 {
			return f.Decoded[:77] + "..."
		}
		return f.Decoded
	}
	if len(f.Payload) > 80 {
		return string(f.Payload[:77]) + "..."
	}
	return string(f.Payload)
}

// ConnectionConfig configures how a client connects to a streaming target.
type ConnectionConfig struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	URL               string            `json:"url"`
	Protocol          Protocol          `json:"protocol"`
	Headers           map[string]string `json:"headers,omitempty"`
	Subprotocols      []string          `json:"subprotocols,omitempty"`
	TLSInsecure       bool              `json:"tls_insecure"`
	AutoReconnect     bool              `json:"auto_reconnect"`
	ReconnectInterval time.Duration     `json:"reconnect_interval"`
	HeartbeatInterval time.Duration     `json:"heartbeat_interval"`
	HandshakeTimeout  time.Duration     `json:"handshake_timeout"`
	RecordFile        string            `json:"record_file,omitempty"`
}

// ConnectionStats tracks real-time session counters and metrics.
type ConnectionStats struct {
	State              ConnectionState `json:"state"`
	ConnectedAt        time.Time       `json:"connected_at,omitempty"`
	UptimeSeconds      float64         `json:"uptime_seconds"`
	FramesReceived     uint64          `json:"frames_received"`
	FramesSent         uint64          `json:"frames_sent"`
	BytesReceived      uint64          `json:"bytes_received"`
	BytesSent          uint64          `json:"bytes_sent"`
	LastHeartbeatAt    time.Time       `json:"last_heartbeat_at,omitempty"`
	HeartbeatLatencyMs float64         `json:"heartbeat_latency_ms"`
	LastError          string          `json:"last_error,omitempty"`
	ReconnectAttempts  uint32          `json:"reconnect_attempts"`
}

// FrameHandler is invoked whenever a new frame is processed.
type FrameHandler func(frame *Frame)

// StateHandler is invoked whenever connection state changes.
type StateHandler func(state ConnectionState, err error)

// MarshalJSON provides clean JSON serialization for Frame with string payload representation.
func (f Frame) MarshalJSON() ([]byte, error) {
	type Alias Frame
	return json.Marshal(&struct {
		Alias
		PayloadStr string `json:"payload_str"`
		LatencyMs  int64  `json:"latency_ms"`
	}{
		Alias:      (Alias)(f),
		PayloadStr: string(f.Payload),
		LatencyMs:  f.Latency.Milliseconds(),
	})
}
