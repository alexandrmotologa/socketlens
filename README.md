<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/images/logo.png">
    <img src="docs/images/logo.png" alt="SocketLens Logo" width="128" style="border-radius: 24px;" />
  </picture>
</p>

<h1 align="center">SocketLens</h1>

<p align="center">
  <strong>High-performance streaming workbench, binary codec inspector, and chaos lab for WebSocket, SSE, and Socket.io.</strong>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.23%2B-blue" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  <a href="https://github.com/alexandrmotologa/socketlens"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey" alt="Platform"></a>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#visual-walkthrough">Visual Walkthrough</a> &bull;
  <a href="#capabilities">Capabilities</a> &bull;
  <a href="#installation">Installation</a> &bull;
  <a href="#architecture-overview">Architecture</a> &bull;
  <a href="#documentation">Documentation</a>
</p>

<p align="center">
  <img src="docs/images/socketlens_demo.gif" alt="SocketLens Interactive Studio Demo" width="100%" />
</p>

---

## Visual Walkthrough

### Interactive Stream Studio
Virtualize tens of thousands of incoming and outgoing frames, inspect binary and JSON payloads, examine live payload diffs, and filter streams by opcode, size, or regex.

![SocketLens Studio Timeline](docs/images/socketlens_studio.png)

### Bi-Directional Interception Proxy & Live Tampering
Place non-blocking or blocking breakpoints on live streams. Intercept payloads in transit, edit contents before upstream delivery, or drop frames on the fly.

![SocketLens Interception Proxy](docs/images/socketlens_proxy.png)

### LLM & AI Stream Telemetry
Measure Time-To-First-Token (TTFT), real-time Tokens Per Second (TPS), and inter-token jitter on Server-Sent Events. Reassemble token fragments into rendered markdown and parse structured tool calls.

![SocketLens LLM Telemetry](docs/images/socketlens_llm.png)

### Real-Time JSON Schema Validation
Validate streaming events against JSON Schema (Draft-07) specifications as frames arrive. Instantly locate breaking contract changes with inline visual diagnostics.

![SocketLens Schema Validation](docs/images/socketlens_schema.png)

### Concurrency Stress Benchmark
Stress-test streaming servers with 100 to 10,000 concurrent client connections. Track latency percentiles (p50, p95, p99), frame distribution, and throughput stability under load.

![SocketLens Stress Runner](docs/images/socketlens_bench.png)

---

## Capabilities

- **Unified Stream Inspector**: Connect to `ws://`, `wss://`, Server-Sent Events (`http://`, `https://`), and Socket.io v4 endpoints with custom headers, query params, subprotocols, and TLS certificates.
- **Bi-Directional Interception Proxy & Live Tampering**: Start a transparent local proxy to capture streaming frames in flight. Pause streams on breakpoints, inspect and modify payloads interactively, or drop packets before upstream forwarding.
- **LLM and AI Stream Telemetry**: Inspect SSE token events with real-time Time-To-First-Token (TTFT in ms), Tokens Per Second (TPS), and inter-token jitter. Automatically reassemble fragmented tokens into full markdown text and parse structured tool calls.
- **In-Flight Semantic Diffing**: Spot state changes across high-frequency message streams with instant structural and value-level JSON diffs.
- **Real-Time JSON Schema Validation**: Enforce JSON Schema (Draft-07) contracts across live streams with visual timeline error flags for broken payload contracts.
- **Event-Driven Auto-Responders**: Build automation rules that match incoming frame conditions and reply immediately with custom payloads.
- **Universal Export**: Save sessions to Chrome DevTools-compatible HAR 1.2 archives or reproduce requests in one click via generated `wscat` and `curl` commands.
- **In-Flight Binary Codecs**: Inspect binary payloads decoded on the fly. Supports MessagePack, CBOR, Protobuf, and Gzip-compressed frames alongside JSON and hex dump views.
- **Embedded Mock Server with Chaos Testing**: Spin up local mock endpoints with configurable responses, token stream generation for simulated LLM output, and chaos options including latency injection, dropped frames, and forced socket resets.
- **Stress and Concurrency Runner**: Benchmark backend stream capacity by driving 100 to 10,000 concurrent client connections with real-time latency percentiles (p50, p95, p99), error rates, and throughput metrics.
- **Session Recording and Replay**: Record live streaming sessions to `.jsonl` archives and replay frame sequences with millisecond accuracy and adjustable speed multipliers.
- **Dual Interface**: Run interactive sessions in your terminal through the built-in TUI or launch the local web studio with virtualized timeline scrolling and Monaco editor integration.
- **Zero Runtime Dependencies**: Packaged as a single Go binary with an embedded React frontend, consuming under 35 MB of RAM at idle.

