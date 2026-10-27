import { describe, expect, it } from "vitest";
import { landingIndex, moveItem, positionsFor } from "./reorder";

describe("moveItem", () => {
  it("moves an item down", () => {
    expect(moveItem(["a", "b", "c"], 0, 2)).toEqual(["b", "c", "a"]);
  });

  it("moves an item up", () => {
    expect(moveItem(["a", "b", "c"], 2, 0)).toEqual(["c", "a", "b"]);
  });

  it("keeps the order when the target is the current index", () => {
    expect(moveItem(["a", "b", "c"], 1, 1)).toEqual(["a", "b", "c"]);
  });

  it("clamps targets past the end", () => {
    expect(moveItem(["a", "b"], 0, 99)).toEqual(["b", "a"]);
  });

  it("ignores a source index outside the list", () => {
    expect(moveItem(["a"], 5, 0)).toEqual(["a"]);
  });

  it("leaves the input untouched", () => {
    const input = ["a", "b", "c"];
    moveItem(input, 0, 2);
    expect(input).toEqual(["a", "b", "c"]);
  });
});

describe("landingIndex", () => {
  const midpoints = [100, 200, 300];

  it("lands first when the pointer is above every row", () => {
    expect(landingIndex(midpoints, 50)).toBe(0);
  });

  it("lands between rows", () => {
    expect(landingIndex(midpoints, 150)).toBe(1);
    expect(landingIndex(midpoints, 250)).toBe(2);
  });

  it("lands last when the pointer is below every row", () => {
    expect(landingIndex(midpoints, 350)).toBe(3);
  });

  it("treats a pointer exactly on a midpoint as still above that row", () => {
    expect(landingIndex(midpoints, 100)).toBe(0);
  });

  it("lands first in an empty list", () => {
    expect(landingIndex([], 42)).toBe(0);
  });
});

describe("positionsFor", () => {
  it("numbers ids from zero in visual order", () => {
    expect([...positionsFor(["x", "y", "z"])]).toEqual([
      ["x", 0],
      ["y", 1],
      ["z", 2],
    ]);
  });
});
