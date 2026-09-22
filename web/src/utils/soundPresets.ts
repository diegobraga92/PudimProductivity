import type { SoundID } from "./audio";

/**
 * Soundscape presets: the saved "mixes" of sounds and volumes.
 *
 * Kept as pure functions over localStorage (mirroring `pomodoroSoundSync`) so
 * the audio engine stays focused on playback and the storage is unit-testable.
 */

/** A saved mix: which sounds were on, their levels, and the master level. */
export interface Preset {
  id: string;
  label: string;
  sounds: Partial<Record<SoundID, boolean>>;
  volumes: Partial<Record<SoundID, number>>;
  masterVolume: number;
}

const LS_PRESETS_KEY = "soundscape_presets";

/** Loads the saved presets (empty when missing or corrupt). */
export function getPresets(): Preset[] {
  try {
    const raw = localStorage.getItem(LS_PRESETS_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? (parsed as Preset[]) : [];
  } catch {
    return [];
  }
}

/** Persists the presets to localStorage. */
function savePresets(presets: Preset[]): void {
  localStorage.setItem(LS_PRESETS_KEY, JSON.stringify(presets));
}

/** Appends the current mix as a preset and returns its generated id. */
export function savePreset(
  label: string,
  sounds: Partial<Record<SoundID, boolean>>,
  volumes: Partial<Record<SoundID, number>>,
  masterVolume: number,
): string {
  const presets = getPresets();
  const id = `preset_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
  presets.push({ id, label, sounds: { ...sounds }, volumes: { ...volumes }, masterVolume });
  savePresets(presets);
  return id;
}

/** Deletes a preset by id (no-op for unknown ids). */
export function deletePreset(id: string): void {
  savePresets(getPresets().filter((p) => p.id !== id));
}
