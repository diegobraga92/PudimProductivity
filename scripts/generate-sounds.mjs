#!/usr/bin/env node
/**
 * Generates the default Soundscape ambient sound library as MP3 loops.
 *
 * These are placeholder loops generated procedurally (no external samples):
 * a distinct texture per sound, engineered to loop seamlessly. They make the
 * backend sound-serving pipeline testable end-to-end. Replace them with
 * properly mastered, license-clean loop files before shipping.
 *
 * Usage: node scripts/generate-sounds.mjs   (requires ffmpeg on PATH)
 * Output: backend/sounds/<sound-id>.mp3
 */
import { execFileSync } from "node:child_process";
import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const outDir = resolve(root, "backend", "sounds");
const tmpDir = join(outDir, ".tmp-wav");

const SAMPLE_RATE = 22050;
const DURATION = 8; // seconds — every modulation LFO below uses an integer # of cycles
const N = SAMPLE_RATE * DURATION;

const rand = Math.random;

// ── DSP building blocks ────────────────────────────────────────────────────

function fillWhite(buf) {
  for (let i = 0; i < N; i++) buf[i] = rand() * 2 - 1;
}

function fillPink(buf) {
  // Paul Kellet's economical pink-noise filter.
  let b0 = 0, b1 = 0, b2 = 0, b3 = 0, b4 = 0, b5 = 0, b6 = 0;
  for (let i = 0; i < N; i++) {
    const w = rand() * 2 - 1;
    b0 = 0.99886 * b0 + w * 0.0555179;
    b1 = 0.99332 * b1 + w * 0.0750759;
    b2 = 0.969 * b2 + w * 0.153852;
    b3 = 0.8665 * b3 + w * 0.3104856;
    b4 = 0.55 * b4 + w * 0.5329522;
    b5 = -0.7616 * b5 - w * 0.016898;
    buf[i] = (b0 + b1 + b2 + b3 + b4 + b5 + b6 + w * 0.5362) * 0.11;
    b6 = w * 0.115926;
  }
}

function fillBrown(buf) {
  let last = 0;
  for (let i = 0; i < N; i++) {
    last = (last + (rand() * 2 - 1) * 0.02) * 0.997;
    buf[i] = last * 3.5;
  }
}

function onePoleLP(buf, alpha) {
  let y = 0;
  for (let i = 0; i < N; i++) {
    y += alpha * (buf[i] - y);
    buf[i] = y;
  }
}

function onePoleHP(buf, alpha) {
  let y = 0, prev = 0;
  for (let i = 0; i < N; i++) {
    const x = buf[i];
    y = alpha * (y + x - prev);
    buf[i] = y;
    prev = x;
  }
}

/** Multiply by a raised sinusoid (0..1), with a whole number of cycles over N. */
function mulSinusoid(buf, freq, phase = 0) {
  for (let i = 0; i < N; i++) {
    buf[i] *= 0.5 + 0.5 * Math.sin((2 * Math.PI * freq * i) / SAMPLE_RATE + phase);
  }
}

function addSinusoid(buf, freq, amp, phase = 0) {
  for (let i = 0; i < N; i++) {
    buf[i] += amp * Math.sin((2 * Math.PI * freq * i) / SAMPLE_RATE + phase);
  }
}

/** Sparse short decaying impulses — fire-like crackle. */
function addCrackle(buf, density) {
  let i = 0;
  while (i < N) {
    if (rand() < density) {
      const amp = 0.4 + rand() * 0.6;
      const len = 60 + Math.floor(rand() * 180);
      for (let k = 0; k < len && i + k < N; k++) {
        buf[i + k] += amp * Math.exp(-k / 30) * (rand() * 2 - 1);
      }
      i += len;
    } else {
      i += 20 + Math.floor(rand() * 400);
    }
  }
}

function normalize(buf, peak = 0.85) {
  let m = 0;
  for (let i = 0; i < N; i++) m = Math.max(m, Math.abs(buf[i]));
  if (m > 0) for (let i = 0; i < N; i++) buf[i] *= peak / m;
}

/** Crossfade the tail into the head so the loop has no seam click. */
function makeSeamless(buf, blendMs = 40) {
  const blend = Math.floor((blendMs / 1000) * SAMPLE_RATE);
  for (let i = 0; i < blend; i++) {
    const t = i / blend;
    buf[i] = buf[i] * t + buf[N - blend + i] * (1 - t);
  }
}

// ── Per-sound generators (ids must match the backend sound catalog) ────────

