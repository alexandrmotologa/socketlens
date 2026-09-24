import React, { useState, useRef, useEffect } from 'react';
import { Frame } from '../types';
import { Search, Trash2, ArrowDown, Pause, Play } from 'lucide-react';

interface FrameTimelineProps {
  frames: Frame[];
  selectedFrame: Frame | null;
  onSelectFrame: (f: Frame) => void;
  onClear: () => void;
}

export const FrameTimeline: React.FC<FrameTimelineProps> = ({
  frames,
  selectedFrame,
  onSelectFrame,
  onClear,
}) => {
  const [filterText, setFilterText] = useState('');
  const [autoScroll, setAutoScroll] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  const filteredFrames = frames.filter((f) => {
    if (!filterText) return true;
    const search = filterText.toLowerCase();
    const content = (f.decoded || f.payload_str || '').toLowerCase();
    const event = (f.event_name || '').toLowerCase();
    return content.includes(search) || event.includes(search) || f.opcode.includes(search);
  });

  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [frames, autoScroll]);

  return (
    <div className="flex-1 flex flex-col min-h-0 bg-slate-900/40">
      {/* Search & Control Filter Bar */}
      <div className="h-10 border-b border-slate-800/80 px-4 flex items-center justify-between gap-3 text-xs bg-slate-950/40">
        <div className="flex items-center gap-2 flex-1 max-w-md">
          <Search className="w-3.5 h-3.5 text-slate-500" />
          <input
            type="text"
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            placeholder="Search frames by text, opcode, or event name..."
            className="w-full bg-transparent border-none text-slate-200 placeholder-slate-500 font-mono text-[11px] outline-none"
          />
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`px-2 py-1 rounded text-[11px] font-medium flex items-center gap-1.5 transition ${
              autoScroll
                ? 'bg-blue-500/10 text-blue-400 border border-blue-500/20'
                : 'bg-slate-800/60 text-slate-400'
            }`}
          >
            {autoScroll ? <ArrowDown className="w-3 h-3" /> : <Pause className="w-3 h-3" />}
            Auto-scroll
          </button>

          <button
            onClick={onClear}
            className="p-1.5 text-slate-400 hover:text-rose-400 hover:bg-slate-800 rounded transition"
            title="Clear timeline"
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Frame Rows */}
      <div ref={containerRef} className="flex-1 overflow-y-auto font-mono text-xs">
        {filteredFrames.length === 0 ? (
          <div className="h-full flex items-center justify-center text-slate-500 text-xs">
            {frames.length === 0 ? 'No frames captured yet. Connect to a stream to begin.' : 'No frames match current filter.'}
          </div>
        ) : (
          filteredFrames.map((frame) => {
            const isSelected = selectedFrame?.id === frame.id;
            const isInbound = frame.direction === 'inbound';
            const timeStr = new Date(frame.timestamp).toLocaleTimeString();

            return (
              <div
                key={frame.id}
                onClick={() => onSelectFrame(frame)}
                className={`grid grid-cols-[70px_85px_80px_1fr_75px] gap-2 px-4 py-2 border-b border-slate-800/40 items-center cursor-pointer transition ${
                  isSelected
                    ? 'bg-blue-600/15 border-l-2 border-l-blue-500 text-slate-100'
                    : 'hover:bg-slate-800/30 text-slate-300'
                }`}
              >
                {/* Direction */}
                <div>
                  {isInbound ? (
                    <span className="text-emerald-400 font-bold text-[11px] flex items-center gap-1">
                      IN ▲
                    </span>
                  ) : (
                    <span className="text-cyan-400 font-bold text-[11px] flex items-center gap-1">
                      OUT ▼
                    </span>
                  )}
                </div>

                {/* Timestamp */}
                <div className="text-[11px] text-slate-400">{timeStr}</div>

                {/* OpCode / Format badge */}
                <div>
                  <span className="px-1.5 py-0.5 rounded bg-slate-800 text-[10px] uppercase font-bold text-slate-300 border border-slate-700/60">
                    {frame.format || frame.opcode}
                  </span>
                </div>

                {/* Payload snippet preview */}
                <div className="truncate text-[11px] text-slate-300 font-normal">
                  {frame.event_name && (
                    <span className="text-amber-400 font-semibold mr-1.5">
                      [{frame.event_name}]
                    </span>
                  )}
                  {frame.decoded || frame.payload_str || ''}
                </div>

                {/* Length */}
                <div className="text-right text-[11px] text-slate-500 font-medium">
                  {frame.length} B
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};
