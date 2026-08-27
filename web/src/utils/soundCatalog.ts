import type { SoundID } from "./audio";

export interface SoundDef {
  id: SoundID;
  labelKey: string;
  icon: string;
}

/**
 * The soundscape catalog shared by the Soundscape page and the Pomodoro page
 * (its "ambient sound" picker). Kept in sync with KNOWN_SOUND_IDS in
 * ./soundFiles and the SoundID union in ./audio.
 */
export const SOUNDS: SoundDef[] = [
  { id: "light-rain", labelKey: "soundscape.lightRain", icon: "🌧️" },
  { id: "rain", labelKey: "soundscape.rainSound", icon: "🌧️" },
  { id: "rain-and-thunder", labelKey: "soundscape.rainAndThunder", icon: "⛈️" },
  { id: "strong-rain", labelKey: "soundscape.strongRain", icon: "🌧️" },
  { id: "stronger-rain", labelKey: "soundscape.strongerRain", icon: "🌩️" },
  { id: "fire", labelKey: "soundscape.fire", icon: "🔥" },
  { id: "fire-and-thunder", labelKey: "soundscape.fireAndThunder", icon: "🔥⛈️" },
  { id: "ocean", labelKey: "soundscape.ocean", icon: "🌊" },
];
