import React, { useState } from 'react';
import { Server, Play, Square, AlertTriangle, Copy, Check } from 'lucide-react';

interface MockServerPanelProps {
  onStart: (cfg: any) => Promise<any>;
  onStop: () => Promise<void>;
  isRunning: boolean;
  endpoints?: { ws_url: string; sse_url: string };
}

export const MockServerPanel: React.FC<MockServerPanelProps> = ({
  onStart,
  onStop,
  isRunning,
  endpoints,
}) => {
  const [port, setPort] = useState(8080);
  const [route, setRoute] = useState('/ws/feed');
  const [mode, setMode] = useState<'echo' | 'broadcast' | 'llm-tokens' | 'ticks'>('echo');
  const [rate, setRate] = useState(10);
  const [latencyMs, setLatencyMs] = useState(0);
  const [dropRate, setDropRate] = useState(0);
  const [copiedWs, setCopiedWs] = useState(false);

  const handleToggle = async () => {
    if (isRunning) {
      await onStop();
    } else {
      await onStart({
        port,
        route,
        mode,
        rate,
        chaos: {
          latency: latencyMs * 1000000, // convert ms to ns
          drop_rate: dropRate,
        },
      });
    }
  };

  const copyEndpoint = (url: string) => {
    navigator.clipboard.writeText(url);
    setCopiedWs(true);
    setTimeout(() => setCopiedWs(false), 1500);
  };

  return (
    <div className="p-4 flex flex-col gap-4 text-xs">
      <div className="flex items-center justify-between border-b border-slate-800 pb-3">
        <div className="flex items-center gap-2">
          <Server className="w-4 h-4 text-blue-400" />
          <span className="font-bold text-slate-100 text-sm">Embedded Mock Stream Server</span>
        </div>
        <span
          className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
            isRunning
              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
              : 'bg-slate-800 text-slate-400'
          }`}
        >
          {isRunning ? 'RUNNING' : 'STOPPED'}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-slate-400 font-semibold mb-1 text-[11px]">Server Mode</label>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value as any)}
            className="w-full bg-slate-950 border border-slate-700/80 rounded-lg p-2 text-slate-200 outline-none"
          >
            <option value="echo">Echo Mode (Client Mirror)</option>
            <option value="broadcast">Broadcast (Pub/Sub Hub)</option>
            <option value="llm-tokens">LLM Stream (Simulated Tokens)</option>
            <option value="ticks">Financial Ticker Generator</option>
          </select>
        </div>

        <div>
          <label className="block text-slate-400 font-semibold mb-1 text-[11px]">Port</label>
          <input
            type="number"
            value={port}
            onChange={(e) => setPort(parseInt(e.target.value) || 8080)}
            className="w-full bg-slate-950 border border-slate-700/80 rounded-lg p-2 text-slate-200 outline-none font-mono"
          />
        </div>

        <div>
          <label className="block text-slate-400 font-semibold mb-1 text-[11px]">Route Prefix</label>
          <input
            type="text"
            value={route}
            onChange={(e) => setRoute(e.target.value)}
            className="w-full bg-slate-950 border border-slate-700/80 rounded-lg p-2 text-slate-200 outline-none font-mono"
          />
        </div>

        <div>
          <label className="block text-slate-400 font-semibold mb-1 text-[11px]">Rate (Events / Sec)</label>
          <input
            type="number"
            value={rate}
            onChange={(e) => setRate(parseInt(e.target.value) || 10)}
            className="w-full bg-slate-950 border border-slate-700/80 rounded-lg p-2 text-slate-200 outline-none font-mono"
          />
        </div>
      </div>

      {/* Chaos dials */}
      <div className="p-3 bg-amber-500/5 border border-amber-500/20 rounded-lg flex flex-col gap-2.5">
        <div className="flex items-center gap-1.5 text-amber-400 font-bold text-[11px]">
          <AlertTriangle className="w-3.5 h-3.5" />
          Chaos Injection Dials
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-slate-400 text-[10px] mb-1">Latency Delay (ms)</label>
            <input
              type="number"
              value={latencyMs}
              onChange={(e) => setLatencyMs(parseInt(e.target.value) || 0)}
              className="w-full bg-slate-950 border border-slate-800 rounded p-1.5 text-slate-200 font-mono text-[11px] outline-none"
            />
          </div>

          <div>
            <label className="block text-slate-400 text-[10px] mb-1">Packet Drop Rate (%)</label>
            <input
              type="number"
              value={dropRate}
              min="0"
              max="100"
              onChange={(e) => setDropRate(parseInt(e.target.value) || 0)}
              className="w-full bg-slate-950 border border-slate-800 rounded p-1.5 text-slate-200 font-mono text-[11px] outline-none"
            />
          </div>
        </div>
      </div>

      <button
        onClick={handleToggle}
        className={`w-full py-2.5 rounded-lg font-bold flex items-center justify-center gap-2 shadow-lg transition ${
          isRunning
            ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-rose-900/20'
            : 'bg-emerald-600 hover:bg-emerald-500 text-white shadow-emerald-900/20'
        }`}
      >
        {isRunning ? (
          <>
            <Square className="w-3.5 h-3.5 fill-current" />
            Stop Mock Server
          </>
        ) : (
          <>
            <Play className="w-3.5 h-3.5 fill-current" />
            Start Mock Server
          </>
        )}
      </button>

      {/* Endpoints display */}
      {isRunning && endpoints && (
        <div className="mt-2 p-3 bg-slate-950 border border-slate-800 rounded-lg flex flex-col gap-2 font-mono text-[11px]">
          <div className="flex items-center justify-between text-slate-300">
            <span>WS: {endpoints.ws_url}</span>
            <button
              onClick={() => copyEndpoint(endpoints.ws_url)}
              className="text-slate-500 hover:text-slate-300 p-1"
            >
              {copiedWs ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
            </button>
          </div>
          <div className="flex items-center justify-between text-slate-300">
            <span>SSE: {endpoints.sse_url}</span>
            <button
              onClick={() => copyEndpoint(endpoints.sse_url)}
              className="text-slate-500 hover:text-slate-300 p-1"
            >
              <Copy className="w-3 h-3" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
