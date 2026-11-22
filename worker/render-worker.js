/**
 * Reel Studio - Render Worker
 * Isolated Node.js process managing Edge / Playwright automation and ffmpeg encoding.
 * Communicates with Go host via newline-delimited JSON over stdin/stdout.
 */

const { chromium } = require('playwright');
const { spawn } = require('child_process');
const fs = require('fs');
const readline = require('readline');

const VIRTUAL_TIME = `(() => {
  const T0 = Date.now();
  let vt = 0, seq = 0;
  const timers = new Map();
  let raf = [];
  performance.now = () => vt;
  const RealDate = Date;
  class VDate extends RealDate {
    constructor(...a) { if (a.length) super(...a); else super(T0 + vt); }
    static now() { return T0 + vt; }
  }
  window.Date = VDate;
  window.setTimeout = (fn, ms = 0, ...args) => { const id = ++seq; timers.set(id, { at: vt + Math.max(0, +ms || 0), fn, args }); return id; };
  window.setInterval = (fn, ms = 0, ...args) => { const id = ++seq; const every = Math.max(1, +ms || 0); timers.set(id, { at: vt + every, every, fn, args }); return id; };
  window.clearTimeout = window.clearInterval = id => timers.delete(id);
  window.requestAnimationFrame = fn => { raf.push(fn); return raf.length; };
  window.cancelAnimationFrame = () => {};

  const anims = new WeakMap();
  function syncAnimations() {
    for (const a of document.getAnimations()) {
      if (!anims.has(a)) { anims.set(a, vt); a.pause(); }
      a.currentTime = vt - anims.get(a);
    }
  }
  window.__vtAdvance = (dt) => {
    const target = vt + dt;
    for (;;) {
      let next = null;
      for (const [id, t] of timers) if (t.at <= target && (!next || t.at < next[1].at)) next = [id, t];
      if (!next) break;
      const [id, t] = next;
      vt = Math.max(vt, t.at);
      if (t.every) t.at += t.every; else timers.delete(id);
      try { t.fn(...t.args); } catch (e) { console.error(e); }
    }
    vt = target;
    const q = raf; raf = [];
    for (const fn of q) { try { fn(vt); } catch (e) { console.error(e); } }
    syncAnimations();
    return vt;
  };
})();`;

function send(msg) {
  try {
    process.stdout.write(JSON.stringify(msg) + '\n');
  } catch (err) {
    // ignore stdout write errors on broken pipe
  }
}

function log(id, level, message) {
  send({ type: 'log', id: id || null, level, message });
}

let activeRender = null; // { id, cancel: () => void }

