import config from "../config";
import type { SoundEntry } from "../api/sounds";
import type { SoundID } from "./audio";

/**
 * Soundscape ambient sounds are served as audio files (e.g. MP3 loops) by the
 * backend:
 *
 *   GET /api/v1/sounds        → { "sounds": [{ "id": "rain", "file": "rain.mp3", … }] }
 *   GET /api/v1/sounds/{file} → audio bytes (Range-capable, CORS-enabled)
 *
 * This module fetches that catalog and maps each sound id to its playable URL.
 * The audio engine is catalog-agnostic: a sound whose file is unknown simply
 * does not play.
 */

let fileBySound: Partial<Record<SoundID, string>> = {};

/** Fetches the backend sound catalog (empty when the backend is unreachable). */
export async function fetchSoundCatalog(): Promise<SoundEntry[]> {
  const res = await fetch(`${config.apiBaseUrl}/sounds`);
  if (!res.ok) return [];
  const data = (await res.json()) as { sounds?: SoundEntry[] };
  return data.sounds ?? [];
}

/** Rebuilds the sound id to playable URL map from a fetched catalog. */
export function setSoundFileMap(entries: SoundEntry[]): void {
  const next: Partial<Record<SoundID, string>> = {};
  for (const entry of entries) {
    if (entry.id && entry.file) {
      next[entry.id] = `${config.apiBaseUrl}/sounds/${entry.file}`;
    }
  }
  fileBySound = next;
}

/** Returns the backend URL for a sound, or undefined when no file is known. */
export function getSoundFile(id: SoundID): string | undefined {
  return fileBySound[id];
}


