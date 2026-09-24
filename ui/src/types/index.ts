export type Protocol = 'ws' | 'wss' | 'sse' | 'socketio';
export type Direction = 'inbound' | 'outbound';
export type OpCode = 'text' | 'binary' | 'ping' | 'pong' | 'close' | 'event';
export type PayloadFormat = 'json' | 'text' | 'msgpack' | 'cbor' | 'protobuf' | 'gzip' | 'raw';
export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error';

export interface Frame {
  id: string;
  connection_id: string;
  sequence: number;
  timestamp: string;
  direction: Direction;
  protocol: Protocol;
  opcode: OpCode;
  payload_str?: string;
  decoded?: string;
  format: PayloadFormat;
  length: number;
  latency_ms?: number;
  event_name?: string;
  metadata?: Record<string, string>;
}

export interface ConnectionConfig {
  id?: string;
  name?: string;
  url: string;
  protocol: Protocol;
  headers?: Record<string, string>;
  subprotocols?: string[];
  tls_insecure?: boolean;
  auto_reconnect?: boolean;
  heartbeat_interval?: number;
}

export interface ConnectionStats {
  state: ConnectionState;
  connected_at?: string;
  uptime_seconds: number;
  frames_received: number;
  frames_sent: number;
  bytes_received: number;
  bytes_sent: number;
  last_heartbeat_at?: string;
  heartbeat_latency_ms: number;
  last_error?: string;
  reconnect_attempts: number;
}

export interface MockServerConfig {
  port: number;
  route: string;
  mode: 'echo' | 'broadcast' | 'llm-tokens' | 'ticks';
  rate: number;
  chaos: {
    latency: number;
    drop_rate: number;
  };
}

export interface BenchmarkReport {
  target_url: string;
  target_clients: number;
  active_clients: number;
  peak_clients: number;
  successful_handshakes: number;
  failed_handshakes: number;
  duration: number;
  p50_latency_ms: number;
  p90_latency_ms: number;
  p95_latency_ms: number;
  p99_latency_ms: number;
  min_latency_ms: number;
  max_latency_ms: number;
  avg_latency_ms: number;
  frames_sent: number;
  frames_received: number;
  messages_per_sec: number;
  bytes_per_sec: number;
}
