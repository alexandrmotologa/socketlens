const puppeteer = require('puppeteer-core');
const fs = require('fs');
const path = require('path');
const { spawn, execSync } = require('child_process');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const BIN_PATH = path.resolve(__dirname, '..', 'bin', 'socketlens.exe');
const STUDIO_PORT = 50070;
const STUDIO_URL = `http://127.0.0.1:${STUDIO_PORT}`;
const IMAGES_DIR = path.resolve(__dirname, '..', 'docs', 'images');
const TEMP_FRAMES_DIR = path.resolve(__dirname, '..', 'temp_gif_frames');

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function clickButtonWithText(page, text) {
  const buttons = await page.$$('button');
  for (const b of buttons) {
    const t = await page.evaluate((el) => el.textContent, b);
    if (t && t.trim().includes(text)) {
      await b.click();
      return true;
    }
  }
  return false;
}

async function waitForServerReady(url, maxRetries = 20) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const res = await fetch(`${url}/api/v1/connections`);
      if (res.ok) return true;
    } catch (e) {}
    await sleep(300);
  }
  return false;
}

async function main() {
  if (!fs.existsSync(IMAGES_DIR)) fs.mkdirSync(IMAGES_DIR, { recursive: true });
  if (!fs.existsSync(TEMP_FRAMES_DIR)) fs.mkdirSync(TEMP_FRAMES_DIR, { recursive: true });

  console.log('1. Spawning SocketLens native daemon...');
  const daemon = spawn(BIN_PATH, ['--port', String(STUDIO_PORT), '--no-browser'], {
    stdio: 'ignore',
  });

  daemon.on('error', (err) => {
    console.error('Failed to spawn socketlens binary:', err);
    process.exit(1);
  });

  const ready = await waitForServerReady(STUDIO_URL);
  if (!ready) {
    console.error('SocketLens server did not become ready.');
    daemon.kill();
    process.exit(1);
  }
  console.log('✓ SocketLens daemon listening on:', STUDIO_URL);

  try {
    // 2. Start mock server on ticks mode
    console.log('2. Starting embedded Mock Server (ticks mode, rate=15/s)...');
    await fetch(`${STUDIO_URL}/api/v1/mock/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        port: 8080,
        route: '/ws/feed',
        mode: 'ticks',
        rate: 15,
      }),
    });
    await sleep(600);

    // 3. Connect Stream client to mock server
    console.log('3. Registering active WebSocket stream connection...');
    await fetch(`${STUDIO_URL}/api/v1/connections`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: 'conn_feed_mock',
        url: 'ws://127.0.0.1:8080/ws/feed',
        protocol: 'ws',
        auto_reconnect: true,
      }),
    });
    await sleep(1500);

    // Send a sample outbound frame
    console.log('4. Dispatching outbound subscribe frame...');
    await fetch(`${STUDIO_URL}/api/v1/connections/conn_feed_mock/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        payload: JSON.stringify({
          action: 'subscribe',
          channel: 'crypto_orderbook',
          symbol: 'BTC-USDT',
          depth: 50,
          timestamp: Date.now(),
        }),
        opcode: 'text',
      }),
    });
    await sleep(1200);

    // 4. Launch Headless Chrome
    console.log('4. Launching headless Chrome via puppeteer-core...');
    const browser = await puppeteer.launch({
      executablePath: CHROME_PATH,
      headless: true,
      defaultViewport: {
        width: 1440,
        height: 900,
        deviceScaleFactor: 2,
      },
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--hide-scrollbars'],
    });

    const page = await browser.newPage();
    await page.goto(STUDIO_URL, { waitUntil: 'networkidle0' });
    await sleep(1500);

    // Connect to stream from UI ConnectBar
    console.log('5. Connecting to mock stream from UI ConnectBar...');
    await clickButtonWithText(page, 'Connect');
    await sleep(3500);

    // Send an outbound frame via MessageComposer
    await clickButtonWithText(page, 'Send Frame');
    await sleep(1500);

    // Click on a timeline frame row to show details
    await page.evaluate(() => {
      const rows = document.querySelectorAll('.grid.grid-cols-\\[70px_85px_80px_1fr_75px\\], .border-b.border-slate-800\\/40');
      if (rows.length > 2) {
        rows[1].click();
      }
    });
    await sleep(1000);

    // Capture 1: Main Studio & Live Timeline
    const studioImgPath = path.join(IMAGES_DIR, 'socketlens_studio.png');
    await page.screenshot({ path: studioImgPath });
    console.log('✓ Captured:', studioImgPath);

    // 5. Proxy Panel with Live Breakpoint
    console.log('6. Configuring Interception Proxy & Breakpoint Tampering...');
    await fetch(`${STUDIO_URL}/api/v1/proxy/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        local_port: 8081,
        target_url: 'ws://127.0.0.1:8080/ws/feed',
        enable_breakpoint: true,
      }),
    });
    await sleep(800);

    // Trigger a frame through proxy to catch in breakpoint
    await fetch(`${STUDIO_URL}/api/v1/connections`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: 'conn_proxy_client',
        url: 'ws://127.0.0.1:8081',
        protocol: 'ws',
        auto_reconnect: false,
      }),
    });
    await sleep(1000);

    await fetch(`${STUDIO_URL}/api/v1/connections/conn_proxy_client/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        payload: JSON.stringify({
          order_id: 'ord_9941',
          side: 'BUY',
          quantity: 2.5,
          price: 64250.0,
        }),
        opcode: 'text',
      }),
    });
    await sleep(1000);

    // Switch to Proxy Tab
    await clickButtonWithText(page, 'Proxy');
    await sleep(1000);

    const proxyImgPath = path.join(IMAGES_DIR, 'socketlens_proxy.png');
    await page.screenshot({ path: proxyImgPath });
    console.log('✓ Captured:', proxyImgPath);

    // 6. LLM Inspector Panel
    console.log('7. Running LLM Stream Telemetry...');
    await fetch(`${STUDIO_URL}/api/v1/mock/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        port: 8080,
        route: '/ws/feed',
        mode: 'llm-tokens',
        rate: 35,
      }),
    });
    await sleep(500);

    await fetch(`${STUDIO_URL}/api/v1/connections`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: 'conn_llm_stream',
        url: 'http://127.0.0.1:8080/ws/feed/sse',
        protocol: 'sse',
        auto_reconnect: false,
      }),
    });
    await sleep(3500);

    await clickButtonWithText(page, 'LLM');
    await sleep(1200);

    const llmImgPath = path.join(IMAGES_DIR, 'socketlens_llm.png');
    await page.screenshot({ path: llmImgPath });
    console.log('✓ Captured:', llmImgPath);

    // 7. Schema Validator Panel
    console.log('8. Capturing Schema Validation...');
    await clickButtonWithText(page, 'Schema');
    await sleep(800);

    await clickButtonWithText(page, 'Apply & Validate');
    await sleep(800);

    await clickButtonWithText(page, 'Run Test');
    await sleep(800);

    const schemaImgPath = path.join(IMAGES_DIR, 'socketlens_schema.png');
    await page.screenshot({ path: schemaImgPath });
    console.log('✓ Captured:', schemaImgPath);

    // 8. Benchmark Runner Panel
    console.log('9. Running Concurrency Stress Benchmark...');
    await fetch(`${STUDIO_URL}/api/v1/bench/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        url: 'ws://127.0.0.1:8080/ws/feed',
        clients: 150,
        duration: 3000000000,
        ramp_up: 500000000,
        send_interval: 200000000,
        payload: '{"type":"ping"}',
      }),
    });
    await sleep(3800);

    await clickButtonWithText(page, 'Bench');
    await sleep(1200);

    const benchImgPath = path.join(IMAGES_DIR, 'socketlens_bench.png');
    await page.screenshot({ path: benchImgPath });
    console.log('✓ Captured:', benchImgPath);

    // 9. Record Animated Demo GIF
    console.log('10. Recording animated frames for GIF demo...');
    await clickButtonWithText(page, 'Stream');
    await sleep(1000);

    const TOTAL_FRAMES = 26;
    for (let i = 0; i < TOTAL_FRAMES; i++) {
      const frameFile = path.join(TEMP_FRAMES_DIR, `frame_${String(i).padStart(3, '0')}.png`);
      await page.screenshot({ path: frameFile });

      if (i % 6 === 0) {
        const rows = await page.$$('.border-b.border-slate-800\\/50');
        const targetRow = Math.min((i / 6) + 1, rows.length - 1);
        if (rows[targetRow]) {
          await rows[targetRow].click();
        }
      }
      await sleep(160);
    }

    try {
      await Promise.race([
        browser.close(),
        new Promise((resolve) => setTimeout(resolve, 2000)),
      ]);
    } catch (e) {}

    // 10. Compile GIF with FFmpeg
    console.log('11. Compiling animated demo GIF with FFmpeg...');
    const gifPath = path.join(IMAGES_DIR, 'socketlens_demo.gif');
    const inputPattern = path.join(TEMP_FRAMES_DIR, 'frame_%03d.png').replace(/\\/g, '/');
    const outputGif = gifPath.replace(/\\/g, '/');
    execSync(
      `ffmpeg -y -framerate 6 -i "${inputPattern}" -vf "scale=1200:-1:flags=lanczos,split[s0][s1];[s0]palettegen=stats_mode=diff[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3" "${outputGif}"`,
      { stdio: 'inherit' }
    );
    console.log('✓ Successfully created demo GIF:', gifPath);

    fs.rmSync(TEMP_FRAMES_DIR, { recursive: true, force: true });
    console.log('✓ Cleaned temp directory.');
  } finally {
    console.log('Terminating SocketLens daemon...');
    try {
      execSync('taskkill /F /IM socketlens.exe /IM chrome.exe', { stdio: 'ignore' });
    } catch (e) {}
  }
}

main().catch((err) => {
  console.error('Fatal error during capture:', err);
  process.exit(1);
});
