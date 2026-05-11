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
import Checkbox from "../components/Checkbox";
import WeekHeatmap from "../components/WeekHeatmap";
import StreakBadge from "../components/StreakBadge";
import ProgressBar from "../components/ProgressBar";
import { computeStreaks } from "../utils/streaks";

type View = "list" | "create" | "detail" | "listDetail";

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

interface TaskListProps {
  onNavigate?: (view: string, taskId?: string) => void;
}

export default function TaskList(_props: TaskListProps) {
  const queryClient = useQueryClient();
  const [view, setView] = useState<View>("list");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [selectedListId, setSelectedListId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<TaskStatus | "">("");
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [newHabitTitle, setNewHabitTitle] = useState("");
  const [newListName, setNewListName] = useState("");
  const [activeTab, setActiveTab] = useState<"todos" | "habits" | "lists">("todos");

  // Fetch unassigned one-off tasks
  const { data: todoTasks = [], isLoading: todoLoading, error: todoError } = useQuery<Task[]>({
    queryKey: ["tasks", "one-off", statusFilter],
    queryFn: () =>
      listTasks(statusFilter ? (statusFilter as TaskStatus) : undefined, "one-off"),
  });

  // Fetch unassigned habit tasks
  const { data: habitTasks = [], isLoading: habitLoading, error: habitError } = useQuery<Task[]>({
    queryKey: ["tasks", "habit", statusFilter],
    queryFn: () =>
      listTasks(statusFilter ? (statusFilter as TaskStatus) : undefined, "habit"),
  });

  // Fetch task lists
  const { data: taskLists = [] } = useQuery<TaskList[]>({
    queryKey: ["taskLists"],
    queryFn: listTaskLists,
  });

  const isLoading = todoLoading || habitLoading;
  const error = todoError || habitError;

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
      queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) =>
      updateTask(id, { status: status === "done" ? "todo" : "done" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
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
      queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
      setNewTodoTitle("");
    },
  });

  const createHabitMutation = useMutation({
    mutationFn: (title: string) =>
      createTask({ title, recurrence_days: ["mon", "tue", "wed", "thu", "fri"] }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
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
          queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
          queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
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
        onUpdated={() => {
          queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
          queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
        }}
        onDeleted={() => {
          queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
          queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
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

  // Compute stats
  const doneTodos = todoTasks.filter((t) => t.status === "done").length;
  const todoProgress = todoTasks.length > 0 ? Math.round((doneTodos / todoTasks.length) * 100) : 0;

  const today = new Date().toISOString().split("T")[0];
  const todayHabitCompletions = Object.values(allCompletions ?? {}).filter((dates) =>
    dates.includes(today)
  ).length;
  const habitProgress = habitTasks.length > 0 ? Math.round((todayHabitCompletions / habitTasks.length) * 100) : 0;

  return (
    <div className="animate-fade-in">
      {/* Page Header */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "var(--space-lg)",
        }}
      >
        <div>
          <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700, marginBottom: "0.2rem" }}>
            📋 Tasks
          </h2>
          <p style={{ margin: 0, fontSize: "var(--font-size-sm)", color: "var(--color-text-secondary)" }}>
            Manage your todos, habits, and lists
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setView("create")}>
          + New Task
        </button>
      </div>

      {/* Filter + Progress Row */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: "var(--space-md)",
          marginBottom: "var(--space-lg)",
          flexWrap: "wrap",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
          <select
            className="select"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as TaskStatus | "")}
          >
            <option value="">All</option>
            <option value="todo">Todo</option>
            <option value="done">Done</option>
          </select>
        </div>

        {/* Mini progress indicators */}
        <div style={{ display: "flex", gap: "var(--space-lg)", alignItems: "center" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
            <span className="badge badge-todo">{doneTodos}/{todoTasks.length}</span>
            <div style={{ width: "80px" }}>
              <ProgressBar value={todoProgress} variant="todo" />
            </div>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
            <span className="badge badge-habit">{todayHabitCompletions}/{habitTasks.length}</span>
            <div style={{ width: "80px" }}>
              <ProgressBar value={habitProgress} variant="habit" />
            </div>
          </div>
        </div>
      </div>

      {isLoading && (
        <div className="card" style={{ textAlign: "center", padding: "var(--space-xl)" }}>
          <p style={{ color: "var(--color-text-secondary)" }}>Loading tasks...</p>
        </div>
      )}
      {error && (
        <div className="card" style={{ borderLeft: "3px solid var(--color-danger)", marginBottom: "var(--space-md)" }}>
          <p style={{ color: "var(--color-danger)", margin: 0 }}>
            Error: {(error as Error).message}
          </p>
        </div>
      )}

      {/* ===== SECTION TABS ===== */}
      <div
        style={{
          display: "flex",
          gap: "0",
          marginBottom: "var(--space-lg)",
          borderBottom: "2px solid var(--color-border-light)",
        }}
      >
        {[
          { id: "todos" as const, label: "To-Dos", icon: "📋", count: todoTasks.length },
          { id: "habits" as const, label: "Habits", icon: "🔄", count: habitTasks.length },
          { id: "lists" as const, label: "Lists", icon: "📁", count: taskLists.length },
        ].map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.35rem",
              padding: "0.6rem 1.2rem",
              background: "transparent",
              border: "none",
              borderBottom: activeTab === tab.id ? "2px solid var(--color-primary)" : "2px solid transparent",
              marginBottom: "-2px",
              cursor: "pointer",
              fontFamily: "var(--font-family)",
              fontSize: "var(--font-size-sm)",
              fontWeight: activeTab === tab.id ? 600 : 400,
              color: activeTab === tab.id ? "var(--color-primary)" : "var(--color-text-secondary)",
              transition: "all var(--transition-fast)",
            }}
          >
            <span>{tab.icon}</span>
            <span>{tab.label}</span>
            <span
              className="badge"
              style={{
                background: activeTab === tab.id ? "var(--color-primary)" : "var(--color-border-light)",
                color: activeTab === tab.id ? "white" : "var(--color-text-secondary)",
                padding: "0.05rem 0.35rem",
                fontSize: "0.65rem",
              }}
            >
              {tab.count}
            </span>
          </button>
        ))}
      </div>

      {/* ===== TODOS TAB ===== */}
      {activeTab === "todos" && (
        <div className="animate-fade-in">
          {/* Quick-add for todos */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (newTodoTitle.trim()) createTodoMutation.mutate(newTodoTitle.trim());
            }}
            style={{ display: "flex", gap: "0.3rem", marginBottom: "var(--space-md)" }}
          >
            <input
              type="text"
              className="input"
              value={newTodoTitle}
              onChange={(e) => setNewTodoTitle(e.target.value)}
              placeholder="Quick add todo..."
            />
            <button
              type="submit"
              className="btn btn-primary"
              disabled={createTodoMutation.isPending || !newTodoTitle.trim()}
            >
              {createTodoMutation.isPending ? "..." : "Add"}
            </button>
          </form>

          {todoTasks.length === 0 && !isLoading && (
            <div className="empty-state">
              <div className="empty-state-icon">📋</div>
              <p className="empty-state-text">No todos yet. Add one above!</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
            {todoTasks.map((task, index) => (
              <div
                key={task.id}
                className={`card card-todo ${task.status === "done" ? "card-done" : ""}`}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "var(--space-sm)",
                  padding: "0.6rem 0.8rem",
                  animationDelay: `${index * 30}ms`,
                }}
              >
                <Checkbox
                  checked={task.status === "done"}
                  onChange={() =>
                    toggleMutation.mutate({ id: task.id, status: task.status })
                  }
                />
                <span
                  style={{
                    flex: 1,
                    cursor: "pointer",
                    textDecoration: task.status === "done" ? "line-through" : "none",
                    color: task.status === "done" ? "var(--color-text-muted)" : "var(--color-text)",
                    fontSize: "var(--font-size-base)",
                    transition: "all var(--transition-fast)",
                  }}
                  onClick={() => {
                    setSelectedTaskId(task.id);
                    setView("detail");
                  }}
                >
                  {task.title}
                </span>
                <button
                  className="btn btn-danger btn-sm"
                  onClick={(e) => {
                    e.stopPropagation();
                    if (confirm("Delete this task?")) {
                      deleteMutation.mutate(task.id);
                    }
                  }}
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ===== HABITS TAB ===== */}
      {activeTab === "habits" && (
        <div className="animate-fade-in">
          {/* Quick-add for habits */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (newHabitTitle.trim()) createHabitMutation.mutate(newHabitTitle.trim());
            }}
            style={{ display: "flex", gap: "0.3rem", marginBottom: "var(--space-md)" }}
          >
            <input
              type="text"
              className="input"
              value={newHabitTitle}
              onChange={(e) => setNewHabitTitle(e.target.value)}
              placeholder="Quick add habit (weekdays)..."
            />
            <button
              type="submit"
              className="btn btn-primary"
              style={{ background: "var(--color-habit)" }}
              disabled={createHabitMutation.isPending || !newHabitTitle.trim()}
            >
              {createHabitMutation.isPending ? "..." : "Add"}
            </button>
          </form>

          {habitTasks.length === 0 && !isLoading && (
            <div className="empty-state">
              <div className="empty-state-icon">🔄</div>
              <p className="empty-state-text">No habits yet. Add one above!</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
            {habitTasks.map((task, index) => {
              const taskCompletions = allCompletions?.[task.id] ?? [];
              const { current, longest } = computeStreaks(taskCompletions);
              const weeklyDone = taskCompletions.filter((d) => d <= today).length;
              const weeklyTotal = task.recurrence_days?.length ?? 0;
              const weeklyPct = weeklyTotal > 0 ? Math.round((weeklyDone / weeklyTotal) * 100) : 0;

              return (
                <div
                  key={task.id}
                  className="card card-habit"
                  style={{
                    padding: "0.7rem 0.8rem",
                    animationDelay: `${index * 30}ms`,
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "var(--space-sm)",
                      marginBottom: "0.4rem",
                    }}
                  >
                    <span
                      style={{
                        flex: 1,
                        cursor: "pointer",
                        fontSize: "var(--font-size-base)",
                        fontWeight: 500,
                      }}
                      onClick={() => {
                        setSelectedTaskId(task.id);
                        setView("detail");
                      }}
                    >
                      {task.title}
                    </span>
                    <StreakBadge current={current} longest={longest} />
                    <span className="badge badge-habit">habit</span>
                    <button
                      className="btn btn-danger btn-sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        if (confirm("Delete this habit?")) {
                          deleteMutation.mutate(task.id);
                        }
                      }}
                    >
                      ✕
                    </button>
                  </div>

                  {/* Weekly progress bar */}
                  <div style={{ marginBottom: "0.4rem" }}>
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        fontSize: "var(--font-size-xs)",
                        color: "var(--color-text-muted)",
                        marginBottom: "0.15rem",
                      }}
                    >
                      <span>This week</span>
                      <span>{weeklyDone}/{weeklyTotal}</span>
                    </div>
                    <ProgressBar value={weeklyPct} variant="habit" />
                  </div>

                  <WeekHeatmap
                    recurrenceDays={task.recurrence_days ?? []}
                    completions={taskCompletions}
                    onToggleDay={(date, completed) => {
                      habitToggleMutation.mutate({ taskId: task.id, date, completed });
                    }}
                    disabled={habitToggleMutation.isPending}
                  />
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ===== LISTS TAB ===== */}
      {activeTab === "lists" && (
        <div className="animate-fade-in">
          {/* Create new list */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (newListName.trim()) createListMutation.mutate(newListName.trim());
            }}
            style={{ display: "flex", gap: "0.3rem", marginBottom: "var(--space-md)" }}
          >
            <input
              type="text"
              className="input"
              value={newListName}
              onChange={(e) => setNewListName(e.target.value)}
              placeholder="New list name (e.g. Shopping List)..."
            />
            <button
              type="submit"
              className="btn btn-primary"
              style={{ background: "var(--color-list)" }}
              disabled={createListMutation.isPending || !newListName.trim()}
            >
              {createListMutation.isPending ? "..." : "Create"}
            </button>
          </form>

          {taskLists.length === 0 && (
            <div className="empty-state">
              <div className="empty-state-icon">📁</div>
              <p className="empty-state-text">No lists yet. Create one above!</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
            {taskLists.map((list) => (
              <div
                key={list.id}
                className="card card-list"
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "var(--space-sm)",
                  padding: "0.7rem 0.8rem",
                  cursor: "pointer",
                }}
                onClick={() => {
                  setSelectedListId(list.id);
                  setView("listDetail");
                }}
              >
                <span style={{ fontSize: "1.2rem" }}>📁</span>
                <span style={{ flex: 1, fontWeight: 600, fontSize: "var(--font-size-base)" }}>
                  {list.name}
                </span>
                <button
                  className="btn btn-danger btn-sm"
                  onClick={(e) => {
                    e.stopPropagation();
                    if (confirm(`Delete list "${list.name}"?`)) {
                      deleteListMutation.mutate(list.id);
                    }
                  }}
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