async function renderJob(params) {
  const {
    id,
    html,
    audio,
    out,
    audioDuration,
    width = 1080,
    height = 1920,
    dpr = 1.0,
    fps = 30,
    crf = 18,
    preset = 'veryfast',
    imageFormat = 'png',
    tail = 1.0,
    ffmpegPath = 'ffmpeg',
    browserChannel = 'msedge',
    browserExecutable = '',
    iframeReplacements = {}
  } = params;

  let cancelled = false;
  let browser = null;
  let ff = null;

  activeRender = {
    id,
    cancel: () => {
      cancelled = true;
      log(id, 'warn', `Cancellation requested for job ${id}`);
      if (ff && ff.exitCode === null) {
        try { ff.kill('SIGKILL'); } catch (e) {}
      }
      if (browser) {
        try { browser.close().catch(() => {}); } catch (e) {}
      }
      if (fs.existsSync(out)) {
        try { fs.unlinkSync(out); } catch (e) {}
      }
    }
  };

  try {
    log(id, 'info', `Starting render job ${id}: ${width}x${height} @ ${fps}fps, crf=${crf}, format=${imageFormat}`);
    send({ type: 'progress', id, phase: 'Initializing browser', pct: 1 });

    const totalDuration = (Number(audioDuration) || 0) + Number(tail || 1.0);
    const totalFrames = Math.ceil(totalDuration * fps);
    const dt = 1000 / fps;

    const launchOptions = {
      args: ['--autoplay-policy=no-user-gesture-required', '--hide-scrollbars', '--disable-web-security']
    };

    if (browserExecutable && fs.existsSync(browserExecutable)) {
      launchOptions.executablePath = browserExecutable;
    } else if (browserChannel) {
      launchOptions.channel = browserChannel;
    }

    try {
      browser = await chromium.launch(launchOptions);
    } catch (launchErr) {
      // If launch failed with specified channel, fallback to default chromium launch
      log(id, 'warn', `Failed launching browser with ${launchOptions.channel || launchOptions.executablePath}: ${launchErr.message}`);
      browser = await chromium.launch({
        args: ['--autoplay-policy=no-user-gesture-required', '--hide-scrollbars', '--disable-web-security']
      });
    }

    if (cancelled) return;

    // Viewport calculation: 1440p uses 1080x1920 layout with 1.3333 scale factor
    const scale = width / 1080;
    const vw = scale > 1 ? 1080 : width;
    const vh = scale > 1 ? 1920 : height;
    const scaleFactor = scale > 1 ? scale : (dpr || 1.0);

    const context = await browser.newContext({
      viewport: { width: vw, height: vh },
      deviceScaleFactor: scaleFactor
    });
    const page = await context.newPage();

    // Strip X-Frame-Options and Content-Security-Policy headers so iframes can embed
    await page.route('**/*', async (route) => {
      try {
        const response = await route.fetch();
        const headers = response.headers();
        delete headers['x-frame-options'];
        delete headers['content-security-policy'];
        delete headers['content-security-policy-report-only'];
        await route.fulfill({
          response,
          headers
        });
      } catch (e) {
        route.continue().catch(() => {});
      }
    });

    await page.addInitScript(VIRTUAL_TIME);

    send({ type: 'progress', id, phase: 'Opening reel', pct: 2 });
    
    // Convert path to file URL
    const fileUrl = 'file:///' + html.replace(/\\/g, '/').replace(/^\/?/, '') + '?autoplay=0';
    log(id, 'info', `Navigating to: ${fileUrl}`);
    await page.goto(fileUrl, { waitUntil: 'load', timeout: 60000 });
    await page.evaluate(() => document.fonts && document.fonts.ready);

    if (cancelled) return;

    // Apply any uploaded image replacements for blocked/broken website mockups
    if (iframeReplacements && Object.keys(iframeReplacements).length > 0) {
      log(id, 'info', `Applying ${Object.keys(iframeReplacements).length} mockup image replacements`);
      await page.evaluate((replacements) => {
        for (const [targetUrl, imageSrc] of Object.entries(replacements)) {
          const iframes = document.querySelectorAll('iframe');
          iframes.forEach(iframe => {
            const dataSrc = iframe.getAttribute('data-src') || '';
            const src = iframe.getAttribute('src') || '';
            if (
              (dataSrc && (dataSrc.includes(targetUrl) || targetUrl.includes(dataSrc))) ||
              (src && (src.includes(targetUrl) || targetUrl.includes(src)))
            ) {
              const img = document.createElement('img');
              img.src = imageSrc;
              img.className = iframe.className;
              img.style.cssText = iframe.style.cssText;
              img.style.width = '100%';
              img.style.height = '100%';
              img.style.objectFit = 'cover';
              img.style.display = 'block';
              img.setAttribute('data-replaced', targetUrl);
              if (iframe.parentNode) {
                iframe.parentNode.replaceChild(img, iframe);
              }
            }
          });
        }
      }, iframeReplacements);
    }

    const isReel = await page.evaluate(() => typeof silent !== 'undefined' && !!document.getElementById('silentBtn'));
    if (!isReel) {
      throw new Error('This HTML is not a ReelSync reel (no silent clock / Preview button found).');
    }

    // Configure silent duration and start reel
    await page.evaluate(d => { silent.duration = d; }, Number(audioDuration));
    await page.evaluate(() => {
      silent.duration = silent.duration;
      audio = silent;
      begin();
    });

    // Loading phase: advance virtual time 50ms per real 50ms until started === true
    send({ type: 'progress', id, phase: 'Loading the websites in the reel', pct: 3 });
    const loadStart = Date.now();
    for (;;) {
      if (cancelled) return;
      await page.evaluate(() => window.__vtAdvance(50));
      const hasStarted = await page.evaluate(() => typeof started !== 'undefined' && started === true);
      if (hasStarted) break;
      if (Date.now() - loadStart > 90000) {
        throw new Error('The reel did not start (loading took more than 90 s).');
      }
      await new Promise(r => setTimeout(r, 50));
    }

    // Frame 0 reset and wait for iframes to paint
    await page.evaluate(() => { silent.currentTime = 0; });
    await new Promise(r => setTimeout(r, 1500));

    if (cancelled) return;

    // Build ffmpeg arguments exactly as specified
    const ffArgs = [
      '-y',
      '-f', 'image2pipe',
      '-framerate', String(fps),
      '-i', 'pipe:0',
      '-i', audio,
      '-map', '0:v:0',
      '-map', '1:a:0',
      '-vf', `scale=${width}:${height}:flags=lanczos,format=yuv420p`,
      '-c:v', 'libx264',
      '-preset', preset,
      '-crf', String(crf),
      '-profile:v', 'high',
      '-level', '4.2',
      '-r', String(fps),
      '-g', String(fps * 2),
      '-bf', '2',
      '-c:a', 'aac',
      '-b:a', '192k',
      '-ar', '48000',
      '-ac', '2',
      '-af', `apad=whole_dur=${totalDuration.toFixed(3)}`,
      '-t', totalDuration.toFixed(3),
      '-movflags', '+faststart',
      out
    ];

    log(id, 'info', `Spawning ffmpeg: ${ffmpegPath}`);
    ff = spawn(ffmpegPath, ffArgs, { stdio: ['pipe', 'ignore', 'pipe'] });
    let fferr = '';
    ff.stderr.on('data', d => {
      fferr = (fferr + d).slice(-4000);
    });

    const ffDone = new Promise((resolve, reject) => {
      ff.on('close', code => {
        if (code === 0) resolve();
        else reject(new Error(`ffmpeg failed (exit code ${code}):\n${fferr}`));
      });
      ff.on('error', err => reject(new Error(`ffmpeg execution error: ${err.message}`)));
    });

    const cdp = await page.context().newCDPSession(page);
    const renderStartTime = Date.now();

    for (let k = 0; k < totalFrames; k++) {
      if (cancelled) return;

      await page.evaluate(d => window.__vtAdvance(d), k === 0 ? 0 : dt);

      const shot = await cdp.send('Page.captureScreenshot', imageFormat === 'png'
        ? { format: 'png', optimizeForSpeed: true }
        : { format: 'jpeg', quality: 95, optimizeForSpeed: true });

      const buf = Buffer.from(shot.data, 'base64');
      if (!ff.stdin.write(buf)) {
        await new Promise(r => ff.stdin.once('drain', r));
      }

      if (k % 5 === 0 || k === totalFrames - 1) {
        const elapsed = (Date.now() - renderStartTime) / 1000;
        const rate = (k + 1) / (elapsed || 0.001);
        const eta = Math.max(0, Math.round((totalFrames - k - 1) / rate));
        const pct = 3 + Math.round(92 * (k + 1) / totalFrames);

        send({
          type: 'progress',
          id,
          phase: 'Rendering frames',
          frame: k + 1,
          frames: totalFrames,
          pct,
          eta
        });
      }
    }

    ff.stdin.end();
    send({ type: 'progress', id, phase: 'Encoding MP4', pct: 96 });

    await ffDone;

    if (cancelled) return;

    const fileStat = fs.statSync(out);
    send({ type: 'progress', id, phase: 'Done', pct: 100 });
    send({
      type: 'done',
      id,
      result: {
        width,
        height,
        fps,
        duration: totalDuration,
        frames: totalFrames,
        size: fileStat.size
      }
    });

  } catch (err) {
    if (!cancelled) {
      log(id, 'error', `Render failed: ${err.message}`);
      send({ type: 'error', id, error: err.message });
    }
  } finally {
    activeRender = null;
    if (ff && ff.exitCode === null && !ff.stdin.writableEnded) {
      try { ff.stdin.end(); } catch (e) {}
    }
    if (browser) {
      try { await browser.close(); } catch (e) {}
    }
  }
}

// Stdin line parser
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

rl.on('line', line => {
  const trimmed = line.trim();
  if (!trimmed) return;

  try {
    const msg = JSON.parse(trimmed);
    if (msg.cmd === 'render') {
      renderJob(msg).catch(err => {
        send({ type: 'error', id: msg.id, error: err.message });
      });
    } else if (msg.cmd === 'cancel') {
      if (activeRender && activeRender.id === msg.id) {
        activeRender.cancel();
      }
    } else if (msg.cmd === 'exit') {
      if (activeRender) activeRender.cancel();
      process.exit(0);
    }
  } catch (err) {
    send({ type: 'error', id: null, error: `Invalid IPC command: ${err.message}` });
  }
});

process.on('SIGINT', () => {
  if (activeRender) activeRender.cancel();
  process.exit(0);
});

process.on('SIGTERM', () => {
  if (activeRender) activeRender.cancel();
  process.exit(0);
});

log(null, 'info', 'Render worker initialized and ready.');
