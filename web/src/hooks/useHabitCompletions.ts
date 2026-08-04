import { useQuery } from "@tanstack/react-query";
import { getAllTaskCompletions, type Task } from "../api/tasks";
import { STREAK_HISTORY_START } from "../utils/constants";

/**
 * Fetches the full completion history for all habit tasks in a single
 * batch request, keyed by task ID.
 */
export function useHabitCompletions(habitTasks: Task[]) {
  const { data } = useQuery({
    queryKey: ["habitCompletions", STREAK_HISTORY_START],
    queryFn: async () => {
      const today = new Date().toISOString().slice(0, 10);
      const completions = await getAllTaskCompletions(STREAK_HISTORY_START, today);
      const results: Record<string, string[]> = {};
      for (const task of habitTasks) {
        results[task.id] = [];
      }
      for (const c of completions) {
        if (results[c.task_id] !== undefined) {
          results[c.task_id].push(c.completed_date);
        }
      }
      return results;
    },
    enabled: habitTasks.length > 0,
  });

  return data ?? {};
}