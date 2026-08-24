/**
 * Web Audio API soundscape engine.
 *
 * Generates ambient sounds in the browser using AudioContext, and plays
 * looping audio files (MP3 loops served by the backend) when they are
 * available — see ./soundFiles for the catalog. File-based and synthesized
 * sounds can be mixed freely, and every file-backed sound falls back to its
 * synthesized version when the file fails to load.
 *
 * Supported sounds:
 *   - white-noise  : flat spectrum, hiss-like
 *   - pink-noise   : -3dB/octave rolloff, warmer
 *   - brown-noise  : -6dB/octave rolloff, deep rumble
 *   - rain         : filtered noise with gentle modulation
 *   - ocean        : filtered noise with slow wave modulation
 *   - wind         : howling wind with occasional gusts
 *   - campfire     : crackling fire with embers and pops
 */

import { getSoundFile } from "./soundFiles";

export type SoundID =
  | "white-noise"
  | "pink-noise"
  | "brown-noise"
  | "rain"
  | "ocean"
  | "wind"
  | "campfire"
  | "binaural-beat"
  | "isochronic-tone"
  | "meditation-bowl"
  | "ambient-pad";

export type PresetID = string;

interface Preset {
  id: PresetID;
  label: string;
  sounds: Partial<Record<SoundID, boolean>>;
  volumes: Partial<Record<SoundID, number>>;
  masterVolume: number;
  rainIntensity: number;
}

interface ActiveSource {
  /** Web Audio source for synthesized sounds, or the media-element node for file sounds. */
  source: AudioBufferSourceNode | MediaElementAudioSourceNode;
  gain: GainNode;
  /** Set when this sound is played from a looping audio file. */
  element?: HTMLAudioElement;
}

/** How long (seconds) to fade in/out a sound when playing/stopping. */
const FADE_DURATION = 0.5;

const CICADA_CARRIER_FREQ = 5500; // Hz
const CICADA_MOD_FREQ = 28; // Hz
const CICADA_BURST_DURATION = 0.15; // seconds per pulse
const CICADA_GAP_MIN = 0.15;
const CICADA_GAP_MAX = 0.4;

const LS_PRESETS_KEY = "soundscape_presets";

class SoundscapeEngine {
  private ctx: AudioContext | null = null;
  private active: Map<SoundID, ActiveSource> = new Map();
  private masterGain: GainNode | null = null;
  private reverbGain: GainNode | null = null;
  private reverbNode: ConvolverNode | null = null;
  private analyserNode: AnalyserNode | null = null;

  // Per-sound state
  private rainIntensity = 0.5;
  private soundVolumes: Partial<Record<SoundID, number>> = {};

  /**
   * Ensure AudioContext is created (must be called from a user gesture).
   * Now includes a shared reverb and analyser node.
   */
  private ensureContext(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext({ latencyHint: "playback" });
      this.masterGain = this.ctx.createGain();
      this.masterGain.gain.value = 0.5;

      // ── Reverb: built from an impulse response generated from filtered noise ──
      this.reverbGain = this.ctx.createGain();
      this.reverbGain.gain.value = 0.15; // wet mix

      this.reverbNode = this.ctx.createConvolver();
      this.reverbNode.buffer = this.buildReverbIR(this.ctx, 2.0, 0.5);

      // Connect: masterGain → reverb (dry) + masterGain → reverbNode → reverbGain (wet)
      // Actually simpler: masterGain → destination directly
      // We'll insert reverb as a send from masterGain instead.
      // Revised routing:
      //   sound sources → individual gain → masterGain → destination
      //   sound sources → reverbSend (new) → reverbNode → reverbGain → destination
      // For simplicity with existing code, we'll put reverb on a separate chain:
      // All sounds connect to masterGain normally.
      // Additionally, we create a reverb send node that gets a portion of each sound's signal.
      // To keep this clean without modifying every sound generator, we'll add reverb after masterGain:
      //   masterGain → split → (dry) destination
      //                    → (wet) reverbNode → reverbGain → destination
      // But this affects masterGain equally. For simplicity with existing architecture,
      // we place reverb as a subtle global effect after master gain.

      // Split master output: one dry path, one wet path through reverb
      // We need a splitter/merger. Most straightforward: masterGain → reverbNode → reverbGain → destination
      // AND masterGain → destination directly. But we can't connect masterGain to two nodes directly
      // in Web Audio without a splitter. The GainNode can only connect to one destination.
      // Actually, GainNode.connect() can connect to multiple! Let's use that.
      this.masterGain.connect(this.ctx.destination); // dry path

      // Wet path
      this.masterGain.connect(this.reverbNode);
      this.reverbNode.connect(this.reverbGain);
      this.reverbGain.connect(this.ctx.destination);

      // ── Analyser ──
      this.analyserNode = this.ctx.createAnalyser();
      this.analyserNode.fftSize = 256;
      this.masterGain.connect(this.analyserNode);
    }
    if (this.ctx.state === "suspended") {
      this.ctx.resume();
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

  /** Set the reverb wet mix (0–1). */
  setReverbMix(v: number): void {
    if (this.reverbGain) {
      this.reverbGain.gain.value = Math.max(0, Math.min(0.5, v));
    }
  }

  /** Get analyser node for visualizer. */
  getAnalyser(): AnalyserNode | null {
    return this.analyserNode ?? null;
  }

  /**
   * Get current frequency data for visualizer (returns Uint8Array of length fftSize/2).
   * Caller should supply a pre-allocated Uint8Array.
   */
  getFrequencyData(data: Uint8Array): void {
    if (this.analyserNode) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (this.analyserNode as AnalyserNode).getByteFrequencyData(data as any);
    }
  }

  /** Set rain intensity (0=drizzle, 1=downpour). */
  setRainIntensity(v: number): void {
    this.rainIntensity = Math.max(0, Math.min(1, v));
  }

  /** Get current rain intensity. */
  getRainIntensity(): number {
    return this.rainIntensity;
  }

  /** Play a sound with optional fade-in. */
  play(id: SoundID, fadeIn = true): void {
    if (this.active.has(id)) return;

    const ctx = this.ensureContext();
    const gain = ctx.createGain();
    gain.gain.value = fadeIn ? 0 : (this.soundVolumes[id] ?? 0.5);
    gain.connect(this.masterGain!);

    // If a sound file is available from the backend catalog, play it as a
    // looping media element routed through the same gain → masterGain graph (so
    // volume, reverb and the visualizer apply). Fall back to synthesis on any
    // failure — including an async play() rejection (autoplay policy, CORS or
    // network error), which would otherwise silently play nothing.
    const file = getSoundFile(id);
    if (file) {
      const element = new Audio();
      // MediaElementAudioSourceNode outputs silence unless the media is
      // CORS-clean. The desktop app loads from app://bundle and fetches the
      // audio from the backend, so request anonymous (no-credential) CORS mode
      // — the backend answers it via the global CORS middleware.
      element.crossOrigin = "anonymous";
      element.loop = true;
      try {
        element.src = file;
        const source = ctx.createMediaElementSource(element);
        source.connect(gain);

        if (fadeIn) {
          const targetVol = this.soundVolumes[id] ?? 0.5;
          gain.gain.setValueAtTime(0, ctx.currentTime);
          gain.gain.linearRampToValueAtTime(targetVol, ctx.currentTime + FADE_DURATION);
        }

        const entry = { source, gain, element };
        this.active.set(id, entry);
        void element.play().catch(() => {
          // Media element failed to load/autoplay (missing file, CORS failure,
          // autoplay policy) — release the element and its source node, but
          // keep the shared gain node wired to masterGain so the synthesized
          // fallback can play through it.
          try {
            element.pause();
          } catch {
            // already stopped
          }
          element.removeAttribute("src");
          element.load();
          source.disconnect();
          this.active.delete(id);
          this.startSynthesized(ctx, id, gain, fadeIn);
        });
        return;
      } catch {
        // File playback unavailable — fall through to synthesis.
      }
    }

    this.startSynthesized(ctx, id, gain, fadeIn);
  }

