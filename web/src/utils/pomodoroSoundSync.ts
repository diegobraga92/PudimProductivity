import type { SessionStatus } from "../api/pomodoro";
import type { SoundID } from "./audio";
import { createExternalStore } from "./externalStore";
import { BUILTIN_SOUND_IDS } from "./soundCatalog";

/** Pomodoro <-> sound sync settings in localStorage (legacy keys kept) + an external store for subscribers. */

export const POMODORO_SYNC_ENABLED_KEY = "soundscape_pomodoro_enabled";
export const POMODORO_SYNC_SOUND_KEY = "soundscape_pomodoro_sound";
export const DEFAULT_SYNC_SOUND: SoundID = "rain";

// Built-in ids are always valid; setValidSoundIds adds the user's ids once the catalog is fetched.
let validSoundIds = new Set<SoundID>(BUILTIN_SOUND_IDS);

const store = createExternalStore();

/** Replaces the known sound ids with the built-ins plus the fetched catalog. */
export function setValidSoundIds(ids: Iterable<SoundID>): void {
  const next = new Set<SoundID>(BUILTIN_SOUND_IDS);
  for (const id of ids) next.add(id);
  validSoundIds = next;
}

export function getPomodoroSyncEnabled(): boolean {
  return localStorage.getItem(POMODORO_SYNC_ENABLED_KEY) === "true";
}

export function setPomodoroSyncEnabled(enabled: boolean): void {
  localStorage.setItem(POMODORO_SYNC_ENABLED_KEY, String(enabled));
  store.emit();
}

export function getPomodoroSyncSound(): SoundID {
  const raw = localStorage.getItem(POMODORO_SYNC_SOUND_KEY);
  const id = raw as SoundID;
  return id && validSoundIds.has(id) ? id : DEFAULT_SYNC_SOUND;
}

export function setPomodoroSyncSound(sound: SoundID): void {
  localStorage.setItem(POMODORO_SYNC_SOUND_KEY, sound);
  store.emit();
}

/** Subscribes to sync-setting changes. Returns an unsubscribe function. */
export const subscribe = store.subscribe;

export type SyncAction = "play" | "stop" | "idle";

/** Action for a timer status: "idle" when disabled, else play while running and stop on any other status. */
export function resolveSyncAction(enabled: boolean, status: SessionStatus | null): SyncAction {
  if (!enabled) return "idle";
  return status === "running" ? "play" : "stop";
}
