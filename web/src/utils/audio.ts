/**
 * Soundscape audio engine.
 *
 * Plays the ambient sound loops (MP3 files served by the backend) through the
 * Web Audio graph so volume, reverb and the frequency visualizer apply. Every
 * sound in the catalog is backed by a real audio file.
 *
 * Sounds:
 *   - light-rain       : light rainfall
 *   - rain             : steady rain
 *   - rain-and-thunder : rain with distant thunder
 *   - strong-rain      : heavy rain
 *   - stronger-rain    : storm downpour
 *   - fire             : crackling fire
 *   - fire-and-thunder : fire with distant thunder
 *   - ocean            : waves
 */

import { getSoundFile } from "./soundFiles";

export type SoundID =
  | "light-rain"
  | "rain"
  | "rain-and-thunder"
  | "strong-rain"
  | "stronger-rain"
  | "fire"
  | "fire-and-thunder"
  | "ocean";

export type PresetID = string;

interface Preset {
  id: PresetID;
  label: string;
  sounds: Partial<Record<SoundID, boolean>>;
  volumes: Partial<Record<SoundID, number>>;
  masterVolume: number;
}

interface ActiveSource {
  /** Media-element node for the looping MP3. */
  source: MediaElementAudioSourceNode;
  gain: GainNode;
  /** The looping audio element. */
  element: HTMLAudioElement;
}

/** How long (seconds) to fade in/out a sound when playing/stopping. */
const FADE_DURATION = 0.5;

const LS_PRESETS_KEY = "soundscape_presets";

class SoundscapeEngine {
  private ctx: AudioContext | null = null;
  private active: Map<SoundID, ActiveSource> = new Map();
  private masterGain: GainNode | null = null;
  private reverbGain: GainNode | null = null;
  private reverbNode: ConvolverNode | null = null;
  private analyserNode: AnalyserNode | null = null;

  // Per-sound volume state
  private soundVolumes: Partial<Record<SoundID, number>> = {};

  /**
   * Ensure the AudioContext exists (must be called from a user gesture).
   * Creates the master gain, a subtle global reverb send and the analyser used
   * by the frequency visualizer.
   */
  private ensureContext(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext({ latencyHint: "playback" });
      this.masterGain = this.ctx.createGain();
      this.masterGain.gain.value = 0.5;

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
   */
  play(id: SoundID, fadeIn = true): boolean {
    if (this.active.has(id)) return true;

    const file = getSoundFile(id);
    if (!file) {
      console.warn(`[soundscape] no audio file available for "${id}"`);
      return false;
    }

    const ctx = this.ensureContext();
    const gain = ctx.createGain();
    gain.gain.value = fadeIn ? 0 : (this.soundVolumes[id] ?? 0.5);
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

      if (fadeIn) {
        const targetVol = this.soundVolumes[id] ?? 0.5;
        gain.gain.setValueAtTime(0, ctx.currentTime);
        gain.gain.linearRampToValueAtTime(targetVol, ctx.currentTime + FADE_DURATION);
      }

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
      this.masterGain.gain.value = Math.max(0, Math.min(1, v));
    }
  }

  /** Set volume for a specific sound (0–1). */
  setSoundVolume(id: SoundID, v: number): void {
    this.soundVolumes[id] = v;
    const entry = this.active.get(id);
    if (entry) {
      entry.gain.gain.value = Math.max(0, Math.min(1, v));
    }
  }

  /** Check if a sound is currently playing. */
  isPlaying(id: SoundID): boolean {
    return this.active.has(id);
  }

  // ─── Presets ───────────────────────────────────────────────

  /** Load presets from localStorage. */
  private loadPresets(): Preset[] {
    try {
      const raw = localStorage.getItem(LS_PRESETS_KEY);
      if (!raw) return [];
      return JSON.parse(raw) as Preset[];
    } catch {
      return [];
    }
  }

  /** Save presets to localStorage. */
  private savePresets(presets: Preset[]): void {
    localStorage.setItem(LS_PRESETS_KEY, JSON.stringify(presets));
  }

  /** Get all saved presets. */
  getPresets(): Preset[] {
    return this.loadPresets();
  }

  /**
   * Save current mix as a preset.
   * Returns the generated id.
   */
  savePreset(
    label: string,
    currentSounds: Partial<Record<SoundID, boolean>>,
    currentVolumes: Partial<Record<SoundID, number>>,
    masterVol: number,
  ): PresetID {
    const presets = this.loadPresets();
    const id = `preset_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
    const preset: Preset = {
      id,
      label,
      sounds: { ...currentSounds },
      volumes: { ...currentVolumes },
      masterVolume: masterVol,
    };
    presets.push(preset);
    this.savePresets(presets);
    return id;
  }

  /** Delete a preset by id. */
  deletePreset(id: PresetID): void {
    const presets = this.loadPresets().filter((p) => p.id !== id);
    this.savePresets(presets);
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
