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
  { id: "white-noise", labelKey: "soundscape.whiteNoise", icon: "🌊", descKey: "soundscape.desc.whiteNoise" },
  { id: "pink-noise", labelKey: "soundscape.pinkNoise", icon: "🌸", descKey: "soundscape.desc.pinkNoise" },
  { id: "brown-noise", labelKey: "soundscape.brownNoise", icon: "🌫️", descKey: "soundscape.desc.brownNoise" },
  { id: "rain", labelKey: "soundscape.rainSound", icon: "🌧️", descKey: "soundscape.desc.rain" },
  { id: "ocean", labelKey: "soundscape.ocean", icon: "🌊", descKey: "soundscape.desc.ocean" },
  { id: "wind", labelKey: "soundscape.wind", icon: "💨", descKey: "soundscape.desc.wind" },
  { id: "campfire", labelKey: "soundscape.campfire", icon: "🔥", descKey: "soundscape.desc.campfire" },
  { id: "binaural-beat", labelKey: "soundscape.binaural", icon: "🎧", descKey: "soundscape.desc.binaural" },
  { id: "isochronic-tone", labelKey: "soundscape.isochronic", icon: "📳", descKey: "soundscape.desc.isochronic" },
  { id: "meditation-bowl", labelKey: "soundscape.meditationBowl", icon: "🕉️", descKey: "soundscape.desc.meditationBowl" },
  { id: "ambient-pad", labelKey: "soundscape.ambientPad", icon: "🎹", descKey: "soundscape.desc.ambientPad" },
];
