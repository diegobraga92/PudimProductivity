import type { SoundID } from "./audio";

/**
 * Optional loopable audio files for the ambient Soundscape sounds.
 *
 * Each value is a URL path served from `web/public/` (Vite serves the public
 * directory at the site root), e.g. `"/sounds/rain.mp3"`.
 *
 * Leave a key unset (or remove it) to keep the synthesized Web Audio version
 * for that sound. When a key is set, the Soundscape engine plays the looping
 * audio file instead of generating the sound in-browser.
 *
 * Example:
 *   AMBIENT_SOUND_FILES = {
 *     rain: "/sounds/rain.mp3",
 *     ocean: "/sounds/ocean.mp3",
 *   };
 */
export const AMBIENT_SOUND_FILES: Partial<Record<SoundID, string>> = {};
