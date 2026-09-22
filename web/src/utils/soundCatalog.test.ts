import { describe, expect, it } from "vitest";
import { BUILTIN_SOUND_IDS, SOUNDS, resolveSounds } from "./soundCatalog";

/** Tiny `t` stub so labels are predictable and independent of the dictionary. */
const t = (key: string) => `t:${key}`;

const entry = (id: string, extra: { label?: string; icon?: string } = {}) => ({
  id,
  file: `${id}.mp3`,
  mime: "audio/mpeg",
  custom: true,
  ...extra,
});

describe("resolveSounds", () => {
  it("renders the built-ins first, in catalog order, with translated labels", () => {
    const sounds = resolveSounds([], t);
    expect(sounds.map((s) => s.id)).toEqual(SOUNDS.map((s) => s.id));
    expect(sounds[0]).toEqual({
      id: SOUNDS[0].id,
      label: `t:${SOUNDS[0].labelKey}`,
      icon: SOUNDS[0].icon,
      custom: false,
    });
    expect(sounds.every((s) => !s.custom)).toBe(true);
  });

  it("appends the user's sounds with their stored name and icon", () => {
    const sounds = resolveSounds([entry("custom-1", { label: "My rain", icon: "🏴" })], t);
    expect(sounds).toHaveLength(SOUNDS.length + 1);
    expect(sounds[sounds.length - 1]).toEqual({
      id: "custom-1",
      label: "My rain",
      icon: "🏴",
      custom: true,
    });
  });

  it("falls back to the id and a default emoji for incomplete entries", () => {
    const sounds = resolveSounds([entry("custom-2")], t);
    const custom = sounds[sounds.length - 1];
    expect(custom.label).toBe("custom-2");
    expect(custom.icon).toBe("🎵");
    expect(custom.custom).toBe(true);
  });

  it("ignores backend entries that would duplicate a built-in id", () => {
    const sounds = resolveSounds([entry("rain", { label: "Shadow" })], t);
    expect(sounds).toHaveLength(SOUNDS.length);
    expect(sounds.filter((s) => s.id === "rain")).toHaveLength(1);
    expect(BUILTIN_SOUND_IDS.has("rain")).toBe(true);
  });
});
