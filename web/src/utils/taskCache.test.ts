import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import type { Task } from "../api/tasks";
import { queryKeys } from "./queryKeys";
import { applyTaskDelete, applyTaskReorder, applyTaskUpsert } from "./taskCache";

function task(overrides: Partial<Task> & { id: string }): Task {
  return {
    title: overrides.id,
    status: "todo",
    position: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function newClient(): QueryClient {
  return new QueryClient();
}

const ids = (tasks: Task[] | undefined) => (tasks ?? []).map((t) => t.id);

describe("applyTaskUpsert", () => {
  it("keeps a list item out of the inbox buckets", () => {
    const queryClient = newClient();
    const inboxTask = task({ id: "inbox-1" });
    queryClient.setQueryData<Task[]>(queryKeys.tasksOneOff, [inboxTask]);
    queryClient.setQueryData<Task[]>(queryKeys.tasksHabit, []);

    applyTaskUpsert(queryClient, task({ id: "list-1", list_id: "list-a", title: "Buy milk" }));

    // The regular tasks view and the Dashboard read these keys; /tasks never
    // returns list items, so mirroring one here renders a phantom row.
    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.tasksOneOff))).toEqual(["inbox-1"]);
    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.tasksHabit))).toEqual([]);
    // Nothing has the list cached, so the event must not fabricate a bucket.
    expect(queryClient.getQueryData(queryKeys.taskListTasks("list-a"))).toBeUndefined();
  });

  it("updates a list bucket that is already cached", () => {
    const queryClient = newClient();
    queryClient.setQueryData<Task[]>(queryKeys.taskListTasks("list-a"), [
      task({ id: "list-1", list_id: "list-a", title: "Old title" }),
      task({ id: "list-2", list_id: "list-a" }),
    ]);

    applyTaskUpsert(queryClient, task({ id: "list-1", list_id: "list-a", title: "New title" }));

    const cached = queryClient.getQueryData<Task[]>(queryKeys.taskListTasks("list-a"));
    expect(ids(cached)).toEqual(["list-1", "list-2"]);
    expect(cached?.[0].title).toBe("New title");
  });

  it("appends a new list item to the cached list bucket", () => {
    const queryClient = newClient();
    queryClient.setQueryData<Task[]>(queryKeys.taskListTasks("list-a"), [task({ id: "list-1", list_id: "list-a" })]);

    applyTaskUpsert(queryClient, task({ id: "list-2", list_id: "list-a" }));

    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.taskListTasks("list-a")))).toEqual([
      "list-1",
      "list-2",
    ]);
  });

  it("routes inbox tasks by habit-ness and evicts them from their old list", () => {
    const queryClient = newClient();
    queryClient.setQueryData<Task[]>(queryKeys.tasksOneOff, [task({ id: "inbox-1" })]);
    queryClient.setQueryData<Task[]>(queryKeys.tasksHabit, []);
    queryClient.setQueryData<Task[]>(queryKeys.taskListTasks("list-a"), [
      task({ id: "moved", list_id: "list-a" }),
      task({ id: "list-1", list_id: "list-a" }),
    ]);

    // The task left the list and became a habit.
    applyTaskUpsert(queryClient, task({ id: "moved", recurrence_days: ["mon"] }));

    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.tasksHabit))).toEqual(["moved"]);
    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.tasksOneOff))).toEqual(["inbox-1"]);
    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.taskListTasks("list-a")))).toEqual([
      "list-1",
    ]);
  });
});

describe("applyTaskReorder", () => {
  it("applies a list drag to the list bucket only", () => {
    const queryClient = newClient();
    queryClient.setQueryData<Task[]>(queryKeys.tasksOneOff, [
      task({ id: "inbox-1", position: 0 }),
      task({ id: "inbox-2", position: 1 }),
    ]);
    queryClient.setQueryData<Task[]>(queryKeys.taskListTasks("list-a"), [
      task({ id: "list-1", list_id: "list-a", position: 0 }),
      task({ id: "list-2", list_id: "list-a", position: 1 }),
    ]);

    applyTaskReorder(queryClient, {
      list_id: "list-a",
      is_habit: false,
      ordered_ids: ["list-2", "list-1"],
    });

    const inbox = queryClient.getQueryData<Task[]>(queryKeys.tasksOneOff) ?? [];
    expect(inbox.map((t) => t.position)).toEqual([0, 1]);

    const list = queryClient.getQueryData<Task[]>(queryKeys.taskListTasks("list-a")) ?? [];
    expect(list.map((t) => [t.id, t.position])).toEqual([
      ["list-1", 1],
      ["list-2", 0],
    ]);
  });

  it("ignores an empty order", () => {
    const queryClient = newClient();
    queryClient.setQueryData<Task[]>(queryKeys.tasksOneOff, [task({ id: "inbox-1", position: 3 })]);

    applyTaskReorder(queryClient, { is_habit: false, ordered_ids: [] });

    expect(queryClient.getQueryData<Task[]>(queryKeys.tasksOneOff)?.[0].position).toBe(3);
  });
});

describe("applyTaskDelete", () => {
  it("removes the task from the inbox and every cached list", () => {
    const queryClient = newClient();
    queryClient.setQueryData<Task[]>(queryKeys.tasksOneOff, [task({ id: "gone" }), task({ id: "inbox-1" })]);
    queryClient.setQueryData<Task[]>(queryKeys.tasksHabit, [task({ id: "gone" })]);
    queryClient.setQueryData<Task[]>(queryKeys.taskListTasks("list-a"), [task({ id: "gone", list_id: "list-a" })]);
    queryClient.setQueryData<Task[]>(queryKeys.taskListTasks("list-b"), [
      task({ id: "gone", list_id: "list-b" }),
      task({ id: "list-1", list_id: "list-b" }),
    ]);

    applyTaskDelete(queryClient, "gone");

    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.tasksOneOff))).toEqual(["inbox-1"]);
    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.tasksHabit))).toEqual([]);
    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.taskListTasks("list-a")))).toEqual([]);
    expect(ids(queryClient.getQueryData<Task[]>(queryKeys.taskListTasks("list-b")))).toEqual(["list-1"]);
  });
});
