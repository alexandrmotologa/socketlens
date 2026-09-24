# Engineering Specification & Implementation Blueprint: SocketLens
> The Modern Local-First Desktop Studio & CLI for WebSocket, Server-Sent Events (SSE), Socket.io & Real-Time Streams (Reimagining wscat & Filling Postman's Streaming Void)

---

## 1. Executive Summary & Market Opportunity

### 1.1 The Market Vacuum
Real-time, bi-directional streaming is the backbone of modern software. In 2026, streaming traffic has surged exponentially due to:
* **AI & LLM Streaming:** Every generative AI interface uses **Server-Sent Events (SSE)** with `text/event-stream` delivering incremental tokens and agent tool-call chunks.
* **Modern Web Collaboration:** Real-time whiteboards, shared code editors, and multi-user apps powered by **WebSockets** and **Socket.io**.
* **Financial & Crypto Feeds:** High-frequency ticker orderbooks, bid/ask streams, and live telemetry running over raw WebSocket frames using binary serialization (MessagePack, Protobuf, CBOR).
* **IoT & Edge Messaging:** MQTT-over-WebSocket telemetry pipelines.

Yet, developer tooling for streaming remains fragmented and deeply frustrating:
* **Postman / Insomnia:** Extremely bloated (>1.2GB RAM usage), requires mandatory cloud login, sends sensitive traffic through third-party servers, and provides clunky streaming support that crashes on high-throughput binary feeds.
* **`wscat`:** Primitive terminal CLI tool that only handles plain text strings, has no timeline view, cannot inspect binary frames, and lacks session recording or replay.
* **Browser DevTools Network Tab:** Read-only, difficult to filter or search through 50,000+ frames, cannot inject custom payloads on the fly, and cannot simulate server-side events or disconnects.

### 1.2 The Solution: SocketLens
**SocketLens** is an ultra-fast, local-first streaming workbench and CLI designed from the ground up for real-time protocols: **WebSocket, SSE, Socket.io v4, and MQTT-over-WS**.

* **Unified Real-Time Inspector:** Seamlessly connect to, inspect, and interact with WebSockets (`ws://`, `wss://`), Server-Sent Events (`http://`, `https://` with SSE), and Socket.io v4 namespaces.
* **Automatic Binary Codec Translation:** In-flight bidirectional decoding of binary formats: **Protocol Buffers, MessagePack, CBOR, BSON, and Gzip-compressed frames** directly into human-readable, formatted JSON with Monaco Editor.
* **1-Click Embedded Mock Stream Server:** Spin up a local mock WebSocket / SSE server on any local port with custom event intervals, heartbeat ping-pong responses, and programmable chaos (abrupt disconnects, network jitter, corrupt frames) to test client resiliency.
* **High-Throughput Connection Flood Runner:** Spin up 1,000 to 10,000 concurrent client connections from your local machine to benchmark your backend's pub/sub scalability and memory limits.
* **Traffic Recording & Time-Accurate Replay:** Export live sessions into structured `.har` or `.jsonl` archives and replay message sequences with millisecond-accurate timing offsets.
* **Single-Binary Zero-Dependency Distribution:** Built in **Go 1.23+** with an embedded **React 19 + Monaco** visual studio via `go:embed`. Memory footprint <35MB RAM, sub-20ms cold boot.

---

## 2. Core Architecture & Tech Stack

```
┌────────────────────────────────────────────────────────────────────────┐
│                         SocketLens Architecture                        │
└────────────────────────────────────────────────────────────────────────┘

[ Web Studio / Desktop Workbench ] (http://localhost:50070)
               │
               ▼  (Internal WebSocket & REST Control Bus)
[ SocketLens Single Binary ] (Go 1.23+ Engine)
   ├── Embedded Web Server: go:embed (Vite + React 19 + Monaco + Lucide)
   ├── Protocol Dispatcher & Connection Hub
   │     ├── WebSocket Client (coder/websocket - high-performance RFC 6455)
   │     ├── Server-Sent Events Client (r3labs/sse event-stream parser)
   │     ├── Socket.io v4 Client (Engine.IO v4 protocol parser)
   │     └── MQTT-over-WS Client (eclipse/paho.mqtt.golang)
   ├── Codec & Message Transformation Pipeline
   │     ├── Protobuf Dynamic Decoder (bufbuild/protocompile)
   │     ├── MessagePack / CBOR / BSON Engine (vmihailenco/msgpack, fxamacker/cbor)
   │     └── JSON Schema Validator & Formatter
   ├── Mock Stream Server & Chaos Generator
   │     ├── In-Memory Dynamic WS / SSE Listener
   │     ├── Scriptable Script Engine (JavaScript / JSON recipes)
   │     └── Chaos Middleware (Jitter, Dropped frames, Zombie sockets)
   └── Telemetry & Stress Engine
         ├── High-Concurrency Worker Pool (epoll / goroutine multiplexer)
         ├── Frame Velocity & Bandwidth Meter (Messages/sec, Byte rate)
         └── Session Recorder & Replayer (.har / .jsonl)
               │
               ▼ (ws://, wss://, http://, https://)
[ Target Server / LLM API / Real-Time Microservice ]
```

### 2.1 Backend Technology
* **Language:** Go 1.23+
* **WebSocket Engine:** `github.com/coder/websocket` (formerly `nhooyr.io/websocket` — the gold standard Go WebSocket library with minimal allocations and zero goroutine leaks).
* **SSE Client:** `github.com/r3labs/sse/v2` (robust `text/event-stream` client with automatic reconnection and event ID tracking).
* **Binary Serialization:**
  * MessagePack: `github.com/vmihailenco/msgpack/v5`
  * CBOR: `github.com/fxamacker/cbor/v2`
  * Protobuf: `google.golang.org/protobuf`
* **HTTP Router:** `github.com/go-chi/chi/v5` + `github.com/go-chi/cors`.
* **CLI Framework:** `github.com/spf13/cobra` + `github.com/charmbracelet/bubbletea` (for terminal interactive mode).

### 2.2 Frontend Technology
* **Core:** Vite + React 19 + TypeScript.
* **Editor:** Monaco Editor (`@monaco-editor/react`) for message payload drafting, JSON formatting, and schema validation.
* **Timeline Virtualization:** `@tanstack/react-virtual` rendering 100,000+ incoming frames at 60 FPS without memory leaks.
* **Charts & Telemetry:** Lightweight SVG velocity charts (messages/sec, throughput in KB/s, latency jitter).
* **Styling:** Tailwind CSS + dark glassmorphic design system.

---

## 3. Key Feature Specifications

### 3.1 Universal Real-Time Connection Studio
* **Multi-Protocol Connect Bar:**
  * Select protocol: `WebSocket (WS/WSS)`, `Server-Sent Events (SSE)`, `Socket.io v4`, or `MQTT`.
  * URL input with environment variable substitution (`{{API_URL}}/stream`).
  * Custom Handshake Headers (`Authorization: Bearer ...`, `Sec-WebSocket-Protocol`, `Origin`, custom cookies).
  * TLS options: Insecure skip verify, custom CA root certificate, client mTLS certificate & key.
* **Connection Lifecycle Controller:**
  * Connect / Disconnect button with auto-reconnect policy (Linear, Exponential Backoff, Fixed interval).
  * Heartbeat / Ping-Pong management: auto-reply to server Pings with Pongs, or emit custom periodic heartbeat frames.

### 3.2 Real-Time Message Timeline & Frame Inspector
* **Color-Coded Directional Timeline:**
  * 🟢 **Inbound (Server ➔ Client)**: Timestamp (µs), opcode (Text/Binary/Ping/Pong), byte size, duration since last frame.
  * 🔵 **Outbound (Client ➔ Server)**: Quick resend, edit-and-resend, delete.
* **Live Search & Filter:**
  * Filter by opcode, regex text search, or JSON path expressions (`payload.type == "order_book"`).
* **Binary View:**
  * Switch between formatted JSON, raw UTF-8 string, Hex dump view, or Base64.
  * Decode Protobuf frames by uploading a `.proto` file or selecting reflected schema.

### 3.3 Message Composer & Automated Dispatcher
* **Dual Composer:**
  * Draft JSON or plain text messages in Monaco Editor with syntax highlighting.
  * Save reusable message templates in categorized collections.
* **Automated Dispatcher (Looping / Interval):**
  * Schedule recurring message broadcasts (e.g., send `{"action": "ping"}` every 5000ms).
  * Sequence runner: execute a scripted series of 5 messages with custom delays.

### 3.4 1-Click Embedded Mock Stream Server
* Select port (e.g. `:8080`) and route path (`/ws/feed` or `/api/chat/stream`).
* Configure response behavior:
  * **Echo Mode:** Returns any received client payload back to the sender.
  * **Broadcast Mode:** Emits received payloads to all connected clients.
  * **Tick Generator:** Emits continuous simulated events (e.g. streaming stock ticks or fake LLM tokens at 20 tokens/sec).
* **Chaos Mode:**
  * Inject latency (add 100–500ms delay to frames).
  * Drop 10% of frames randomly.
  * Trigger sudden TCP reset (`RST`) after 15 seconds to test client recovery.

### 3.5 High-Concurrency Stress Runner
* Test backend WebSocket / SSE scalability from the terminal or UI:
  * Configure: Target URL, Concurrency (`100` to `10,000` clients), Ramp-up duration, and message cadence.
  * Live metrics: Connected sockets, Handshake latency (p50, p95, p99), Disconnect rate, throughput.
  * Generates markdown performance benchmark summary.

---

## 4. CLI Command-Line Specification

```bash
# Launch SocketLens visual studio (http://localhost:50070)
socketlens

# Connect directly to a WebSocket in interactive terminal TUI mode
socketlens connect wss://echo.websocket.events

# Connect to an LLM Server-Sent Events stream and follow live tokens
socketlens connect https://api.openai.com/v1/chat/completions \
  --proto sse \
  --header "Authorization: Bearer $OPENAI_API_KEY"

# Start an embedded mock WebSocket server on port 8080
socketlens mock --port 8080 --route "/ws/feed" --mode echo

# Run a high-concurrency connection stress test (1,000 clients for 60s)
socketlens bench wss://api.example.com/stream \
  --clients 1000 \
  --duration 60s \
  --ramp-up 10s \
  --report benchmark.json

# Record a live session to a JSONL archive
socketlens record wss://api.example.com/feed --out ./session.jsonl

# Replay a recorded session back to a local server
socketlens replay ./session.jsonl --target ws://localhost:3000/feed
```

---

## 5. Complete Project Directory Layout

```
socketlens/
├── cmd/
│   └── socketlens/
│       └── main.go                         # Cobra CLI entrypoint
├── pkg/
│   ├── client/
│   │   ├── client.go                       # Universal connection manager
│   │   ├── websocket.go                    # coder/websocket implementation
│   │   ├── sse.go                          # Server-Sent Events stream client
│   │   ├── socketio.go                     # Socket.io v4 Engine.IO protocol adapter
│   │   └── types.go                        # Frame, OpCode, Event, ConnectionState
│   ├── codec/
│   │   ├── detector.go                     # Auto-detection of payload format
│   │   ├── protobuf.go                     # Dynamic Protobuf decoder
│   │   ├── msgpack.go                      # MessagePack encoder/decoder
│   │   ├── cbor.go                         # CBOR encoder/decoder
│   │   └── json.go                         # High-performance JSON formatter
│   ├── mock/
│   │   ├── server.go                       # Embedded mock HTTP/WS/SSE server
│   │   ├── echo_handler.go                 # Echo & broadcast handlers
│   │   ├── generator.go                    # Fake LLM token & ticker generators
│   │   └── chaos.go                        # Latency, packet drop, and disconnect simulator
│   ├── stress/
│   │   ├── runner.go                       # Concurrent connection pool engine
│   │   ├── worker.go                       # Client connection worker goroutine
│   │   └── metrics.go                      # Latency histograms, percentiles, error rates
│   ├── session/
│   │   ├── recorder.go                     # Stream recorder writing to JSONL
│   │   └── replayer.go                     # Time-accurate message replayer
│   └── server/
│       ├── api.go                          # REST & internal control bus router
│       ├── hub.go                          # UI WebSocket bridge for streaming timeline
│       └── static.go                       # go:embed static production frontend
├── ui/
│   ├── index.html                          # Web studio entrypoint
│   ├── package.json                        # Vite, React 19, Monaco, Tailwind
│   ├── tsconfig.json                       # Strict TypeScript configuration
│   ├── src/
│   │   ├── api/
│   │   │   ├── client.ts                   # REST API client
│   │   │   └── stream.ts                   # Internal event-stream listener
│   │   ├── components/
│   │   │   ├── ConnectBar.tsx              # URL, protocol, headers, connect toggle
│   │   │   ├── FrameTimeline.tsx           # Virtualized chronological frame list
│   │   │   ├── FrameDetailModal.tsx        # JSON, Hex, Raw, Decoded Protobuf view
│   │   │   ├── MessageComposer.tsx         # Monaco editor, template library, dispatch
│   │   │   ├── MockServerPanel.tsx         # Embedded mock server controller & chaos dials
│   │   │   ├── StressRunnerPanel.tsx       # Benchmarking dashboard with real-time graphs
│   │   │   └── StatsBar.tsx                # Messages/s, KB/s, ping latency pill
│   │   ├── types/
│   │   │   └── index.ts                    # TypeScript interface definitions
│   │   ├── App.tsx                         # Main workbench shell
│   │   └── main.tsx                        # React 19 mount point
├── Makefile                                # Build, test, and release targets
├── go.mod                                  # Go dependencies
├── go.sum                                  # Checksums
└── README.md                               # Documentation & animated demo
```

---

## 6. Implementation Roadmap & Execution Checklist

### Phase 1: Go Module Setup & Multi-Protocol Client Core
- [x] 1.1 Initialize Go module `github.com/alexandrmotologa/socketlens` and configure dependencies (`coder/websocket`, `r3labs/sse/v2`, `spf13/cobra`, `go-chi/chi/v5`, `go-chi/cors`, `vmihailenco/msgpack/v5`, `fxamacker/cbor/v2`).
- [x] 1.2 Implement `pkg/client/types.go` modeling Frame, Direction, Protocol, OpCode, PayloadFormat, and ConnectionState.
- [x] 1.3 Implement `pkg/client/websocket.go` with RFC 6455 support, ping/pong latency measurement, and automatic reconnection.
- [x] 1.4 Implement `pkg/client/sse.go` supporting `text/event-stream` line parsing, event names, retry directives, and Last-Event-ID recovery.
- [x] 1.5 Implement `pkg/client/socketio.go` handling Engine.IO v4 handshake, packet framing, and namespace event envelopes.
- [x] 1.6 Implement `pkg/client/client.go` universal connection manager dispatching across protocols.
- [x] 1.7 Write automated tests in `pkg/client/client_test.go` verifying WebSocket, SSE, and reconnection behavior against test servers.

### Phase 2: Binary Codec Pipeline & Serialization Engine
- [x] 2.1 Implement `pkg/codec/detector.go` sniffing magic bytes for MessagePack, CBOR, Protobuf wire format, Gzip, and JSON.
- [x] 2.2 Implement `pkg/codec/msgpack.go` converting MessagePack buffers into structured JSON AST.
- [x] 2.3 Implement `pkg/codec/cbor.go` converting CBOR binary frames into JSON AST.
- [x] 2.4 Implement `pkg/codec/protobuf.go` dynamic wire format decoder parsing varints, fixed64, length-delimited, and fixed32 tags without requiring static proto stubs.
- [x] 2.5 Implement `pkg/codec/gzip.go` and `pkg/codec/hex.go` for decompression and 16-byte hex dump views.
- [x] 2.6 Write automated tests in `pkg/codec/codec_test.go` verifying round-trip serialization and detection.

### Phase 3: Embedded Mock Server & Chaos Middleware
- [x] 3.1 Implement `pkg/mock/server.go` starting an in-memory HTTP/WS/SSE server on configurable ports.
- [x] 3.2 Implement Echo and Broadcast handlers for WebSocket connections.
- [x] 3.3 Implement continuous event generators:
  - LLM token stream generator emitting tokens at configurable rates (1 to 100 tokens/s).
  - Financial orderbook ticker generator.
- [x] 3.4 Implement Chaos Middleware introducing configurable latency (50 to 500ms), packet loss (0 to 100%), and abrupt connection resets.
- [x] 3.5 Write unit and integration tests in `pkg/mock/mock_test.go` verifying mock streaming behavior and chaos disruption.

### Phase 4: Stress Benchmarking & Session Recorder / Replayer
- [x] 4.1 Implement `pkg/stress/runner.go` managing concurrent connection pools (100 to 10,000 workers).
- [x] 4.2 Implement `pkg/stress/metrics.go` calculating p50, p90, p95, p99 handshake latency, throughput in msg/s and KB/s, and error rates.
- [x] 4.3 Implement `pkg/session/recorder.go` streaming captured frames into newline-delimited JSON (`.jsonl`).
- [x] 4.4 Implement `pkg/session/replayer.go` reading captured sessions and dispatching frames with millisecond-exact relative timing and speed multipliers (0.1x to 10x).
- [x] 4.5 Write tests validating stress test execution and recorder/replayer timing accuracy.

### Phase 5: CLI Architecture & Interactive Terminal TUI Mode
- [x] 5.1 Implement Cobra CLI entrypoint in `cmd/socketlens/main.go` and subcommands: `connect`, `mock`, `bench`, `record`, `replay`.
- [x] 5.2 Implement Bubbletea interactive TUI in `pkg/tui/` with dual-pane layout (Connection stats on top, stream timeline on bottom).
- [x] 5.3 Implement keyboard controls (arrow navigation, pause Space, inspect Enter, quit q) with color-coded directional frames.

### Phase 6: Embedded Web Studio & Single-Binary Delivery
- [x] 6.1 Scaffold Vite + React 19 + TypeScript + Tailwind CSS in `ui/`.
- [x] 6.2 Build high-performance Frame Timeline with virtualized rendering for 50k+ frames.
- [x] 6.3 Implement Monaco Message Composer with custom JSON templates and auto-dispatch intervals.
- [x] 6.4 Implement Mock Server & Stress Runner dashboards with real-time SVG charts.
- [x] 6.5 Build internal WebSocket control bus in `pkg/server/hub.go` bridging the Go engine to the browser studio.
- [x] 6.6 Embed frontend assets into Go binary using `go:embed`.
- [x] 6.7 Package single-binary release and verify complete end-to-end functionality across macOS, Linux, and Windows.

### Phase 7: Advanced Interception Proxy, AI/LLM Stream Telemetry & Studio Enhancements
- [x] 7.1 **Bi-directional Interception Proxy & Live Tampering (`pkg/proxy`, `socketlens proxy`, `ProxyPanel`):**
  - Transparent forward proxy for WebSockets with breakpoint support.
  - Allows developers to suspend frames in flight, inspect, modify payload bytes, or drop frames before upstream forwarding.
- [x] 7.2 **LLM & AI Stream Inspector (`pkg/llm`, `LLMInspectorPanel`):**
  - Real-time SSE streaming metrics: Time-To-First-Token (TTFT in ms), Tokens Per Second (TPS), average inter-token jitter.
  - Live token assembler with auto-reconstructed message markdown and structured Tool-Call argument extraction (OpenAI, Anthropic, Ollama).
- [x] 7.3 **In-Flight State Diffing (`pkg/codec/diff.go`, `FrameDetailPanel`):**
  - Structural and value-level JSON diffing comparing sequential streaming frames to pinpoint delta changes instantly.
- [x] 7.4 **Real-Time JSON Schema Validation (`pkg/schema`, `SchemaValidatorPanel`):**
  - Live validation of all incoming and outgoing stream frames against JSON Schema Draft-07.
  - Real-time visual error flags in the timeline for breaking contract changes and missing fields.
- [x] 7.5 **Event-Driven Auto-Responders & Automation Rules (`pkg/rules`, `AutomationRulesPanel`):**
  - Scriptable If-This-Then-That automation rules for immediate mock replies, heartbeat ping-pong, and contract simulation.
- [x] 7.6 **Universal Export (`pkg/session/har.go`, `StatsBar`, `ConnectBar`):**
  - Export live stream sessions into Chrome DevTools-compatible HAR 1.2 archives.
  - Instant 1-click CLI command reproduction (`wscat` and `curl -N`).
  - Real-time SVG throughput sparklines visualizing stream velocity.


---

## 7. Verification & Acceptance Criteria
1. **Multi-Protocol Fidelity:** Must connect to, send messages, and parse frames from standard WebSockets (`wss://`), SSE endpoints (`text/event-stream`), and Socket.io servers.
2. **High Throughput:** Must comfortably process and render 10,000 incoming frames/second without memory leaks or UI freezes.
3. **Binary Codec Support:** Must automatically detect and render MessagePack and CBOR binary frames into readable JSON.
4. **Mock Server Resilience:** Mock server must support concurrent clients and reliably simulate network jitter and drops.
5. **Zero External Dependencies:** Single compiled binary (<25MB) with zero external runtime requirements.

---

## 8. Kick-Off Prompt for Subagent

```markdown
Please read [SOCKETLENS_SPEC_AND_ROADMAP.md](file:///B:/workgit/socketlens/SOCKETLENS_SPEC_AND_ROADMAP.md) in full and execute Phase 1: scaffold the Go 1.23+ project in `B:\workgit\socketlens`, configure core dependencies (`coder/websocket`, `r3labs/sse/v2`, `spf13/cobra`, `go-chi/chi/v5`), implement the universal WebSocket and Server-Sent Events client engine, build the core frame data models, and write automated tests verifying connection and message receipt against local test endpoints.
```
