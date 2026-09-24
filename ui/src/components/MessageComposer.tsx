import React, { useState, useEffect } from 'react';
import { PayloadFormat } from '../types';
import { Send, Clock, Play, Square, Sparkles } from 'lucide-react';

interface MessageComposerProps {
  onSend: (payload: string, format: PayloadFormat) => void;
  disabled: boolean;
}

const templates = [
  {
    name: 'WebSocket Ping',
    format: 'json' as PayloadFormat,
    payload: JSON.stringify({ type: 'ping', timestamp: Date.now() }, null, 2),
  },
  {
    name: 'Crypto Ticker Subscribe',
    format: 'json' as PayloadFormat,
    payload: JSON.stringify(
      {
        action: 'subscribe',
        channel: 'trades',
        symbols: ['BTC-USDT', 'ETH-USDT'],
      },
      null,
      2
    ),
  },
  {
    name: 'Socket.io Chat Event',
    format: 'json' as PayloadFormat,
    payload: JSON.stringify(['chat:message', { text: 'Hello SocketLens!', user: 'alex' }], null, 2),
  },
  {
    name: 'Binary MessagePack Payload',
    format: 'msgpack' as PayloadFormat,
    payload: JSON.stringify({ op: 1, v: '1.0', data: { status: 'heartbeat' } }, null, 2),
  },
];

export const MessageComposer: React.FC<MessageComposerProps> = ({ onSend, disabled }) => {
  const [payload, setPayload] = useState(templates[0].payload);
  const [format, setFormat] = useState<PayloadFormat>('json');
  const [intervalMs, setIntervalMs] = useState<number>(3000);
  const [isLooping, setIsLooping] = useState(false);

  useEffect(() => {
    let timer: any = null;
    if (isLooping && !disabled) {
      timer = setInterval(() => {
        onSend(payload, format);
      }, intervalMs);
    }
    return () => {
      if (timer) clearInterval(timer);
    };
  }, [isLooping, disabled, payload, format, intervalMs]);

  const handleSend = () => {
    onSend(payload, format);
  };

  const handleSelectTemplate = (t: typeof templates[0]) => {
    setPayload(t.payload);
    setFormat(t.format);
  };

  return (
    <div className="flex flex-col h-full bg-slate-900/50 p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <select
            value={format}
            onChange={(e) => setFormat(e.target.value as PayloadFormat)}
            className="bg-slate-950 border border-slate-700/80 rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-200 outline-none"
          >
            <option value="json">Format: Plain / JSON</option>
            <option value="msgpack">Format: MessagePack (Binary)</option>
            <option value="cbor">Format: CBOR (Binary)</option>
          </select>

          <div className="relative group">
            <button className="px-2.5 py-1.5 rounded-lg bg-slate-800 text-xs font-medium text-slate-300 hover:text-slate-100 flex items-center gap-1">
              <Sparkles className="w-3 h-3 text-amber-400" />
              Templates
            </button>
            <div className="hidden group-hover:block absolute left-0 top-full mt-1 w-52 bg-slate-950 border border-slate-800 rounded-lg shadow-xl z-20 py-1">
              {templates.map((t) => (
                <button
                  key={t.name}
                  onClick={() => handleSelectTemplate(t)}
                  className="w-full text-left px-3 py-1.5 text-xs text-slate-300 hover:bg-slate-800 hover:text-white"
                >
                  {t.name}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Dispatch Button */}
        <button
          onClick={handleSend}
          disabled={disabled}
          className="px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-bold text-xs flex items-center gap-1.5 shadow-md shadow-blue-900/20"
        >
          <Send className="w-3.5 h-3.5" />
          Send Frame
        </button>
      </div>

      {/* Monaco / Textarea editor */}
      <textarea
        value={payload}
        onChange={(e) => setPayload(e.target.value)}
        rows={6}
        className="w-full flex-1 bg-slate-950 border border-slate-800 rounded-lg p-3 font-mono text-xs text-slate-100 outline-none focus:border-blue-500 resize-none leading-relaxed"
        placeholder='{"type": "subscribe", "channel": "ticker"}'
      />

      {/* Looping / Recurring Dispatch Bar */}
      <div className="mt-3 pt-3 border-t border-slate-800/80 flex items-center justify-between text-xs text-slate-400">
        <div className="flex items-center gap-2">
          <Clock className="w-3.5 h-3.5 text-slate-500" />
          <span>Repeat every</span>
          <input
            type="number"
            value={intervalMs}
            onChange={(e) => setIntervalMs(Math.max(100, parseInt(e.target.value) || 1000))}
            className="w-16 bg-slate-950 border border-slate-800 rounded px-2 py-1 text-center text-xs font-mono text-slate-200 outline-none"
          />
          <span>ms</span>
        </div>

        <button
          onClick={() => setIsLooping(!isLooping)}
          disabled={disabled}
          className={`px-3 py-1 rounded-md font-semibold text-xs flex items-center gap-1.5 transition ${
            isLooping
              ? 'bg-rose-500/20 border border-rose-500/30 text-rose-400'
              : 'bg-slate-800 text-slate-300 hover:text-white'
          }`}
        >
          {isLooping ? (
            <>
              <Square className="w-3 h-3 fill-current" />
              Stop Loop
            </>
          ) : (
            <>
              <Play className="w-3 h-3 fill-current" />
              Start Loop
            </>
          )}
        </button>
      </div>
    </div>
  );
};
