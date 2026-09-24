import React, { useState, useEffect } from 'react';
import { AutomationRule } from '../types';
import { Bot, Plus, Trash2, ArrowRight, Zap, CheckCircle2 } from 'lucide-react';

export const AutomationRulesPanel: React.FC = () => {
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [name, setName] = useState('');
  const [condition, setCondition] = useState<'contains' | 'exact' | 'event_name' | 'opcode'>('contains');
  const [pattern, setPattern] = useState('');
  const [response, setResponse] = useState('');
  const [message, setMessage] = useState<string | null>(null);

  const fetchRules = async () => {
    try {
      const res = await fetch('/api/v1/rules');
      if (res.ok) {
        const data: AutomationRule[] = await res.json();
        setRules(data);
      }
    } catch (err) {}
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const handleAddRule = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !pattern || !response) return;

    try {
      const res = await fetch('/api/v1/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name,
          condition,
          pattern,
          action: 'reply',
          response,
        }),
      });
      if (res.ok) {
        setName('');
        setPattern('');
        setResponse('');
        setMessage('Automation rule active!');
        setTimeout(() => setMessage(null), 3000);
        fetchRules();
      }
    } catch (err) {}
  };

  const handleDeleteRule = async (id?: string) => {
    if (!id) return;
    try {
      await fetch(`/api/v1/rules/${id}`, { method: 'DELETE' });
      fetchRules();
    } catch (err) {}
  };

  return (
    <div className="flex flex-col h-full bg-slate-900/60 overflow-y-auto p-4 space-y-4 text-xs">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Bot className="w-4 h-4 text-amber-400" />
          <span className="font-bold text-slate-200 text-sm">Event-Driven Auto-Responders</span>
        </div>

        <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30">
          {rules.length} Active Rules
        </span>
      </div>

      <p className="text-slate-400 text-[11px] leading-relaxed">
        Define real-time automation scripts and auto-responders. When an inbound frame matches a condition, SocketLens replies instantaneously without manual intervention.
      </p>

      {message && (
        <div className="bg-slate-950 border border-emerald-500/40 text-emerald-300 p-2.5 rounded text-[11px] flex items-center gap-2">
          <CheckCircle2 className="w-3.5 h-3.5 flex-shrink-0" />
          <span>{message}</span>
        </div>
      )}

      {/* Add New Rule Form */}
      <form
        onSubmit={handleAddRule}
        className="bg-slate-950/80 border border-slate-800 rounded-lg p-3.5 space-y-3"
      >
        <span className="font-bold text-slate-300 text-xs flex items-center gap-1.5">
          <Zap className="w-3.5 h-3.5 text-amber-400" />
          Create If-This-Then-That Rule
        </span>

        <div className="space-y-1">
          <label className="text-slate-400 font-medium text-[10px] uppercase">Rule Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Heartbeat Ping Responder"
            className="w-full bg-slate-900 border border-slate-700 rounded px-2.5 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400"
          />
        </div>

        <div className="grid grid-cols-3 gap-2">
          <div className="space-y-1">
            <label className="text-slate-400 font-medium text-[10px] uppercase">Match On</label>
            <select
              value={condition}
              onChange={(e) => setCondition(e.target.value as any)}
              className="w-full bg-slate-900 border border-slate-700 rounded px-2 py-1.5 text-slate-200 focus:outline-none focus:border-amber-400"
            >
              <option value="contains">Payload Contains</option>
              <option value="exact">Payload Exact</option>
              <option value="event_name">Event Name</option>
              <option value="opcode">OpCode</option>
            </select>
          </div>

          <div className="col-span-2 space-y-1">
            <label className="text-slate-400 font-medium text-[10px] uppercase">Trigger Pattern</label>
            <input
              type="text"
              value={pattern}
              onChange={(e) => setPattern(e.target.value)}
              placeholder="e.g. ping or trade_update"
              className="w-full bg-slate-900 border border-slate-700 rounded px-2.5 py-1.5 text-slate-200 font-mono focus:outline-none focus:border-amber-400"
            />
          </div>
        </div>

        <div className="space-y-1">
          <label className="text-slate-400 font-medium text-[10px] uppercase">
            Auto-Reply Payload
          </label>
          <textarea
            rows={2}
            value={response}
            onChange={(e) => setResponse(e.target.value)}
            placeholder='e.g. {"type":"pong"} or pong'
            className="w-full bg-slate-900 border border-slate-700 rounded p-2 text-slate-200 font-mono text-[11px] focus:outline-none focus:border-amber-400 resize-none"
          />
        </div>

        <div className="flex justify-end pt-1">
          <button
            type="submit"
            disabled={!name || !pattern || !response}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-500 hover:bg-amber-400 disabled:opacity-50 text-slate-950 font-bold rounded transition shadow"
          >
            <Plus className="w-3.5 h-3.5" />
            Add Automation Rule
          </button>
        </div>
      </form>

      {/* Rules List */}
      <div className="space-y-2">
        <span className="font-bold text-slate-300 text-xs">Configured Triggers</span>

        {rules.length === 0 ? (
          <div className="p-6 text-center border border-dashed border-slate-800 rounded-lg text-slate-500 text-xs">
            No automation rules active. Add a rule above to automate stream replies.
          </div>
        ) : (
          <div className="space-y-2">
            {rules.map((r) => (
              <div
                key={r.id}
                className="bg-slate-950 border border-slate-800/80 rounded-lg p-3 space-y-2"
              >
                <div className="flex items-center justify-between">
                  <span className="font-bold text-slate-200">{r.name}</span>
                  <button
                    onClick={() => handleDeleteRule(r.id)}
                    className="text-slate-500 hover:text-rose-400 transition"
                    title="Delete rule"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                <div className="bg-slate-900/80 border border-slate-800 rounded p-2 text-[11px] font-mono flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1 text-slate-300 truncate">
                    <span className="text-amber-400 font-semibold">{r.condition}:</span>
                    <span className="truncate text-slate-200 font-bold">"{r.pattern}"</span>
                  </div>

                  <ArrowRight className="w-3 h-3 text-slate-500 flex-shrink-0" />

                  <div className="text-emerald-400 truncate font-semibold">
                    reply: "{r.response}"
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
