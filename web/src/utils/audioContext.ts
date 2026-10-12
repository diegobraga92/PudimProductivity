/** Singleton AudioContext for short UI sounds, created on first use and resumed if suspended. */
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