// Generate a 2.0-second 440Hz sine wave WAV file for testing
const fs = require('fs');
const path = require('path');

const sampleRate = 44100;
const durationSec = 2.0;
const numSamples = Math.floor(sampleRate * durationSec);
const dataSize = numSamples * 2; // 16-bit mono = 2 bytes per sample
const fileSize = 44 + dataSize;

const buffer = Buffer.alloc(fileSize);

// RIFF header
buffer.write('RIFF', 0);
buffer.writeUInt32LE(fileSize - 8, 4);
buffer.write('WAVE', 8);

// fmt chunk
buffer.write('fmt ', 12);
buffer.writeUInt32LE(16, 16);      // Subchunk1Size (16 for PCM)
buffer.writeUInt16LE(1, 20);       // AudioFormat (1 = PCM)
buffer.writeUInt16LE(1, 22);       // NumChannels (1 = Mono)
buffer.writeUInt32LE(sampleRate, 24); // SampleRate
buffer.writeUInt32LE(sampleRate * 2, 28); // ByteRate (SampleRate * NumChannels * BitsPerSample/8)
buffer.writeUInt16LE(2, 32);       // BlockAlign (NumChannels * BitsPerSample/8)
buffer.writeUInt16LE(16, 34);      // BitsPerSample

// data chunk
buffer.write('data', 36);
buffer.writeUInt32LE(dataSize, 40);

const freq = 440.0;
for (let i = 0; i < numSamples; i++) {
  const t = i / sampleRate;
  const sample = Math.sin(2 * Math.PI * freq * t) * 0.5; // amplitude 0.5
  const intSample = Math.max(-32768, Math.min(32767, Math.floor(sample * 32767)));
  buffer.writeInt16LE(intSample, 44 + i * 2);
}

const outPath = path.join(__dirname, 'sample-audio.wav');
fs.writeFileSync(outPath, buffer);
console.log(`Generated ${outPath} (${fileSize} bytes, ${durationSec}s duration)`);
