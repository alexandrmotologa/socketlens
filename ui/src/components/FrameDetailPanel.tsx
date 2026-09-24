import React, { useState, useEffect } from 'react';
import { Frame, DiffResult } from '../types';
import { Copy, Check, Send, Code, Terminal, GitCompare, AlertTriangle, Plus, Minus, Edit3 } from 'lucide-react';

interface FrameDetailPanelProps {
  frame: Frame | null;
  previousFrame?: Frame | null;
  onResend?: (payload: string) => void;
}

export const FrameDetailPanel: React.FC<FrameDetailPanelProps> = ({
  frame,
  previousFrame,
  onResend,
}) => {
  const [viewMode, setViewMode] = useState<'formatted' | 'raw' | 'diff'>('formatted');
  const [copied, setCopied] = useState(false);
  const [diffResult, setDiffResult] = useState<DiffResult | null>(null);

  const payload = frame?.decoded || frame?.payload_str || '';
  const prevPayload = previousFrame?.decoded || previousFrame?.payload_str || '';

  useEffect(() => {
    if (viewMode === 'diff' && frame && previousFrame) {
      fetch('/api/v1/codec/diff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          old_payload: prevPayload,
          new_payload: payload,
        }),
      })
        .then((r) => r.json())
        .then((data) => setDiffResult(data))
        .catch(() => setDiffResult(null));
    }
  }, [viewMode, frame, previousFrame, payload, prevPayload]);

  if (!frame) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-slate-500 text-xs p-6 text-center">
        <Code className="w-8 h-8 mb-2 opacity-40 text-slate-400" />
        Select a streaming frame from the timeline to inspect its decoded contents, schema status, and diff.
      </div>
    );
  }

  const handleCopy = () => {
    navigator.clipboard.writeText(payload);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="h-full flex flex-col bg-slate-950/60 border-l border-slate-800/80">
      {/* Detail Header */}
      <div className="h-12 border-b border-slate-800/80 px-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xs font-bold text-slate-200">Frame #{frame.sequence}</span>
          <span
            className={`text-[10px] font-bold px-1.5 py-0.5 rounded uppercase ${
              frame.direction === 'inbound'
                ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                : 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/20'
            }`}
          >
            {frame.direction}
          </span>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={handleCopy}
            className="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded transition"
            title="Copy payload"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
          </button>

          {onResend && (
            <button
              onClick={() => onResend(payload)}
              className="p-1.5 text-slate-400 hover:text-blue-400 hover:bg-slate-800 rounded transition"
              title="Resend this payload"
            >
              <Send className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Schema Errors Alert Banner */}
      {frame.schema_errors && frame.schema_errors.length > 0 && (
        <div className="bg-rose-950/40 border-b border-rose-800/50 p-2.5 space-y-1">
          <div className="flex items-center gap-1.5 text-rose-400 font-bold text-[11px]">
            <AlertTriangle className="w-3.5 h-3.5" />
            <span>Schema Violations ({frame.schema_errors.length})</span>
          </div>
          <ul className="list-disc list-inside space-y-0.5 text-rose-300 font-mono text-[10px]">
            {frame.schema_errors.map((err, idx) => (
              <li key={idx}>{err}</li>
            ))}
          </ul>
        </div>
      )}

      {/* Metadata Overview */}
      <div className="p-3 bg-slate-900/40 border-b border-slate-800/80 text-[11px] font-mono grid grid-cols-2 gap-2 text-slate-400">
        <div>
          <span className="text-slate-500">Opcode:</span> <b className="text-slate-200">{frame.opcode}</b>
        </div>
        <div>
          <span className="text-slate-500">Format:</span> <b className="text-blue-400">{frame.format || 'text'}</b>
        </div>
        <div>
          <span className="text-slate-500">Length:</span> <b className="text-slate-200">{frame.length} bytes</b>
        </div>
        <div>
          <span className="text-slate-500">Protocol:</span> <b className="text-slate-200">{frame.protocol.toUpperCase()}</b>
        </div>
        {frame.event_name && (
          <div className="col-span-2">
            <span className="text-slate-500">Event Name:</span> <b className="text-amber-400">{frame.event_name}</b>
          </div>
        )}
      </div>

      {/* Body View Mode Tabs */}
      <div className="flex border-b border-slate-800/80 bg-slate-900/30 px-3">
        <button
          onClick={() => setViewMode('formatted')}
          className={`px-3 py-1.5 text-xs font-semibold border-b-2 flex items-center gap-1.5 transition ${
            viewMode === 'formatted'
              ? 'border-blue-500 text-slate-100'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Code className="w-3 h-3" />
          Formatted
        </button>

        <button
          onClick={() => setViewMode('raw')}
          className={`px-3 py-1.5 text-xs font-semibold border-b-2 flex items-center gap-1.5 transition ${
            viewMode === 'raw'
              ? 'border-blue-500 text-slate-100'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <Terminal className="w-3 h-3" />
          Raw String
        </button>

        <button
          onClick={() => setViewMode('diff')}
          disabled={!previousFrame}
          className={`px-3 py-1.5 text-xs font-semibold border-b-2 flex items-center gap-1.5 transition disabled:opacity-40 ${
            viewMode === 'diff'
              ? 'border-purple-500 text-purple-300'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
          title={previousFrame ? 'Diff with previous frame in sequence' : 'Select a frame that has a predecessor'}
        >
          <GitCompare className="w-3 h-3" />
          Diff
        </button>
      </div>

      {/* Payload Display or Diff Viewer */}
      <div className="flex-1 overflow-auto p-4 bg-slate-950 font-mono text-xs">
        {viewMode === 'diff' ? (
          <div className="space-y-3">
            <div className="text-slate-400 text-[11px] pb-2 border-b border-slate-800">
              Diff against previous frame #{previousFrame?.sequence}:
            </div>

            {!diffResult ? (
              <div className="text-slate-500 italic">Computing differences...</div>
            ) : !diffResult.has_changes ? (
              <div className="text-emerald-400 text-xs">Identical payload structure (No changes detected).</div>
            ) : (
              <div className="space-y-1.5">
                {diffResult.changes.map((c, i) => (
                  <div
                    key={i}
                    className={`p-2 rounded border text-[11px] flex items-start gap-2 ${
                      c.kind === 'added'
                        ? 'bg-emerald-950/30 border-emerald-800/40 text-emerald-300'
                        : c.kind === 'removed'
                        ? 'bg-rose-950/30 border-rose-800/40 text-rose-300'
                        : 'bg-amber-950/30 border-amber-800/40 text-amber-300'
                    }`}
                  >
                    {c.kind === 'added' && <Plus className="w-3.5 h-3.5 flex-shrink-0 text-emerald-400 mt-0.5" />}
                    {c.kind === 'removed' && <Minus className="w-3.5 h-3.5 flex-shrink-0 text-rose-400 mt-0.5" />}
                    {c.kind === 'modified' && <Edit3 className="w-3.5 h-3.5 flex-shrink-0 text-amber-400 mt-0.5" />}

                    <div className="overflow-hidden">
                      <div className="font-bold">{c.path}</div>
                      {c.kind === 'modified' ? (
                        <div className="text-[10px] space-y-0.5">
                          <span className="line-through text-rose-400 mr-2">{JSON.stringify(c.old_value)}</span>
                          <span className="text-emerald-400">{JSON.stringify(c.new_value)}</span>
                        </div>
                      ) : (
                        <div className="text-[10px]">
                          {JSON.stringify(c.kind === 'added' ? c.new_value : c.old_value)}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : (
          <pre className="text-slate-200 whitespace-pre-wrap break-all leading-relaxed">
            {viewMode === 'formatted' ? payload : (frame.payload_str || payload)}
          </pre>
        )}
      </div>
    </div>
  );
};
