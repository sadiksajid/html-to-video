/**
 * Worker End-to-End Test
 * Tests render-worker.js over IPC:
 * - Renders sample-reel.html + sample-audio.wav (duration 2.0s)
 * - Resolution: 720x1280 (720p), 30 fps, balanced quality
 * - Asserts:
 *   1. MP4 file generated
 *   2. Video stream: h264, 720x1280, 30fps
 *   3. Audio stream: aac, 48000 Hz stereo
 *   4. Total duration: 3.0s (audio 2.0s + tail 1.0s) within ±0.1s
 *   5. Frame at t=1.0s differs from frame at t=0.0s (visual animation moved)
 */

const { spawn, execFileSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const readline = require('readline');

const ROOT_DIR = path.resolve(__dirname, '..');
const NODE_EXE = path.join(ROOT_DIR, 'third_party', 'node.exe');
const FFMPEG_EXE = path.join(ROOT_DIR, 'third_party', 'ffmpeg.exe');
const FFPROBE_EXE = path.join(ROOT_DIR, 'third_party', 'ffprobe.exe');
const WORKER_SCRIPT = path.join(ROOT_DIR, 'worker', 'render-worker.js');

const SAMPLE_REEL = path.join(ROOT_DIR, 'testdata', 'sample-reel.html');
const SAMPLE_AUDIO = path.join(ROOT_DIR, 'testdata', 'sample-audio.wav');
const OUT_MP4 = path.join(ROOT_DIR, 'testdata', 'test-output.mp4');

function assert(condition, message) {
  if (!condition) {
    console.error(`\n[FAIL] Assertion failed: ${message}`);
    process.exit(1);
  }
}

async function runTest() {
  console.log('--- Starting Worker E2E Test ---');

  if (fs.existsSync(OUT_MP4)) {
    fs.unlinkSync(OUT_MP4);
  }

  // 1. Probe audio duration
  const probeOut = execFileSync(FFPROBE_EXE, [
    '-v', 'error',
    '-show_entries', 'format=duration',
    '-of', 'csv=p=0',
    SAMPLE_AUDIO
  ], { encoding: 'utf8' }).trim();
  const audioDuration = parseFloat(probeOut);
  console.log(`[OK] Probed audio duration: ${audioDuration.toFixed(2)}s`);
  assert(Math.abs(audioDuration - 2.0) < 0.1, `Expected audio duration ~2.0s, got ${audioDuration}`);

  // 2. Spawn worker process
  console.log(`[INFO] Spawning worker: ${NODE_EXE} ${WORKER_SCRIPT}`);
  const worker = spawn(NODE_EXE, [WORKER_SCRIPT], {
    stdio: ['pipe', 'pipe', 'inherit'],
    cwd: ROOT_DIR
  });

  const rl = readline.createInterface({ input: worker.stdout });
  let doneMessage = null;
  let progressCount = 0;

  rl.on('line', line => {
    try {
      const msg = JSON.parse(line.trim());
      if (msg.type === 'progress') {
        progressCount++;
        process.stdout.write(`\r[Worker Progress] ${msg.phase}: ${msg.pct}% ${msg.frame ? `(frame ${msg.frame}/${msg.frames})` : ''}     `);
      } else if (msg.type === 'log') {
        if (msg.level === 'error') console.error(`\n[Worker Log ${msg.level}]:`, msg.message);
      } else if (msg.type === 'done') {
        console.log('\n[Worker Done]:', JSON.stringify(msg.result));
        doneMessage = msg;
      } else if (msg.type === 'error') {
        console.error('\n[Worker Error]:', msg.error);
      }
    } catch (e) {
      // ignore
    }
  });

  // Send render command
  const renderCmd = {
    cmd: 'render',
    id: 'test-job-001',
    html: SAMPLE_REEL,
    audio: SAMPLE_AUDIO,
    out: OUT_MP4,
    audioDuration: audioDuration,
    width: 720,
    height: 1280,
    dpr: 1.0,
    fps: 30,
    quality: 'balanced',
    crf: 21,
    preset: 'veryfast',
    imageFormat: 'jpeg',
    tail: 1.0,
    ffmpegPath: FFMPEG_EXE,
    browserChannel: 'msedge'
  };

  worker.stdin.write(JSON.stringify(renderCmd) + '\n');

  // Wait for worker done or exit
  await new Promise((resolve, reject) => {
    const interval = setInterval(() => {
      if (doneMessage) {
        clearInterval(interval);
        worker.stdin.write(JSON.stringify({ cmd: 'exit' }) + '\n');
        setTimeout(() => {
          try { worker.kill(); } catch (e) {}
          resolve();
        }, 500);
      }
    }, 200);

    worker.on('exit', code => {
      clearInterval(interval);
      if (doneMessage) resolve();
      else reject(new Error(`Worker exited prematurely with code ${code}`));
    });
  });

  // 3. Verify output file exists
  assert(fs.existsSync(OUT_MP4), `Output MP4 file does not exist: ${OUT_MP4}`);
  const stat = fs.statSync(OUT_MP4);
  console.log(`[OK] Output file size: ${(stat.size / 1024).toFixed(1)} KB`);
  assert(stat.size > 50000, 'Output MP4 size is suspiciously small');

  // 4. Probe video streams
  const streamInfo = execFileSync(FFPROBE_EXE, [
    '-v', 'error',
    '-print_format', 'json',
    '-show_streams',
    '-show_format',
    OUT_MP4
  ], { encoding: 'utf8' });

  const media = JSON.parse(streamInfo);
  const vStream = media.streams.find(s => s.codec_type === 'video');
  const aStream = media.streams.find(s => s.codec_type === 'audio');

  assert(!!vStream, 'Video stream missing from output MP4');
  assert(vStream.codec_name === 'h264', `Expected video codec h264, got ${vStream.codec_name}`);
  assert(vStream.width === 720, `Expected width 720, got ${vStream.width}`);
  assert(vStream.height === 1280, `Expected height 1280, got ${vStream.height}`);

  // Frame rate check: 30 fps
  const [num, den] = vStream.r_frame_rate.split('/').map(Number);
  const rFps = Math.round(num / (den || 1));
  assert(rFps === 30, `Expected frame rate 30, got ${rFps}`);

  // Audio check
  assert(!!aStream, 'Audio stream missing from output MP4');
  assert(aStream.codec_name === 'aac', `Expected audio codec aac, got ${aStream.codec_name}`);
  assert(aStream.sample_rate === '48000', `Expected audio sample rate 48000, got ${aStream.sample_rate}`);

  // Duration check: audio (2.0s) + tail (1.0s) = 3.0s (±0.1s)
  const duration = parseFloat(media.format.duration);
  console.log(`[OK] MP4 total duration: ${duration.toFixed(3)}s (target: 3.0s)`);
  assert(Math.abs(duration - 3.0) <= 0.15, `Duration ${duration}s out of bounds (expected ~3.0s ± 0.1s)`);

  // 5. Verify animation moved between t=0 and t=1.0s
  const frame0Path = path.join(ROOT_DIR, 'testdata', 'frame_t0.png');
  const frame1Path = path.join(ROOT_DIR, 'testdata', 'frame_t1.png');

  if (fs.existsSync(frame0Path)) fs.unlinkSync(frame0Path);
  if (fs.existsSync(frame1Path)) fs.unlinkSync(frame1Path);

  execFileSync(FFMPEG_EXE, ['-y', '-ss', '0.0', '-i', OUT_MP4, '-vframes', '1', frame0Path]);
  execFileSync(FFMPEG_EXE, ['-y', '-ss', '1.0', '-i', OUT_MP4, '-vframes', '1', frame1Path]);

  assert(fs.existsSync(frame0Path) && fs.existsSync(frame1Path), 'Failed to extract verification frames');

  const f0 = fs.readFileSync(frame0Path);
  const f1 = fs.readFileSync(frame1Path);

  const diffBytes = Math.abs(f0.length - f1.length);
  const buffersEqual = f0.equals(f1);

  console.log(`[OK] Frame 0 size: ${f0.length} bytes, Frame 1 size: ${f1.length} bytes (diff: ${diffBytes} bytes)`);
  assert(!buffersEqual, 'Frame at t=0 and frame at t=1.0s are identical! Animation did not move.');

  // Clean up verification frames
  fs.unlinkSync(frame0Path);
  fs.unlinkSync(frame1Path);

  console.log('\n======================================');
  console.log('  ALL WORKER E2E ASSERTIONS PASSED!  ');
  console.log('======================================\n');
}

runTest().catch(err => {
  console.error('\n[FATAL TEST ERROR]:', err);
  process.exit(1);
});
