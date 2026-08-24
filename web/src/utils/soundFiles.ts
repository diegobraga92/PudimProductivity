import type { SoundID } from "./audio";
import config from "../config";

/**
 * Soundscape ambient sounds are served as audio files (MP3 loops) by the
 * backend:
 *
 *   GET /api/v1/sounds        → { "sounds": [{ "id": "rain", "file": "rain.mp3", … }] }
 *   GET /api/v1/sounds/{file} → audio bytes (Range-capable, CORS-enabled)
 *
 * This module fetches the catalog once and maps each known SoundID to its
 * playable URL. The Soundscape engine keeps synthesizing any sound whose file
 * is unknown or fails to load, so a missing/empty catalog silently degrades to
 * the in-browser synthesized versions (and the feature keeps working offline or
 * when the backend is unreachable).
 */

export interface SoundCatalogEntry {
  id: string;
  file: string;
  mime: string;
}

/** SoundIDs the engine knows how to play (must match web/src/utils/audio.ts). */
const KNOWN_SOUND_IDS: ReadonlySet<SoundID> = new Set<SoundID>([
  "white-noise",
  "pink-noise",
  "brown-noise",
  "rain",
  "ocean",
  "wind",
  "campfire",
  "binaural-beat",
  "isochronic-tone",
  "meditation-bowl",
  "ambient-pad",
]);

/** Populated once the backend catalog is fetched; SoundID → playable URL. */
let fileBySound: Partial<Record<SoundID, string>> = {};
let loadPromise: Promise<void> | null = null;

/**
 * Fetch the backend sound catalog. Safe to call multiple times — the network
 * request runs at most once and never throws (failures keep the synthesized
 * fallbacks active).
 */
export function loadSoundCatalog(): Promise<void> {
  if (!loadPromise) {
    loadPromise = (async () => {
      try {
        const res = await fetch(`${config.apiBaseUrl}/sounds`);
        if (!res.ok) return;
        const data = (await res.json()) as { sounds?: SoundCatalogEntry[] };
        const next: Partial<Record<SoundID, string>> = {};
        for (const entry of data.sounds ?? []) {
          const id = entry.id as SoundID;
          if (KNOWN_SOUND_IDS.has(id) && entry.file) {
            next[id] = `${config.apiBaseUrl}/sounds/${entry.file}`;
          }
        }
        fileBySound = next;
      } catch {
        // Backend unreachable or invalid response — keep synthesized sounds.
      }
    })();
  }
  return loadPromise;
}

/** Returns the backend URL for a sound, or undefined when no file is known. */
export function getSoundFile(id: SoundID): string | undefined {
  return fileBySound[id];
}

