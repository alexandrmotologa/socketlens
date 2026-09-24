import React, { useState, useEffect } from 'react';
import { Frame, ConnectionConfig, ConnectionState, PayloadFormat } from './types';
import { StatsBar } from './components/StatsBar';
import { ConnectBar } from './components/ConnectBar';
import { FrameTimeline } from './components/FrameTimeline';
import { FrameDetailPanel } from './components/FrameDetailPanel';
import { MessageComposer } from './components/MessageComposer';
import { MockServerPanel } from './components/MockServerPanel';
import { StressRunnerPanel } from './components/StressRunnerPanel';
import { ProxyPanel } from './components/ProxyPanel';
import { LLMInspectorPanel } from './components/LLMInspectorPanel';
import { SchemaValidatorPanel } from './components/SchemaValidatorPanel';
import { AutomationRulesPanel } from './components/AutomationRulesPanel';
import { Radio, Server, Gauge, Shuffle, Sparkles, ShieldCheck, Bot } from 'lucide-react';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<
    'stream' | 'proxy' | 'llm' | 'schema' | 'rules' | 'mock' | 'bench'
  >('stream');
  const [activeConnId, setActiveConnId] = useState<string | null>(null);
  const [connState, setConnState] = useState<ConnectionState>('disconnected');
  const [frames, setFrames] = useState<Frame[]>([]);
  const [selectedFrame, setSelectedFrame] = useState<Frame | null>(null);
  const [framesIn, setFramesIn] = useState(0);
  const [framesOut, setFramesOut] = useState(0);
  const [totalBytes, setTotalBytes] = useState(0);
  const [mockRunning, setMockRunning] = useState(false);
  const [mockEndpoints, setMockEndpoints] = useState<any>(null);

  // Connect to internal control WebSocket
  useEffect(() => {
    let ws: WebSocket | null = null;
    let timer: any = null;

    const connectControl = () => {
      const loc = window.location;
      const wsProto = loc.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${wsProto}//${loc.host}/ws/control`;

      ws = new WebSocket(wsUrl);

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === 'history') {
            ingestFrames(msg.frames);
          } else if (msg.type === 'batch') {
            ingestFrames(msg.frames);
          } else if (msg.type === 'state') {
            setConnState(msg.state);
          }
        } catch (err) {}
      };

      ws.onclose = () => {
        timer = setTimeout(connectControl, 2000);
      };
    };

    connectControl();
    return () => {
      if (ws) ws.close();
      if (timer) clearTimeout(timer);
    };
  }, []);

  const ingestFrames = (newFrames: Frame[]) => {
    setFrames((prev) => {
      const updated = [...prev, ...newFrames];
      if (updated.length > 5000) {
        return updated.slice(updated.length - 5000);
      }
      return updated;
    });

    let incIn = 0;
    let incOut = 0;
    let bytes = 0;

    newFrames.forEach((f) => {
      if (f.direction === 'inbound') incIn++;
      else incOut++;
      bytes += f.length || 0;
    });

    setFramesIn((p) => p + incIn);
    setFramesOut((p) => p + incOut);
    setTotalBytes((p) => p + bytes);
  };

  const handleConnect = async (cfg: ConnectionConfig) => {
    setConnState('connecting');
    try {
      const res = await fetch('/api/v1/connections', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(cfg),
      });
      const data = await res.json();
      setActiveConnId(data.connection_id);
    } catch (err) {
      setConnState('error');
    }
  };

  const handleDisconnect = async () => {
    if (activeConnId) {
      await fetch(`/api/v1/connections/${activeConnId}`, { method: 'DELETE' });
      setActiveConnId(null);
      setConnState('disconnected');
    }
  };

  const handleSend = async (payload: string, format: PayloadFormat) => {
    if (!activeConnId) return;
    await fetch(`/api/v1/connections/${activeConnId}/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ payload, format }),
    });
  };

  const handleStartMock = async (cfg: any) => {
    const res = await fetch('/api/v1/mock/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(cfg),
    });
    const data = await res.json();
    setMockRunning(true);
    setMockEndpoints(data);
  };

  const handleStopMock = async () => {
    await fetch('/api/v1/mock/stop', { method: 'POST' });
    setMockRunning(false);
  };

  const handleClear = () => {
    setFrames([]);
    setSelectedFrame(null);
    setFramesIn(0);
    setFramesOut(0);
    setTotalBytes(0);
  };

  // Find previous frame in sequence for diffing
  const selectedIndex = selectedFrame
    ? frames.findIndex((f) => f.id === selectedFrame.id)
    : -1;
  const previousFrame = selectedIndex > 0 ? frames[selectedIndex - 1] : null;

  return (
    <div className="flex flex-col h-screen w-screen bg-background overflow-hidden text-slate-100">
      {/* Top Telemetry Header */}
      <StatsBar
        state={connState}
        framesIn={framesIn}
        framesOut={framesOut}
        totalBytes={totalBytes}
      />

      {/* Main Studio 3-Column Layout */}
      <div className="flex-1 grid grid-cols-[400px_1fr_420px] min-h-0 divide-x divide-slate-800/80">
        {/* Left Column: Workbench Subsystems */}
        <div className="flex flex-col bg-slate-900/60 min-h-0">
          {/* Subsystem Navigation Tabs */}
          <div className="flex border-b border-slate-800/80 bg-slate-950/70 overflow-x-auto scrollbar-none">
            <button
              onClick={() => setActiveTab('stream')}
              className={`px-3 py-2.5 text-[11px] font-bold flex items-center gap-1.5 border-b-2 whitespace-nowrap transition ${
                activeTab === 'stream'
                  ? 'border-blue-500 text-blue-400 bg-blue-500/10'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <Radio className="w-3.5 h-3.5" />
              Stream
            </button>

            <button
              onClick={() => setActiveTab('proxy')}
              className={`px-3 py-2.5 text-[11px] font-bold flex items-center gap-1.5 border-b-2 whitespace-nowrap transition ${
                activeTab === 'proxy'
                  ? 'border-purple-500 text-purple-400 bg-purple-500/10'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <Shuffle className="w-3.5 h-3.5" />
              Proxy
            </button>

            <button
              onClick={() => setActiveTab('llm')}
              className={`px-3 py-2.5 text-[11px] font-bold flex items-center gap-1.5 border-b-2 whitespace-nowrap transition ${
                activeTab === 'llm'
                  ? 'border-indigo-500 text-indigo-400 bg-indigo-500/10'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <Sparkles className="w-3.5 h-3.5" />
              LLM
            </button>

            <button
              onClick={() => setActiveTab('schema')}
              className={`px-3 py-2.5 text-[11px] font-bold flex items-center gap-1.5 border-b-2 whitespace-nowrap transition ${
                activeTab === 'schema'
                  ? 'border-emerald-500 text-emerald-400 bg-emerald-500/10'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <ShieldCheck className="w-3.5 h-3.5" />
              Schema
            </button>

            <button
              onClick={() => setActiveTab('rules')}
              className={`px-3 py-2.5 text-[11px] font-bold flex items-center gap-1.5 border-b-2 whitespace-nowrap transition ${
                activeTab === 'rules'
                  ? 'border-amber-500 text-amber-400 bg-amber-500/10'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <Bot className="w-3.5 h-3.5" />
              Rules
            </button>

            <button
              onClick={() => setActiveTab('mock')}
              className={`px-3 py-2.5 text-[11px] font-bold flex items-center gap-1.5 border-b-2 whitespace-nowrap transition ${
                activeTab === 'mock'
                  ? 'border-emerald-500 text-emerald-400 bg-emerald-500/10'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <Server className="w-3.5 h-3.5" />
              Mock
            </button>

            <button
              onClick={() => setActiveTab('bench')}
              className={`px-3 py-2.5 text-[11px] font-bold flex items-center gap-1.5 border-b-2 whitespace-nowrap transition ${
                activeTab === 'bench'
                  ? 'border-cyan-500 text-cyan-400 bg-cyan-500/10'
                  : 'border-transparent text-slate-400 hover:text-slate-200'
              }`}
            >
              <Gauge className="w-3.5 h-3.5" />
              Bench
            </button>
          </div>

          {/* Subsystem Panels */}
          <div className="flex-1 overflow-y-auto min-h-0">
            {activeTab === 'stream' && (
              <div className="flex flex-col h-full">
                <ConnectBar
                  onConnect={handleConnect}
                  onDisconnect={handleDisconnect}
                  isConnected={connState === 'connected'}
                  isConnecting={connState === 'connecting'}
                />
                <div className="flex-1 min-h-[300px]">
                  <MessageComposer
                    onSend={handleSend}
                    disabled={connState !== 'connected'}
                  />
                </div>
              </div>
            )}

            {activeTab === 'proxy' && <ProxyPanel />}

            {activeTab === 'llm' && <LLMInspectorPanel />}

            {activeTab === 'schema' && <SchemaValidatorPanel />}

            {activeTab === 'rules' && <AutomationRulesPanel />}

            {activeTab === 'mock' && (
              <MockServerPanel
                onStart={handleStartMock}
                onStop={handleStopMock}
                isRunning={mockRunning}
                endpoints={mockEndpoints}
              />
            )}

            {activeTab === 'bench' && <StressRunnerPanel />}
          </div>
        </div>

        {/* Center Column: Frame Timeline */}
        <div className="flex flex-col min-h-0">
          <FrameTimeline
            frames={frames}
            selectedFrame={selectedFrame}
            onSelectFrame={setSelectedFrame}
            onClear={handleClear}
          />
        </div>

        {/* Right Column: Frame Detail & Diff Inspector */}
        <div className="flex flex-col min-h-0">
          <FrameDetailPanel
            frame={selectedFrame}
            previousFrame={previousFrame}
            onResend={(payload) => handleSend(payload, 'json')}
          />
        </div>
      </div>
    </div>
  );
};
export default App;
