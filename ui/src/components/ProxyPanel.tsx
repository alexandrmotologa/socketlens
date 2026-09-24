import React, { useState, useEffect } from 'react';
import { ProxyStatus, BreakpointItem } from '../types';
import { Play, Square, PauseCircle, CheckCircle2, Trash2, ArrowUpRight, ArrowDownLeft, ShieldAlert } from 'lucide-react';

export const ProxyPanel: React.FC = () => {
  const [targetUrl, setTargetUrl] = useState('wss://echo.websocket.events');
  const [localPort, setLocalPort] = useState(8081);
  const [breakpointsEnabled, setBreakpointsEnabled] = useState(true);
  const [status, setStatus] = useState<ProxyStatus>({ running: false });
  const [editedPayloads, setEditedPayloads] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  const fetchStatus = async () => {
    try {
      const res = await fetch('/api/v1/proxy/status');
      if (res.ok) {
        const data: ProxyStatus = await res.json();
        setStatus(data);
        // Initialize edited payloads for newly intercepted breakpoints
        if (data.pending_breakpoints) {
          setEditedPayloads((prev) => {
            const next = { ...prev };
            data.pending_breakpoints?.forEach((bp) => {
              if (next[bp.id] === undefined) {
                next[bp.id] = bp.payload;
              }
            });
            return next;
          });
        }
      }
    } catch (err) {}
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 1000);
    return () => clearInterval(interval);
  }, []);

  const handleStart = async () => {
    setLoading(true);
    try {
      await fetch('/api/v1/proxy/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          local_port: Number(localPort),
          target_url: targetUrl,
          enable_breakpoint: breakpointsEnabled,
        }),
      });
      await fetchStatus();
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    setLoading(true);
    try {
      await fetch('/api/v1/proxy/stop', { method: 'POST' });
      await fetchStatus();
    } finally {
      setLoading(false);
    }
  };

  const handleResume = async (id: string, drop: boolean) => {
    const payload = editedPayloads[id] || '';
    try {
      await fetch(`/api/v1/proxy/breakpoints/${id}/resume`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ payload, drop }),
      });
      setEditedPayloads((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
      await fetchStatus();
    } catch (err) {}
  };

  return (
    <div className="flex flex-col h-full bg-slate-900/60 overflow-y-auto p-4 space-y-4 text-xs">
      {/* Proxy Configuration Card */}
      <div className="bg-slate-950/80 border border-slate-800 rounded-lg p-4 space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-purple-400 font-bold text-sm tracking-wide">
              Interception Proxy & Tamper
            </span>
            <span
              className={`px-2 py-0.5 rounded text-[10px] font-semibold uppercase ${
                status.running
                  ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                  : 'bg-slate-800 text-slate-400'
              }`}
            >
              {status.running ? 'Active Relay' : 'Stopped'}
            </span>
          </div>

          {status.running ? (
            <button
              onClick={handleStop}
              disabled={loading}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-rose-600/20 hover:bg-rose-600/30 text-rose-400 border border-rose-500/30 rounded font-medium transition"
            >
              <Square className="w-3.5 h-3.5" />
              Stop Proxy
            </button>
          ) : (
            <button
              onClick={handleStart}
              disabled={loading || !targetUrl}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-purple-600 hover:bg-purple-500 text-white rounded font-medium transition shadow-lg shadow-purple-900/20"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              Start Proxy
            </button>
          )}
        </div>

        <p className="text-slate-400 text-[11px] leading-relaxed">
          Transparent bidirectional proxy that intercepts frames in flight, allowing you to pause, inspect, modify, or drop payloads before forwarding.
        </p>

        <div className="grid grid-cols-3 gap-2">
          <div className="col-span-2 space-y-1">
            <label className="text-slate-400 font-medium">Upstream Target URL</label>
            <input
              type="text"
              value={targetUrl}
              onChange={(e) => setTargetUrl(e.target.value)}
              disabled={status.running}
              placeholder="wss://api.example.com/stream"
              className="w-full bg-slate-900 border border-slate-700/80 rounded px-2.5 py-1.5 text-slate-200 font-mono focus:outline-none focus:border-purple-500 disabled:opacity-50"
            />
          </div>

          <div className="space-y-1">
            <label className="text-slate-400 font-medium">Local Port</label>
            <input
              type="number"
              value={localPort}
              onChange={(e) => setLocalPort(Number(e.target.value))}
              disabled={status.running}
              className="w-full bg-slate-900 border border-slate-700/80 rounded px-2.5 py-1.5 text-slate-200 font-mono focus:outline-none focus:border-purple-500 disabled:opacity-50"
            />
          </div>
        </div>

        <label className="flex items-center gap-2 pt-1 cursor-pointer">
          <input
            type="checkbox"
            checked={breakpointsEnabled}
            onChange={(e) => setBreakpointsEnabled(e.target.checked)}
            disabled={status.running}
            className="rounded border-slate-700 text-purple-600 focus:ring-0 bg-slate-900"
          />
          <span className="text-slate-300">
            Pause frames at breakpoint for live payload editing
          </span>
        </label>

        {status.running && (
          <div className="bg-purple-950/30 border border-purple-900/40 rounded p-2.5 text-[11px] text-purple-300 flex items-center justify-between">
            <span>Connect client apps to:</span>
            <code className="font-mono bg-purple-900/40 px-2 py-0.5 rounded text-purple-200">
              ws://127.0.0.1:{status.local_port || localPort}
            </code>
          </div>
        )}
      </div>

      {/* Pending Intercepted Breakpoints */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="font-bold text-slate-300 flex items-center gap-1.5">
            <PauseCircle className="w-3.5 h-3.5 text-amber-400" />
            Held Frames ({status.pending_breakpoints?.length || 0})
          </span>
          {status.pending_breakpoints && status.pending_breakpoints.length > 0 && (
            <span className="text-[10px] text-amber-400 animate-pulse">
              Stream paused on breakpoint
            </span>
          )}
        </div>

        {(!status.pending_breakpoints || status.pending_breakpoints.length === 0) ? (
          <div className="p-8 text-center border border-dashed border-slate-800 rounded-lg text-slate-500 text-xs">
            No frames currently suspended. When frames arrive, they will pause here for inspection and tampering.
          </div>
        ) : (
          <div className="space-y-3">
            {status.pending_breakpoints.map((bp) => (
              <div
                key={bp.id}
                className="bg-slate-950 border border-amber-500/40 rounded-lg p-3 space-y-2.5 shadow-lg shadow-amber-950/20"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span
                      className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-bold ${
                        bp.direction === 'outbound'
                          ? 'bg-blue-500/20 text-blue-400'
                          : 'bg-emerald-500/20 text-emerald-400'
                      }`}
                    >
                      {bp.direction === 'outbound' ? (
                        <ArrowUpRight className="w-3 h-3" />
                      ) : (
                        <ArrowDownLeft className="w-3 h-3" />
                      )}
                      {bp.direction.toUpperCase()}
                    </span>
                    <span className="font-mono text-slate-400 text-[10px]">{bp.id}</span>
                  </div>
                  <span className="text-slate-500 text-[10px] font-mono">
                    {new Date(bp.timestamp).toLocaleTimeString()}
                  </span>
                </div>

                <div className="space-y-1">
                  <label className="text-[10px] text-slate-400 font-semibold uppercase">
                    Tamper Payload Before Forwarding
                  </label>
                  <textarea
                    rows={4}
                    value={editedPayloads[bp.id] !== undefined ? editedPayloads[bp.id] : bp.payload}
                    onChange={(e) =>
                      setEditedPayloads({ ...editedPayloads, [bp.id]: e.target.value })
                    }
                    className="w-full bg-slate-900 border border-slate-700 rounded p-2 text-slate-200 font-mono text-[11px] focus:outline-none focus:border-amber-400 resize-y"
                  />
                </div>

                <div className="flex items-center justify-end gap-2 pt-1">
                  <button
                    onClick={() => handleResume(bp.id, true)}
                    className="flex items-center gap-1 px-2.5 py-1 text-rose-400 hover:bg-rose-500/10 rounded transition font-medium"
                  >
                    <Trash2 className="w-3 h-3" />
                    Drop Frame
                  </button>
                  <button
                    onClick={() => handleResume(bp.id, false)}
                    className="flex items-center gap-1 px-3 py-1 bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold rounded transition shadow"
                  >
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    Forward Modified
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
