import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  listTasks,
  deleteTask,
  updateTask,
  completeTask,
  uncompleteTask,
  getTaskCompletions,
  createTask,
  type Task,
  type TaskStatus,
  type RecurrenceDay,
} from "../api/tasks";
import {
  listTaskLists,
  createTaskList,
  deleteTaskList,
  type TaskList,
} from "../api/taskLists";
import TaskCreate from "./TaskCreate";
import TaskDetail from "./TaskDetail";
import TaskListDetail from "./TaskListDetail";

type View = "list" | "create" | "detail" | "listDetail";

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
  const dayOfWeek = now.getDay();
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
  const day = d.getDay();
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
        const isToday = date === new Date().toISOString().split("T")[0];

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
  const [selectedListId, setSelectedListId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<TaskStatus | "">("");
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [newHabitTitle, setNewHabitTitle] = useState("");
  const [newListName, setNewListName] = useState("");

  // Fetch unassigned tasks (not in any list)
  const { data: tasks, isLoading, error } = useQuery<Task[]>({
    queryKey: ["tasks", statusFilter],
    queryFn: () =>
      listTasks(statusFilter ? (statusFilter as TaskStatus) : undefined),
  });

  // Fetch task lists
  const { data: taskLists = [] } = useQuery<TaskList[]>({
    queryKey: ["taskLists"],
    queryFn: listTaskLists,
  });

  // Separate tasks into todos and habits
  const todoTasks = tasks?.filter((t) => !t.recurrence_days || t.recurrence_days.length === 0) ?? [];
  const habitTasks = tasks?.filter((t) => t.recurrence_days && t.recurrence_days.length > 0) ?? [];

  // Fetch completions for all habit tasks
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

  // Mutations
  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) =>
      updateTask(id, { status: status === "done" ? "todo" : "done" }),
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

  const createTodoMutation = useMutation({
    mutationFn: (title: string) => createTask({ title }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      setNewTodoTitle("");
    },
  });

  const createHabitMutation = useMutation({
    mutationFn: (title: string) =>
      createTask({ title, recurrence_days: ["mon", "tue", "wed", "thu", "fri"] }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      setNewHabitTitle("");
    },
  });

  const createListMutation = useMutation({
    mutationFn: (name: string) => createTaskList({ name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskLists"] });
      setNewListName("");
    },
  });

  const deleteListMutation = useMutation({
    mutationFn: deleteTaskList,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskLists"] });
    },
  });

  // View routing
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

  if (view === "listDetail" && selectedListId) {
    return (
      <TaskListDetail
        listId={selectedListId}
        onBack={() => setView("list")}
        onDeleted={() => {
          queryClient.invalidateQueries({ queryKey: ["taskLists"] });
          setView("list");
          setSelectedListId(null);
        }}
      />
    );
  }

  return (
    <div style={{ maxWidth: "900px", margin: "0 auto", padding: "1rem" }}>
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

      {/* Two-column layout */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: "1.5rem",
          marginBottom: "2rem",
        }}
      >
        {/* LEFT COLUMN: Todo tasks */}
        <div>
          <h3 style={{ margin: "0 0 0.75rem 0", color: "#333" }}>
            📋 To-Do
          </h3>

          {/* Quick-add for todos */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (newTodoTitle.trim()) createTodoMutation.mutate(newTodoTitle.trim());
            }}
            style={{ display: "flex", gap: "0.3rem", marginBottom: "0.75rem" }}
          >
            <input
              type="text"
              value={newTodoTitle}
              onChange={(e) => setNewTodoTitle(e.target.value)}
              placeholder="Quick add todo..."
              style={{
                flex: 1,
                padding: "0.4rem",
                border: "1px solid #ccc",
                borderRadius: "4px",
                fontSize: "0.9rem",
              }}
            />
            <button
              type="submit"
              disabled={createTodoMutation.isPending || !newTodoTitle.trim()}
              style={{
                padding: "0.4rem 0.7rem",
                background:
                  createTodoMutation.isPending || !newTodoTitle.trim()
                    ? "#6c757d"
                    : "#007bff",
                color: "white",
                border: "none",
                borderRadius: "4px",
                cursor:
                  createTodoMutation.isPending || !newTodoTitle.trim()
                    ? "not-allowed"
                    : "pointer",
                fontSize: "0.85rem",
              }}
            >
              Add
            </button>
          </form>

          {todoTasks.length === 0 && !isLoading && (
            <p style={{ color: "#666", fontSize: "0.9rem" }}>No todos yet.</p>
          )}

          <ul style={{ listStyle: "none", padding: 0 }}>
            {todoTasks.map((task) => (
              <li
                key={task.id}
                style={{
                  padding: "0.5rem 0.75rem",
                  marginBottom: "0.3rem",
                  border: "1px solid #ddd",
                  borderRadius: "4px",
                  display: "flex",
                  alignItems: "center",
                  gap: "0.5rem",
                  background: task.status === "done" ? "#f9f9f9" : "white",
                }}
              >
                <input
                  type="checkbox"
                  checked={task.status === "done"}
                  onChange={() =>
                    toggleMutation.mutate({ id: task.id, status: task.status })
                  }
                  style={{ width: "1.1rem", height: "1.1rem", cursor: "pointer" }}
                />
                <span
                  style={{
                    flex: 1,
                    cursor: "pointer",
                    textDecoration:
                      task.status === "done" ? "line-through" : "none",
                    color: task.status === "done" ? "#888" : "inherit",
                    fontSize: "0.9rem",
                  }}
                  onClick={() => {
                    setSelectedTaskId(task.id);
                    setView("detail");
                  }}
                >
                  {task.title}
                </span>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    if (confirm("Delete this task?")) {
                      deleteMutation.mutate(task.id);
                    }
                  }}
                  style={{
                    padding: "0.2rem 0.5rem",
                    background: "#dc3545",
                    color: "white",
                    border: "none",
                    borderRadius: "3px",
                    cursor: "pointer",
                    fontSize: "0.75rem",
                  }}
                >
                  Del
                </button>
              </li>
            ))}
          </ul>
        </div>

        {/* RIGHT COLUMN: Habits */}
        <div>
          <h3 style={{ margin: "0 0 0.75rem 0", color: "#333" }}>
            🔄 Habits
          </h3>

          {/* Quick-add for habits */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (newHabitTitle.trim()) createHabitMutation.mutate(newHabitTitle.trim());
            }}
            style={{ display: "flex", gap: "0.3rem", marginBottom: "0.75rem" }}
          >
            <input
              type="text"
              value={newHabitTitle}
              onChange={(e) => setNewHabitTitle(e.target.value)}
              placeholder="Quick add habit (weekdays)..."
              style={{
                flex: 1,
                padding: "0.4rem",
                border: "1px solid #ccc",
                borderRadius: "4px",
                fontSize: "0.9rem",
              }}
            />
            <button
              type="submit"
              disabled={createHabitMutation.isPending || !newHabitTitle.trim()}
              style={{
                padding: "0.4rem 0.7rem",
                background:
                  createHabitMutation.isPending || !newHabitTitle.trim()
                    ? "#6c757d"
                    : "#28a745",
                color: "white",
                border: "none",
                borderRadius: "4px",
                cursor:
                  createHabitMutation.isPending || !newHabitTitle.trim()
                    ? "not-allowed"
                    : "pointer",
                fontSize: "0.85rem",
              }}
            >
              Add
            </button>
          </form>

          {habitTasks.length === 0 && !isLoading && (
            <p style={{ color: "#666", fontSize: "0.9rem" }}>No habits yet.</p>
          )}

          <ul style={{ listStyle: "none", padding: 0 }}>
            {habitTasks.map((task) => {
              const taskCompletions = allCompletions?.[task.id] ?? [];

              return (
                <li
                  key={task.id}
                  style={{
                    padding: "0.5rem 0.75rem",
                    marginBottom: "0.3rem",
                    border: "1px solid #ddd",
                    borderRadius: "4px",
                    background: "#f0f8ff",
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "0.5rem",
                    }}
                  >
                    <span
                      style={{
                        flex: 1,
                        cursor: "pointer",
                        fontSize: "0.9rem",
                      }}
                      onClick={() => {
                        setSelectedTaskId(task.id);
                        setView("detail");
                      }}
                    >
                      {task.title}
                      <span
                        style={{
                          marginLeft: "0.4rem",
                          fontSize: "0.7rem",
                          color: "#007bff",
                          background: "#e6f2ff",
                          padding: "0.1rem 0.3rem",
                          borderRadius: "3px",
                        }}
                      >
                        habit
                      </span>
                    </span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        if (confirm("Delete this habit?")) {
                          deleteMutation.mutate(task.id);
                        }
                      }}
                      style={{
                        padding: "0.2rem 0.5rem",
                        background: "#dc3545",
                        color: "white",
                        border: "none",
                        borderRadius: "3px",
                        cursor: "pointer",
                        fontSize: "0.75rem",
                      }}
                    >
                      Del
                    </button>
                  </div>

                  <HabitWeekRow
                    task={task}
                    completions={taskCompletions}
                    onToggleDay={(taskId, date, completed) => {
                      habitToggleMutation.mutate({ taskId, date, completed });
                    }}
                  />
                </li>
              );
            })}
          </ul>
        </div>
      </div>

      {/* TASK LISTS SECTION */}
      <hr style={{ margin: "1.5rem 0" }} />
      <div>
        <h3 style={{ margin: "0 0 0.75rem 0", color: "#333" }}>
          📁 Task Lists
        </h3>

        {/* Create new list */}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (newListName.trim()) createListMutation.mutate(newListName.trim());
          }}
          style={{ display: "flex", gap: "0.3rem", marginBottom: "0.75rem" }}
        >
          <input
            type="text"
            value={newListName}
            onChange={(e) => setNewListName(e.target.value)}
            placeholder="New list name (e.g. Shopping List)..."
            style={{
              flex: 1,
              padding: "0.4rem",
              border: "1px solid #ccc",
              borderRadius: "4px",
              fontSize: "0.9rem",
            }}
          />
          <button
            type="submit"
            disabled={createListMutation.isPending || !newListName.trim()}
            style={{
              padding: "0.4rem 0.7rem",
              background:
                createListMutation.isPending || !newListName.trim()
                  ? "#6c757d"
                  : "#007bff",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor:
                createListMutation.isPending || !newListName.trim()
                  ? "not-allowed"
                  : "pointer",
              fontSize: "0.85rem",
            }}
          >
            Create
          </button>
        </form>

        {taskLists.length === 0 && (
          <p style={{ color: "#666", fontSize: "0.9rem" }}>
            No lists yet. Create one above!
          </p>
        )}

        <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
          {taskLists.map((list) => (
            <div
              key={list.id}
              style={{
                padding: "0.5rem 0.75rem",
                border: "1px solid #ddd",
                borderRadius: "6px",
                background: "white",
                display: "flex",
                alignItems: "center",
                gap: "0.5rem",
                cursor: "pointer",
              }}
              onClick={() => {
                setSelectedListId(list.id);
                setView("listDetail");
              }}
            >
              <span style={{ fontWeight: "bold", fontSize: "0.9rem" }}>
                {list.name}
              </span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  if (confirm(`Delete list "${list.name}"?`)) {
                    deleteListMutation.mutate(list.id);
                  }
                }}
                style={{
                  padding: "0.15rem 0.4rem",
                  background: "#dc3545",
                  color: "white",
                  border: "none",
                  borderRadius: "3px",
                  cursor: "pointer",
                  fontSize: "0.7rem",
                }}
              >
                ✕
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
