import { getSharedAudioContext, scheduleContextCleanup } from "./audioContext";

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
    const ctx = getSharedAudioContext();

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
    scheduleContextCleanup(totalDuration * 1000 + 100);
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
    const ctx = getSharedAudioContext();

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

    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable — silently ignore
  }
}

/**
 * Play a short two-note descending chime to signal a habit uncompletion.
 * The mirror of playHabitCompletionSound: starts on E5 (659.25 Hz) and
 * descends to C5 (523.25 Hz), giving an intuitive "undo" feel.
 */
export function playHabitUncompletionSound(): void {
  try {
    const ctx = getSharedAudioContext();

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

    // --- First note: E5 (659.25 Hz) ---
    const osc1 = ctx.createOscillator();
    osc1.type = "sine";
    osc1.frequency.value = 659.25;
    osc1.connect(gain);
    osc1.start(now);
    osc1.stop(now + noteDuration);

    // --- Second note: C5 (523.25 Hz) ---
    const osc2 = ctx.createOscillator();
    osc2.type = "sine";
    osc2.frequency.value = 523.25;
    osc2.connect(gain);
    osc2.start(now + noteDuration + gap);
    osc2.stop(now + noteDuration * 2 + gap);

    // Clean up the context after the sound finishes
    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable — silently ignore
  }
}

/**
 * Play a repeating three-beep alarm sound to signal a scheduled habit is
 * coming up. Uses a distinctive ascending pattern (E5 → G5 → A5) with
 * short gaps, distinct from the completion/uncompletion chimes.
 *
 * Note: This function is async because an AudioContext created outside of a
 * user gesture (e.g. from a setTimeout/setInterval callback) starts in a
 * "suspended" state in modern browsers, and must be resumed explicitly via
 * ctx.resume() before any scheduled oscillators will produce sound.
 */
export async function playAlarmSound(): Promise<void> {
  try {
    const ctx = getSharedAudioContext();

    // Resume the context first — critical for alarms fired from timers
    // (no user gesture available). Without this, oscillators are scheduled
    // on a suspended context and produce silence.
    if (ctx.state === "suspended") {
      await ctx.resume();
    }

    const now = ctx.currentTime;
    const noteDuration = 0.18;
    const gap = 0.12;
    const totalDuration = noteDuration * 3 + gap * 2 + 0.05;

    // Envelope — moderate peak, clear and audible
    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.35, now + 0.008);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    // --- Three ascending notes: E5 (659.25 Hz), G5 (784.00 Hz), A5 (880.00 Hz) ---
    const notes = [659.25, 784.0, 880.0];
    notes.forEach((freq, i) => {
      const osc = ctx.createOscillator();
      osc.type = "sine";
      osc.frequency.value = freq;
      osc.connect(gain);
      const start = now + i * (noteDuration + gap);
      osc.start(start);
      osc.stop(start + noteDuration);
    });

    // Clean up the context after the sound finishes
    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable — silently ignore
  }
}

/**
 * Play a short low single-note ping to signal a regular to-do task uncompletion.
 * Uses G4 (392 Hz) with a quick attack and decay — a lower, softer tone
 * distinct from the completion ping, signaling the action was reversed.
 */
export function playTodoUncompletionSound(): void {
  try {
    const ctx = getSharedAudioContext();

    const now = ctx.currentTime;
    const totalDuration = 0.2;

    // Envelope — smooth bell-like attack and decay
    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.2, now + 0.005);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    // --- Single note: G4 (392 Hz) with a slightly softer timbre ---
    const osc = ctx.createOscillator();
    osc.type = "triangle";
    osc.frequency.value = 392;
    osc.connect(gain);
    osc.start(now);
    osc.stop(now + totalDuration);

    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable — silently ignore
  }
}