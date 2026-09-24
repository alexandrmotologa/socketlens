# Protocol Specifications and API Reference

This document outlines the protocols supported by SocketLens and specifies the internal HTTP and WebSocket control APIs exposed by the embedded server.

---

## 1. Supported Streaming Protocols

### 1.1 WebSocket (RFC 6455)

SocketLens connects to standard `ws://` and `wss://` endpoints.

- **Handshake**: Transmits standard HTTP Upgrade headers. Supports custom headers, authorization tokens, cookies, and `Sec-WebSocket-Protocol` subprotocol negotiation.
- **Framing**: Handles Text (0x1), Binary (0x2), Close (0x8), Ping (0x9), and Pong (0xA) frames.
- **Heartbeat**: Configurable interval ping frames. Measures round-trip time between ping transmission and pong reception.
- **TLS Configuration**: Allows bypassing invalid certificates (`--insecure`) and mounting custom CA bundles or client certificates for mutual TLS (mTLS).

### 1.2 Server-Sent Events (SSE)

SocketLens reads `text/event-stream` feeds over HTTP/1.1 and HTTP/2.

- **Request Headers**: Automatically supplies `Accept: text/event-stream` and `Cache-Control: no-cache`.
- **Parsing**: Separates fields by newline characters. Merges consecutive `data:` fields with standard newline joins.
- **Reconnection**: Remembers the most recent `id:` value and populates the `Last-Event-ID` header on automatic reconnection attempts.

### 1.3 Socket.io v4

SocketLens connects to Socket.io servers implementing the Engine.IO v4 protocol over WebSocket transport.

- **Handshake URL**: `ws://<host>:<port>/socket.io/?EIO=4&transport=websocket`
- **Session Negotiation**: Reads the opening Engine.IO packet (`0{"sid":"...","upgrades":[],"pingInterval":25000,"pingTimeout":20000}`).
- **Event Dispatch**: Formats client emissions as type 42 message packets: `42["eventName", { ...payload }]`.

---

## 2. Embedded Server REST API

The embedded server runs on port 50070 by default (`http://127.0.0.1:50070`).

### 2.1 Connection Management

#### `POST /api/v1/connections`
Establishes a new stream connection.

**Request Payload:**
```json
{
  "protocol": "ws",
  "url": "wss://echo.websocket.events",
  "headers": {
    "Authorization": "Bearer token123"
  },
  "subprotocols": ["chat.v1"],
  "tls_insecure": false,
  "auto_reconnect": true
}
```

**Response (200 OK):**
```json
{
  "connection_id": "conn_98a7f1bc",
  "status": "connected",
  "url": "wss://echo.websocket.events",
  "created_at": "2026-09-25T00:20:00Z"
}
```

#### `GET /api/v1/connections`
Lists all active and previous stream sessions.

#### `DELETE /api/v1/connections/:id`
Terminates an active connection.

---

### 2.2 Frame Transmission

#### `POST /api/v1/connections/:id/send`
Dispatches a frame to the target stream.

**Request Payload:**
```json
{
  "opcode": "text",
  "payload": "{\"event\": \"subscribe\", \"channel\": \"orders\"}"
}
```

**Response (200 OK):**
```json
{
  "status": "sent",
  "bytes": 42,
  "timestamp": "2026-09-25T00:20:05Z"
}
```

---

### 2.3 Mock Server Control

#### `POST /api/v1/mock/start`
Starts a local mock stream server.

**Request Payload:**
```json
{
  "port": 8080,
  "route": "/ws/feed",
  "mode": "echo",
  "chaos": {
    "latency_ms": 100,
    "drop_percentage": 5
  }
}
```

#### `POST /api/v1/mock/stop`
Stops the running mock server.

---

### 2.4 Benchmarking Control

#### `POST /api/v1/bench/start`
Starts a concurrent load test.

**Request Payload:**
```json
{
  "url": "ws://localhost:8080/feed",
  "clients": 500,
  "duration_seconds": 30,
  "ramp_up_seconds": 5
}
```

#### `GET /api/v1/bench/status`
Returns real-time benchmark metrics including active connections, handshake latencies, and message throughput.

---

## 3. Internal Control WebSocket

The visual studio communicates with the Go backend over a local WebSocket control channel at `ws://127.0.0.1:50070/ws/control`.

### Event Types Dispatched by Server:

- `timeline_batch`: An array of serialized incoming or outgoing `Frame` objects.
- `connection_state`: Updates to connection status (connecting, connected, disconnected, error).
- `metrics_tick`: 1-second interval telemetry with frame rate, bandwidth usage, and latency metrics.
- `bench_update`: Live progress updates during stress test execution.
