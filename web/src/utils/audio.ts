/**
 * Soundscape audio engine.
 *
 * Plays the ambient sound loops (audio files served by the backend) through the
 * Web Audio graph so volume, reverb and the frequency visualizer apply.
 *
 * The engine is catalog-agnostic: it plays any id it is given as long as
 * ./soundFiles knows a file for it. The built-in ids and their labels live in
 * ./soundCatalog, and saved mixes live in ./soundPresets.
 */

import { getSoundFile } from "./soundFiles";

/**
 * Identifier of a sound in the library. Built-in ids are the names shipped with
 * the app; sounds added by the user get their own ids from the backend catalog.
 */
export type SoundID = string;

/** Default level of a sound's own volume slider (0–1). */
export const DEFAULT_SOUND_VOLUME = 0.5;

/** Default master output level (0–1), matching the initial slider position. */
export const DEFAULT_MASTER_VOLUME = 0.5;

interface ActiveSource {
  /** Media-element node for the looping audio file. */
  source: MediaElementAudioSourceNode;
  gain: GainNode;
  /** The looping audio element. */
  element: HTMLAudioElement;
}

/** How long (seconds) to fade in/out a sound when playing/stopping. */
const FADE_DURATION = 0.5;

/** Clamps a gain level into the valid 0–1 range. */
function clamp01(v: number): number {
  return Math.max(0, Math.min(1, v));
}

class SoundscapeEngine {
  private ctx: AudioContext | null = null;
  private active: Map<SoundID, ActiveSource> = new Map();
  private masterGain: GainNode | null = null;
  private reverbGain: GainNode | null = null;
  private reverbNode: ConvolverNode | null = null;
  private analyserNode: AnalyserNode | null = null;

  /**
   * Ensure the AudioContext exists (must be called from a user gesture).
   * Creates the master gain, a subtle global reverb send and the analyser used
   * by the frequency visualizer.
   */
  private ensureContext(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext({ latencyHint: "playback" });
      this.masterGain = this.ctx.createGain();
      this.masterGain.gain.value = DEFAULT_MASTER_VOLUME;

      // Reverb: subtle wet send off the master bus.
      this.reverbGain = this.ctx.createGain();
      this.reverbGain.gain.value = 0.15;
      this.reverbNode = this.ctx.createConvolver();
      this.reverbNode.buffer = this.buildReverbIR(this.ctx, 2.0, 0.5);

      this.masterGain.connect(this.ctx.destination); // dry path
      this.masterGain.connect(this.reverbNode); // wet path
      this.reverbNode.connect(this.reverbGain);
      this.reverbGain.connect(this.ctx.destination);

      // Analyser for the visualizer.
      this.analyserNode = this.ctx.createAnalyser();
      this.analyserNode.fftSize = 256;
      this.masterGain.connect(this.analyserNode);
    }
    if (this.ctx.state === "suspended") {
      void this.ctx.resume();
    }
    return this.ctx;
  }

  /** Build a simple impulse response for reverb: decaying exponential noise. */
  private buildReverbIR(ctx: AudioContext, duration: number, decay: number): AudioBuffer {
    const sampleRate = ctx.sampleRate;
    const length = sampleRate * duration;
    const buffer = ctx.createBuffer(2, length, sampleRate);
    for (let ch = 0; ch < 2; ch++) {
      const data = buffer.getChannelData(ch);
      for (let i = 0; i < length; i++) {
        const t = i / sampleRate;
        const envelope = Math.exp(-t * decay);
        data[i] = (Math.random() * 2 - 1) * envelope;
      }
    }
    return buffer;
  }

  /**
   * Get current frequency data for the visualizer (returns Uint8Array of
   * length fftSize/2). Caller should supply a pre-allocated Uint8Array.
   */
  getFrequencyData(data: Uint8Array): void {
    if (this.analyserNode) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      this.analyserNode.getByteFrequencyData(data as any);
    }
  }

  /**
   * Play a looping audio file for a sound, routing it through the shared
   * gain → masterGain graph (volume, reverb and the visualizer apply).
   *
   * The caller owns the level: pass the sound's current volume so the fade-in
   * lands on it directly instead of jumping once the slider state catches up.
   */
  play(id: SoundID, volume = DEFAULT_SOUND_VOLUME): boolean {
    if (this.active.has(id)) return true;

    const file = getSoundFile(id);
    if (!file) {
      console.warn(`[soundscape] no audio file available for "${id}"`);
      return false;
    }

    const level = clamp01(volume);
    const ctx = this.ensureContext();
    const gain = ctx.createGain();
    gain.gain.value = 0;
    gain.connect(this.masterGain!);

    let entry: ActiveSource;
    try {
      // MediaElementAudioSourceNode outputs silence unless the media is
      // CORS-clean. The desktop app loads from app://bundle and fetches the
      // audio from the backend, so request anonymous (no-credential) CORS mode.
      const element = new Audio();
      element.crossOrigin = "anonymous";
      element.loop = true;
      element.src = file;
      const source = ctx.createMediaElementSource(element);
      source.connect(gain);

      gain.gain.setValueAtTime(0, ctx.currentTime);
      gain.gain.linearRampToValueAtTime(level, ctx.currentTime + FADE_DURATION);

      entry = { source, gain, element };
    } catch {
      // File playback unavailable.
      gain.disconnect();
      return false;
    }

    this.active.set(id, entry);
    void entry.element.play().catch(() => {
      // The file failed to load/autoplay (missing file, CORS failure or
      // autoplay policy).
      this.releaseEntry(entry);
      this.active.delete(id);
    });
    return true;
  }

  /** Stop a specific sound with optional fade-out. */
  stop(id: SoundID, fadeOut = true): void {
    const entry = this.active.get(id);
    if (!entry) return;

    if (fadeOut && this.ctx) {
      const ctx = this.ctx;
      const gain = entry.gain;
      const currentTime = ctx.currentTime;
      gain.gain.setValueAtTime(gain.gain.value, currentTime);
      gain.gain.linearRampToValueAtTime(0, currentTime + FADE_DURATION);

      // Schedule the actual stop + cleanup after the fade completes.
      setTimeout(() => {
        this.releaseEntry(entry);
        this.active.delete(id);
      }, FADE_DURATION * 1000 + 50);
    } else {
      this.releaseEntry(entry);
      this.active.delete(id);
    }
  }

  /** Stop all sounds. */
  stopAll(): void {
    for (const id of this.active.keys()) {
      this.stop(id, false);
    }
  }

  /** Pause and free a looping audio element, disconnecting its node chain. */
  private releaseEntry(entry: ActiveSource): void {
    entry.element.pause();
    entry.element.removeAttribute("src");
    entry.element.load();
    entry.source.disconnect();
    entry.gain.disconnect();
  }

  /** Set master volume (0–1). */
  setVolume(v: number): void {
    if (this.masterGain) {
      this.masterGain.gain.value = clamp01(v);
    }
  }

  /**
   * Live-updates the volume of a playing sound (0–1). No-op when it is not
   * playing: the caller owns the level and passes it to `play` next time.
   */
  setSoundVolume(id: SoundID, v: number): void {
    const entry = this.active.get(id);
    if (entry) {
      entry.gain.gain.value = clamp01(v);
    }
  }

  /** Check if a sound is currently playing. */
  isPlaying(id: SoundID): boolean {
    return this.active.has(id);
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
