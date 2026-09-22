import { describe, expect, it, vi } from "vitest";

// config.ts reads `window` while it is evaluated, so stub it before the module
// under test (and its config import) is loaded. Vitest runs in a node
// environment, which has no window.
vi.hoisted(() => {
  Object.defineProperty(globalThis, "window", { configurable: true, value: {} });
});

import config from "../config";
import { getSoundFile, setSoundFileMap } from "./soundFiles";

const url = (file: string) => `${config.apiBaseUrl}/sounds/${file}`;

describe("sound file map", () => {
  it("maps catalog ids to playable backend URLs", () => {
    setSoundFileMap([
      { id: "rain", file: "rain.mp3?v=2", mime: "audio/mpeg" },
      { id: "custom-1", file: "custom-1.mp3", mime: "audio/mpeg" },
    ]);

    expect(getSoundFile("rain")).toBe(url("rain.mp3?v=2"));
    expect(getSoundFile("custom-1")).toBe(url("custom-1.mp3"));
  });

  it("forgets ids that are no longer in the catalog", () => {
    setSoundFileMap([{ id: "rain", file: "rain.mp3", mime: "audio/mpeg" }]);
    expect(getSoundFile("rain")).toBeDefined();

    setSoundFileMap([]);
    expect(getSoundFile("rain")).toBeUndefined();
  });

  it("ignores entries without an id or file", () => {
    setSoundFileMap([
      { id: "", file: "x.mp3", mime: "audio/mpeg" },
      { id: "y", file: "", mime: "audio/mpeg" },
    ]);

    expect(getSoundFile("")).toBeUndefined();
    expect(getSoundFile("y")).toBeUndefined();
  });
});
