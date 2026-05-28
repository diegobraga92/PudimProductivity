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

export type SoundID = "white-noise" | "pink-noise" | "brown-noise" | "rain" | "ocean" | "wind" | "campfire";

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
      case "wind":
        return this.makeWindSource(ctx, gain);
      case "campfire":
        return this.makeCampfireSource(ctx, gain);
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

  /** Rain — layered: background wash + randomized droplets with variable intensity. */
  private makeRainSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Background wash ──
    // A filtered noise bed that provides the "hiss" of falling rain
    const washBuffer = this.makeNoiseBuffer(ctx, 4);
    const washSource = ctx.createBufferSource();
    washSource.buffer = washBuffer;
    washSource.loop = true;

    const washGain = ctx.createGain();
    washGain.gain.value = 0.3;

    // Two cascading low-pass filters for a softer, more natural wash
    const washLP1 = ctx.createBiquadFilter();
    washLP1.type = "lowpass";
    washLP1.frequency.value = 4000;
    washLP1.Q.value = 0.5;

    const washLP2 = ctx.createBiquadFilter();
    washLP2.type = "lowpass";
    washLP2.frequency.value = 2000;
    washLP2.Q.value = 0.3;

    // Slow LFO to gently modulate the wash intensity (rain showers vary)
    const washLFO = ctx.createOscillator();
    washLFO.frequency.value = 0.08; // ~12-second cycle
    const washLFOGain = ctx.createGain();
    washLFOGain.gain.value = 0.08;
    washLFO.connect(washLFOGain);
    washLFOGain.connect(washGain.gain);
    washLFO.start();

    washSource.connect(washLP1);
    washLP1.connect(washLP2);
    washLP2.connect(washGain);
    washGain.connect(gain);
    washSource.start();

    // ── Layer 2: Droplet scheduler ──
    // Creates short noise bursts at random intervals with randomized pitch and size.
    // Uses a bimodal distribution: many small drops + occasional large drops.
    const dropletGain = ctx.createGain();
    dropletGain.gain.value = 0.45;
    dropletGain.connect(gain);

    let dropletInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleDroplet = () => {
      if (!this.active.has("rain" as SoundID)) return;

      // Bimodal droplet size: 80% small/medium, 20% large
      const isLarge = Math.random() < 0.2;
      const duration = isLarge
        ? 0.08 + Math.random() * 0.12 // 80-200ms (large drops)
        : 0.02 + Math.random() * 0.05; // 20-70ms (small/medium drops)

      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const dSource = ctx.createBufferSource();
      dSource.buffer = buf;

      // Bandpass filter: large drops are lower-pitched, small drops are higher-pitched
      const bp = ctx.createBiquadFilter();
      bp.type = "bandpass";
      if (isLarge) {
        bp.frequency.value = 800 + Math.random() * 1200; // 800-2000Hz — deeper "plop"
        bp.Q.value = 0.8 + Math.random() * 1.2;
      } else {
        bp.frequency.value = 2500 + Math.random() * 3500; // 2500-6000Hz — sharper "tick"
        bp.Q.value = 1.5 + Math.random() * 2.5;
      }

      // Quick gain envelope with faster attack for more percussive feel
      const env = ctx.createGain();
      const peak = isLarge ? 0.5 + Math.random() * 0.5 : 0.3 + Math.random() * 0.4;
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(peak, ctx.currentTime + 0.003);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      dSource.connect(bp);
      bp.connect(env);
      env.connect(dropletGain);
      dSource.start(ctx.currentTime);
      dSource.stop(ctx.currentTime + duration);

      // Schedule next droplet: faster for small drops, slower for large
      const nextDelay = isLarge
        ? 80 + Math.random() * 200  // 80-280ms between large drops
        : 5 + Math.random() * 25;    // 5-30ms between small drops

      dropletInterval = setTimeout(scheduleDroplet, nextDelay);
    };

    // Start scheduling droplets
    dropletInterval = setTimeout(scheduleDroplet, 50);

    // Store the interval so we can clear it on stop
    const originalStop = washSource.stop.bind(washSource);
    washSource.stop = ((...args: [when?: number]) => {
      if (dropletInterval) {
        clearTimeout(dropletInterval);
        dropletInterval = null;
      }
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return washSource;
  }

  /** Ocean — layered: deep wash with complex swell + wave sets with foam. */
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

    // Complex swell: two LFOs at different rates for more organic motion
    const swellLFO1 = ctx.createOscillator();
    swellLFO1.frequency.value = 0.03; // ~33-second cycle
    const swellGain1 = ctx.createGain();
    swellGain1.gain.value = 0.15;
    swellLFO1.connect(swellGain1);
    swellGain1.connect(washGain.gain);

    const swellLFO2 = ctx.createOscillator();
    swellLFO2.frequency.value = 0.07; // ~14-second cycle
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
    // Uses "wave sets" — groups of 3-5 waves with increasing then decreasing intensity
    const crashGain = ctx.createGain();
    crashGain.gain.value = 0.5;
    crashGain.connect(gain);

    let crashInterval: ReturnType<typeof setTimeout> | null = null;
    let waveSetCount = 0;

    const scheduleCrash = () => {
      if (!this.active.has("ocean" as SoundID)) return;

      // Wave set logic: 3-5 waves per set
      const wavesPerSet = 3 + Math.floor(Math.random() * 3); // 3-5
      const progress = waveSetCount / wavesPerSet; // 0 to 1
      const intensity = Math.sin(progress * Math.PI); // bell curve: 0 → 1 → 0

      const duration = 1.5 + intensity * 2.5; // 1.5-4 seconds, longer for bigger waves
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);

      // Crash sound: noise burst with exponential decay
      for (let i = 0; i < length; i++) {
        const t = i / length;
        const envelope = Math.exp(-t * (3 + intensity * 2)); // faster decay for bigger waves
        data[i] = (Math.random() * 2 - 1) * envelope;
      }

      const cSource = ctx.createBufferSource();
      cSource.buffer = buf;

      // Low-pass that opens briefly then closes (wave crash)
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

      cSource.connect(lp);
      lp.connect(env);
      env.connect(crashGain);
      cSource.start(ctx.currentTime);
      cSource.stop(ctx.currentTime + duration);

      // ── Layer 3: Foam "shhh" after each crash ──
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

      fSource.connect(foamLP);
      foamLP.connect(foamEnv);
      foamEnv.connect(crashGain);
      fSource.start(ctx.currentTime + 0.1);
      fSource.stop(ctx.currentTime + 0.1 + foamDuration);

      // Advance wave set counter
      waveSetCount += 1;
      if (waveSetCount >= wavesPerSet) {
        waveSetCount = 0;
      }

      // Next crash: 3-8 seconds between waves, shorter within a set
      const nextDelay = 3000 + Math.random() * 5000;
      crashInterval = setTimeout(scheduleCrash, nextDelay);
    };

    crashInterval = setTimeout(scheduleCrash, 1000);

    // Clean up scheduler on stop
    const originalStop = washSource.stop.bind(washSource);
    washSource.stop = ((...args: [when?: number]) => {
      if (crashInterval) {
        clearTimeout(crashInterval);
        crashInterval = null;
      }
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return washSource;
  }

  /**
   * Wind — filtered noise with howling resonance and gust simulation.
   *
   * Three layers:
   *   1. Brown noise base (low rumble)
   *   2. Band-passed noise with drifting center frequency (the "howl")
   *   3. Gust events — brief volume swells with filter opening
   */
  private makeWindSource(ctx: AudioContext, gain: GainNode): AudioBufferSourceNode {
    // ── Layer 1: Brown noise base (deep wind rumble) ──
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
    // Band-passed noise with a slow LFO sweeping the center frequency
    const howlBuffer = this.makeNoiseBuffer(ctx, 4);
    const howlSource = ctx.createBufferSource();
    howlSource.buffer = howlBuffer;
    howlSource.loop = true;

    const howlBP = ctx.createBiquadFilter();
    howlBP.type = "bandpass";
    howlBP.frequency.value = 400;
    howlBP.Q.value = 3;

    // LFO to sweep the howl frequency up and down
    const howlLFO = ctx.createOscillator();
    howlLFO.frequency.value = 0.05; // 20-second cycle
    const howlLFOGain = ctx.createGain();
    howlLFOGain.gain.value = 250; // sweep range: ±250Hz
    howlLFO.connect(howlLFOGain);
    howlLFOGain.connect(howlBP.frequency);
    howlLFO.start();

    // Second LFO for Q modulation (resonance varies)
    const qLFO = ctx.createOscillator();
    qLFO.frequency.value = 0.03; // ~33-second cycle
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
    // Brief events where volume swells and filter opens
    const gustGain = ctx.createGain();
    gustGain.gain.value = 0;
    gustGain.connect(gain);

    let gustInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleGust = () => {
      if (!this.active.has("wind" as SoundID)) return;

      const gustDuration = 1 + Math.random() * 3; // 1-4 seconds
      const gustLength = ctx.sampleRate * gustDuration;
      const buf = ctx.createBuffer(1, gustLength, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < gustLength; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const gSource = ctx.createBufferSource();
      gSource.buffer = buf;

      // Band-pass that opens during gust
      const gBP = ctx.createBiquadFilter();
      gBP.type = "bandpass";
      gBP.frequency.setValueAtTime(200, ctx.currentTime);
      gBP.frequency.linearRampToValueAtTime(800 + Math.random() * 1200, ctx.currentTime + 0.5);
      gBP.frequency.exponentialRampToValueAtTime(200, ctx.currentTime + gustDuration);
      gBP.Q.value = 1.5;

      // Volume swell envelope
      const gEnv = ctx.createGain();
      gEnv.gain.setValueAtTime(0, ctx.currentTime);
      gEnv.gain.linearRampToValueAtTime(0.3 + Math.random() * 0.3, ctx.currentTime + 0.4);
      gEnv.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + gustDuration);

      gSource.connect(gBP);
      gBP.connect(gEnv);
      gEnv.connect(gustGain);
      gSource.start(ctx.currentTime);
      gSource.stop(ctx.currentTime + gustDuration);

      // Next gust in 5-20 seconds
      const nextDelay = 5000 + Math.random() * 15000;
      gustInterval = setTimeout(scheduleGust, nextDelay);
    };

    gustInterval = setTimeout(scheduleGust, 3000);

    // Clean up on stop
    const originalStop = baseSource.stop.bind(baseSource);
    baseSource.stop = ((...args: [when?: number]) => {
      if (gustInterval) {
        clearTimeout(gustInterval);
        gustInterval = null;
      }
      howlLFO.stop();
      qLFO.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return baseSource;
  }

  /**
   * Campfire — crackling embers, pops, and low rumble.
   *
   * Three layers:
   *   1. Low rumble (brown noise, band-passed for the fire's roar)
   *   2. Crackles — very short noise bursts at random intervals
   *   3. Pops — occasional louder mid-frequency bursts
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

    // Slow LFO to make the fire "breathe"
    const fireLFO = ctx.createOscillator();
    fireLFO.frequency.value = 0.1; // 10-second cycle
    const fireLFOGain = ctx.createGain();
    fireLFOGain.gain.value = 0.1;
    fireLFO.connect(fireLFOGain);
    fireLFOGain.connect(rumbleGain.gain);
    fireLFO.start();

    rumbleSource.connect(rumbleBP);
    rumbleBP.connect(rumbleHP);
    rumbleHP.connect(rumbleGain);
    rumbleGain.connect(gain);
    rumbleSource.start();

    // ── Layer 2: Crackle scheduler ──
    // Very short, high-frequency noise bursts (the "crackling" sound)
    const crackleGain = ctx.createGain();
    crackleGain.gain.value = 0.3;
    crackleGain.connect(gain);

    let crackleInterval: ReturnType<typeof setTimeout> | null = null;

    const scheduleCrackle = () => {
      if (!this.active.has("campfire" as SoundID)) return;

      const duration = 0.005 + Math.random() * 0.03; // 5-35ms
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const cSource = ctx.createBufferSource();
      cSource.buffer = buf;

      // High-pass filter for bright, crispy crackle
      const crackleHP = ctx.createBiquadFilter();
      crackleHP.type = "highpass";
      crackleHP.frequency.value = 2000 + Math.random() * 4000;
      crackleHP.Q.value = 0.5;

      // Fast attack/decay envelope
      const env = ctx.createGain();
      const peak = 0.3 + Math.random() * 0.5;
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(peak, ctx.currentTime + 0.002);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      cSource.connect(crackleHP);
      crackleHP.connect(env);
      env.connect(crackleGain);
      cSource.start(ctx.currentTime);
      cSource.stop(ctx.currentTime + duration);

      // Random panning for spatial feel
      const panner = ctx.createStereoPanner();
      panner.pan.value = (Math.random() * 2 - 1) * 0.6;
      // We can't easily insert panner after env, so we'll just let it be mono
      // The crackles naturally feel spatial due to randomness

      // Next crackle in 20-200ms
      const nextDelay = 20 + Math.random() * 180;
      crackleInterval = setTimeout(scheduleCrackle, nextDelay);
    };

    crackleInterval = setTimeout(scheduleCrackle, 100);

    // ── Layer 3: Pop scheduler ──
    // Occasional louder, lower-pitched bursts
    const popGain = ctx.createGain();
    popGain.gain.value = 0.4;
    popGain.connect(gain);

    let popInterval: ReturnType<typeof setTimeout> | null = null;

    const schedulePop = () => {
      if (!this.active.has("campfire" as SoundID)) return;

      const duration = 0.05 + Math.random() * 0.1; // 50-150ms
      const length = ctx.sampleRate * duration;
      const buf = ctx.createBuffer(1, length, ctx.sampleRate);
      const data = buf.getChannelData(0);
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1;
      }

      const pSource = ctx.createBufferSource();
      pSource.buffer = buf;

      // Band-pass for a fuller "pop" sound
      const popBP = ctx.createBiquadFilter();
      popBP.type = "bandpass";
      popBP.frequency.value = 500 + Math.random() * 1500;
      popBP.Q.value = 2 + Math.random() * 2;

      // Envelope: fast attack, medium decay
      const env = ctx.createGain();
      const peak = 0.4 + Math.random() * 0.4;
      env.gain.setValueAtTime(0, ctx.currentTime);
      env.gain.linearRampToValueAtTime(peak, ctx.currentTime + 0.005);
      env.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

      pSource.connect(popBP);
      popBP.connect(env);
      env.connect(popGain);
      pSource.start(ctx.currentTime);
      pSource.stop(ctx.currentTime + duration);

      // Next pop in 1-6 seconds
      const nextDelay = 1000 + Math.random() * 5000;
      popInterval = setTimeout(schedulePop, nextDelay);
    };

    popInterval = setTimeout(schedulePop, 500);

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
      fireLFO.stop();
      return originalStop(...args);
    }) as AudioScheduledSourceNode["stop"];

    return rumbleSource;
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
