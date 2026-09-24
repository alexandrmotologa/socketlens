# Benchmarking and Stress Testing Guide

SocketLens includes a built-in stress runner (`socketlens bench`) capable of maintaining thousands of simultaneous WebSocket connections from a single developer machine. This guide outlines host configuration and metrics interpretation.

---

## 1. Operating System Tuning for High Concurrency

Operating systems place conservative defaults on open file descriptors and ephemeral network ports. When running tests with more than 1,024 concurrent connections, apply the following adjustments.

### Linux and macOS

Check current open file descriptor limits:

```bash
ulimit -n
```

Temporarily increase descriptor limits in your terminal session before launching the test:

```bash
ulimit -n 65535
```

For persistent high-throughput testing on Linux, raise ephemeral port ranges and enable socket reuse:

```bash
# Expand available client port range
sudo sysctl -w net.ipv4.ip_local_port_range="1024 65535"

# Allow rapid recycling of sockets in TIME_WAIT state
sudo sysctl -w net.ipv4.tcp_tw_reuse=1
```

### Windows

On Windows hosts, increase the dynamic port allocation range through PowerShell with administrator rights:

```powershell
netsh int ipv4 set dynamicport tcp start=10000 num=55535
```

---

## 2. Benchmark Execution Lifecycle

A benchmark run proceeds through three distinct stages:

1. **Ramp-Up Phase**: Client goroutines connect incrementally across the specified ramp duration. This prevents connection stampedes and measures server accept queue stability.
2. **Sustain Phase**: All connected clients maintain their connections and transmit scheduled ping or data frames at the requested cadence.
3. **Teardown Phase**: Connections close gracefully, and final latency percentiles are compiled.

---

## 3. Interpreting Benchmark Metrics

When a test completes, SocketLens outputs a summary report:

```
=== SocketLens Benchmark Results ===
Target:             wss://api.example.com/stream
Target Concurrency: 1000 clients
Active Connections: 998 clients (99.8%)
Failed Handshakes:  2

Handshake Latency:
  p50:              14.2 ms
  p90:              28.7 ms
  p95:              41.1 ms
  p99:              89.4 ms
  Max:              182.0 ms

Throughput:
  Frames Sent:      29,840
  Frames Received:  29,820
  Message Rate:     994.0 msgs/sec
  Bandwidth:        1.42 MB/sec
```

### Metric Definitions:

- **p50 Latency**: The median duration required to complete the TCP, TLS, and WebSocket upgrade handshake. Half of all connections connected faster than this value.
- **p95 and p99 Latency**: The tail latencies experienced during peak connection spikes. Tail latency spikes often point to thread pool contention, slow database lookups during auth token verification, or saturated TLS termination proxies.
- **Failed Handshakes**: Connections that encountered TCP resets, TLS handshake timeouts, or non-101 HTTP status responses during upgrade.
