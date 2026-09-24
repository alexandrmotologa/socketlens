import React, { useState } from 'react';
import { ConnectionConfig, Protocol } from '../types';
import { Play, Square, Settings2, ShieldAlert } from 'lucide-react';

interface ConnectBarProps {
  onConnect: (cfg: ConnectionConfig) => void;
  onDisconnect: () => void;
  isConnected: boolean;
  isConnecting: boolean;
}

export const ConnectBar: React.FC<ConnectBarProps> = ({
  onConnect,
  onDisconnect,
  isConnected,
  isConnecting,
}) => {
  const [protocol, setProtocol] = useState<Protocol>('ws');
  const [url, setUrl] = useState('wss://echo.websocket.events');
  const [headersJson, setHeadersJson] = useState('');
  const [showOptions, setShowOptions] = useState(false);
  const [tlsInsecure, setTlsInsecure] = useState(false);
  const [autoReconnect, setAutoReconnect] = useState(true);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isConnected || isConnecting) {
      onDisconnect();
      return;
    }

    let parsedHeaders: Record<string, string> = {};
    if (headersJson.trim()) {
      try {
        parsedHeaders = JSON.parse(headersJson);
      } catch (err) {
        alert('Invalid JSON in headers field');
        return;
      }
    }

    onConnect({
      url,
      protocol,
      headers: parsedHeaders,
      tls_insecure: tlsInsecure,
      auto_reconnect: autoReconnect,
      heartbeat_interval: 15,
    });
  };

  const handleProtocolChange = (p: Protocol) => {
    setProtocol(p);
    if (p === 'sse' && url.startsWith('ws')) {
      setUrl('http://localhost:8080/ws/feed/sse');
    } else if (p === 'ws' && url.startsWith('http')) {
      setUrl('wss://echo.websocket.events');
    }
  };

  return (
    <div className="bg-slate-900/60 p-4 border-b border-slate-800/80">
      <form onSubmit={handleSubmit} className="flex flex-col gap-3">
        <div className="flex gap-2">
          <select
            value={protocol}
            onChange={(e) => handleProtocolChange(e.target.value as Protocol)}
            className="w-36 bg-slate-950 border border-slate-700/80 rounded-lg px-3 py-2 text-xs font-semibold text-slate-200 outline-none focus:border-blue-500"
          >
            <option value="ws">WebSocket (WS)</option>
            <option value="sse">Server-Sent Events</option>
            <option value="socketio">Socket.io v4</option>
          </select>

          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="Target stream URL (e.g. wss://echo.websocket.events)"
            className="flex-1 bg-slate-950 border border-slate-700/80 rounded-lg px-3 py-2 text-xs font-mono text-slate-100 placeholder-slate-500 outline-none focus:border-blue-500"
          />

          <button
            type="button"
            onClick={() => setShowOptions(!showOptions)}
            className={`p-2 rounded-lg border text-xs font-medium flex items-center gap-1 transition ${
              showOptions
                ? 'bg-slate-800 border-blue-500/50 text-blue-400'
                : 'bg-slate-950 border-slate-700/80 text-slate-400 hover:text-slate-200'
            }`}
          >
            <Settings2 className="w-4 h-4" />
          </button>

          <button
            type="submit"
            className={`px-5 py-2 rounded-lg text-xs font-bold flex items-center gap-2 shadow-lg transition ${
              isConnected || isConnecting
                ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-900/20'
                : 'bg-blue-600 hover:bg-blue-500 text-white shadow-blue-900/20'
            }`}
          >
            {isConnected || isConnecting ? (
              <>
                <Square className="w-3.5 h-3.5 fill-current" />
                Disconnect
              </>
            ) : (
              <>
                <Play className="w-3.5 h-3.5 fill-current" />
                Connect
              </>
            )}
          </button>
        </div>

        {showOptions && (
          <div className="p-3 bg-slate-950/80 border border-slate-800 rounded-lg flex flex-col gap-2.5 text-xs">
            <div className="flex gap-4 items-center">
              <label className="flex items-center gap-2 text-slate-300 cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoReconnect}
                  onChange={(e) => setAutoReconnect(e.target.checked)}
                  className="rounded border-slate-700 bg-slate-900 text-blue-600 focus:ring-0"
                />
                Auto-reconnect on disconnect
              </label>

              <label className="flex items-center gap-2 text-slate-300 cursor-pointer">
                <input
                  type="checkbox"
                  checked={tlsInsecure}
                  onChange={(e) => setTlsInsecure(e.target.checked)}
                  className="rounded border-slate-700 bg-slate-900 text-blue-600 focus:ring-0"
                />
                <span className="flex items-center gap-1 text-amber-400">
                  <ShieldAlert className="w-3.5 h-3.5" /> Skip TLS verification
                </span>
              </label>
            </div>

            <div>
              <span className="block text-[11px] font-semibold text-slate-400 mb-1">
                Custom Headers (JSON format)
              </span>
              <textarea
                value={headersJson}
                onChange={(e) => setHeadersJson(e.target.value)}
                placeholder='{"Authorization": "Bearer token123", "X-Custom-Client": "SocketLens"}'
                rows={2}
                className="w-full bg-slate-900 border border-slate-800 rounded-md p-2 font-mono text-[11px] text-slate-200 outline-none focus:border-blue-500"
              />
            </div>
          </div>
        )}
      </form>
    </div>
  );
};
