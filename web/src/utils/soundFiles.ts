import type { SoundID } from "./audio";
import config from "../config";

/**
 * Soundscape ambient sounds are served as audio files (MP3 loops) by the
 * backend:
 *
 *   GET /api/v1/sounds        → { "sounds": [{ "id": "rain", "file": "rain.mp3", … }] }
 *   GET /api/v1/sounds/{file} → audio bytes (Range-capable, CORS-enabled)
 *
 * This module fetches the catalog once and maps each SoundID to its playable
 * URL. Every sound in the catalog is file-backed; a sound whose file is
 * unknown or fails to load simply does not play.
 */

interface SoundCatalogEntry {
  id: string;
  file: string;
  mime: string;
}

/** SoundIDs the engine knows how to play (must match web/src/utils/audio.ts). */
const KNOWN_SOUND_IDS: ReadonlySet<SoundID> = new Set<SoundID>([
  "ambient-pad",
  "light-rain",
  "rain",
  "rain-and-thunder",
  "strong-rain",
  "stronger-rain",
  "fire",
  "fire-and-thunder",
  "ocean",
]);

/** Populated once the backend catalog is fetched; SoundID → playable URL. */
let fileBySound: Partial<Record<SoundID, string>> = {};
let loadPromise: Promise<void> | null = null;

/**
 * Fetch the backend sound catalog. Safe to call multiple times — the network
 * request runs at most once and never throws (failures leave the catalog
 * empty).
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
        // Backend unreachable or invalid response — leave the catalog empty.
      }
    })();
  }
  return loadPromise;
}

/** Returns the backend URL for a sound, or undefined when no file is known. */
export function getSoundFile(id: SoundID): string | undefined {
  return fileBySound[id];
}

