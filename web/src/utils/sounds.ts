import { getSharedAudioContext, scheduleContextCleanup } from "./audioContext";

/** Short UI feedback sounds synthesized with the Web Audio API. */

/** Two-note ascending chime (C5 → E5) signalling a habit/task completion. */
export function playHabitCompletionSound(): void {
  try {
    const ctx = getSharedAudioContext();

    const now = ctx.currentTime;
    const noteDuration = 0.12;
    const gap = 0.06;
    const totalDuration = noteDuration * 2 + gap + 0.05;

    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.3, now + 0.005);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    const osc1 = ctx.createOscillator();
    osc1.type = "sine";
    osc1.frequency.value = 523.25;
    osc1.connect(gain);
    osc1.start(now);
    osc1.stop(now + noteDuration);

    const osc2 = ctx.createOscillator();
    osc2.type = "sine";
    osc2.frequency.value = 659.25;
    osc2.connect(gain);
    osc2.start(now + noteDuration + gap);
    osc2.stop(now + noteDuration * 2 + gap);

    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable.
  }
}

/** Single soft E5 ping signalling a regular to-do completion. */
export function playTodoCompletionSound(): void {
  try {
    const ctx = getSharedAudioContext();

    const now = ctx.currentTime;
    const totalDuration = 0.2;

    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.25, now + 0.005);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    const osc = ctx.createOscillator();
    osc.type = "triangle";
    osc.frequency.value = 659.25;
    osc.connect(gain);
    osc.start(now);
    osc.stop(now + totalDuration);

    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable.
  }
}

/** Two-note descending chime (E5 → C5) signalling a habit uncompletion. */
export function playHabitUncompletionSound(): void {
  try {
    const ctx = getSharedAudioContext();

    const now = ctx.currentTime;
    const noteDuration = 0.12;
    const gap = 0.06;
    const totalDuration = noteDuration * 2 + gap + 0.05;

    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.3, now + 0.005);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    const osc1 = ctx.createOscillator();
    osc1.type = "sine";
    osc1.frequency.value = 659.25;
    osc1.connect(gain);
    osc1.start(now);
    osc1.stop(now + noteDuration);

    const osc2 = ctx.createOscillator();
    osc2.type = "sine";
    osc2.frequency.value = 523.25;
    osc2.connect(gain);
    osc2.start(now + noteDuration + gap);
    osc2.stop(now + noteDuration * 2 + gap);

    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable.
  }
}

/** Repeating three-beep alarm (E5-G5-A5); async because a timer-created AudioContext starts suspended. */
export async function playAlarmSound(): Promise<void> {
  try {
    const ctx = getSharedAudioContext();

    if (ctx.state === "suspended") {
      await ctx.resume();
    }

    const now = ctx.currentTime;
    const noteDuration = 0.18;
    const gap = 0.12;
    const totalDuration = noteDuration * 3 + gap * 2 + 0.05;

    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.35, now + 0.008);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

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

    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable.
  }
}

/** Single low G4 ping signalling a regular to-do uncompletion. */
export function playTodoUncompletionSound(): void {
  try {
    const ctx = getSharedAudioContext();

    const now = ctx.currentTime;
    const totalDuration = 0.2;

    const gain = ctx.createGain();
    gain.connect(ctx.destination);
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.2, now + 0.005);
    gain.gain.exponentialRampToValueAtTime(0.001, now + totalDuration);

    const osc = ctx.createOscillator();
    osc.type = "triangle";
    osc.frequency.value = 392;
    osc.connect(gain);
    osc.start(now);
    osc.stop(now + totalDuration);

    scheduleContextCleanup(totalDuration * 1000 + 100);
  } catch {
    // Web Audio API unavailable.
  }
}
