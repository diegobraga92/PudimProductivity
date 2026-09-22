import { beforeEach, describe, expect, it } from "vitest";
import { deletePreset, getPresets, savePreset } from "./soundPresets";

// Vitest runs in a node environment where `localStorage` is unavailable, so
// provide a minimal in-memory implementation (same approach as
// pomodoroSoundSync.test.ts).
const storage = new Map<string, string>();
Object.defineProperty(globalThis, "localStorage", {
  configurable: true,
  value: {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, String(value));
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => {
      storage.clear();
    },
  },
});

describe("soundscape presets", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("starts empty", () => {
    expect(getPresets()).toEqual([]);
  });

  it("round-trips a saved mix", () => {
    const id = savePreset("Focus", { rain: true, ocean: false }, { rain: 0.3 }, 0.8);

    const presets = getPresets();
    expect(presets).toHaveLength(1);
    expect(presets[0]).toEqual({
      id,
      label: "Focus",
      sounds: { rain: true, ocean: false },
      volumes: { rain: 0.3 },
      masterVolume: 0.8,
    });
  });

  it("copies the mix so later edits do not leak into the stored preset", () => {
    const sounds = { rain: true };
    const volumes = { rain: 0.5 };
    savePreset("Copy", sounds, volumes, 0.5);
    sounds.rain = false;
    volumes.rain = 0.1;

    expect(getPresets()[0].sounds).toEqual({ rain: true });
    expect(getPresets()[0].volumes).toEqual({ rain: 0.5 });
  });

  it("keeps presets in insertion order and deletes only the requested one", () => {
    const first = savePreset("One", {}, {}, 0.5);
    const second = savePreset("Two", {}, {}, 0.5);
    expect(getPresets().map((p) => p.id)).toEqual([first, second]);

    deletePreset(first);
    expect(getPresets().map((p) => p.id)).toEqual([second]);
  });

  it("ignores corrupt stored data", () => {
    localStorage.setItem("soundscape_presets", "{not json");
    expect(getPresets()).toEqual([]);

    localStorage.setItem("soundscape_presets", JSON.stringify({ sounds: {} }));
    expect(getPresets()).toEqual([]);
  });
});
