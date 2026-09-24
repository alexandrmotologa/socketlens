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
  schema_errors?: string[];
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

export interface BreakpointItem {
  id: string;
  direction: Direction;
  opcode: OpCode;
  payload: string;
  timestamp: string;
}

export interface ProxyStatus {
  running: boolean;
  local_port?: number;
  target_url?: string;
  breakpoints_enabled?: boolean;
  pending_breakpoints?: BreakpointItem[];
}

export interface ToolCall {
  id: string;
  name: string;
  arguments: string;
}

export interface LLMReport {
  model: string;
  started_at: string;
  first_token_at?: string;
  finished_at?: string;
  ttft_ms: number;
  total_tokens: number;
  tokens_per_sec: number;
  avg_jitter_ms: number;
  reconstructed_text: string;
  tool_calls?: ToolCall[];
  is_finished: boolean;
}

export interface AutomationRule {
  id?: string;
  name: string;
  enabled?: boolean;
  condition: 'contains' | 'exact' | 'event_name' | 'opcode';
  pattern: string;
  action: 'reply' | 'drop' | 'alert';
  response?: string;
}

export interface SchemaViolation {
  property: string;
  expected: string;
  actual: string;
  message: string;
}

export interface DiffChange {
  path: string;
  kind: 'added' | 'removed' | 'modified';
  old_value?: any;
  new_value?: any;
}

export interface DiffResult {
  has_changes: boolean;
  changes: DiffChange[];
}

