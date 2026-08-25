import type { SoundID } from "./audio";

export interface SoundDef {
  id: SoundID;
  labelKey: string;
  icon: string;
  descKey: string;
}

/**
 * The soundscape catalog shared by the Soundscape page and the Pomodoro page
 * (its "ambient sound" picker). Kept in sync with KNOWN_SOUND_IDS in
 * ./soundFiles and the SoundID union in ./audio.
 */
export const SOUNDS: SoundDef[] = [
  { id: "ambient-pad", labelKey: "soundscape.ambientPad", icon: "🎹", descKey: "soundscape.desc.ambientPad" },
  { id: "light-rain", labelKey: "soundscape.lightRain", icon: "🌧️", descKey: "soundscape.desc.lightRain" },
  { id: "rain", labelKey: "soundscape.rainSound", icon: "🌧️", descKey: "soundscape.desc.rain" },
  { id: "rain-and-thunder", labelKey: "soundscape.rainAndThunder", icon: "⛈️", descKey: "soundscape.desc.rainAndThunder" },
  { id: "strong-rain", labelKey: "soundscape.strongRain", icon: "🌧️", descKey: "soundscape.desc.strongRain" },
  { id: "stronger-rain", labelKey: "soundscape.strongerRain", icon: "🌩️", descKey: "soundscape.desc.strongerRain" },
  { id: "fire", labelKey: "soundscape.fire", icon: "🔥", descKey: "soundscape.desc.fire" },
  { id: "fire-and-thunder", labelKey: "soundscape.fireAndThunder", icon: "🔥⛈️", descKey: "soundscape.desc.fireAndThunder" },
  { id: "ocean", labelKey: "soundscape.ocean", icon: "🌊", descKey: "soundscape.desc.ocean" },
];
