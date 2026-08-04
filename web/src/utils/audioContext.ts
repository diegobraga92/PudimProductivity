/**
 * Shared lazy-initialized AudioContext for short UI feedback sounds.
 *
 * Creating/closing an AudioContext per sound is wasteful and can cause
 * performance issues. This singleton reuses a single context across all
 * UI sound effects, creating it on first use and resuming it if suspended.
 */
let sharedCtx: AudioContext | null = null;

/** Get (or lazily create) the shared AudioContext. */
export function getSharedAudioContext(): AudioContext {
  if (!sharedCtx) {
    sharedCtx = new AudioContext({ latencyHint: "interactive" });
  }
  if (sharedCtx.state === "suspended") {
    sharedCtx.resume();
  }
  return sharedCtx;
}

/** Schedule a cleanup timer to close the shared context after a delay. */
export function scheduleContextCleanup(delayMs: number): void {
  setTimeout(() => {
    if (sharedCtx && sharedCtx.state !== "closed") {
      sharedCtx.close();
      sharedCtx = null;
    }
  }, delayMs);
}