  /**
   * Start the synthesized (Web Audio) version of a sound through an already
   * wired gain node, applying the same fade-in the file path uses.
   */
  private startSynthesized(ctx: AudioContext, id: SoundID, gain: GainNode, fadeIn: boolean): void {
    const source = this.createSource(ctx, id, gain);
    if (!source) return;

    if (fadeIn) {
      const targetVol = this.soundVolumes[id] ?? 0.5;
      gain.gain.setValueAtTime(0, ctx.currentTime);
      gain.gain.linearRampToValueAtTime(targetVol, ctx.currentTime + FADE_DURATION);
    }

    this.active.set(id, { source, gain });
  }

  /** Stop a specific sound with optional fade-out. */
  stop(id: SoundID, fadeOut = true): void {
    const entry = this.active.get(id);
    if (!entry) return;

    if (fadeOut && this.ctx) {
      // Fade out, then disconnect
      const ctx = this.ctx;
      const gain = entry.gain;
      const currentTime = ctx.currentTime;
      gain.gain.setValueAtTime(gain.gain.value, currentTime);
      gain.gain.linearRampToValueAtTime(0, currentTime + FADE_DURATION);

      // Schedule the actual stop + cleanup after fade
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

  /**
   * Release an active sound's underlying resources: pause and free a looping
   * audio element for file-backed sounds, or stop the buffer source for
   * synthesized ones. Disconnects the node chain in both cases.
   */
  private releaseEntry(entry: ActiveSource): void {
    if (entry.element) {
      entry.element.pause();
      entry.element.removeAttribute("src");
      entry.element.load();
    } else {
      try {
        (entry.source as AudioBufferSourceNode).stop();
      } catch {
        // already stopped
      }
    }
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

  /** Clean up everything. */
  destroy(): void {
    this.stopAll();
    if (this.ctx) {
      this.ctx.close();
      this.ctx = null;
      this.masterGain = null;
      this.reverbNode = null;
      this.reverbGain = null;
      this.analyserNode = null;
    }
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
    rainIntensity: number,
  ): PresetID {
    const presets = this.loadPresets();
    const id = `preset_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
    const preset: Preset = {
      id,
      label,
      sounds: { ...currentSounds },
      volumes: { ...currentVolumes },
      masterVolume: masterVol,
      rainIntensity,
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
      case "wind":
        return this.makeWindSource(ctx, gain);
      case "campfire":
        return this.makeCampfireSource(ctx, gain);
      case "binaural-beat":
        return this.makeBinauralSource(ctx, gain);
      case "isochronic-tone":
        return this.makeIsochronicSource(ctx, gain);
      case "meditation-bowl":
        return this.makeMeditationBowlSource(ctx, gain);
      case "ambient-pad":
        return this.makeAmbientPadSource(ctx, gain);
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

  /**
   * Rain — layered: background wash + randomized droplets with variable intensity.
   *
   * Improvements:
   *   - LFO-driven wash filter drift for more organic texture
   *   - Stereo-panned droplets for spatial immersion
   *   - Thunder events: sharp crack + brown-noise sub-bass rumble
   *   - Rain intensity slider (drizzle ↔ downpour)
   */
  private makeRainSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Background wash ──
    const washBuffer = this.makeNoiseBuffer(ctx, 4);
    const washSource = ctx.createBufferSource();
    washSource.buffer = washBuffer;
    washSource.loop = true;

    const washGain = ctx.createGain();
    washGain.gain.value = 0.18;

    const washLP1 = ctx.createBiquadFilter();
    washLP1.type = "lowpass";
    washLP1.frequency.value = 3000;
    washLP1.Q.value = 0.5;

    const washLP2 = ctx.createBiquadFilter();
    washLP2.type = "lowpass";
    washLP2.frequency.value = 1200;
    washLP2.Q.value = 0.3;

    // Intensity modulation LFO (deeper variation to reduce static feel)
    const washLFO = ctx.createOscillator();
    washLFO.frequency.value = 0.08;
    const washLFOGain = ctx.createGain();
    washLFOGain.gain.value = 0.12;
    washLFO.connect(washLFOGain);
    washLFOGain.connect(washGain.gain);
    washLFO.start();

    // Filter drift LFO
    const filterDriftLFO = ctx.createOscillator();
    filterDriftLFO.frequency.value = 0.02;
    const filterDriftGain = ctx.createGain();
    filterDriftGain.gain.value = 800;
    filterDriftLFO.connect(filterDriftGain);
    filterDriftGain.connect(washLP1.frequency);
    filterDriftLFO.start();

    washSource.connect(washLP1);
    washLP1.connect(washLP2);
    washLP2.connect(washGain);
    washGain.connect(gain);
    washSource.start();

    // ── Layer 2: Droplet scheduler ──
    const dropletGain = ctx.createGain();
    dropletGain.gain.value = 0.35;
    dropletGain.connect(gain);

    let dropletInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleDroplet = () => {
      if (!this.active.has("rain" as SoundID)) return;

      const intensity = this.rainIntensity; // 0-1
      // Density scales with intensity: drizzle (low) → downpour (high)
      const isLarge = Math.random() < 0.15 + intensity * 0.15; // 15-30% large drops

      const duration = isLarge
        ? 0.08 + Math.random() * 0.12
        : 0.02 + Math.random() * 0.05;

      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const dSource = ctx.createBufferSource();
      dSource.buffer = buf;

      // Bandpass: lower Q to avoid metallic ringing
      const bp = ctx.createBiquadFilter();
      bp.type = "bandpass";
      if (isLarge) {
        bp.frequency.value = 600 + Math.random() * 800;
        bp.Q.value = 0.3 + Math.random() * 0.7;
      } else {
        bp.frequency.value = 2000 + Math.random() * 2000;
        bp.Q.value = 0.5 + Math.random() * 1.0;
      }

      // Gentle low-pass sweep: high end starts open and closes (softens the impact)
      const dripLp = ctx.createBiquadFilter();
      dripLp.type = "lowpass";
      dripLp.frequency.setValueAtTime(8000, ctx.currentTime);
      dripLp.frequency.exponentialRampToValueAtTime(3000, ctx.currentTime + duration);

      const env = ctx.createGain();
      const peak = isLarge ? 0.5 + Math.random() * 0.5 : 0.3 + Math.random() * 0.4;
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(peak, ctx.currentTime + 0.003);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      const panner = ctx.createStereoPanner();
      panner.pan.value = Math.random() * 2 - 1;

      dSource.connect(bp);
      bp.connect(dripLp);
      dripLp.connect(env);
      env.connect(panner);
      panner.connect(dropletGain);
      dSource.start(ctx.currentTime);
      dSource.stop(ctx.currentTime + duration);

      // Droplet delay scales inversely with intensity
      const baseMin = 10 - intensity * 8;
      const baseMax = 40 - intensity * 30;
      const largeMin = 100 - intensity * 60;
      const largeMax = 300 - intensity * 200;
      const nextDelay = isLarge
        ? Math.max(5, largeMin + Math.random() * (largeMax - largeMin))
        : Math.max(2, baseMin + Math.random() * (baseMax - baseMin));

      dropletInterval = setTimeout(scheduleDroplet, nextDelay);
    };

    dropletInterval = setTimeout(scheduleDroplet, 50);

    // ── Layer 3: Thunder scheduler ──
    const thunderGain = ctx.createGain();
    thunderGain.gain.value = 0;
    thunderGain.connect(gain);

    let thunderInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleThunder = () => {
      if (!this.active.has("rain" as SoundID)) return;

      const startTime = ctx.currentTime;

      // ── Sub-layer 3a: Sharp crack (wide-spectrum, 50ms) ──
      const crackDuration = 0.05;
      const crackLength = ctx.sampleRate * crackDuration;
      const crackBuf = ctx.createBuffer(1, crackLength, ctx.sampleRate);
      const crackData = crackBuf.getChannelData(0);
      for (let i = 0; i < crackLength; i++) {
        crackData[i] = Math.random() * 2 - 1;
      }

      const crackSource = ctx.createBufferSource();
      crackSource.buffer = crackBuf;

      // Fast envelope for sharp crack
      const crackEnv = ctx.createGain();
      crackEnv.gain.setValueAtTime(0, startTime);
      crackEnv.gain.linearRampToValueAtTime(0.5, startTime + 0.002);
      crackEnv.gain.exponentialRampToValueAtTime(0.001, startTime + crackDuration);

      const crackPanner = ctx.createStereoPanner();
      crackPanner.pan.value = (Math.random() - 0.5) * 0.8;

      crackSource.connect(crackEnv);
      crackEnv.connect(crackPanner);
      crackPanner.connect(gain);
      crackSource.start(startTime);
      crackSource.stop(startTime + crackDuration);

      // ── Sub-layer 3b: Sub-bass sine rumble (25-40Hz) ──
      const subDuration = 2 + Math.random() * 3; // 2-5 seconds
      const subOsc = ctx.createOscillator();
      subOsc.type = "sine";
      const subFreq = 25 + Math.random() * 15; // 25-40Hz
      subOsc.frequency.value = subFreq;

      const subEnv = ctx.createGain();
      const subPeak = 0.15 + Math.random() * 0.1;
      subEnv.gain.setValueAtTime(0, startTime + 0.01);
      subEnv.gain.linearRampToValueAtTime(subPeak, startTime + 0.5 + Math.random() * 0.3);
      subEnv.gain.setValueAtTime(subPeak, startTime + subDuration * 0.6);
      subEnv.gain.exponentialRampToValueAtTime(0.001, startTime + subDuration);

      const subPanner = ctx.createStereoPanner();
      subPanner.pan.value = (Math.random() - 0.5) * 0.4;

      subOsc.connect(subEnv);
      subEnv.connect(subPanner);
      subPanner.connect(gain);
      subOsc.start(startTime + 0.01);
      subOsc.stop(startTime + subDuration);

      // ── Sub-layer 3c: Brown noise rumble (existing, slightly delayed) ──
      const rumbleDelay = 0.1;
      const rumbleDuration = subDuration - 0.2;
      if (rumbleDuration > 0.5) {
        const rLength = ctx.sampleRate * rumbleDuration;
        const rBuf = ctx.createBuffer(1, rLength, ctx.sampleRate);
        const rData = rBuf.getChannelData(0);
        let lastOut = 0;
        for (let i = 0; i < rLength; i++) {
          const white = Math.random() * 2 - 1;
          lastOut = (lastOut + 0.01 * white) / 1.01;
          rData[i] = lastOut * 5;
        }

        const rSource = ctx.createBufferSource();
        rSource.buffer = rBuf;

        const rLP = ctx.createBiquadFilter();
        rLP.type = "lowpass";
        rLP.frequency.value = 60 + Math.random() * 60;
        rLP.Q.value = 0.5;

        const rEnv = ctx.createGain();
        const rPeakVol = 0.4 + Math.random() * 0.3;
        rEnv.gain.setValueAtTime(0, startTime + rumbleDelay);
        rEnv.gain.linearRampToValueAtTime(rPeakVol, startTime + rumbleDelay + 0.6 + Math.random() * 0.3);
        rEnv.gain.exponentialRampToValueAtTime(0.001, startTime + rumbleDelay + rumbleDuration);

        const rPanner = ctx.createStereoPanner();
        rPanner.pan.value = (Math.random() - 0.5) * 0.6;

        rSource.connect(rLP);
        rLP.connect(rEnv);
        rEnv.connect(rPanner);
        rPanner.connect(thunderGain);
        rSource.start(startTime + rumbleDelay);
        rSource.stop(startTime + rumbleDelay + rumbleDuration);
      }

      // Next thunder: 30-120s, but more frequent at higher intensity
      const baseDelay = 60000 - this.rainIntensity * 30000;
      const nextDelay = baseDelay + Math.random() * 60000;
      thunderInterval = setTimeout(scheduleThunder, nextDelay);
    };

    thunderInterval = setTimeout(scheduleThunder, 20000 + Math.random() * 30000);

    const originalStop = washSource.stop.bind(washSource);
    washSource.stop = ((...args: [when?: number]) => {
      if (dropletInterval) {
        clearTimeout(dropletInterval);
        dropletInterval = null;
      }
      if (thunderInterval) {
        clearTimeout(thunderInterval);
        thunderInterval = null;
      }
      filterDriftLFO.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return washSource;
  }

  /**
   * Ocean — layered: deep wash with complex swell + wave sets with foam.
   *
   * Improvements:
   *   - Stereo-panned wave crashes and foam for shore-like spread
   *   - Wave set timing refined: closer waves within sets, longer breaks between
   *   - Occasional seagull calls for coastal atmosphere
   *   - Distant fog horn
   */
  private makeOceanSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Deep wash ──
    const washBuffer = this.makeNoiseBuffer(ctx, 4);
    const washSource = ctx.createBufferSource();
    washSource.buffer = washBuffer;
    washSource.loop = true;

    const washLP = ctx.createBiquadFilter();
    washLP.type = "lowpass";
    washLP.frequency.value = 500;
    washLP.Q.value = 0.7;

    const washGain = ctx.createGain();
    washGain.gain.value = 0.35;

    const swellLFO1 = ctx.createOscillator();
    swellLFO1.frequency.value = 0.03;
    const swellGain1 = ctx.createGain();
    swellGain1.gain.value = 0.15;
    swellLFO1.connect(swellGain1);
    swellGain1.connect(washGain.gain);

    const swellLFO2 = ctx.createOscillator();
    swellLFO2.frequency.value = 0.07;
    const swellGain2 = ctx.createGain();
    swellGain2.gain.value = 0.08;
    swellLFO2.connect(swellGain2);
    swellGain2.connect(washGain.gain);

    swellLFO1.start();
    swellLFO2.start();

    washSource.connect(washLP);
    washLP.connect(washGain);
    washGain.connect(gain);
    washSource.start();

    // ── Layer 2: Wave crash scheduler ──
    const crashGain = ctx.createGain();
    crashGain.gain.value = 0.4;
    crashGain.connect(gain);

    let crashInterval: ReturnType<typeof setTimeout> | null = null;
    let waveSetCount = 0;
    let inSetBreak = false;

    const scheduleCrash = () => {
      if (!this.active.has("ocean" as SoundID)) return;

      const wavesPerSet = 2 + Math.floor(Math.random() * 3); // 2-4 waves
      const progress = waveSetCount / wavesPerSet;
      const intensity = Math.sin(progress * Math.PI);

      const duration = 1.5 + intensity * 2.5;
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);

      for (let i = 0; i < length; i++) {
        const t = i / length;
        const envelope = Math.exp(-t * (3 + intensity * 2));
        data[i] = (Math.random() * 2 - 1) * envelope;
      }

      const cSource = ctx.createBufferSource();
      cSource.buffer = buf;

      const lp = ctx.createBiquadFilter();
      lp.type = "lowpass";
      const peakFreq = 1500 + intensity * 1500;
      lp.frequency.setValueAtTime(150, ctx.currentTime);
      lp.frequency.linearRampToValueAtTime(peakFreq, ctx.currentTime + 0.25);
      lp.frequency.exponentialRampToValueAtTime(250, ctx.currentTime + duration * 0.5);

      const env = ctx.createGain();
      const peakVol = 0.5 + intensity * 0.5;
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(peakVol, ctx.currentTime + 0.12);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      const crashPanner = ctx.createStereoPanner();
      crashPanner.pan.value = (Math.random() - 0.5) * 0.8;

      cSource.connect(lp);
      lp.connect(env);
      env.connect(crashPanner);
      crashPanner.connect(crashGain);
      cSource.start(ctx.currentTime);
      cSource.stop(ctx.currentTime + duration);

      // ── Layer 3: Foam ──
      const foamDuration = 0.3 + intensity * 0.5;
      const foamLength = ctx.sampleRate * foamDuration;
      const foamBuf = ctx.createBuffer(1, foamLength, ctx.sampleRate);
      const foamData = foamBuf.getChannelData(0);
      for (let i = 0; i < foamLength; i++) {
        const t = i / foamLength;
        const envelope = Math.exp(-t * 3);
        foamData[i] = (Math.random() * 2 - 1) * envelope;
      }

      const fSource = ctx.createBufferSource();
      fSource.buffer = foamBuf;

      const foamLP = ctx.createBiquadFilter();
      foamLP.type = "lowpass";
      foamLP.frequency.value = 4000 + intensity * 2000;
      foamLP.Q.value = 0.3;

      const foamEnv = ctx.createGain();
      foamEnv.gain.setValueAtTime(0, ctx.currentTime + 0.1);
      foamEnv.gain.linearRampToValueAtTime(0.15 + intensity * 0.15, ctx.currentTime + 0.2);
      foamEnv.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.1 + foamDuration);

      const foamPanner = ctx.createStereoPanner();
      foamPanner.pan.value = crashPanner.pan.value + (Math.random() - 0.5) * 0.2;

      fSource.connect(foamLP);
      foamLP.connect(foamEnv);
      foamEnv.connect(foamPanner);
      foamPanner.connect(crashGain);
      fSource.start(ctx.currentTime + 0.1);
      fSource.stop(ctx.currentTime + 0.1 + foamDuration);

      waveSetCount += 1;
      if (waveSetCount >= wavesPerSet) {
        waveSetCount = 0;
        inSetBreak = true;
      }

      let nextDelay: number;
      if (inSetBreak) {
        nextDelay = 12000 + Math.random() * 13000; // 12-25 seconds between sets
        inSetBreak = false;
      } else {
        nextDelay = 4000 + Math.random() * 3000; // 4-7 seconds within a set
      }
      crashInterval = setTimeout(scheduleCrash, nextDelay);
    };

    crashInterval = setTimeout(scheduleCrash, 1000);

    // ── Layer 4: Seagull scheduler ──
    const seagullGain = ctx.createGain();
    seagullGain.gain.value = 0;
    seagullGain.connect(gain);

    let seagullInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleSeagull = () => {
      if (!this.active.has("ocean" as SoundID)) return;

      const callDuration = 1 + Math.random() * 1.5;
      const startTime = ctx.currentTime;

      const osc1 = ctx.createOscillator();
      osc1.type = "sine";
      osc1.frequency.setValueAtTime(1200 + Math.random() * 800, startTime);
      osc1.frequency.linearRampToValueAtTime(1800 + Math.random() * 600, startTime + 0.2);
      osc1.frequency.linearRampToValueAtTime(800 + Math.random() * 400, startTime + callDuration * 0.5);
      osc1.frequency.linearRampToValueAtTime(1500 + Math.random() * 500, startTime + callDuration * 0.7);
      osc1.frequency.linearRampToValueAtTime(1000 + Math.random() * 300, startTime + callDuration);

      const osc2 = ctx.createOscillator();
      osc2.type = "sine";
      osc2.frequency.setValueAtTime(1400 + Math.random() * 600, startTime);
      osc2.frequency.linearRampToValueAtTime(2000 + Math.random() * 400, startTime + 0.15);
      osc2.frequency.linearRampToValueAtTime(900 + Math.random() * 300, startTime + callDuration * 0.4);
      osc2.frequency.linearRampToValueAtTime(1600 + Math.random() * 400, startTime + callDuration * 0.65);

      const callEnv = ctx.createGain();
      callEnv.gain.setValueAtTime(0, startTime);
      callEnv.gain.linearRampToValueAtTime(0.04 + Math.random() * 0.03, startTime + 0.1);
      callEnv.gain.setValueAtTime(0.04 + Math.random() * 0.03, startTime + callDuration * 0.6);
      callEnv.gain.exponentialRampToValueAtTime(0.001, startTime + callDuration);

      const sPanner = ctx.createStereoPanner();
      sPanner.pan.value = (Math.random() - 0.5) * 1.2;

      osc1.connect(callEnv);
      osc2.connect(callEnv);
      callEnv.connect(sPanner);
      sPanner.connect(seagullGain);
      osc1.start(startTime);
      osc2.start(startTime);
      osc1.stop(startTime + callDuration);
      osc2.stop(startTime + callDuration);

      const nextDelay = 30000 + Math.random() * 90000;
      seagullInterval = setTimeout(scheduleSeagull, nextDelay);
    };

    seagullInterval = setTimeout(scheduleSeagull, 15000 + Math.random() * 20000);

    // ── Layer 5: Fog horn scheduler ──
    const fogGain = ctx.createGain();
    fogGain.gain.value = 0;
    fogGain.connect(gain);

    let fogInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleFogHorn = () => {
      if (!this.active.has("ocean" as SoundID)) return;

      const startTime = ctx.currentTime;
      const hornDuration = 3 + Math.random() * 2; // 3-5 seconds

      // Slow sine at ~150Hz with tremolo
      const hornOsc = ctx.createOscillator();
      hornOsc.type = "sine";
      hornOsc.frequency.value = 120 + Math.random() * 60; // 120-180Hz

      // Tremolo LFO
      const tremLFO = ctx.createOscillator();
      tremLFO.frequency.value = 0.3 + Math.random() * 0.4; // 0.3-0.7Hz
      const tremGain = ctx.createGain();
      tremGain.gain.value = 0.3; // modulation depth
      tremLFO.connect(tremGain);
      tremGain.connect(hornOsc.frequency); // frequency vibrato
      tremLFO.start();

      const hornEnv = ctx.createGain();
      hornEnv.gain.setValueAtTime(0, startTime);
      hornEnv.gain.linearRampToValueAtTime(0.06 + Math.random() * 0.04, startTime + 0.5);
      hornEnv.gain.setValueAtTime(0.06 + Math.random() * 0.04, startTime + hornDuration - 0.5);
      hornEnv.gain.exponentialRampToValueAtTime(0.001, startTime + hornDuration);

      const hornPanner = ctx.createStereoPanner();
      hornPanner.pan.value = (Math.random() - 0.5) * 1.4; // far to one side

      hornOsc.connect(hornEnv);
      hornEnv.connect(hornPanner);
      hornPanner.connect(fogGain);
      hornOsc.start(startTime);
      hornOsc.stop(startTime + hornDuration);

      tremLFO.stop(startTime + hornDuration);

      const nextDelay = 15000 + Math.random() * 20000; // 15-35 seconds
      fogInterval = setTimeout(scheduleFogHorn, nextDelay);
    };

    fogInterval = setTimeout(scheduleFogHorn, 20000 + Math.random() * 15000);

    // Clean up on stop
    const originalStop = washSource.stop.bind(washSource);
    washSource.stop = ((...args: [when?: number]) => {
      if (crashInterval) {
        clearTimeout(crashInterval);
        crashInterval = null;
      }
      if (seagullInterval) {
        clearTimeout(seagullInterval);
        seagullInterval = null;
      }
      if (fogInterval) {
        clearTimeout(fogInterval);
        fogInterval = null;
      }
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return washSource;
  }

  /**
   * Wind — filtered noise with howling resonance and gust simulation.
   */
  private makeWindSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Brown noise base ──
    const baseBuffer = this.makeNoiseBuffer(ctx, 4);
    const baseSource = ctx.createBufferSource();
    baseSource.buffer = baseBuffer;
    baseSource.loop = true;

    const baseLP = ctx.createBiquadFilter();
    baseLP.type = "lowpass";
    baseLP.frequency.value = 300;
    baseLP.Q.value = 0.5;

    const baseHP = ctx.createBiquadFilter();
    baseHP.type = "highpass";
    baseHP.frequency.value = 60;

    const baseGain = ctx.createGain();
    baseGain.gain.value = 0.35;

    baseSource.connect(baseLP);
    baseLP.connect(baseHP);
    baseHP.connect(baseGain);
    baseGain.connect(gain);
    baseSource.start();

    // ── Layer 2: Howling resonance ──
    const howlBuffer = this.makeNoiseBuffer(ctx, 4);
    const howlSource = ctx.createBufferSource();
    howlSource.buffer = howlBuffer;
    howlSource.loop = true;

    const howlBP = ctx.createBiquadFilter();
    howlBP.type = "bandpass";
    howlBP.frequency.value = 400;
    howlBP.Q.value = 3;

    const howlLFO = ctx.createOscillator();
    howlLFO.frequency.value = 0.05;
    const howlLFOGain = ctx.createGain();
    howlLFOGain.gain.value = 250;
    howlLFO.connect(howlLFOGain);
    howlLFOGain.connect(howlBP.frequency);

    const rateModLFO = ctx.createOscillator();
    rateModLFO.frequency.value = 0.008;
    const rateModGain = ctx.createGain();
    rateModGain.gain.value = 0.025;
    rateModLFO.connect(rateModGain);
    rateModGain.connect(howlLFO.frequency);
    rateModLFO.start();

    howlLFO.start();

    const qLFO = ctx.createOscillator();
    qLFO.frequency.value = 0.03;
    const qLFOGain = ctx.createGain();
    qLFOGain.gain.value = 1.5;
    qLFO.connect(qLFOGain);
    qLFOGain.connect(howlBP.Q);
    qLFO.start();

    const howlGain = ctx.createGain();
    howlGain.gain.value = 0.25;

    howlSource.connect(howlBP);
    howlBP.connect(howlGain);
    howlGain.connect(gain);
    howlSource.start();

    // ── Layer 3: Gust scheduler ──
    const gustGain = ctx.createGain();
    gustGain.gain.value = 0;
    gustGain.connect(gain);

    let gustInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleGust = () => {
      if (!this.active.has("wind" as SoundID)) return;

      const gustDuration = 1 + Math.random() * 3;
      const gustStrength = 0.3 + Math.random() * 0.7;
      const gustLength = ctx.sampleRate * gustDuration;
      const buf = ctx.createBuffer(1, gustLength, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < gustLength; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const gSource = ctx.createBufferSource();
      gSource.buffer = buf;

      const gBP = ctx.createBiquadFilter();
      gBP.type = "bandpass";
      gBP.frequency.setValueAtTime(200, ctx.currentTime);
      gBP.frequency.linearRampToValueAtTime(
        400 + gustStrength * 1600,
        ctx.currentTime + 0.5
      );
      gBP.frequency.exponentialRampToValueAtTime(200, ctx.currentTime + gustDuration);
      gBP.Q.value = 1.5;

      const gEnv = ctx.createGain();
      gEnv.gain.setValueAtTime(0, ctx.currentTime);
      gEnv.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.05);
      gEnv.gain.linearRampToValueAtTime(gustStrength * 0.6, ctx.currentTime + 0.6);
      gEnv.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + gustDuration);

      const gPanner = ctx.createStereoPanner();
      gPanner.pan.value = (Math.random() - 0.5) * 1.0;

      gSource.connect(gBP);
      gBP.connect(gEnv);
      gEnv.connect(gPanner);
      gPanner.connect(gustGain);
      gSource.start(ctx.currentTime);
      gSource.stop(ctx.currentTime + gustDuration);

      // Whistle during strong gusts
      if (gustStrength > 0.6 && Math.random() < 0.4) {
        const whistleDuration = 0.5 + Math.random() * 1.5;
        const whistleLength = ctx.sampleRate * whistleDuration;
        const whistleBuf = ctx.createBuffer(1, whistleLength, ctx.sampleRate);
        const whistleData = whistleBuf.getChannelData(0);
        for (let i = 0; i < whistleLength; i++) {
          whistleData[i] = Math.random() * 2 - 1;
        }

        const wSource = ctx.createBufferSource();
        wSource.buffer = whistleBuf;

        const wBP = ctx.createBiquadFilter();
        wBP.type = "bandpass";
        wBP.frequency.setValueAtTime(500 + Math.random() * 500, ctx.currentTime);
        wBP.frequency.linearRampToValueAtTime(
          1000 + Math.random() * 2000,
          ctx.currentTime + whistleDuration * 0.3
        );
        wBP.frequency.exponentialRampToValueAtTime(600, ctx.currentTime + whistleDuration);
        wBP.Q.value = 8 + Math.random() * 6;

        const wEnv = ctx.createGain();
        wEnv.gain.setValueAtTime(0, ctx.currentTime);
        wEnv.gain.linearRampToValueAtTime(0.15 + Math.random() * 0.15, ctx.currentTime + 0.2);
        wEnv.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + whistleDuration);

        const wPanner = ctx.createStereoPanner();
        wPanner.pan.value = gPanner.pan.value + (Math.random() - 0.5) * 0.3;

        wSource.connect(wBP);
        wBP.connect(wEnv);
        wEnv.connect(wPanner);
        wPanner.connect(gustGain);
        wSource.start(ctx.currentTime);
        wSource.stop(ctx.currentTime + whistleDuration);
      }

      const nextDelay = (1.0 - gustStrength * 0.5) * (5000 + Math.random() * 15000);
      gustInterval = setTimeout(scheduleGust, nextDelay);
    };

    gustInterval = setTimeout(scheduleGust, 3000);

    const originalStop = baseSource.stop.bind(baseSource);
    baseSource.stop = ((...args: [when?: number]) => {
      if (gustInterval) {
        clearTimeout(gustInterval);
        gustInterval = null;
      }
      howlLFO.stop();
      rateModLFO.stop();
      qLFO.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return baseSource;
  }

  /**
   * Campfire — crackling embers, pops, low rumble, and crickets.
   */
  private makeCampfireSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Fire rumble ──
    const rumbleBuffer = this.makeNoiseBuffer(ctx, 4);
    const rumbleSource = ctx.createBufferSource();
    rumbleSource.buffer = rumbleBuffer;
    rumbleSource.loop = true;

    const rumbleBP = ctx.createBiquadFilter();
    rumbleBP.type = "bandpass";
    rumbleBP.frequency.value = 180;
    rumbleBP.Q.value = 1.2;

    const rumbleHP = ctx.createBiquadFilter();
    rumbleHP.type = "highpass";
    rumbleHP.frequency.value = 80;

    const rumbleGain = ctx.createGain();
    rumbleGain.gain.value = 0.4;

    const fireLFO = ctx.createOscillator();
    fireLFO.frequency.value = 0.1;
    const fireLFOGain = ctx.createGain();
    fireLFOGain.gain.value = 0.1;
    fireLFO.connect(fireLFOGain);
    fireLFOGain.connect(rumbleGain.gain);

    const fireLFO2 = ctx.createOscillator();
    fireLFO2.frequency.value = 0.07;
    const fireLFOGain2 = ctx.createGain();
    fireLFOGain2.gain.value = 0.06;
    fireLFO2.connect(fireLFOGain2);
    fireLFOGain2.connect(rumbleGain.gain);
    fireLFO2.start();

    fireLFO.start();

    rumbleSource.connect(rumbleBP);
    rumbleBP.connect(rumbleHP);
    rumbleHP.connect(rumbleGain);
    rumbleGain.connect(gain);
    rumbleSource.start();

    // ── Layer 2: Crackle scheduler ──
    const crackleGain = ctx.createGain();
    crackleGain.gain.value = 0.25;
    crackleGain.connect(gain);

    const densityLFO = ctx.createOscillator();
    densityLFO.frequency.value = 0.03;
    const densityLFOGain = ctx.createGain();
    densityLFOGain.gain.value = 0.3;
    densityLFO.connect(densityLFOGain);
    densityLFO.start();

    let crackleInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleCrackle = () => {
      if (!this.active.has("campfire" as SoundID)) return;

      const duration = 0.005 + Math.random() * 0.03;
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const cSource = ctx.createBufferSource();
      cSource.buffer = buf;

      const crackleHP = ctx.createBiquadFilter();
      crackleHP.type = "highpass";
      crackleHP.frequency.value = 2000 + Math.random() * 4000;
      crackleHP.Q.value = 0.5;

      const env = ctx.createGain();
      const peak = 0.3 + Math.random() * 0.5;
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(peak, ctx.currentTime + 0.002);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      const cPanner = ctx.createStereoPanner();
      cPanner.pan.value = Math.random() * 2 - 1;

      cSource.connect(crackleHP);
      crackleHP.connect(env);
      env.connect(cPanner);
      cPanner.connect(crackleGain);
      cSource.start(ctx.currentTime);
      cSource.stop(ctx.currentTime + duration);

      const densityMod = 0.5 + densityLFOGain.gain.value;
      const minDelay = Math.max(5, 60 * densityMod);
      const maxDelay = Math.max(minDelay + 10, 400 * densityMod);
      const nextDelay = minDelay + Math.random() * (maxDelay - minDelay);
      crackleInterval = setTimeout(scheduleCrackle, nextDelay);
    };

    crackleInterval = setTimeout(scheduleCrackle, 100);

    // ── Layer 3: Pop scheduler ──
    const popGain = ctx.createGain();
    popGain.gain.value = 0.4;
    popGain.connect(gain);

    let popInterval: ReturnType<typeof setTimeout> | null = null;

    const schedulePop = () => {
      if (!this.active.has("campfire" as SoundID)) return;

      const isSharpPop = Math.random() < 0.2;
      const duration = 0.05 + Math.random() * 0.1;
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const pSource = ctx.createBufferSource();
      pSource.buffer = buf;

      const popBP = ctx.createBiquadFilter();
      popBP.type = "bandpass";
      popBP.frequency.value = 500 + Math.random() * 1500;
      popBP.Q.value = isSharpPop
        ? 8 + Math.random() * 4
        : 2 + Math.random() * 2;

      const env = ctx.createGain();
      const peak = 0.4 + Math.random() * 0.4;
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(peak, ctx.currentTime + 0.005);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      const pPanner = ctx.createStereoPanner();
      pPanner.pan.value = Math.random() * 2 - 1;

      pSource.connect(popBP);
      popBP.connect(env);
      env.connect(pPanner);
      pPanner.connect(popGain);
      pSource.start(ctx.currentTime);
      pSource.stop(ctx.currentTime + duration);

      const nextDelay = 1000 + Math.random() * 5000;
      popInterval = setTimeout(schedulePop, nextDelay);
    };

    popInterval = setTimeout(schedulePop, 500);

    // ── Layer 4: Log shift scheduler ──
    const logGain = ctx.createGain();
    logGain.gain.value = 0;
    logGain.connect(gain);

    let logInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleLogShift = () => {
      if (!this.active.has("campfire" as SoundID)) return;

      const duration = 0.3 + Math.random() * 0.5;
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);

      let lastOut = 0;
      for (let i = 0; i < length; i++) {
        const white = Math.random() * 2 - 1;
        lastOut = (lastOut + 0.03 * white) / 1.03;
        data[i] = lastOut * 4;
      }

      const lSource = ctx.createBufferSource();
      lSource.buffer = buf;

      const lBP = ctx.createBiquadFilter();
      lBP.type = "bandpass";
      lBP.frequency.value = 100 + Math.random() * 200;
      lBP.Q.value = 0.8 + Math.random() * 1.0;

      const lEnv = ctx.createGain();
      const peak = 0.5 + Math.random() * 0.3;
      lEnv.gain.setValueAtTime(0, ctx.currentTime);
      lEnv.gain.linearRampToValueAtTime(peak, ctx.currentTime + 0.08);
      lEnv.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      const lPanner = ctx.createStereoPanner();
      lPanner.pan.value = (Math.random() - 0.5) * 0.4;

      lSource.connect(lBP);
      lBP.connect(lEnv);
      lEnv.connect(lPanner);
      lPanner.connect(logGain);
      lSource.start(ctx.currentTime);
      lSource.stop(ctx.currentTime + duration);

      const nextDelay = 15000 + Math.random() * 45000;
      logInterval = setTimeout(scheduleLogShift, nextDelay);
    };

    logInterval = setTimeout(scheduleLogShift, 10000 + Math.random() * 20000);

    // ── Layer 5: Cricket / Cicada scheduler ──
    // High-frequency pulse train at ~5.5kHz carrier, 28Hz modulation
    const cricketGain = ctx.createGain();
    cricketGain.gain.value = 0;
    cricketGain.connect(gain);

    let cricketInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleCricketChirp = () => {
      if (!this.active.has("campfire" as SoundID)) return;

      const startTime = ctx.currentTime;
      const chirpCount = 3 + Math.floor(Math.random() * 6); // 3-8 chirps

      for (let i = 0; i < chirpCount; i++) {
        const chirpStart = startTime + i * (CICADA_BURST_DURATION + CICADA_GAP_MIN + Math.random() * (CICADA_GAP_MAX - CICADA_GAP_MIN));

        // Amplitude-modulated sine: carrier at ~5.5kHz modulated at ~28Hz
        const chirpLength = CICADA_BURST_DURATION * (0.5 + Math.random() * 1.0);
        const chirpSamples = ctx.sampleRate * chirpLength;
        const chirpBuf = ctx.createBuffer(1, chirpSamples, ctx.sampleRate);
        const chirpData = chirpBuf.getChannelData(0);

        const carrier = CICADA_CARRIER_FREQ + (Math.random() - 0.5) * 200;
        const modFreq = CICADA_MOD_FREQ + (Math.random() - 0.5) * 8;

        for (let j = 0; j < chirpSamples; j++) {
          const t = j / ctx.sampleRate;
          const ampMod = 0.5 + 0.5 * Math.sin(2 * Math.PI * modFreq * t);
          const sample = Math.sin(2 * Math.PI * carrier * t) * ampMod;
          chirpData[j] = sample * 0.3;
        }

        const cSource = ctx.createBufferSource();
        cSource.buffer = chirpBuf;

        const cEnv = ctx.createGain();
        cEnv.gain.setValueAtTime(0, chirpStart);
        cEnv.gain.linearRampToValueAtTime(0.04 + Math.random() * 0.03, chirpStart + 0.005);
        cEnv.gain.exponentialRampToValueAtTime(0.001, chirpStart + chirpLength);

        // High-pass to ensure only the piercing upper frequencies pass
        const cHP = ctx.createBiquadFilter();
        cHP.type = "highpass";
        cHP.frequency.value = 3000;

        const cPanner = ctx.createStereoPanner();
        cPanner.pan.value = Math.random() * 2 - 1;

        cSource.connect(cHP);
        cHP.connect(cEnv);
        cEnv.connect(cPanner);
        cPanner.connect(cricketGain);
        cSource.start(chirpStart);
        cSource.stop(chirpStart + chirpLength);
      }

      // Next cricket burst in 2-8 seconds
      const nextDelay = 2000 + Math.random() * 6000;
      cricketInterval = setTimeout(scheduleCricketChirp, nextDelay);
    };

    cricketInterval = setTimeout(scheduleCricketChirp, 3000 + Math.random() * 5000);

    // Clean up on stop
    const originalStop = rumbleSource.stop.bind(rumbleSource);
    rumbleSource.stop = ((...args: [when?: number]) => {
      if (crackleInterval) {
        clearTimeout(crackleInterval);
        crackleInterval = null;
      }
      if (popInterval) {
        clearTimeout(popInterval);
        popInterval = null;
      }
      if (logInterval) {
        clearTimeout(logInterval);
        logInterval = null;
      }
      if (cricketInterval) {
        clearTimeout(cricketInterval);
        cricketInterval = null;
      }
      densityLFO.stop();
      fireLFO.stop();
      fireLFO2.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return rumbleSource;
  }

  // ─── Binaural Beat ──────────────────────────────────────────
  /**
   * Binaural beat — two sine oscillators panned hard left/right.
   * Left: 200Hz. Right: 200Hz + beatFrequency.
   * Set beat frequency via setBinauralRate().
   */
  private binauralRate = 10; // Hz (Alpha: 8-14, Beta: 14-30, Theta: 4-8, Gamma: 30-50)

  setBinauralRate(rate: number): void {
    this.binauralRate = Math.max(4, Math.min(50, rate));
  }

  getBinauralRate(): number {
    return this.binauralRate;
  }

  private makeBinauralSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // Dummy source that doesn't play audio — we use oscillators instead.
    const buffer = this.makeNoiseBuffer(ctx, 4);
    const dummy = ctx.createBufferSource();
    dummy.buffer = buffer;
    dummy.loop = true;
    const dummyGain = ctx.createGain();
    dummyGain.gain.value = 0; // silent
    dummy.connect(dummyGain);
    dummyGain.connect(gain);
    dummy.start();

    const baseFreq = 150;
    const beatFreq = this.binauralRate;

    // Soft low-pass filter to remove harshness
    const lpfL = ctx.createBiquadFilter();
    lpfL.type = "lowpass";
    lpfL.frequency.value = 3000;
    lpfL.Q.value = 0.5;

    const lpfR = ctx.createBiquadFilter();
    lpfR.type = "lowpass";
    lpfR.frequency.value = 3000;
    lpfR.Q.value = 0.5;

    // Left channel oscillators (fundamental + gentle 2nd harmonic for warmth)
    const oscL1 = ctx.createOscillator();
    oscL1.type = "sine";
    oscL1.frequency.value = baseFreq;

    const oscL2 = ctx.createOscillator();
    oscL2.type = "sine";
    oscL2.frequency.value = baseFreq * 2;

    const gainL = ctx.createGain();
    gainL.gain.value = 0.2;

    const gainL2 = ctx.createGain();
    gainL2.gain.value = 0.03; // subtle second harmonic

    const pannerL = ctx.createStereoPanner();
    pannerL.pan.value = -1;

    oscL1.connect(gainL);
    oscL2.connect(gainL2);
    gainL.connect(lpfL);
    gainL2.connect(lpfL);
    lpfL.connect(pannerL);
    pannerL.connect(gain);

    // Right channel oscillators (offset by beat frequency)
    const oscR1 = ctx.createOscillator();
    oscR1.type = "sine";
    oscR1.frequency.value = baseFreq + beatFreq;

    const oscR2 = ctx.createOscillator();
    oscR2.type = "sine";
    oscR2.frequency.value = (baseFreq + beatFreq) * 2;

    const gainR = ctx.createGain();
    gainR.gain.value = 0.2;

    const gainR2 = ctx.createGain();
    gainR2.gain.value = 0.03;

    const pannerR = ctx.createStereoPanner();
    pannerR.pan.value = 1;

    oscR1.connect(gainR);
    oscR2.connect(gainR2);
    gainR.connect(lpfR);
    gainR2.connect(lpfR);
    lpfR.connect(pannerR);
    pannerR.connect(gain);

    oscL1.start();
    oscL2.start();
    oscR1.start();
    oscR2.start();

    // Cleanup on stop
    const originalStop = dummy.stop.bind(dummy);
    dummy.stop = ((...args: [when?: number]) => {
      oscL1.stop();
      oscL2.stop();
      oscR1.stop();
      oscR2.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return dummy;
  }

  // ─── Isochronic Tone ────────────────────────────────────────
  /**
   * Isochronic tone — a single sine wave amplitude-modulated by a square wave
   * at the target beat frequency. Works without headphones.
   */
  private isochronicRate = 10; // Hz

  setIsochronicRate(rate: number): void {
    this.isochronicRate = Math.max(4, Math.min(50, rate));
  }

  getIsochronicRate(): number {
    return this.isochronicRate;
  }

  private makeIsochronicSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // Dummy buffer source to return (carries the stop lifecycle)
    const buffer = this.makeNoiseBuffer(ctx, 4);
    const dummy = ctx.createBufferSource();
    dummy.buffer = buffer;
    dummy.loop = true;
    const dummyGain2 = ctx.createGain();
    dummyGain2.gain.value = 0;
    dummy.connect(dummyGain2);
    dummyGain2.connect(gain);
    dummy.start();

    const carrierFreq = 180; // Hz (lower = warmer)
    const modFreq = this.isochronicRate;

    // Carrier oscillator — sine at 180Hz
    const carrier = ctx.createOscillator();
    carrier.type = "sine";
    carrier.frequency.value = carrierFreq;

    // Soft low-pass filter for the output to remove harsh edges
    const isoLpf = ctx.createBiquadFilter();
    isoLpf.type = "lowpass";
    isoLpf.frequency.value = 4000;
    isoLpf.Q.value = 0.5;

    // Gain node that acts as the amplitude modulation VCA
    const modGain = ctx.createGain();
    modGain.gain.value = 0;

    // Smoothed isochronic pulse: use a triangle-like envelope (fade in/out)
    // instead of a hard square wave, to avoid clicking.
    const modLength = ctx.sampleRate / modFreq; // samples per cycle
    const modBuffer = ctx.createBuffer(1, Math.ceil(modLength), ctx.sampleRate);
    const modData = modBuffer.getChannelData(0);
    const halfSamples = Math.floor(modLength * 0.35); // 35% duty cycle
    const fadeSamples = Math.floor(modLength * 0.1);  // 10% fade on each edge
    for (let i = 0; i < modData.length; i++) {
      if (i < fadeSamples) {
        // Fade in
        modData[i] = i / fadeSamples;
      } else if (i < halfSamples - fadeSamples) {
        // Sustain
        modData[i] = 1;
      } else if (i < halfSamples) {
        // Fade out
        modData[i] = (halfSamples - i) / fadeSamples;
      } else {
        // Off
        modData[i] = 0;
      }
    }

    const modSource = ctx.createBufferSource();
    modSource.buffer = modBuffer;
    modSource.loop = true;

    const modGainAmount = ctx.createGain();
    modGainAmount.gain.value = 0.25;

    modSource.connect(modGainAmount);
    modGainAmount.connect(modGain.gain);
    modSource.start();

    carrier.connect(modGain);
    modGain.connect(isoLpf);
    isoLpf.connect(gain);

    carrier.start();

    // Cleanup
    const originalStop = dummy.stop.bind(dummy);
    dummy.stop = ((...args: [when?: number]) => {
      carrier.stop();
      modSource.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return dummy;
  }

  // ─── Meditation Bowl (Singing Bowl) ─────────────────────────
  /**
   * Meditation bowl — multiple harmonic sine oscillators with exponential decay,
   * re-triggered periodically to create a resonant bell-like tone.
   *
   * Harmonics: 1.0, 2.8, 5.4, 8.5 times the fundamental (~260Hz)
   */
  private makeMeditationBowlSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // Dummy buffer source for lifecycle
    const buffer = this.makeNoiseBuffer(ctx, 4);
    const dummy = ctx.createBufferSource();
    dummy.buffer = buffer;
    dummy.loop = true;
    const dummyGain3 = ctx.createGain();
    dummyGain3.gain.value = 0;
    dummy.connect(dummyGain3);
    dummyGain3.connect(gain);
    dummy.start();

    const fundamental = 260; // Hz
    const harmonics = [1.0, 2.8, 5.4, 8.5];
    const amplitudes = [0.2, 0.12, 0.06, 0.03];
    const decayTime = 4; // seconds
    const minInterval = 4000; // ms between strikes
    const maxInterval = 8000;

    let strikeInterval: ReturnType<typeof setTimeout> | null = null;

    const strike = () => {
      if (!this.active.has("meditation-bowl" as SoundID)) return;

      const startTime = ctx.currentTime;

      for (let h = 0; h < harmonics.length; h++) {
        const osc = ctx.createOscillator();
        osc.type = "sine";
        osc.frequency.value = fundamental * harmonics[h];

        const env = ctx.createGain();
        env.gain.setValueAtTime(amplitudes[h], startTime);
        env.gain.exponentialRampToValueAtTime(0.001, startTime + decayTime);

        const panner = ctx.createStereoPanner();
        panner.pan.value = (Math.random() - 0.5) * 0.3;

        osc.connect(env);
        env.connect(panner);
        panner.connect(gain);
        osc.start(startTime);
        osc.stop(startTime + decayTime + 0.1);
      }

      const nextDelay = minInterval + Math.random() * (maxInterval - minInterval);
      strikeInterval = setTimeout(strike, nextDelay);
    };

    strikeInterval = setTimeout(strike, 500);

    // Cleanup
    const originalStop = dummy.stop.bind(dummy);
    dummy.stop = ((...args: [when?: number]) => {
      if (strikeInterval) {
        clearTimeout(strikeInterval);
        strikeInterval = null;
      }
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return dummy;
  }
  // ─── Ambient Pad (Focus Music Drone) ────────────────────────
  /**
   * Evolving chord drone inspired by YouTube focus music.
   *
   * Design:
   *  - 4 detuned triangle oscillators per note for lush chorus
   *  - Slow 6-note chord (root, 5th, octave, 10th, 12th, 15th)
   *  - Individual volume LFOs at different rates for organic evolution
   *  - Slow low-pass filter sweep (400-800Hz over 60s)
   *  - Subtle tape warble (pitch modulation ±1Hz)
   */
  private makeAmbientPadSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    const buffer = this.makeNoiseBuffer(ctx, 4);
    const dummy = ctx.createBufferSource();
    dummy.buffer = buffer;
    dummy.loop = true;
    const dGain = ctx.createGain();
    dGain.gain.value = 0;
    dummy.connect(dGain);
    dGain.connect(gain);
    dummy.start();

    const rootFreq = 130.81; // C3

    // Chord intervals: root, 5th, octave, 10th, 12th, 15th
    const intervals = [1, 1.5, 2, 2.5, 3, 4];
    // Detune amounts (cents) for 4 voices per note
    const detunes = [-5, -1.5, 1.5, 5];

    // Shared slow filter sweep
    const padFilter = ctx.createBiquadFilter();
    padFilter.type = "lowpass";
    padFilter.frequency.setValueAtTime(400, ctx.currentTime);
    padFilter.frequency.linearRampToValueAtTime(800, ctx.currentTime + 60);
    padFilter.Q.value = 0.7;
    padFilter.connect(gain);

    // Tape warble: slow pitch modulation
    const warbleLFO = ctx.createOscillator();
    warbleLFO.frequency.value = 0.1; // 10-second cycle
    const warbleGain = ctx.createGain();
    warbleGain.gain.value = 1; // ±1Hz pitch modulation
    warbleLFO.connect(warbleGain);
    warbleLFO.start();

    const allOscs: OscillatorNode[] = [];

    for (let i = 0; i < intervals.length; i++) {
      const baseFreq = rootFreq * intervals[i];

      // Individual volume LFO at unique slow rate
      const volLFO = ctx.createOscillator();
      volLFO.frequency.value = 0.02 + (i * 0.005); // 20-50s cycles
      const volLFOGain = ctx.createGain();
      volLFOGain.gain.value = 0.15; // modulation depth
      volLFO.connect(volLFOGain);
      volLFO.start();

      for (let d = 0; d < detunes.length; d++) {
        const osc = ctx.createOscillator();
        osc.type = "triangle"; // warmer than sine
        osc.frequency.value = baseFreq * Math.pow(2, detunes[d] / 1200);

        // Apply warble
        warbleGain.connect(osc.frequency);

        const oscGain = ctx.createGain();
        const baseAmp = 0.08 / (d + 1); // fundamental strongest, voices quieter
        oscGain.gain.setValueAtTime(baseAmp, ctx.currentTime);
        volLFOGain.connect(oscGain.gain); // modulate amplitude

        osc.connect(oscGain);
        oscGain.connect(padFilter);
        osc.start();
        allOscs.push(osc);
      }
    }

    // Cleanup on stop
    const originalStop = dummy.stop.bind(dummy);
    dummy.stop = ((...args: [when?: number]) => {
      for (const osc of allOscs) {
        osc.stop();
      }
      warbleLFO.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return dummy;
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