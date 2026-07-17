/**
 * Lightweight utility for short UI feedback sounds.
 * Uses Web Audio API — no external files needed.
 */

/**
 * Play a short two-note ascending chime to signal a habit/task completion.
 * The chime consists of C5 (523 Hz) followed by E5 (659 Hz), each with a
 * quick attack and decay.
 */
export function playHabitCompletionSound(): void {
  try {
    const ctx = new AudioContext({ latencyHint: "interactive" });

    const now = ctx.currentTime;
    const noteDuration = 0.12;
    const gap = 0.06;
    const totalDuration = noteDuration * 2 + gap + 0.05;

    // Envelope
    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.3, now + 0.005);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    // --- First note: C5 (523.25 Hz) ---
    const osc1 = ctx.createOscillator();
    osc1.type = "sine";
    osc1.frequency.value = 523.25;
    osc1.connect(gain);
    osc1.start(now);
    osc1.stop(now + noteDuration);

    // --- Second note: E5 (659.25 Hz) ---
    const osc2 = ctx.createOscillator();
    osc2.type = "sine";
    osc2.frequency.value = 659.25;
    osc2.connect(gain);
    osc2.start(now + noteDuration + gap);
    osc2.stop(now + noteDuration * 2 + gap);

    // Clean up the context after the sound finishes
    setTimeout(() => {
      ctx.close();
    }, totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable — silently ignore
  }
}

/**
 * Play a short single-note ping to signal a regular to-do task completion.
 * Uses E5 (659.25 Hz) with a quick attack and longer tail, distinct from
 * the two-note ascending chime used for habits.
 */
export function playTodoCompletionSound(): void {
  try {
    const ctx = new AudioContext({ latencyHint: "interactive" });

    const now = ctx.currentTime;
    const totalDuration = 0.2;

    // Envelope — smooth bell-like attack and decay
    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.25, now + 0.005);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    // --- Single note: E5 (659.25 Hz) with a slightly softer timbre ---
    const osc = ctx.createOscillator();
    osc.type = "triangle";
    osc.frequency.value = 659.25;
    osc.connect(gain);
    osc.start(now);
    osc.stop(now + totalDuration);

    setTimeout(() => {
      ctx.close();
    }, totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable — silently ignore
  }
}