---

## Installation

### From Source

Ensure Go 1.23 or newer is installed:

```bash
git clone https://github.com/alexandrmotologa/socketlens.git
cd socketlens
go install ./cmd/socketlens
```

### Prebuilt Binaries

Download compiled binaries for Linux, macOS, and Windows directly from the GitHub releases page.

---

## Quick Start

### 1. Launch the Web Studio

Start the local server and open the studio interface in your default browser:

```bash
socketlens
# Serves the web interface on http://127.0.0.1:50070
```

### 2. Connect via Terminal TUI

Connect directly to a WebSocket server in interactive terminal mode:

```bash
socketlens connect wss://echo.websocket.events
```

### 3. Stream an LLM Server-Sent Events Feed

Follow real-time token output from an SSE endpoint:

```bash
socketlens connect https://api.openai.com/v1/chat/completions \
  --proto sse \
  --header "Authorization: Bearer $OPENAI_API_KEY"
```

### 4. Run an Embedded Mock Server

Spin up an echo server with chaos simulation enabled:

```bash
socketlens mock --port 8080 --route "/ws/feed" --mode echo --chaos-latency 150ms --chaos-drop 5
```

### 5. Benchmark Connection Capacity

Run a concurrent load test with 500 connections for 30 seconds:

```bash
socketlens bench wss://api.example.com/stream \
  --clients 500 \
  --duration 30s \
  --ramp-up 5s \
  --report benchmark.json
```

### 6. Record and Replay Traffic

Save incoming and outgoing frames to disk and replay them against a local development server:

```bash
# Record live stream to file
socketlens record wss://stream.binance.com:9443/ws/btcusdt@trade --out btc_trades.jsonl

# Replay recorded frames at 2x speed
socketlens replay btc_trades.jsonl --target ws://localhost:8080/feed --speed 2.0
```

### 7. Intercept and Tamper Live Streams

Start a transparent interception proxy on port 8081 forwarding to an upstream endpoint with breakpoint inspection:

```bash
socketlens proxy wss://echo.websocket.events --port 8081 --breakpoint
```


---

## Architecture Overview

```
[ Web Studio (React 19 + Monaco) ] <--> [ Embedded Go Server (:50070) ]
                                                |
        +---------------------------------------+---------------------------------------+
        |                                       |                                       |
  [ Protocol Clients ]                   [ Codec Pipeline ]                   [ Mock & Chaos ]
  - WebSocket (coder/websocket)          - MessagePack                        - Echo / Broadcast
  - Server-Sent Events (r3labs/sse)      - CBOR                               - Token & Ticker Generator
  - Socket.io v4 Engine                  - Protobuf Wire Decoder              - Latency & Drop Simulator
        |                                - Gzip Decompressor                            |
        v                                       |                                       v
[ Remote Endpoints ]                     [ Timeline Bus ]                      [ Local Test Sockets ]
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for full subsystem details.

---

## Documentation

- [Architecture Guide](docs/ARCHITECTURE.md): Go core engine, protocol clients, virtualized timeline, and codec pipelines.
- [Protocol Specifications](docs/API_AND_PROTOCOLS.md): Details on RFC 6455 WebSocket handling, SSE chunking, and Socket.io handshakes.
- [CLI Reference](docs/CLI_REFERENCE.md): Flags and commands for `connect`, `mock`, `bench`, `record`, and `replay`.
- [Benchmarking Guide](docs/BENCHMARKING.md): Tuning OS file descriptors and interpreting latency percentiles.

---

## Contributing

1. Fork the repository.
2. Create a feature branch: `git checkout -b feature/new-codec`.
3. Commit your changes: `git commit -m "feat(codec): add avro binary decoder"`.
4. Push to your branch and open a pull request.

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
