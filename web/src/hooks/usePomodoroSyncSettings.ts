import { useCallback, useSyncExternalStore } from "react";
import type { SoundID } from "../utils/audio";
import {
  getPomodoroSyncEnabled,
  getPomodoroSyncSound,
  setPomodoroSyncEnabled,
  setPomodoroSyncSound,
  subscribe,
} from "../utils/pomodoroSoundSync";

export interface PomodoroSyncSettings {
  /** Whether a sound auto-plays while the pomodoro timer runs. */
  enabled: boolean;
  /** The sound to auto-play. */
  sound: SoundID;
  setEnabled: (enabled: boolean) => void;
  setSound: (sound: SoundID) => void;
}

/** Reactive pomodoro <-> sound sync settings, shared by the Pomodoro page and the app-root hook. */
export function usePomodoroSyncSettings(): PomodoroSyncSettings {
  const enabled = useSyncExternalStore(subscribe, getPomodoroSyncEnabled);
  const sound = useSyncExternalStore(subscribe, getPomodoroSyncSound);

  const setEnabled = useCallback((v: boolean) => setPomodoroSyncEnabled(v), []);
  const setSound = useCallback((id: SoundID) => setPomodoroSyncSound(id), []);

  return { enabled, sound, setEnabled, setSound };
}
