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

  /** Rain — band-passed noise with slow modulation. */
  private makeRainSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    const buffer = this.makeNoiseBuffer(ctx, 4);
    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.loop = true;

    // Band-pass filter centered around rain frequencies
    const bp = ctx.createBiquadFilter();
    bp.type = "bandpass";
    bp.frequency.value = 2000;
    bp.Q.value = 0.5;

    // Slow LFO to modulate the filter frequency (gives movement)
    const lfo = ctx.createOscillator();
    lfo.frequency.value = 0.1; // very slow
    const lfoGain = ctx.createGain();
    lfoGain.gain.value = 800;
    lfo.connect(lfoGain);
    lfoGain.connect(bp.frequency);
    lfo.start();

    source.connect(bp);
    bp.connect(gain);
    source.start();
    return source;
  }

  /** Ocean — filtered noise with very slow wave-like modulation. */
  private makeOceanSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    const buffer = this.makeNoiseBuffer(ctx, 4);
    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.loop = true;

    // Low-pass filter
    const lp = ctx.createBiquadFilter();
    lp.type = "lowpass";
    lp.frequency.value = 800;
    lp.Q.value = 0.7;

    // Very slow LFO on gain for wave-like swell
    const lfo = ctx.createOscillator();
    lfo.frequency.value = 0.05; // 20-second cycle
    const lfoGain = ctx.createGain();
    lfoGain.gain.value = 0.3;
    lfo.connect(lfoGain);
    lfoGain.connect(gain.gain);
    lfo.start();

    source.connect(lp);
    lp.connect(gain);
    source.start();
    return source;
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
