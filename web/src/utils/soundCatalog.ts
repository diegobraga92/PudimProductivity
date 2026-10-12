import type { SoundEntry } from "../api/sounds";
import type { SoundID } from "./audio";

/** A sound shipped with the app: its id, translation key and emoji icon. */
export interface BuiltinSound {
  id: SoundID;
  labelKey: string;
  icon: string;
}

/** Built-in sounds. Ids must match the backend DefaultCatalog (sounds/domain.go); labels come from labelKey. */
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

/** Built-in ids - the always-valid fallback before the backend catalog loads. */
export const BUILTIN_SOUND_IDS: ReadonlySet<SoundID> = new Set(SOUNDS.map((s) => s.id));

/** A sound ready to render: resolved label/icon plus where it came from. */
export interface ResolvedSound {
  id: SoundID;
  /** Translated name (built-ins) or the user-supplied name (added sounds). */
  label: string;
  icon: string;
  /** True for sounds added by the user, rather than shipped with the app. */
  custom: boolean;
}

/** Emoji used when a user-added sound has no icon of its own. */
const FALLBACK_ICON = "🎵";

/** Built-ins (translated, catalog order) then user sounds; backend entries reusing a built-in id are dropped. */
export function resolveSounds(
  entries: readonly SoundEntry[],
  t: (key: string) => string,
): ResolvedSound[] {
  const builtinIds = new Set<SoundID>(SOUNDS.map((s) => s.id));
  const resolved: ResolvedSound[] = SOUNDS.map((s) => ({
    id: s.id,
    label: t(s.labelKey),
    icon: s.icon,
    custom: false,
  }));
  for (const entry of entries) {
    if (builtinIds.has(entry.id)) continue;
    resolved.push({
      id: entry.id,
      label: entry.label || entry.id,
      icon: entry.icon || FALLBACK_ICON,
      custom: entry.custom ?? true,
    });
  }
  return resolved;
}

