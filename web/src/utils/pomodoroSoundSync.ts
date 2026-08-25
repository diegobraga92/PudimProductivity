import type { SessionStatus } from "../api/pomodoro";
import type { SoundID } from "./audio";
import { SOUNDS } from "./soundCatalog";

/**
 * Pomodoro ↔ Soundscape sync settings.
 *
 * Single source of truth for "auto-play a sound while the pomodoro timer is
 * running". Values are persisted in localStorage (legacy keys are kept so
 * existing settings survive), and a tiny external store lets both the Pomodoro
 * page and the app-root automation hook react to changes.
 */

export const POMODORO_SYNC_ENABLED_KEY = "soundscape_pomodoro_enabled";
export const POMODORO_SYNC_SOUND_KEY = "soundscape_pomodoro_sound";
export const DEFAULT_SYNC_SOUND: SoundID = "rain";

const VALID_SOUND_IDS = new Set<SoundID>(SOUNDS.map((s) => s.id));

export function getPomodoroSyncEnabled(): boolean {
  return localStorage.getItem(POMODORO_SYNC_ENABLED_KEY) === "true";
}

export function setPomodoroSyncEnabled(enabled: boolean): void {
  localStorage.setItem(POMODORO_SYNC_ENABLED_KEY, String(enabled));
  emit();
}

export function getPomodoroSyncSound(): SoundID {
  const raw = localStorage.getItem(POMODORO_SYNC_SOUND_KEY);
  const id = raw as SoundID;
  return id && VALID_SOUND_IDS.has(id) ? id : DEFAULT_SYNC_SOUND;
}

export function setPomodoroSyncSound(sound: SoundID): void {
  localStorage.setItem(POMODORO_SYNC_SOUND_KEY, sound);
  emit();
}

type Listener = () => void;
const listeners = new Set<Listener>();

/** Subscribes to sync-setting changes. Returns an unsubscribe function. */
export function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function emit(): void {
  for (const listener of listeners) listener();
}

export type SyncAction = "play" | "stop" | "idle";

/**
 * Decides what the automation hook should do for a given timer status.
 * Disabled sync never touches the engine ("idle"); when enabled, the sound
 * plays while the timer runs and stops on any other status (paused, completed,
 * cancelled, no session).
 */
export function resolveSyncAction(enabled: boolean, status: SessionStatus | null): SyncAction {
  if (!enabled) return "idle";
  return status === "running" ? "play" : "stop";
}
