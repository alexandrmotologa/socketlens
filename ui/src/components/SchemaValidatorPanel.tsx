import React, { useState } from 'react';
import { SchemaViolation } from '../types';
import { ShieldCheck, CheckCircle2, AlertTriangle, FileCode2, Play, Trash2 } from 'lucide-react';

const DEFAULT_SCHEMA = JSON.stringify(
  {
    type: 'object',
    required: ['event', 'data'],
    properties: {
      event: { type: 'string' },
      data: {
        type: 'object',
        required: ['id'],
        properties: {
          id: { type: 'string' },
          price: { type: 'number' },
        },
      },
    },
  },
  null,
  2
);

export const SchemaValidatorPanel: React.FC = () => {
  const [schemaText, setSchemaText] = useState(DEFAULT_SCHEMA);
  const [isActive, setIsActive] = useState(false);
  const [testPayload, setTestPayload] = useState('{"event": "trade", "data": {"id": "btc-101", "price": 64200}}');
  const [violations, setViolations] = useState<SchemaViolation[] | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const handleApply = async () => {
    try {
      const res = await fetch('/api/v1/schema/set', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: schemaText,
      });
      if (res.ok) {
        setIsActive(true);
        setMessage('JSON Schema active and enforcing on all incoming stream frames!');
        setTimeout(() => setMessage(null), 3500);
      } else {
        const err = await res.text();
        setMessage(`Failed to apply schema: ${err}`);
      }
    } catch (err: any) {
      setMessage(`Error: ${err.message}`);
    }
  };

  const handleClear = async () => {
    try {
      await fetch('/api/v1/schema/clear', { method: 'POST' });
      setIsActive(false);
      setViolations(null);
      setMessage('Schema validation deactivated.');
      setTimeout(() => setMessage(null), 3000);
    } catch (err) {}
  };

  const handleTest = async () => {
    try {
      const res = await fetch('/api/v1/schema/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ payload: testPayload }),
      });
      const data = await res.json();
      setViolations(data.violations || []);
    } catch (err) {}
  };

  return (
    <div className="flex flex-col h-full bg-slate-900/60 overflow-y-auto p-4 space-y-4 text-xs">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ShieldCheck className="w-4 h-4 text-emerald-400" />
          <span className="font-bold text-slate-200 text-sm">Real-Time JSON Schema Validation</span>
        </div>

        <span
          className={`px-2 py-0.5 rounded text-[10px] font-semibold uppercase ${
            isActive
              ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
              : 'bg-slate-800 text-slate-400'
          }`}
        >
          {isActive ? 'Enforcing Live' : 'Inactive'}
        </span>
      </div>

      <p className="text-slate-400 text-[11px] leading-relaxed">
        Incoming WebSocket & SSE payloads will be validated against this schema in real-time. Frames with missing fields or type mismatches are highlighted with error flags in the timeline.
      </p>

      {message && (
        <div className="bg-slate-950 border border-emerald-500/40 text-emerald-300 p-2.5 rounded text-[11px] flex items-center gap-2">
          <CheckCircle2 className="w-3.5 h-3.5 flex-shrink-0" />
          <span>{message}</span>
        </div>
      )}

      {/* Schema Editor Card */}
      <div className="space-y-1.5 flex flex-col">
        <div className="flex items-center justify-between">
          <span className="text-slate-400 font-semibold uppercase text-[10px] flex items-center gap-1">
            <FileCode2 className="w-3.5 h-3.5 text-blue-400" />
            JSON Schema Definition (Draft-07)
          </span>
          <button
            onClick={() => setSchemaText(DEFAULT_SCHEMA)}
            className="text-[10px] text-blue-400 hover:text-blue-300 transition"
          >
            Reset Template
          </button>
        </div>

        <textarea
          rows={10}
          value={schemaText}
          onChange={(e) => setSchemaText(e.target.value)}
          className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 font-mono text-slate-200 text-[11px] focus:outline-none focus:border-emerald-500 resize-y"
        />

        <div className="flex items-center justify-end gap-2 pt-1">
          {isActive && (
            <button
              onClick={handleClear}
              className="flex items-center gap-1 px-3 py-1.5 text-rose-400 hover:bg-rose-500/10 rounded font-medium transition"
            >
              <Trash2 className="w-3.5 h-3.5" />
              Clear Schema
            </button>
          )}
          <button
            onClick={handleApply}
            className="flex items-center gap-1.5 px-4 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white font-bold rounded shadow transition shadow-emerald-900/20"
          >
            <ShieldCheck className="w-3.5 h-3.5" />
            Apply & Validate Live Stream
          </button>
        </div>
      </div>

      {/* Live Payload Tester */}
      <div className="bg-slate-950/70 border border-slate-800 rounded-lg p-3.5 space-y-2.5">
        <span className="text-slate-300 font-bold text-[11px] flex items-center gap-1.5">
          <Play className="w-3 h-3 text-cyan-400 fill-current" />
          Test Payload Against Schema
        </span>

        <textarea
          rows={3}
          value={testPayload}
          onChange={(e) => setTestPayload(e.target.value)}
          placeholder='{"event": "ping", ...}'
          className="w-full bg-slate-900 border border-slate-700 rounded p-2 font-mono text-[11px] text-slate-200 focus:outline-none focus:border-cyan-400"
        />

        <div className="flex items-center justify-between">
          <button
            onClick={handleTest}
            disabled={!isActive}
            className="px-3 py-1 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white font-medium rounded transition"
          >
            Run Test
          </button>

          {!isActive && (
            <span className="text-slate-500 text-[10px]">
              Apply schema first to enable testing
            </span>
          )}
        </div>

        {violations !== null && (
          <div className="pt-2 border-t border-slate-800 space-y-1.5">
            {violations.length === 0 ? (
              <div className="flex items-center gap-1.5 text-emerald-400 text-[11px]">
                <CheckCircle2 className="w-3.5 h-3.5" />
                Payload is fully compliant with schema!
              </div>
            ) : (
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-rose-400 font-semibold text-[11px]">
                  <AlertTriangle className="w-3.5 h-3.5" />
                  {violations.length} Violation(s) Detected:
                </div>
                {violations.map((v, i) => (
                  <div
                    key={i}
                    className="bg-rose-950/30 border border-rose-900/40 rounded p-1.5 font-mono text-[10px] text-rose-300"
                  >
                    <span className="font-bold text-rose-200">[{v.property}]</span> {v.message}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
