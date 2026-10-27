import { describe, expect, it } from "vitest";
import { sortTasks, VALID_SORT_OPTIONS } from "./sort";
import type { Task } from "../api/tasks";

function task(id: string, position: number, createdAt: string): Task {
  return {
    id,
    title: id,
    status: "todo",
    position,
    created_at: createdAt,
    updated_at: createdAt,
  };
}

describe("sortTasks", () => {
  it("orders manual by position", () => {
    const tasks = [task("c", 2, "2026-01-03T00:00:00Z"), task("a", 0, "2026-01-01T00:00:00Z"), task("b", 1, "2026-01-02T00:00:00Z")];
    expect(sortTasks(tasks, "manual").map((t) => t.id)).toEqual(["a", "b", "c"]);
  });

  it("breaks manual ties newest first", () => {
    const older = task("older", 0, "2026-01-01T00:00:00Z");
    const newer = task("newer", 0, "2026-02-01T00:00:00Z");
    expect(sortTasks([older, newer], "manual").map((t) => t.id)).toEqual(["newer", "older"]);
  });

  it("does not mutate the input", () => {
    const tasks = [task("b", 1, "2026-01-02T00:00:00Z"), task("a", 0, "2026-01-01T00:00:00Z")];
    sortTasks(tasks, "manual");
    expect(tasks.map((t) => t.id)).toEqual(["b", "a"]);
  });

  it("accepts manual as a persisted option", () => {
    expect(VALID_SORT_OPTIONS.has("manual")).toBe(true);
  });
});
