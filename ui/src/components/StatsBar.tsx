import React, { useState, useEffect } from 'react';
import { ConnectionState } from '../types';
import { Activity, ArrowDownLeft, ArrowUpRight, HardDrive, Download } from 'lucide-react';

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
  const [sparklineData, setSparklineData] = useState<number[]>([0, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
  const [lastTotalFrames, setLastTotalFrames] = useState(framesIn + framesOut);

  // Update sparkline points every 1 second
  useEffect(() => {
    const interval = setInterval(() => {
      const currentTotal = framesIn + framesOut;
      const delta = Math.max(0, currentTotal - lastTotalFrames);
      setLastTotalFrames(currentTotal);
      setSparklineData((prev) => [...prev.slice(1), delta]);
    }, 1000);
    return () => clearInterval(interval);
  }, [framesIn, framesOut, lastTotalFrames]);

  const maxVal = Math.max(...sparklineData, 5);

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

  const handleDownloadHAR = () => {
    window.location.href = '/api/v1/export/har';
  };

  return (
    <header className="h-14 bg-slate-900/90 backdrop-blur-md border-b border-slate-800/80 px-5 flex items-center justify-between z-20">
      <div className="flex items-center gap-3">
        <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-blue-600 via-indigo-600 to-cyan-500 flex items-center justify-center font-black text-white text-base shadow-lg shadow-blue-500/20">
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
        {/* Real-time Sparkline Throughput Chart */}
        <div className="flex items-center gap-2 px-3 py-1 rounded-lg bg-slate-800/40 border border-slate-700/40" title="Stream Pulse (Msg/sec)">
          <span className="text-slate-400 text-[10px] uppercase font-semibold">Activity</span>
          <svg className="w-20 h-6 overflow-visible">
            <polyline
              fill="none"
              stroke="#38bdf8"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              points={sparklineData
                .map((val, idx) => {
                  const x = (idx / (sparklineData.length - 1)) * 80;
                  const y = 24 - (val / maxVal) * 20 - 2;
                  return `${x},${y}`;
                })
                .join(' ')}
            />
          </svg>
        </div>

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

        {/* HAR Export Button */}
        <button
          onClick={handleDownloadHAR}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 border border-slate-700 hover:border-slate-600 rounded-lg text-slate-200 transition font-sans font-medium shadow"
          title="Export current session frames as Chrome DevTools HAR 1.2"
        >
          <Download className="w-3.5 h-3.5 text-blue-400" />
          <span>HAR 1.2</span>
        </button>

        {getStatusBadge()}
      </div>
    </header>
  );
};
