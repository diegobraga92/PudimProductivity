import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  listTasks,
  deleteTask,
  updateTask,
  completeTask,
  uncompleteTask,
  getTaskCompletions,
  type Task,
  type TaskStatus,
  type RecurrenceDay,
} from "../api/tasks";
import TaskCreate from "./TaskCreate";
import TaskDetail from "./TaskDetail";

type View = "list" | "create" | "detail";

const DAY_ORDER: RecurrenceDay[] = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
const DAY_SHORT: Record<RecurrenceDay, string> = {
  mon: "M",
  tue: "T",
  wed: "W",
  thu: "T",
  fri: "F",
  sat: "S",
  sun: "S",
};

function getWeekDates(): string[] {
  const dates: string[] = [];
  const now = new Date();
  const dayOfWeek = now.getDay(); // 0=Sun, 1=Mon, ...
  // Find Monday of this week
  const monday = new Date(now);
  monday.setDate(now.getDate() - ((dayOfWeek + 6) % 7));
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    dates.push(d.toISOString().split("T")[0]);
  }
  return dates;
}

function getDayName(dateStr: string): RecurrenceDay {
  const d = new Date(dateStr + "T00:00:00");
  const day = d.getDay(); // 0=Sun
  return DAY_ORDER[day === 0 ? 6 : day - 1];
}

interface HabitWeekRowProps {
  task: Task;
  completions: string[];
  onToggleDay: (taskId: string, date: string, completed: boolean) => void;
}

function HabitWeekRow({ task, completions, onToggleDay }: HabitWeekRowProps) {
  const weekDates = getWeekDates();
  const completedSet = new Set(completions);

  return (
    <div style={{ display: "flex", gap: "0.25rem", marginTop: "0.3rem" }}>
      {weekDates.map((date) => {
        const dayName = getDayName(date);
        const isScheduled = task.recurrence_days?.includes(dayName);
        const isCompleted = completedSet.has(date);
        const isToday =
          date === new Date().toISOString().split("T")[0];

        return (
          <button
            key={date}
            onClick={() => {
              if (isScheduled || isCompleted) {
                onToggleDay(task.id, date, isCompleted);
              }
            }}
            title={`${dayName} ${date}`}
            style={{
              width: "1.8rem",
              height: "1.8rem",
              border: isToday ? "2px solid #007bff" : "1px solid #ccc",
              borderRadius: "4px",
              background: isCompleted
                ? "#28a745"
                : isScheduled
                ? "#fff3cd"
                : "#f5f5f5",
              color: isCompleted ? "white" : isScheduled ? "#856404" : "#ccc",
              cursor: isScheduled || isCompleted ? "pointer" : "default",
              fontSize: "0.7rem",
              fontWeight: "bold",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              padding: 0,
            }}
          >
            {DAY_SHORT[dayName]}
          </button>
        );
      })}
    </div>
  );
}

