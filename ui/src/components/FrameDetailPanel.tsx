import React, { useState } from 'react';
import { Frame } from '../types';
import { Copy, Check, Send, Code, Terminal, FileText } from 'lucide-react';

interface FrameDetailPanelProps {
  frame: Frame | null;
  onResend?: (payload: string) => void;
}

export const FrameDetailPanel: React.FC<FrameDetailPanelProps> = ({ frame, onResend }) => {
  const [viewMode, setViewMode] = useState<'formatted' | 'raw'>('formatted');
  const [copied, setCopied] = useState(false);

  if (!frame) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-slate-500 text-xs p-6 text-center">
        <Code className="w-8 h-8 mb-2 opacity-40 text-slate-400" />
        Select a streaming frame from the timeline to inspect its decoded contents and headers.
      </div>
    );
  }

  const payload = frame.decoded || frame.payload_str || '';

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

      {/* Metadata Overview */}
      <div className="p-3.5 bg-slate-900/40 border-b border-slate-800/80 text-[11px] font-mono grid grid-cols-2 gap-2 text-slate-400">
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
      </div>

      {/* Payload Display */}
      <div className="flex-1 overflow-auto p-4 bg-slate-950 font-mono text-xs">
        <pre className="text-slate-200 whitespace-pre-wrap break-all leading-relaxed">
          {viewMode === 'formatted' ? payload : (frame.payload_str || payload)}
        </pre>
      </div>
    </div>
  );
};
