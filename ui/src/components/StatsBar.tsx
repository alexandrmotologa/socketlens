import React from 'react';
import { ConnectionState } from '../types';
import { Activity, ArrowDownLeft, ArrowUpRight, HardDrive, Wifi, WifiOff } from 'lucide-react';

interface StatsBarProps {
  state: ConnectionState;
  framesIn: number;
  framesOut: number;
  totalBytes: number;
  heartbeatLatencyMs?: number;
}

export const StatsBar: React.FC<StatsBarProps> = ({
  state,
  framesIn,
  framesOut,
  totalBytes,
  heartbeatLatencyMs,
}) => {
  const getStatusBadge = () => {
    switch (state) {
      case 'connected':
        return (
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs font-semibold">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
            CONNECTED
          </div>
        );
      case 'connecting':
      case 'reconnecting':
        return (
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-amber-500/10 border border-amber-500/30 text-amber-400 text-xs font-semibold">
            <span className="w-2 h-2 rounded-full bg-amber-500 animate-ping"></span>
            {state.toUpperCase()}
          </div>
        );
      default:
        return (
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-rose-500/10 border border-rose-500/30 text-rose-400 text-xs font-semibold">
            <span className="w-2 h-2 rounded-full bg-rose-500"></span>
            DISCONNECTED
          </div>
        );
    }
  };

  return (
    <header className="h-14 bg-slate-900/90 backdrop-blur-md border-b border-slate-800/80 px-5 flex items-center justify-between z-20">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-blue-600 to-cyan-500 flex items-center justify-center font-black text-white text-base shadow-lg shadow-blue-500/20">
          S
        </div>
        <div className="flex items-baseline gap-2">
          <span className="font-extrabold text-slate-100 tracking-tight text-lg">SocketLens</span>
          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 border border-blue-500/30 uppercase tracking-widest">
            Studio
          </span>
        </div>
      </div>

      <div className="flex items-center gap-4 text-xs font-mono">
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-800/50 border border-slate-700/50">
          <ArrowDownLeft className="w-3.5 h-3.5 text-emerald-400" />
          <span className="text-slate-400">IN:</span>
          <span className="text-slate-100 font-semibold">{framesIn.toLocaleString()}</span>
        </div>

        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-800/50 border border-slate-700/50">
          <ArrowUpRight className="w-3.5 h-3.5 text-cyan-400" />
          <span className="text-slate-400">OUT:</span>
          <span className="text-slate-100 font-semibold">{framesOut.toLocaleString()}</span>
        </div>

        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-800/50 border border-slate-700/50">
          <HardDrive className="w-3.5 h-3.5 text-blue-400" />
          <span className="text-slate-400">BANDWIDTH:</span>
          <span className="text-slate-100 font-semibold">{(totalBytes / 1024).toFixed(1)} KB</span>
        </div>

        {heartbeatLatencyMs !== undefined && heartbeatLatencyMs > 0 && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-800/50 border border-slate-700/50">
            <Activity className="w-3.5 h-3.5 text-amber-400" />
            <span className="text-slate-400">PING:</span>
            <span className="text-slate-100 font-semibold">{heartbeatLatencyMs.toFixed(1)} ms</span>
          </div>
        )}

        {getStatusBadge()}
      </div>
    </header>
  );
};
