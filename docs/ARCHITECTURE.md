# Architecture Guide

SocketLens is structured around an asynchronous Go engine paired with an embedded browser-based frontend. This document describes the internal components, data flows, and concurrency model.

---

## 1. High-Level Subsystems

The system consists of five primary layers:

1. **Protocol Engine (`pkg/client`)**: Manages network transports for WebSocket (RFC 6455), Server-Sent Events (`text/event-stream`), and Socket.io v4.
2. **Codec Pipeline (`pkg/codec`)**: Detects payload encodings and decodes binary wire formats into structured JSON or formatted text representations.
3. **Mock and Chaos Engine (`pkg/mock`)**: Provides an in-memory HTTP/WS server capable of scripted frame delivery, token streaming, and simulated transport failures.
4. **Stress and Telemetry Engine (`pkg/stress`, `pkg/session`)**: Runs concurrent connection pools, aggregates latency histograms, and records or replays frame sequences.
5. **Control Server and Studio (`pkg/server`, `ui`)**: Exposes a local REST API and internal WebSocket bridge to stream timeline frames to the React visual studio.

---

## 2. Protocol Engine

### WebSocket Client (`pkg/client/websocket.go`)

The WebSocket implementation uses `github.com/coder/websocket`. It operates with explicit context cancellation, connection timeouts, and customizable handshake headers. 

- Frames arrive as either text or binary opcodes.
- Outbound frames queue through a buffered channel to prevent blocking the caller during high network backpressure.
- Ping and pong handlers calculate round-trip heartbeat latency and maintain connection health statistics.
- Automatic reconnection strategies include constant interval and exponential backoff with jitter.

### Server-Sent Events Client (`pkg/client/sse.go`)

The SSE client implements line-delimited event stream parsing according to the W3C EventSource standard:

- Buffers incoming chunks and parses `event:`, `data:`, `id:`, and `retry:` fields.
- Tracks the `Last-Event-ID` header and automatically re-supplies it when reconnecting after network interruptions.
- Handles multiline data payloads by joining lines with newline delimiters before frame dispatch.

### Socket.io v4 Adapter (`pkg/client/socketio.go`)

The adapter manages Engine.IO packet framing and Socket.io v4 message envelopes:

- Executes the Engine.IO handshake (`/socket.io/?EIO=4&transport=websocket`).
- Parses packet types (open, close, ping, pong, message, upgrade, noop).
- Enforces message routing across namespaces and decodes JSON event arrays (`["event_name", payload]`).

---

## 3. Codec Pipeline

Frames received over the wire pass through a detection and transformation pipeline:

1. **Format Sniffing**: Inspects magic bytes and initial byte patterns to distinguish JSON, MessagePack, CBOR, Protobuf wire format, and Gzip compression.
2. **Decompression**: Inflates Gzip or Deflate payloads before handing the buffer to serial decoders.
3. **Binary Transcoding**: Decodes MessagePack and CBOR into standard Go maps and slices, which serialize to formatted JSON for the timeline inspector.
4. **Protobuf Inspection**: Decodes field tags, wire types (varint, fixed64, length-delimited, fixed32), and values without requiring ahead-of-time compiled schemas, while supporting uploaded `.proto` definitions for full field naming.
5. **Hex Representation**: Generates traditional 16-byte offset hex dumps with ASCII sidebar views for unparsed binary payloads.

---

## 4. Timeline Data Flow and Internal Hub

Incoming and outgoing frames generate an immutable `Frame` struct:

```go
type Frame struct {
    ID        string        `json:"id"`
    ConnID    string        `json:"connection_id"`
    Timestamp time.Time     `json:"timestamp"`
    Direction Direction     `json:"direction"` // Inbound or Outbound
    Protocol  ProtocolType  `json:"protocol"`  // ws, sse, socketio
    OpCode    OpCode        `json:"opcode"`    // Text, Binary, Ping, Pong, Close
    Payload   []byte        `json:"payload"`
    Decoded   string        `json:"decoded,omitempty"`
    Format    PayloadFormat `json:"format"`    // json, msgpack, cbor, proto, raw
    Length    int           `json:"length"`
    Latency   time.Duration `json:"latency,omitempty"`
}
```

Frames publish to the `TimelineHub` (`pkg/server/hub.go`). The hub broadcasts new events to connected web clients over a local WebSocket control channel (`ws://127.0.0.1:50070/ws/control`).

To maintain smooth rendering during high-throughput tests (e.g. 10,000 frames per second), the hub batches timeline dispatches in 20-millisecond windows. The browser client receives array batches instead of individual socket events.

---

## 5. Mock Server and Chaos Simulation

The embedded mock server (`pkg/mock/server.go`) runs an HTTP listener on a requested local port:

- **Echo Mode**: Reflects received text or binary frames back to the sender.
- **Broadcast Mode**: Distributes received payloads to all active connections.
- **Token Generator**: Generates incremental text chunks mimicking LLM completions at a configured token velocity (e.g. 20 tokens per second).
- **Chaos Middleware**:
  - Latency injection: Delays frame writes using a configurable millisecond range.
  - Drop rate: Discards a percentage of inbound or outbound frames.
  - Connection truncation: Abruptly terminates underlying TCP connections after a timeout to test client reconnection handling.

---

## 6. Frontend Architecture

The web studio is located in `ui/` and builds with Vite, React 19, TypeScript, and Tailwind CSS:

- **Timeline Virtualization**: Uses `@tanstack/react-virtual` to display up to 100,000 frames without DOM bloat.
- **Payload Inspection**: Integrates Monaco Editor for JSON formatting, syntax validation, and message drafting.
- **State Management**: Uses lightweight reactive stores to isolate frame ingestion from inspector panel re-renders, preventing UI stutter during high-frequency streaming.

---

## 7. Advanced Interception & Telemetry Subsystems

### Bi-directional Interception Proxy (`pkg/proxy`)
Operates as a local TCP/WebSocket forward proxy. Incoming frames can be intercepted on breakpoints, holding execution in a suspended state (`resumeCh` / `dropCh`). Developers can inspect payloads, modify byte contents in the web studio or CLI, and resume forwarding or drop frames entirely.

### LLM Stream Inspector (`pkg/llm`)
Parses Server-Sent Events token deltas (OpenAI, Anthropic, Ollama) on the fly:
- Computes **Time-To-First-Token (TTFT)** measuring the elapsed time between request dispatch and the initial token frame.
- Calculates **Tokens Per Second (TPS)** throughput and inter-token arrival jitter.
- Reconstructs full response markdown text in memory while extracting structured tool calls (`id`, `name`, `arguments`).

### In-Flight State Diffing (`pkg/codec/diff.go`)
Recursively traverses JSON key-value maps and arrays between sequential frames. Emits structured changes categorized as `added`, `removed`, or `modified` with previous and current values to surface delta state mutations.

### Real-Time JSON Schema Validation (`pkg/schema`)
Compiles JSON Schema definitions (Draft-07) and validates payload bytes on incoming and outgoing frames. Discrepancies generate violation records attached directly to the `Frame` metadata and flagged in the UI timeline.

### Event-Driven Automation Rules (`pkg/rules`)
Enforces scriptable If-This-Then-That rules evaluated on inbound frames. Supports string matching, exact equality, event name matching, and opcode filtering, triggering immediate automated replies via the connection manager.

### Universal Export (`pkg/session/har.go`)
Converts recorded streaming sessions into HTTP Archive 1.2 (`.har`) format according to the Chrome DevTools network specification, preserving millisecond timestamps, opcodes, and directionality.

