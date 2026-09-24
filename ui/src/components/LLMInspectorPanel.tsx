import React, { useState, useEffect } from 'react';
import { LLMReport } from '../types';
import { Sparkles, RefreshCw, Cpu, Activity, Clock, Wrench, Copy, Check } from 'lucide-react';

export const LLMInspectorPanel: React.FC = () => {
  const [report, setReport] = useState<LLMReport | null>(null);
  const [copiedText, setCopiedText] = useState(false);
  const [copiedToolId, setCopiedToolId] = useState<string | null>(null);

  const fetchReport = async () => {
    try {
      const res = await fetch('/api/v1/llm/report');
      if (res.ok) {
        const data: LLMReport = await res.json();
        setReport(data);
      }
    } catch (err) {}
  };

  useEffect(() => {
    fetchReport();
    const interval = setInterval(fetchReport, 800);
    return () => clearInterval(interval);
  }, []);

  const handleReset = async () => {
    try {
      await fetch('/api/v1/llm/reset', { method: 'POST' });
      await fetchReport();
    } catch (err) {}
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedText(true);
    setTimeout(() => setCopiedText(false), 2000);
  };

  const copyToolArgs = (id: string, args: string) => {
    navigator.clipboard.writeText(args);
    setCopiedToolId(id);
    setTimeout(() => setCopiedToolId(null), 2000);
  };

  return (
    <div className="flex flex-col h-full bg-slate-900/60 overflow-y-auto p-4 space-y-4 text-xs">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Sparkles className="w-4 h-4 text-indigo-400" />
          <span className="font-bold text-slate-200 text-sm">LLM & AI Stream Inspector</span>
        </div>

        <button
          onClick={handleReset}
          className="flex items-center gap-1 px-2.5 py-1 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded transition"
          title="Reset metrics"
        >
          <RefreshCw className="w-3.5 h-3.5" />
          Reset
        </button>
      </div>

      {/* Telemetry Metrics Grid */}
      <div className="grid grid-cols-2 gap-2">
        {/* TTFT Card */}
        <div className="bg-slate-950/80 border border-slate-800/80 rounded-lg p-3 space-y-1">
          <div className="flex items-center justify-between text-slate-400 text-[11px]">
            <span className="flex items-center gap-1">
              <Clock className="w-3 h-3 text-cyan-400" />
              TTFT
            </span>
            <span className="text-[10px]">First Token</span>
          </div>
          <div className="text-xl font-bold font-mono text-cyan-400">
            {report?.ttft_ms ? `${report.ttft_ms.toFixed(1)} ms` : '--'}
          </div>
        </div>

        {/* TPS Card */}
        <div className="bg-slate-950/80 border border-slate-800/80 rounded-lg p-3 space-y-1">
          <div className="flex items-center justify-between text-slate-400 text-[11px]">
            <span className="flex items-center gap-1">
              <Activity className="w-3 h-3 text-emerald-400" />
              Throughput
            </span>
            <span className="text-[10px]">Tokens/Sec</span>
          </div>
          <div className="text-xl font-bold font-mono text-emerald-400">
            {report?.tokens_per_sec ? `${report.tokens_per_sec.toFixed(1)} tok/s` : '--'}
          </div>
        </div>

        {/* Total Tokens Card */}
        <div className="bg-slate-950/80 border border-slate-800/80 rounded-lg p-3 space-y-1">
          <div className="flex items-center justify-between text-slate-400 text-[11px]">
            <span className="flex items-center gap-1">
              <Cpu className="w-3 h-3 text-purple-400" />
              Total Tokens
            </span>
            <span className="text-[10px]">Assembled</span>
          </div>
          <div className="text-xl font-bold font-mono text-purple-400">
            {report?.total_tokens || 0}
          </div>
        </div>

        {/* Avg Jitter Card */}
        <div className="bg-slate-950/80 border border-slate-800/80 rounded-lg p-3 space-y-1">
          <div className="flex items-center justify-between text-slate-400 text-[11px]">
            <span className="flex items-center gap-1">
              <Activity className="w-3 h-3 text-amber-400" />
              Avg Jitter
            </span>
            <span className="text-[10px]">Interval</span>
          </div>
          <div className="text-xl font-bold font-mono text-amber-400">
            {report?.avg_jitter_ms ? `${report.avg_jitter_ms.toFixed(1)} ms` : '--'}
          </div>
        </div>
      </div>

      {/* Model & Status Bar */}
      <div className="bg-slate-950/60 border border-slate-800 rounded-md px-3 py-2 flex items-center justify-between text-[11px]">
        <div className="flex items-center gap-2">
          <span className="text-slate-400">Detected Model:</span>
          <span className="font-mono font-bold text-indigo-300">
            {report?.model || 'Auto-detecting...'}
          </span>
        </div>

        <span
          className={`px-2 py-0.5 rounded text-[10px] font-semibold uppercase ${
            report?.is_finished
              ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
              : report?.total_tokens && report.total_tokens > 0
              ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30 animate-pulse'
              : 'bg-slate-800 text-slate-400'
          }`}
        >
          {report?.is_finished ? 'Complete' : report?.total_tokens ? 'Streaming' : 'Idle'}
        </span>
      </div>

      {/* Live Reconstructed Text */}
      <div className="space-y-1.5 flex-1 flex flex-col min-h-[220px]">
        <div className="flex items-center justify-between">
          <span className="text-slate-400 font-semibold uppercase text-[10px]">
            Reconstructed Response Text
          </span>
          {report?.reconstructed_text && (
            <button
              onClick={() => copyToClipboard(report.reconstructed_text)}
              className="flex items-center gap-1 text-[11px] text-indigo-400 hover:text-indigo-300 transition"
            >
              {copiedText ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
              {copiedText ? 'Copied' : 'Copy Text'}
            </button>
          )}
        </div>

        <div className="flex-1 bg-slate-950 border border-slate-800 rounded-lg p-3 font-mono text-slate-200 text-xs leading-relaxed overflow-y-auto whitespace-pre-wrap select-text">
          {report?.reconstructed_text ? (
            report.reconstructed_text
          ) : (
            <span className="text-slate-600 italic">
              Awaiting SSE completion tokens (e.g. OpenAI chat.completion.chunk or Anthropic stream)...
            </span>
          )}
        </div>
      </div>

      {/* Tool Calls Section */}
      {report?.tool_calls && report.tool_calls.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-1.5 text-amber-400 font-semibold">
            <Wrench className="w-3.5 h-3.5" />
            <span>Extracted Tool Calls ({report.tool_calls.length})</span>
          </div>

          <div className="space-y-2">
            {report.tool_calls.map((tc, idx) => (
              <div
                key={tc.id || idx}
                className="bg-slate-950 border border-slate-800 rounded-lg p-2.5 space-y-1.5"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="font-mono font-bold text-amber-300 text-xs">
                      {tc.name}()
                    </span>
                    {tc.id && (
                      <span className="text-[10px] text-slate-500 font-mono">#{tc.id}</span>
                    )}
                  </div>
                  <button
                    onClick={() => copyToolArgs(tc.id || String(idx), tc.arguments)}
                    className="flex items-center gap-1 text-[10px] text-slate-400 hover:text-slate-200"
                  >
                    {copiedToolId === (tc.id || String(idx)) ? (
                      <Check className="w-3 h-3 text-emerald-400" />
                    ) : (
                      <Copy className="w-3 h-3" />
                    )}
                    Copy Args
                  </button>
                </div>
                <pre className="bg-slate-900 border border-slate-800/80 rounded p-2 text-[11px] font-mono text-slate-300 overflow-x-auto">
                  {tc.arguments || '{}'}
                </pre>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
