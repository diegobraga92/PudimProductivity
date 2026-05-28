/**
 * Web Audio API soundscape engine.
 *
 * Generates ambient sounds entirely in the browser using AudioContext.
 * No external files, no streaming, no third-party services.
 *
 * Supported sounds:
 *   - white-noise  : flat spectrum, hiss-like
 *   - pink-noise   : -3dB/octave rolloff, warmer
 *   - brown-noise  : -6dB/octave rolloff, deep rumble
 *   - rain         : filtered noise with gentle modulation
 *   - ocean        : filtered noise with slow wave modulation
 */

export type SoundID = "white-noise" | "pink-noise" | "brown-noise" | "rain" | "ocean";

interface ActiveSource {
  source: AudioBufferSourceNode;
  gain: GainNode;
}

class SoundscapeEngine {
  private ctx: AudioContext | null = null;
  private active: Map<SoundID, ActiveSource> = new Map();
  private masterGain: GainNode | null = null;

  /** Ensure AudioContext is created (must be called from a user gesture). */
  private ensureContext(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext();
      this.masterGain = this.ctx.createGain();
      this.masterGain.gain.value = 0.5;
      this.masterGain.connect(this.ctx.destination);
    }
    if (this.ctx.state === "suspended") {
      this.ctx.resume();
    }
    return this.ctx;
  }

  /** Play a sound. If already playing, it's a no-op. */
  play(id: SoundID): void {
    if (this.active.has(id)) return;

    const ctx = this.ensureContext();
    const gain = ctx.createGain();
    gain.gain.value = 0.5;
    gain.connect(this.masterGain!);

    const source = this.createSource(ctx, id, gain);
    if (!source) return;

    this.active.set(id, { source, gain });
  }

  /** Stop a specific sound. */
  stop(id: SoundID): void {
    const entry = this.active.get(id);
    if (!entry) return;
    try {
      entry.source.stop();
    } catch {
      // already stopped
    }
    entry.source.disconnect();
    entry.gain.disconnect();
    this.active.delete(id);
  }

  /** Stop all sounds. */
  stopAll(): void {
    for (const id of this.active.keys()) {
      this.stop(id);
    }
  }

  /** Set master volume (0–1). */
  setVolume(v: number): void {
    if (this.masterGain) {
      this.masterGain.gain.value = Math.max(0, Math.min(1, v));
    }
  }

  /** Set volume for a specific sound (0–1). */
  setSoundVolume(id: SoundID, v: number): void {
    const entry = this.active.get(id);
    if (entry) {
      entry.gain.gain.value = Math.max(0, Math.min(1, v));
    }
  }

  /** Check if a sound is currently playing. */
  isPlaying(id: SoundID): boolean {
    return this.active.has(id);
  }

  /** Clean up everything. */
  destroy(): void {
    this.stopAll();
    if (this.ctx) {
      this.ctx.close();
      this.ctx = null;
      this.masterGain = null;
    }
  }

  // ─── Sound generators ──────────────────────────────────────

  private createSource(ctx: AudioContext, id: SoundID, gain: GainNode): AudioBufferSourceNode | null {
    switch (id) {
      case "white-noise":
        return this.makeNoiseSource(ctx, gain, 1.0);
      case "pink-noise":
        return this.makePinkNoiseSource(ctx, gain);
      case "brown-noise":
        return this.makeBrownNoiseSource(ctx, gain);
      case "rain":
        return this.makeRainSource(ctx, gain);
      case "ocean":
        return this.makeOceanSource(ctx, gain);
      default:
        return null;
    }
  }

  /** Create an infinite-looping buffer of white noise. */
  private makeNoiseBuffer(ctx: AudioContext, duration: number): AudioBuffer {
    const sampleRate = ctx.sampleRate;
    const length = sampleRate * duration;
    const buffer = ctx.createBuffer(1, length, sampleRate);
    const data = buffer.getChannelData(0);
    for (let i = 0; i < length; i++) {
      data[i] = Math.random() * 2 - 1;
    }
    return buffer;
  }

  /** White noise — flat spectrum. */
  private makeNoiseSource(ctx: AudioContext, gain: GainNode, filterAmount: number): AudioBufferSourceNode {
    const buffer = this.makeNoiseBuffer(ctx, 4);
    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.loop = true;

    // Optional low-pass filter to tame high end
    if (filterAmount < 1) {
      const filter = ctx.createBiquadFilter();
      filter.type = "lowpass";
      filter.frequency.value = 8000 * filterAmount + 2000;
      source.connect(filter);
      filter.connect(gain);
    } else {
      source.connect(gain);
    }

    source.start();
    return source;
  }

  /** Pink noise — -3dB/octave rolloff via Voss-McCartney algorithm. */
  private makePinkNoiseSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    const sampleRate = ctx.sampleRate;
    const duration = 4;
    const length = sampleRate * duration;
    const buffer = ctx.createBuffer(1, length, sampleRate);
    const data = buffer.getChannelData(0);

    // Voss-McCartney pink noise (simplified 7-octave)
    const b: number[] = [0, 0, 0, 0, 0, 0, 0];
    let white: number;
    for (let i = 0; i < length; i++) {
      white = Math.random() * 2 - 1;
      for (let j = 0; j < 7; j++) {
        const mask = 1 << j;
        if ((i & mask) === 0) b[j] = Math.random() * 2 - 1;
      }
      let sum = 0;
      for (let j = 0; j < 7; j++) sum += b[j];
      data[i] = (white + sum) / 8;
    }

    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.loop = true;
    source.connect(gain);
    source.start();
    return source;
  }

  /** Brown noise — -6dB/octave, integrated random walk. */
  private makeBrownNoiseSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    const sampleRate = ctx.sampleRate;
    const duration = 4;
    const length = sampleRate * duration;
    const buffer = ctx.createBuffer(1, length, sampleRate);
    const data = buffer.getChannelData(0);

    let lastOut = 0;
    for (let i = 0; i < length; i++) {
      const white = Math.random() * 2 - 1;
      lastOut = (lastOut + 0.02 * white) / 1.02;
      data[i] = lastOut * 3.5; // amplify
    }

    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.loop = true;

    // High-pass to remove DC offset
    const hp = ctx.createBiquadFilter();
    hp.type = "highpass";
    hp.frequency.value = 40;

    source.connect(hp);
    hp.connect(gain);
    source.start();
    return source;
  }

  /** Rain — layered: background wash + randomized droplets. */
  private makeRainSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Background wash ──
    const washBuffer = this.makeNoiseBuffer(ctx, 4);
    const washSource = ctx.createBufferSource();
    washSource.buffer = washBuffer;
    washSource.loop = true;

    const washGain = ctx.createGain();
    washGain.gain.value = 0.35;

    const washLP = ctx.createBiquadFilter();
    washLP.type = "lowpass";
    washLP.frequency.value = 3000;
    washLP.Q.value = 0.5;

    washSource.connect(washLP);
    washLP.connect(washGain);
    washGain.connect(gain);
    washSource.start();

    // ── Layer 2: Droplet scheduler ──
    // Creates short noise bursts at random intervals with randomized pitch.
    const dropletGain = ctx.createGain();
    dropletGain.gain.value = 0.5;
    dropletGain.connect(gain);

    let dropletInterval: ReturnType<typeof setInterval> | null = null;

    const scheduleDroplet = () => {
      if (!this.active.has("rain" as SoundID)) return;

      const duration = 0.04 + Math.random() * 0.06; // 40-100ms
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const dSource = ctx.createBufferSource();
      dSource.buffer = buf;

      // Randomized bandpass — higher = sharper "tick", lower = softer "tap"
      const bp = ctx.createBiquadFilter();
      bp.type = "bandpass";
      bp.frequency.value = 1800 + Math.random() * 3200;
      bp.Q.value = 1.5 + Math.random() * 2;

      // Quick gain envelope
      const env = ctx.createGain();
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(0.6 + Math.random() * 0.4, ctx.currentTime + 0.005);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      dSource.connect(bp);
      bp.connect(env);
      env.connect(dropletGain);
      dSource.start(ctx.currentTime);
      dSource.stop(ctx.currentTime + duration);

      // Schedule next droplet in 10-40ms
      const nextDelay = 10 + Math.random() * 30;
      dropletInterval = setTimeout(scheduleDroplet, nextDelay);
    };

    // Start scheduling droplets
    dropletInterval = setTimeout(scheduleDroplet, 50);

    // Store the interval so we can clear it on stop
    // We override the source's stop to also clear the scheduler
    const originalStop = washSource.stop.bind(washSource);
    washSource.stop = ((...args: any[]) => {
      if (dropletInterval) {
        clearTimeout(dropletInterval);
        dropletInterval = null;
      }
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return washSource;
  }

  /** Ocean — layered: low-frequency wash + randomized wave crashes. */
  private makeOceanSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Deep wash ──
    const washBuffer = this.makeNoiseBuffer(ctx, 4);
    const washSource = ctx.createBufferSource();
    washSource.buffer = washBuffer;
    washSource.loop = true;

    const washLP = ctx.createBiquadFilter();
    washLP.type = "lowpass";
    washLP.frequency.value = 600;
    washLP.Q.value = 0.7;

    const washGain = ctx.createGain();
    washGain.gain.value = 0.4;

    // Slow LFO on wash gain for gentle swell
    const swellLFO = ctx.createOscillator();
    swellLFO.frequency.value = 0.04; // 25-second cycle
    const swellGain = ctx.createGain();
    swellGain.gain.value = 0.2;
    swellLFO.connect(swellGain);
    swellGain.connect(washGain.gain);
    swellLFO.start();

    washSource.connect(washLP);
    washLP.connect(washGain);
    washGain.connect(gain);
    washSource.start();

    // ── Layer 2: Wave crash scheduler ──
    const crashGain = ctx.createGain();
    crashGain.gain.value = 0.6;
    crashGain.connect(gain);

    let crashInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleCrash = () => {
      if (!this.active.has("ocean" as SoundID)) return;

      const duration = 1.5 + Math.random() * 2.5; // 1.5-4 seconds
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);

      // Crash sound: noise burst with exponential decay
      for (let i = 0; i < length; i++) {
        const t = i / length;
        const envelope = Math.exp(-t * 4); // fast decay
        data[i] = (Math.random() * 2 - 1) * envelope;
      }

      const cSource = ctx.createBufferSource();
      cSource.buffer = buf;

      // Low-pass that opens briefly then closes (wave crash)
      const lp = ctx.createBiquadFilter();
      lp.type = "lowpass";
      lp.frequency.setValueAtTime(200, ctx.currentTime);
      lp.frequency.linearRampToValueAtTime(2500, ctx.currentTime + 0.3);
      lp.frequency.exponentialRampToValueAtTime(300, ctx.currentTime + duration * 0.6);

      const env = ctx.createGain();
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(0.8, ctx.currentTime + 0.15);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      cSource.connect(lp);
      lp.connect(env);
      env.connect(crashGain);
      cSource.start(ctx.currentTime);
      cSource.stop(ctx.currentTime + duration);

      // Next crash in 4-12 seconds
      const nextDelay = 4000 + Math.random() * 8000;
      crashInterval = setTimeout(scheduleCrash, nextDelay);
    };

    crashInterval = setTimeout(scheduleCrash, 1000);

    // Clean up scheduler on stop
    const originalStop = washSource.stop.bind(washSource);
    washSource.stop = ((...args: any[]) => {
      if (crashInterval) {
        clearTimeout(crashInterval);
        crashInterval = null;
      }
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return washSource;
  }
}

/** Singleton instance. */
let instance: SoundscapeEngine | null = null;

export function getSoundscape(): SoundscapeEngine {
  if (!instance) {
    instance = new SoundscapeEngine();
  }
  return instance;
}

export function destroySoundscape(): void {
  if (instance) {
    instance.destroy();
    instance = null;
  }
}
