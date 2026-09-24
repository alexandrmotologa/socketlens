# CLI Reference

SocketLens provides both an interactive terminal interface and direct subcommands for automation and scripting.

---

## Global Flags

The following flags apply to all commands:

```
  -h, --help            Help for socketlens
  -v, --version         Print version information
      --config string   Path to YAML or JSON configuration file
      --verbose         Enable debug logging to stderr
```

---

## 1. Default Command (`socketlens`)

Launches the local web studio and starts the internal server.

```bash
socketlens [flags]
```

### Flags:
- `--port int`: Port to bind the web studio (default: `50070`).
- `--host string`: Host interface to bind (default: `127.0.0.1`).
- `--no-browser`: Do not automatically open the default web browser on startup.

---

## 2. `socketlens connect`

Connects directly to a streaming endpoint in terminal mode.

```bash
socketlens connect <URL> [flags]
```

### Examples:
```bash
# Connect to an echo WebSocket server
socketlens connect wss://echo.websocket.events

# Connect to an SSE endpoint with authorization headers
socketlens connect https://api.openai.com/v1/chat/completions \
  --proto sse \
  --header "Authorization: Bearer $OPENAI_API_KEY"

# Connect with auto-reconnection and custom heartbeat interval
socketlens connect wss://stream.binance.com:9443/ws/btcusdt@trade \
  --reconnect \
  --heartbeat 15s
```

### Flags:
- `--proto string`: Protocol override (`ws`, `sse`, `socketio`). Defaults to auto-detection from URL scheme.
- `-H, --header strings`: Custom HTTP headers in `Key: Value` format.
- `--subprotocol strings`: WebSocket subprotocols to request during handshake.
- `--insecure`: Allow connections to servers with self-signed or invalid TLS certificates.
- `--reconnect`: Automatically attempt reconnection on unexpected disconnects.
- `--heartbeat duration`: Interval to send periodic WebSocket ping frames.
- `--record string`: Write all session traffic directly to the specified `.jsonl` file.

---

## 3. `socketlens proxy`

Starts a transparent bidirectional proxy forwarding client traffic to an upstream server, logging frames in real time with optional breakpoint tampering.

```bash
socketlens proxy <target-URL> [flags]
```

### Examples:
```bash
# Proxy local connections to a remote WebSocket server
socketlens proxy wss://echo.websocket.events --port 8081

# Intercept frames with breakpoint inspection enabled
socketlens proxy wss://api.example.com/feed --port 8081 --breakpoint
```

### Flags:
- `-p, --port int`: Local port to bind the proxy listener (default: `8081`).
- `-b, --breakpoint`: Enable frame breakpoint suspension for live inspection and payload tampering before forwarding.

---

## 4. `socketlens mock`


Starts a local mock stream server for integration testing and client resilience checks.

```bash
socketlens mock [flags]
```

### Examples:
```bash
# Start an echo WebSocket server on port 8080
socketlens mock --port 8080 --route "/ws/feed" --mode echo

# Start an SSE mock streaming fake LLM tokens at 25 tokens/s
socketlens mock --port 9000 --route "/stream" --mode llm-tokens --rate 25

# Simulate an unreliable network with latency and packet drops
socketlens mock --port 8080 --mode echo --chaos-latency 200ms --chaos-drop 10
```

### Flags:
- `-p, --port int`: Port to listen on (default: `8080`).
- `-r, --route string`: Path prefix for the stream handler (default: `/ws`).
- `-m, --mode string`: Stream generation mode (`echo`, `broadcast`, `llm-tokens`, `ticks`).
- `--rate int`: Token or tick emission frequency in events per second (default: `10`).
- `--chaos-latency duration`: Artificial latency added to frame dispatches.
- `--chaos-drop int`: Percentage of frames to drop randomly (0 to 100).
- `--chaos-reset duration`: Abruptly reset client sockets after this duration.

---

## 5. `socketlens bench`

Executes high-concurrency connection stress tests against a target streaming endpoint.

```bash
socketlens bench <URL> [flags]
```

### Examples:
```bash
# Connect 1,000 clients over a 15-second ramp-up period
socketlens bench wss://api.example.com/stream \
  --clients 1000 \
  --ramp-up 15s \
  --duration 60s \
  --report benchmark_results.json
```

### Flags:
- `-c, --clients int`: Number of concurrent connections to maintain (default: `100`).
- `-d, --duration duration`: Total benchmark run time (default: `30s`).
- `--ramp-up duration`: Time window to gradually spin up connections (default: `5s`).
- `--send-interval duration`: Frequency at which each client transmits messages.
- `--payload string`: Payload string or path to JSON file dispatched by clients.
- `--report string`: Destination path to save summary metrics in JSON format.

---

## 6. `socketlens record`

Captures frames from a live stream into a newline-delimited JSON archive.

```bash
socketlens record <URL> --out <file.jsonl> [flags]
```

### Flags:
- `-o, --out string`: Path to output file (required).
- `-H, --header strings`: Custom headers for the recording session.
- `--max-frames int`: Stop recording automatically after capturing this many frames.

---

## 7. `socketlens replay`


Replays a captured session file against a target endpoint with time-accurate offsets.

```bash
socketlens replay <FILE.jsonl> --target <URL> [flags]
```

### Flags:
- `-t, --target string`: Target URL to receive the replayed frames (required).
- `-s, --speed float`: Playback speed multiplier (e.g. `2.0` for double speed, `0.5` for half speed; default: `1.0`).
- `--loop`: Repeat playback indefinitely until interrupted.