export default function TaskList() {
  const queryClient = useQueryClient();
  const [view, setView] = useState<View>("list");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<TaskStatus | "">("");

  const { data: tasks, isLoading, error } = useQuery<Task[]>({
    queryKey: ["tasks", statusFilter],
    queryFn: () =>
      listTasks(statusFilter ? (statusFilter as TaskStatus) : undefined),
  });

  // Fetch completions for all habit tasks
  const habitTasks = tasks?.filter((t) => t.recurrence_days && t.recurrence_days.length > 0) ?? [];
  const weekDates = getWeekDates();
  const from = weekDates[0];
  const to = weekDates[6];

  const { data: allCompletions } = useQuery({
    queryKey: ["habitCompletions", from, to],
    queryFn: async () => {
      const results: Record<string, string[]> = {};
      for (const task of habitTasks) {
        try {
          const completions = await getTaskCompletions(task.id, from, to);
          results[task.id] = completions.map((c) => c.completed_date);
        } catch {
          results[task.id] = [];
        }
      }
      return results;
    },
    enabled: habitTasks.length > 0,
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) =>
      updateTask(id, {
        status: status === "done" ? "todo" : "done",
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
    },
  });

  const habitToggleMutation = useMutation({
    mutationFn: async ({ taskId, completed }: { taskId: string; date: string; completed: boolean }) => {
      if (completed) {
        await uncompleteTask(taskId);
      } else {
        await completeTask(taskId);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["habitCompletions"] });
    },
  });

  if (view === "create") {
    return (
      <TaskCreate
        onCreated={() => {
          queryClient.invalidateQueries({ queryKey: ["tasks"] });
          setView("list");
        }}
        onCancel={() => setView("list")}
      />
    );
  }

  if (view === "detail" && selectedTaskId) {
    return (
      <TaskDetail
        taskId={selectedTaskId}
        onUpdated={() => queryClient.invalidateQueries({ queryKey: ["tasks"] })}
        onDeleted={() => {
          queryClient.invalidateQueries({ queryKey: ["tasks"] });
          setView("list");
          setSelectedTaskId(null);
        }}
        onBack={() => setView("list")}
      />
    );
  }

  return (
    <div style={{ maxWidth: "600px", margin: "0 auto", padding: "1rem" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "1rem",
        }}
      >
        <h2 style={{ margin: 0 }}>Tasks</h2>
        <button
          onClick={() => setView("create")}
          style={{
            padding: "0.5rem 1rem",
            background: "#007bff",
            color: "white",
            border: "none",
            borderRadius: "4px",
            cursor: "pointer",
          }}
        >
          + New Task
        </button>
      </div>

      {/* Filter */}
      <div style={{ marginBottom: "1rem" }}>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as TaskStatus | "")}
          style={{ padding: "0.3rem" }}
        >
          <option value="">All</option>
          <option value="todo">Todo</option>
          <option value="done">Done</option>
        </select>
      </div>

      {isLoading && <p>Loading tasks...</p>}

      {error && <p style={{ color: "red" }}>Error: {(error as Error).message}</p>}

      {tasks && tasks.length === 0 && (
        <p style={{ color: "#666" }}>No tasks yet. Create one!</p>
      )}

      {tasks && tasks.length > 0 && (
        <ul style={{ listStyle: "none", padding: 0 }}>
          {tasks.map((task) => {
            const isHabit = task.recurrence_days && task.recurrence_days.length > 0;
            const taskCompletions = allCompletions?.[task.id] ?? [];

            return (
              <li
                key={task.id}
                style={{
                  padding: "0.75rem",
                  marginBottom: "0.5rem",
                  border: "1px solid #ddd",
                  borderRadius: "6px",
                  background: isHabit ? "#f0f8ff" : task.status === "done" ? "#f9f9f9" : "white",
                }}
              >
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "0.75rem",
                  }}
                >
                  {!isHabit && (
                    <input
                      type="checkbox"
                      checked={task.status === "done"}
                      onChange={() =>
                        toggleMutation.mutate({ id: task.id, status: task.status })
                      }
                      style={{ width: "1.2rem", height: "1.2rem", cursor: "pointer" }}
                    />
                  )}
                  <span
                    style={{
                      flex: 1,
                      cursor: "pointer",
                      textDecoration:
                        task.status === "done" ? "line-through" : "none",
                      color: task.status === "done" ? "#888" : "inherit",
                    }}
                    onClick={() => {
                      setSelectedTaskId(task.id);
                      setView("detail");
                    }}
                  >
                    {task.title}
                    {isHabit && (
                      <span
                        style={{
                          marginLeft: "0.5rem",
                          fontSize: "0.75rem",
                          color: "#007bff",
                          background: "#e6f2ff",
                          padding: "0.1rem 0.4rem",
                          borderRadius: "3px",
                        }}
                      >
                        habit
                      </span>
                    )}
                  </span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      if (confirm("Delete this task?")) {
                        deleteMutation.mutate(task.id);
                      }
                    }}
                    style={{
                      padding: "0.3rem 0.6rem",
                      background: "#dc3545",
                      color: "white",
                      border: "none",
                      borderRadius: "4px",
                      cursor: "pointer",
                      fontSize: "0.8rem",
                    }}
                  >
                    Delete
                  </button>
                </div>

                {/* Weekly habit streak */}
                {isHabit && (
                  <HabitWeekRow
                    task={task}
                    completions={taskCompletions}
                    onToggleDay={(taskId, date, completed) => {
                      habitToggleMutation.mutate({ taskId, date, completed });
                    }}
                  />
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
