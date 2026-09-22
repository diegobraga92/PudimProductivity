import type { SoundID } from "./audio";

/** A sound shipped with the app: its id, translation key and emoji icon. */
export interface BuiltinSound {
  id: SoundID;
  labelKey: string;
  icon: string;
}

/**
 * The built-in sound library.
 *
 * Ids are the stable identifiers bundled by the backend (`DefaultCatalog` in
 * `backend/internal/contexts/content/sounds/domain.go`). Labels are translated
 * through `labelKey`, sounds the user added themselves come from the backend
 * catalog (see `hooks/useSounds`) and carry their own stored name instead.
 */
export const SOUNDS: BuiltinSound[] = [
  { id: "light-rain", labelKey: "soundscape.lightRain", icon: "🌧️" },
  { id: "rain", labelKey: "soundscape.rainSound", icon: "🌧️" },
  { id: "rain-and-thunder", labelKey: "soundscape.rainAndThunder", icon: "⛈️" },
  { id: "strong-rain", labelKey: "soundscape.strongRain", icon: "🌧️" },
  { id: "stronger-rain", labelKey: "soundscape.strongerRain", icon: "🌩️" },
  { id: "fire", labelKey: "soundscape.fire", icon: "🔥" },
  { id: "fire-and-thunder", labelKey: "soundscape.fireAndThunder", icon: "🔥⛈️" },
  { id: "ocean", labelKey: "soundscape.ocean", icon: "🌊" },
];

/**
 * Ids of the built-in sounds. Used as the always-valid fallback for persisted
 * settings before the backend catalog has loaded.
 */
export const BUILTIN_SOUND_IDS: ReadonlySet<SoundID> = new Set(SOUNDS.map((s) => s.id));