const generators = {
  "white-noise": () => {
    const b = new Float32Array(N);
    fillWhite(b);
    return b;
  },
  "pink-noise": () => {
    const b = new Float32Array(N);
    fillPink(b);
    normalize(b);
    return b;
  },
  "brown-noise": () => {
    const b = new Float32Array(N);
    fillBrown(b);
    normalize(b);
    return b;
  },
  rain: () => {
    const b = new Float32Array(N);
    fillWhite(b);
    onePoleLP(b, 0.25); // dulled hiss
    mulSinusoid(b, 0.5); // 4 swell cycles per loop
    normalize(b);
    return b;
  },
  ocean: () => {
    const b = new Float32Array(N);
    fillPink(b);
    mulSinusoid(b, 0.125); // one slow swell per loop
    normalize(b);
    return b;
  },
  wind: () => {
    const b = new Float32Array(N);
    fillWhite(b);
    onePoleLP(b, 0.12);
    onePoleHP(b, 0.35); // band-ish howl
    mulSinusoid(b, 0.25); // gust layers (integer cycles)
    mulSinusoid(b, 0.125, 1.0);
    normalize(b);
    return b;
  },
  campfire: () => {
    const b = new Float32Array(N);
    fillBrown(b);
    onePoleLP(b, 0.2);
    mulSinusoid(b, 0.25);
    addCrackle(b, 0.02);
    normalize(b);
    return b;
  },
  "binaural-beat": () => {
    const b = new Float32Array(N);
    addSinusoid(b, 200, 0.45);
    addSinusoid(b, 210, 0.45); // 10 Hz beat pair
    return b;
  },
  "isochronic-tone": () => {
    const b = new Float32Array(N);
    const carrier = 200;
    const pulse = 10; // integer cycles over 8 s
    for (let i = 0; i < N; i++) {
      const t = i / SAMPLE_RATE;
      const env = 0.5 + 0.5 * Math.sin(2 * Math.PI * pulse * t);
      b[i] = 0.6 * env * Math.sin(2 * Math.PI * carrier * t);
    }
    return b;
  },
  "meditation-bowl": () => {
    const b = new Float32Array(N);
    const partials = [
      [260, 0.5],
      [728, 0.18],
      [1404, 0.09],
      [2210, 0.045],
    ];
    for (const [f, a] of partials) addSinusoid(b, f, a);
    mulSinusoid(b, 0.125, 0.5); // gentle shimmer
    return b;
  },
  "ambient-pad": () => {
    const b = new Float32Array(N);
    // A2 + E3 + A3 + ~C#4, frequencies chosen for integer cycles per loop.
    const chord = [
      [110, 0.25],
      [165, 0.2],
      [220, 0.18],
      [275, 0.14],
    ];
    for (const [f, a] of chord) addSinusoid(b, f, a);
    mulSinusoid(b, 0.25, 1.0);
    return b;
  },
};

const SOUND_IDS = [
  "white-noise",
  "pink-noise",
  "brown-noise",
  "rain",
  "ocean",
  "wind",
  "campfire",
  "binaural-beat",
  "isochronic-tone",
  "meditation-bowl",
  "ambient-pad",
];

// ── WAV writer (16-bit PCM, mono) ──────────────────────────────────────────

function writeWav(path, buf) {
  const dataSize = buf.length * 2;
  const header = Buffer.alloc(44);
  header.write("RIFF", 0);
  header.writeUInt32LE(36 + dataSize, 4);
  header.write("WAVE", 8);
  header.write("fmt ", 12);
  header.writeUInt32LE(16, 16); // fmt chunk size
  header.writeUInt16LE(1, 20); // PCM
  header.writeUInt16LE(1, 22); // mono
  header.writeUInt32LE(SAMPLE_RATE, 24);
  header.writeUInt32LE(SAMPLE_RATE * 2, 28); // byte rate
  header.writeUInt16LE(2, 32); // block align
  header.writeUInt16LE(16, 34); // bits per sample
  header.write("data", 36);
  header.writeUInt32LE(dataSize, 40);
  const pcm = Buffer.alloc(dataSize);
  for (let i = 0; i < buf.length; i++) {
    const s = Math.max(-1, Math.min(1, buf[i]));
    pcm.writeInt16LE(Math.round(s * 32767), i * 2);
  }
  writeFileSync(path, Buffer.concat([header, pcm]));
}

// ── Main ───────────────────────────────────────────────────────────────────

mkdirSync(outDir, { recursive: true });
mkdirSync(tmpDir, { recursive: true });
try {
  for (const id of SOUND_IDS) {
    const buf = generators[id]();
    makeSeamless(buf);
    normalize(buf);
    const wav = join(tmpDir, `${id}.wav`);
    writeWav(wav, buf);
    const mp3 = join(outDir, `${id}.mp3`);
    execFileSync("ffmpeg", [
      "-y",
      "-loglevel",
      "error",
      "-i",
      wav,
      "-ac",
      "2",
      "-ar",
      "44100",
      "-b:a",
      "128k",
      mp3,
    ]);
    console.log(`generated ${mp3}`);
  }
} finally {
  rmSync(tmpDir, { recursive: true, force: true });
}
console.log(`\nDone: ${SOUND_IDS.length} loops written to ${outDir}`);

