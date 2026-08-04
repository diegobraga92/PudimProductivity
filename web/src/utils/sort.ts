import type { Task } from "../api/tasks";

export type SortOption =
  | "alpha-asc"
  | "alpha-desc"
  | "created-asc"
  | "created-desc"
  | "time-asc"
  | "time-desc";

export const SORT_LABELS: Record<SortOption, string> = {
  "alpha-asc": "Name A-Z",
  "alpha-desc": "Name Z-A",
  "created-asc": "Oldest first",
  "created-desc": "Newest first",
  "time-asc": "Time ↑",
  "time-desc": "Time ↓",
};

export const VALID_SORT_OPTIONS = new Set<string>([
  "alpha-asc",
  "alpha-desc",
  "created-asc",
  "created-desc",
  "time-asc",
  "time-desc",
]);

export function sortTasks(tasks: Task[], option: SortOption): Task[] {
  const sorted = [...tasks];

  switch (option) {
    case "alpha-asc": {
      return sorted.sort((a, b) => a.title.localeCompare(b.title));
    }
    case "alpha-desc": {
      return sorted.sort((a, b) => b.title.localeCompare(a.title));
    }
    case "created-asc": {
      return sorted.sort((a, b) => (a.created_at < b.created_at ? -1 : a.created_at > b.created_at ? 1 : 0));
    }
    case "created-desc": {
      return sorted.sort((a, b) => (b.created_at < a.created_at ? -1 : b.created_at > a.created_at ? 1 : 0));
    }
    case "time-asc":
    case "time-desc": {
      const scheduled = sorted.filter((t) => t.start_time != null);
      const unscheduled = sorted.filter((t) => t.start_time == null);
      scheduled.sort((a, b) =>
        option === "time-asc"
          ? (a.start_time ?? "").localeCompare(b.start_time ?? "")
          : (b.start_time ?? "").localeCompare(a.start_time ?? "")
      );
      return [...scheduled, ...unscheduled];
    }
    default: {
      return sorted;
    }
  }
}