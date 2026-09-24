import React, { useState } from 'react';
import { BenchmarkReport } from '../types';
import { Gauge, Play, CheckCircle2, TrendingUp, Cpu } from 'lucide-react';

export const StressRunnerPanel: React.FC = () => {
  const [url, setUrl] = useState('ws://127.0.0.1:8080/ws/feed');
  const [clients, setClients] = useState(100);
  const [duration, setDuration] = useState(10);
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<BenchmarkReport | null>(null);

  const startBench = async () => {
    setLoading(true);
    setReport(null);

    try {
      await fetch('/api/v1/bench/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          url,
          clients,
          duration: duration * 1000000000,
          ramp_up: 2000000000,
        }),
      });

      const poller = setInterval(async () => {
        const res = await fetch('/api/v1/bench/status');
        const data = await res.json();
        if (data.status === 'complete' && data.report) {
          clearInterval(poller);
          setReport(data.report);
          setLoading(false);
        }
      }, 1000);
    } catch (err) {
      setLoading(false);
    }
  };

  return (
    <div className="p-4 flex flex-col gap-4 text-xs overflow-y-auto">
      <div className="flex items-center justify-between border-b border-slate-800 pb-3">
        <div className="flex items-center gap-2">
          <Gauge className="w-4 h-4 text-cyan-400" />
          <span className="font-bold text-slate-100 text-sm">Concurrency Stress Benchmarker</span>
        </div>
      </div>

      <div className="flex flex-col gap-3">
        <div>
          <label className="block text-slate-400 font-semibold mb-1 text-[11px]">Target URL</label>
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className="w-full bg-slate-950 border border-slate-700/80 rounded-lg p-2 text-slate-200 font-mono outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-slate-400 font-semibold mb-1 text-[11px]">
              Concurrent Clients
            </label>
            <input
              type="number"
              value={clients}
              onChange={(e) => setClients(parseInt(e.target.value) || 50)}
              className="w-full bg-slate-950 border border-slate-700/80 rounded-lg p-2 text-slate-200 font-mono outline-none"
            />
          </div>

          <div>
            <label className="block text-slate-400 font-semibold mb-1 text-[11px]">
              Duration (Seconds)
            </label>
            <input
              type="number"
              value={duration}
              onChange={(e) => setDuration(parseInt(e.target.value) || 10)}
              className="w-full bg-slate-950 border border-slate-700/80 rounded-lg p-2 text-slate-200 font-mono outline-none"
            />
          </div>
        </div>

        <button
          onClick={startBench}
          disabled={loading}
          className="w-full py-2.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white font-bold flex items-center justify-center gap-2 shadow-lg shadow-cyan-900/20 transition"
        >
          {loading ? (
            <div className="flex items-center gap-2">
              <span className="w-3.5 h-3.5 border-2 border-white/20 border-t-white rounded-full animate-spin"></span>
              Benchmarking in Progress...
            </div>
          ) : (
            <>
              <Play className="w-3.5 h-3.5 fill-current" />
              Launch Concurrency Test
            </>
          )}
        </button>
      </div>

      {report && (
        <div className="mt-2 p-3.5 bg-slate-950 border border-slate-800 rounded-lg flex flex-col gap-3 font-mono text-[11px]">
          <div className="flex items-center justify-between text-emerald-400 font-bold border-b border-slate-800/80 pb-2">
            <span className="flex items-center gap-1.5">
              <CheckCircle2 className="w-4 h-4" /> Benchmark Complete
            </span>
            <span>{report.peak_clients} / {report.target_clients} connected</span>
          </div>

          <div className="grid grid-cols-2 gap-2 text-slate-300">
            <div>
              <span className="text-slate-500">p50 Latency:</span>{' '}
              <b className="text-slate-100">{report.p50_latency_ms.toFixed(2)} ms</b>
            </div>
            <div>
              <span className="text-slate-500">p95 Latency:</span>{' '}
              <b className="text-amber-400">{report.p95_latency_ms.toFixed(2)} ms</b>
            </div>
            <div>
              <span className="text-slate-500">p99 Latency:</span>{' '}
              <b className="text-rose-400">{report.p99_latency_ms.toFixed(2)} ms</b>
            </div>
            <div>
              <span className="text-slate-500">Avg Latency:</span>{' '}
              <b className="text-slate-100">{report.avg_latency_ms.toFixed(2)} ms</b>
            </div>
            <div>
              <span className="text-slate-500">Message Rate:</span>{' '}
              <b className="text-cyan-400">{report.messages_per_sec.toFixed(1)} msgs/s</b>
            </div>
            <div>
              <span className="text-slate-500">Bandwidth:</span>{' '}
              <b className="text-slate-100">{(report.bytes_per_sec / 1024).toFixed(1)} KB/s</b>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
