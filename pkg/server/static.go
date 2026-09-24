package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:static_dist
var embeddedFS embed.FS

// FileServerHandler serves embedded frontend assets with fallback to SPA index.html.
func FileServerHandler() http.Handler {
	distFS, err := fs.Sub(embeddedFS, "static_dist")
	if err != nil {
		return http.HandlerFunc(serveFallbackStudio)
	}

	// Check if index.html exists in embedded files
	if _, err := distFS.Open("index.html"); err != nil {
		return http.HandlerFunc(serveFallbackStudio)
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if _, err := distFS.Open(path); err != nil {
			// SPA fallback
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})
}

// serveFallbackStudio serves a self-contained, high-performance HTML/JS studio.
func serveFallbackStudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fallbackStudioHTML))
}

const fallbackStudioHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>SocketLens — Streaming Studio</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=Plus+Jakarta+Sans:wght@400;500;600;700;800&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #090D16;
      --card-bg: rgba(17, 24, 39, 0.85);
      --border: rgba(255, 255, 255, 0.08);
      --accent-blue: #3B82F6;
      --accent-cyan: #06B6D4;
      --accent-green: #10B981;
      --accent-red: #EF4444;
      --accent-amber: #F59E0B;
      --text-main: #F8FAFC;
      --text-muted: #94A3B8;
      --font-ui: 'Plus Jakarta Sans', system-ui, sans-serif;
      --font-mono: 'JetBrains Mono', monospace;
    }

    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text-main);
      font-family: var(--font-ui);
      height: 100vh;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      -webkit-font-smoothing: antialiased;
    }

    header {
      height: 56px;
      background: rgba(15, 23, 42, 0.95);
      backdrop-filter: blur(12px);
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 20px;
      z-index: 10;
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      font-weight: 800;
      font-size: 1.15rem;
      letter-spacing: -0.02em;
    }
    .brand-badge {
      background: linear-gradient(135deg, #2563EB, #06B6D4);
      padding: 3px 8px;
      border-radius: 6px;
      font-size: 0.7rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .stats-bar {
      display: flex;
      align-items: center;
      gap: 18px;
      font-size: 0.82rem;
      font-family: var(--font-mono);
    }
    .stat-pill {
      background: rgba(255, 255, 255, 0.05);
      border: 1px solid var(--border);
      padding: 4px 10px;
      border-radius: 6px;
      display: flex;
      gap: 8px;
    }
    .stat-pill span { color: var(--text-muted); }

    .main-grid {
      flex: 1;
      display: grid;
      grid-template-columns: 360px 1fr 380px;
      height: calc(100vh - 56px);
      overflow: hidden;
    }

    .panel {
      background: var(--card-bg);
      border-right: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
    .panel:last-child { border-right: none; border-left: 1px solid var(--border); }

    .panel-header {
      padding: 14px 18px;
      border-bottom: 1px solid var(--border);
      font-size: 0.85rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--text-muted);
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .form-group { margin-bottom: 14px; }
    .form-group label {
      display: block;
      font-size: 0.78rem;
      font-weight: 600;
      color: var(--text-muted);
      margin-bottom: 6px;
    }
    input, select, textarea {
      width: 100%;
      background: rgba(0, 0, 0, 0.4);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 9px 12px;
      color: var(--text-main);
      font-family: var(--font-mono);
      font-size: 0.82rem;
      outline: none;
      transition: border-color 0.2s;
    }
    input:focus, select:focus, textarea:focus { border-color: var(--accent-blue); }

    .btn {
      background: var(--accent-blue);
      color: white;
      border: none;
      border-radius: 8px;
      padding: 10px 16px;
      font-size: 0.82rem;
      font-weight: 600;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      transition: opacity 0.2s, transform 0.1s;
    }
    .btn:hover { opacity: 0.9; }
    .btn:active { transform: scale(0.98); }
    .btn-danger { background: var(--accent-red); }
    .btn-green { background: var(--accent-green); }
    .btn-secondary { background: rgba(255,255,255,0.08); color: var(--text-main); }

    .timeline-container {
      flex: 1;
      overflow-y: auto;
      font-family: var(--font-mono);
      font-size: 0.8rem;
    }

    .frame-row {
      display: grid;
      grid-template-columns: 80px 100px 90px 1fr;
      padding: 8px 16px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.03);
      cursor: pointer;
      align-items: center;
      transition: background 0.15s;
    }
    .frame-row:hover { background: rgba(255, 255, 255, 0.04); }
    .frame-row.selected { background: rgba(59, 130, 246, 0.15); border-left: 3px solid var(--accent-blue); }

    .tag-in { color: var(--accent-green); font-weight: 700; }
    .tag-out { color: var(--accent-cyan); font-weight: 700; }
    .tag-fmt { background: rgba(255,255,255,0.06); padding: 2px 6px; border-radius: 4px; font-size: 0.72rem; }

    .detail-view {
      flex: 1;
      padding: 16px;
      overflow-y: auto;
      background: rgba(0, 0, 0, 0.25);
    }
    .detail-view pre {
      font-family: var(--font-mono);
      font-size: 0.8rem;
      white-space: pre-wrap;
      word-break: break-all;
      color: #E2E8F0;
    }

    .tabs { display: flex; border-bottom: 1px solid var(--border); }
    .tab {
      padding: 10px 16px;
      font-size: 0.8rem;
      font-weight: 600;
      color: var(--text-muted);
      cursor: pointer;
      border-bottom: 2px solid transparent;
    }
    .tab.active { color: var(--text-main); border-bottom-color: var(--accent-blue); }

    .pill-status {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      font-size: 0.75rem;
      font-weight: 700;
      padding: 4px 8px;
      border-radius: 6px;
    }
    .status-connected { background: rgba(16, 185, 129, 0.15); color: var(--accent-green); }
    .status-disconnected { background: rgba(239, 68, 68, 0.15); color: var(--accent-red); }
  </style>
</head>
<body>
  <header>
    <div class="brand">
      <span>SocketLens</span>
      <span class="brand-badge">Studio</span>
    </div>
    <div class="stats-bar">
      <div class="stat-pill"><span>FRAMES IN</span> <b id="stat-in">0</b></div>
      <div class="stat-pill"><span>FRAMES OUT</span> <b id="stat-out">0</b></div>
      <div class="stat-pill"><span>BYTES</span> <b id="stat-bytes">0 B</b></div>
      <div id="conn-badge" class="pill-status status-disconnected">DISCONNECTED</div>
    </div>
  </header>

  <div class="main-grid">
    <!-- Left Panel: Connection & Control -->
    <div class="panel">
      <div class="tabs">
        <div class="tab active" onclick="switchLeftTab('connect')">Connect</div>
        <div class="tab" onclick="switchLeftTab('mock')">Mock Server</div>
        <div class="tab" onclick="switchLeftTab('bench')">Benchmark</div>
      </div>

      <div id="tab-connect" style="padding: 18px; overflow-y: auto;">
        <div class="form-group">
          <label>Protocol</label>
          <select id="conn-proto">
            <option value="ws">WebSocket (WS / WSS)</option>
            <option value="sse">Server-Sent Events (SSE)</option>
            <option value="socketio">Socket.io v4</option>
          </select>
        </div>
        <div class="form-group">
          <label>Target URL</label>
          <input type="text" id="conn-url" value="wss://echo.websocket.events" placeholder="wss://... or http://...">
        </div>
        <div class="form-group">
          <label>Custom Headers (JSON)</label>
          <textarea id="conn-headers" rows="2" placeholder='{"Authorization": "Bearer token"}'></textarea>
        </div>
        <button id="btn-connect" class="btn" style="width: 100%;" onclick="toggleConnect()">Connect Stream</button>

        <hr style="border: 0; border-top: 1px solid var(--border); margin: 20px 0;">

        <div class="form-group">
          <label>Message Composer</label>
          <select id="msg-format" style="margin-bottom: 8px;">
            <option value="json">Format: Plain / JSON</option>
            <option value="msgpack">Format: MessagePack (Binary)</option>
            <option value="cbor">Format: CBOR (Binary)</option>
          </select>
          <textarea id="composer-payload" rows="6" placeholder='{"action": "ping", "data": "hello"}'>{"event": "ping", "timestamp": 123456}</textarea>
        </div>
        <button class="btn btn-secondary" style="width: 100%;" onclick="sendFrame()">Dispatch Frame</button>
      </div>

      <div id="tab-mock" style="padding: 18px; display: none; overflow-y: auto;">
        <div class="form-group">
          <label>Mock Mode</label>
          <select id="mock-mode">
            <option value="echo">Echo Mode (Reflects Payload)</option>
            <option value="broadcast">Broadcast (All Clients)</option>
            <option value="llm-tokens">LLM Stream (Simulated Tokens)</option>
            <option value="ticks">Financial Ticker Generator</option>
          </select>
        </div>
        <div class="form-group">
          <label>Port</label>
          <input type="number" id="mock-port" value="8080">
        </div>
        <div class="form-group">
          <label>Route Prefix</label>
          <input type="text" id="mock-route" value="/ws/feed">
        </div>
        <div class="form-group">
          <label>Chaos Latency (ms)</label>
          <input type="number" id="mock-chaos-latency" value="0">
        </div>
        <div class="form-group">
          <label>Chaos Drop Rate (%)</label>
          <input type="number" id="mock-chaos-drop" value="0">
        </div>
        <button id="btn-mock" class="btn btn-green" style="width: 100%;" onclick="toggleMock()">Start Mock Server</button>
      </div>

      <div id="tab-bench" style="padding: 18px; display: none; overflow-y: auto;">
        <div class="form-group">
          <label>Target URL</label>
          <input type="text" id="bench-url" value="ws://127.0.0.1:8080/ws/feed">
        </div>
        <div class="form-group">
          <label>Concurrent Clients</label>
          <input type="number" id="bench-clients" value="100">
        </div>
        <div class="form-group">
          <label>Duration (Seconds)</label>
          <input type="number" id="bench-duration" value="10">
        </div>
        <button id="btn-bench" class="btn btn-green" style="width: 100%;" onclick="startBenchmark()">Start Stress Run</button>
        <div id="bench-results" style="margin-top: 15px; font-family: var(--font-mono); font-size: 0.78rem; white-space: pre-wrap; color: var(--text-muted);"></div>
      </div>
    </div>

    <!-- Center Panel: Timeline -->
    <div class="panel">
      <div class="panel-header">
        <span>Timeline Inspector</span>
        <div style="display: flex; gap: 8px;">
          <input type="text" id="filter-input" placeholder="Search payloads..." style="padding: 4px 8px; width: 180px; font-size: 0.75rem;" oninput="renderTimeline()">
          <button class="btn btn-secondary" style="padding: 4px 8px; font-size: 0.75rem;" onclick="clearTimeline()">Clear</button>
        </div>
      </div>
      <div class="timeline-container" id="timeline-list"></div>
    </div>

    <!-- Right Panel: Frame Details -->
    <div class="panel">
      <div class="panel-header">
        <span>Payload Inspector</span>
        <span id="detail-meta" style="font-size: 0.75rem; text-transform: none; color: var(--accent-cyan);">Select a frame</span>
      </div>
      <div class="detail-view">
        <pre id="detail-content">// Click any timeline frame to inspect decoded JSON, headers, and metadata</pre>
      </div>
    </div>
  </div>

  <script>
    let activeConnId = null;
    let frames = [];
    let selectedFrame = null;
    let framesInCount = 0;
    let framesOutCount = 0;
    let totalBytes = 0;
    let isMockRunning = false;

    // Connect to internal control WebSocket
    function connectControlWS() {
      const loc = window.location;
      const wsProto = loc.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = wsProto + '//' + loc.host + '/ws/control';
      const ws = new WebSocket(wsUrl);

      ws.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data);
          if (msg.type === 'history') {
            msg.frames.forEach(addFrame);
          } else if (msg.type === 'batch') {
            msg.frames.forEach(addFrame);
          } else if (msg.type === 'state') {
            updateConnState(msg.state);
          }
        } catch (err) {}
      };

      ws.onclose = () => {
        setTimeout(connectControlWS, 2000);
      };
    }
    connectControlWS();

    function addFrame(f) {
      frames.push(f);
      if (f.direction === 'inbound') framesInCount++;
      else framesOutCount++;
      totalBytes += f.length || 0;

      document.getElementById('stat-in').innerText = framesInCount;
      document.getElementById('stat-out').innerText = framesOutCount;
      document.getElementById('stat-bytes').innerText = (totalBytes / 1024).toFixed(1) + ' KB';

      renderTimeline();
    }

    function renderTimeline() {
      const list = document.getElementById('timeline-list');
      const filter = document.getElementById('filter-input').value.toLowerCase();
      const filtered = frames.filter(f => {
        if (!filter) return true;
        const text = (f.decoded || f.payload_str || '').toLowerCase();
        return text.includes(filter);
      });

      const slice = filtered.slice(-300); // virtualized window
      let html = '';
      slice.forEach((f, idx) => {
        const isSelected = selectedFrame === f ? 'selected' : '';
        const tag = f.direction === 'inbound' ? '<span class="tag-in">IN  ▲</span>' : '<span class="tag-out">OUT ▼</span>';
        const ts = new Date(f.timestamp).toLocaleTimeString();
        const fmt = f.format ? '<span class="tag-fmt">' + f.format.toUpperCase() + '</span>' : '';
        const preview = (f.decoded || f.payload_str || '').substring(0, 80).replace(/</g, '&lt;');

        html += '<div class="frame-row ' + isSelected + '" onclick="selectFrame(' + (frames.indexOf(f)) + ')">' +
          '<div>' + tag + '</div>' +
          '<div>' + ts + '</div>' +
          '<div>' + fmt + '</div>' +
          '<div style="overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">' + preview + '</div>' +
        '</div>';
      });
      list.innerHTML = html;
      list.scrollTop = list.scrollHeight;
    }

    function selectFrame(index) {
      selectedFrame = frames[index];
      renderTimeline();
      if (!selectedFrame) return;

      document.getElementById('detail-meta').innerText = '#' + selectedFrame.sequence + ' | ' + selectedFrame.length + ' B';
      const content = selectedFrame.decoded || selectedFrame.payload_str || '';
      document.getElementById('detail-content').innerText = content;
    }

    function clearTimeline() {
      frames = [];
      framesInCount = 0;
      framesOutCount = 0;
      totalBytes = 0;
      document.getElementById('stat-in').innerText = '0';
      document.getElementById('stat-out').innerText = '0';
      document.getElementById('stat-bytes').innerText = '0 B';
      renderTimeline();
      document.getElementById('detail-content').innerText = '// Timeline cleared';
    }

    async function toggleConnect() {
      const btn = document.getElementById('btn-connect');
      if (activeConnId) {
        await fetch('/api/v1/connections/' + activeConnId, { method: 'DELETE' });
        activeConnId = null;
        btn.innerText = 'Connect Stream';
        btn.className = 'btn';
        updateConnState('disconnected');
        return;
      }

      let headers = {};
      try {
        const hText = document.getElementById('conn-headers').value.trim();
        if (hText) headers = JSON.parse(hText);
      } catch (err) {
        alert('Invalid JSON in headers field');
        return;
      }

      const res = await fetch('/api/v1/connections', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          url: document.getElementById('conn-url').value,
          protocol: document.getElementById('conn-proto').value,
          headers: headers,
          auto_reconnect: true
        })
      });

      const data = await res.json();
      activeConnId = data.connection_id;
      btn.innerText = 'Disconnect';
      btn.className = 'btn btn-danger';
      updateConnState('connecting');
    }

    async function sendFrame() {
      if (!activeConnId) {
        alert('Please connect to a streaming endpoint first');
        return;
      }

      const payload = document.getElementById('composer-payload').value;
      const format = document.getElementById('msg-format').value;

      await fetch('/api/v1/connections/' + activeConnId + '/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          payload: payload,
          format: format
        })
      });
    }

    async function toggleMock() {
      const btn = document.getElementById('btn-mock');
      if (isMockRunning) {
        await fetch('/api/v1/mock/stop', { method: 'POST' });
        isMockRunning = false;
        btn.innerText = 'Start Mock Server';
        btn.className = 'btn btn-green';
        return;
      }

      const port = parseInt(document.getElementById('mock-port').value, 10);
      const route = document.getElementById('mock-route').value;
      const mode = document.getElementById('mock-mode').value;
      const latency = parseInt(document.getElementById('mock-chaos-latency').value, 10) * 1000000;
      const drop = parseInt(document.getElementById('mock-chaos-drop').value, 10);

      const res = await fetch('/api/v1/mock/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          port: port,
          route: route,
          mode: mode,
          chaos: { latency: latency, drop_rate: drop }
        })
      });

      if (res.ok) {
        isMockRunning = true;
        btn.innerText = 'Stop Mock Server';
        btn.className = 'btn btn-danger';
      }
    }

    async function startBenchmark() {
      const resDiv = document.getElementById('bench-results');
      resDiv.innerText = 'Running stress test... please wait...';

      const url = document.getElementById('bench-url').value;
      const clients = parseInt(document.getElementById('bench-clients').value, 10);
      const duration = parseInt(document.getElementById('bench-duration').value, 10) * 1000000000;

      await fetch('/api/v1/bench/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          url: url,
          clients: clients,
          duration: duration
        })
      });

      const interval = setInterval(async () => {
        const poll = await fetch('/api/v1/bench/status');
        const st = await poll.json();
        if (st.status === 'complete' && st.report) {
          clearInterval(interval);
          const r = st.report;
          resDiv.innerText = 'Benchmark Complete!\n' +
            'Active/Target: ' + r.peak_clients + ' / ' + r.target_clients + '\n' +
            'p50 Latency: ' + r.p50_latency_ms.toFixed(2) + ' ms\n' +
            'p95 Latency: ' + r.p95_latency_ms.toFixed(2) + ' ms\n' +
            'p99 Latency: ' + r.p99_latency_ms.toFixed(2) + ' ms\n' +
            'Message Rate: ' + r.messages_per_sec.toFixed(1) + ' msg/s';
        }
      }, 1000);
    }

    function updateConnState(s) {
      const b = document.getElementById('conn-badge');
      if (s === 'connected') {
        b.className = 'pill-status status-connected';
        b.innerText = 'CONNECTED';
      } else {
        b.className = 'pill-status status-disconnected';
        b.innerText = (s || 'DISCONNECTED').toUpperCase();
      }
    }

    function switchLeftTab(tab) {
      document.querySelectorAll('.tabs .tab').forEach(t => t.classList.remove('active'));
      document.getElementById('tab-connect').style.display = 'none';
      document.getElementById('tab-mock').style.display = 'none';
      document.getElementById('tab-bench').style.display = 'none';

      if (tab === 'connect') {
        document.querySelectorAll('.tabs .tab')[0].classList.add('active');
        document.getElementById('tab-connect').style.display = 'block';
      } else if (tab === 'mock') {
        document.querySelectorAll('.tabs .tab')[1].classList.add('active');
        document.getElementById('tab-mock').style.display = 'block';
      } else {
        document.querySelectorAll('.tabs .tab')[2].classList.add('active');
        document.getElementById('tab-bench').style.display = 'block';
      }
    }
  </script>
</body>
</html>
`
