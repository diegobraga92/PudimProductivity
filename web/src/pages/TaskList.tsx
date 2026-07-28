import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import { useConfirm } from "../components/ConfirmProvider";
import {
  listTasks,
  deleteTask,
  updateTask,
  completeTask,
  uncompleteTask,
  getAllTaskCompletions,
  createTask,
  type Task,
  type TaskStatus,
  type RecurrenceDay,
} from "../api/tasks";
import {
  listTaskLists,
  createTaskList,
  deleteTaskList,
  getTaskList,
  listTasksByListID,
  updateTaskList,
  type TaskList,
} from "../api/taskLists";
import TaskCreate from "./TaskCreate";
import TaskDetail from "./TaskDetail";
import Checkbox from "../components/Checkbox";
import WeekHeatmap from "../components/WeekHeatmap";
import StreakBadge from "../components/StreakBadge";
import ProgressBar from "../components/ProgressBar";
import { computeStreaks } from "../utils/streaks";
import { getWeekDates, getToday, formatWeekRange } from "../utils/dates";
import { playHabitCompletionSound, playTodoCompletionSound } from "../utils/sounds";

type View = "list" | "create" | "detail";

const DAY_ORDER: RecurrenceDay[] = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];

export default function TaskList() {
  const queryClient = useQueryClient();
  const [view, setView] = useState<View>("list");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [selectedListForDetail, setSelectedListForDetail] = useState<string | null>(null);
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [newHabitTitle, setNewHabitTitle] = useState("");
  const [newListName, setNewListName] = useState("");
  const [weekOffset, setWeekOffset] = useState(0);
  const handleWeekOffsetChange = useCallback((newOffset: number) => {
    setWeekOffset(newOffset);
  }, []);

  // Fetch unassigned one-off tasks
  const { data: todoTasks = [], isLoading: todoLoading, error: todoError } = useQuery<Task[]>({
    queryKey: ["tasks", "one-off"],
    queryFn: () => listTasks(undefined, "one-off"),
  });

  // Fetch unassigned habit tasks
  const { data: habitTasks = [], isLoading: habitLoading, error: habitError } = useQuery<Task[]>({
    queryKey: ["tasks", "habit"],
    queryFn: () => listTasks(undefined, "habit"),
  });

  // Fetch task lists
  const { data: taskLists = [] } = useQuery<TaskList[]>({
    queryKey: ["taskLists"],
    queryFn: listTaskLists,
  });

  const isLoading = todoLoading || habitLoading;
  const error = todoError || habitError;

  // Fetch completions for all habit tasks using a single batch request
  const weekDates = getWeekDates(weekOffset);
  const from = weekDates[0];
  const to = weekDates[6];

  const { data: allCompletions } = useQuery({
    queryKey: ["habitCompletions", from, to],
    queryFn: async () => {
      const completions = await getAllTaskCompletions(from, to);
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

  const confirm = useConfirm();

  // Mutations
  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ id, status }: { id: string; status: TaskStatus }) => {
      if (status === "todo") {
        playTodoCompletionSound();
      }
      return updateTask(id, { status: status === "done" ? "todo" : "done" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
      queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
    },
  });

  const habitToggleMutation = useMutation({
    mutationFn: async ({ taskId, date, completed }: { taskId: string; date: string; completed: boolean }) => {
      if (completed) {
        await uncompleteTask(taskId, date);
      } else {
        playHabitCompletionSound();
        await completeTask(taskId, date);
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
      setSelectedListForDetail(null);
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

  // Compute stats
  const doneTodos = todoTasks.filter((t) => t.status === "done").length;
  const todoProgress = todoTasks.length > 0 ? Math.round((doneTodos / todoTasks.length) * 100) : 0;

  const today = getToday();
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
          alignItems: "flex-start",
          marginBottom: "var(--space-xl)",
        }}
      >
        <div>
          <h1 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700, marginBottom: "0.25rem" }}>
            📋 Tasks
          </h1>
          <p style={{ margin: 0, fontSize: "var(--font-size-sm)", color: "var(--color-text-muted)" }}>
            Manage your todos, habits, and lists
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setView("create")}>
          + New Task
        </button>
      </div>

      {/* Progress Row */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "var(--space-lg)",
          marginBottom: "var(--space-xl)",
          flexWrap: "wrap",
          background: "var(--color-surface)",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-md)",
          padding: "var(--space-md) var(--space-lg)",
        }}
      >
        <span style={{ fontSize: "var(--font-size-sm)", fontWeight: 600, color: "var(--color-text-secondary)" }}>
          Today's Progress
        </span>
        <div style={{ display: "flex", gap: "var(--space-lg)", alignItems: "center" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
            <span className="badge badge-todo">{doneTodos}/{todoTasks.length}</span>
            <div style={{ width: "100px" }}>
              <ProgressBar value={todoProgress} variant="todo" />
            </div>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
            <span className="badge badge-habit">{todayHabitCompletions}/{habitTasks.length}</span>
            <div style={{ width: "100px" }}>
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

      {/* ===== TOP: TO-DOS + HABITS (2-COLUMN GRID) ===== */}
      <div className="task-columns">
        {/* ===== TODOS COLUMN ===== */}
        <div className="task-column">
          <div className="task-column-header">
            <h2 className="section-header" style={{ marginBottom: 0 }}>
              📋 To-Dos <span className="badge badge-todo">{todoTasks.length}</span>
            </h2>
          </div>

          {/* Quick-add for todos */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (newTodoTitle.trim()) createTodoMutation.mutate(newTodoTitle.trim());
            }}
            style={{ display: "flex", gap: "0.5rem", marginBottom: "var(--space-md)" }}
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
              <p className="empty-state-text">No todos yet.<br />Add your first one above!</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
            {todoTasks.map((task, index) => (
              <div
                key={task.id}
                className={`card card-todo ${task.status === "done" ? "card-done" : ""}`}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "var(--space-sm)",
                  padding: "0.75rem 1rem",
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
                    fontWeight: 500,
                    transition: "all var(--transition-fast)",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
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
                  onClick={async (e) => {
                    e.stopPropagation();
                    const ok = await confirm({
                      title: "Delete this task?",
                      confirmLabel: "Delete",
                      confirmVariant: "danger",
                    });
                    if (ok) {
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

        {/* ===== HABITS COLUMN ===== */}
        <div className="task-column">
          <div className="task-column-header">
            <h2 className="section-header" style={{ marginBottom: 0 }}>
              🔄 Habits <span className="badge badge-habit">{habitTasks.length}</span>
            </h2>
          </div>

          {/* Shared week navigation for all habits */}
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: "var(--space-md)",
              padding: "0.4rem 0",
              borderBottom: "1px solid var(--color-border-light)",
            }}
          >
            <button
              className="btn btn-ghost"
              style={{ padding: "0.2rem 0.5rem", fontSize: "var(--font-size-sm)" }}
              onClick={() => handleWeekOffsetChange(weekOffset - 1)}
              aria-label="Previous week"
            >
              &larr; Prev Week
            </button>
            <span
              style={{
                fontSize: "var(--font-size-sm)",
                fontWeight: 600,
                color: "var(--color-text-secondary)",
              }}
            >
              {weekOffset === 0
                ? "This Week"
                : formatWeekRange(weekDates)}
            </span>
            <button
              className="btn btn-ghost"
              style={{ padding: "0.2rem 0.5rem", fontSize: "var(--font-size-sm)" }}
              onClick={() => handleWeekOffsetChange(weekOffset + 1)}
              disabled={weekOffset >= 0}
              aria-label="Next week"
            >
              Next Week &rarr;
            </button>
          </div>

          {/* Quick-add for habits */}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (newHabitTitle.trim()) createHabitMutation.mutate(newHabitTitle.trim());
            }}
            style={{ display: "flex", gap: "0.5rem", marginBottom: "var(--space-md)" }}
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
              disabled={createHabitMutation.isPending || !newHabitTitle.trim()}
            >
              {createHabitMutation.isPending ? "..." : "Add"}
            </button>
          </form>

          {habitTasks.length === 0 && !isLoading && (
            <div className="empty-state">
              <div className="empty-state-icon">🔄</div>
              <p className="empty-state-text">No habits yet.<br />Start building one above!</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
            {habitTasks.map((task, index) => {
              const taskCompletions = allCompletions?.[task.id] ?? [];
              const { current, longest } = computeStreaks(taskCompletions);
              const weekScheduledDates = (task.recurrence_days ?? []).map(
                (day) => weekDates[DAY_ORDER.indexOf(day)]
              ).filter((d): d is string => d !== undefined);
              const weeklyDone = taskCompletions.filter((d) => weekScheduledDates.includes(d) && d <= today).length;
              const weeklyTotal = weekScheduledDates.filter((d) => d <= today).length;
              const weeklyPct = weeklyTotal > 0 ? Math.round((weeklyDone / weeklyTotal) * 100) : 0;
              const allDone = weeklyTotal > 0 && weeklyDone >= weeklyTotal;

              return (
                <div
                  key={task.id}
                  className="card card-habit"
                  style={{
                    animationDelay: `${index * 30}ms`,
                  }}
                >
                  {/* Row 1: Title + Streak + Kebab (hover reveal) */}
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "var(--space-sm)",
                      marginBottom: "0.35rem",
                    }}
                  >
                    <span
                      style={{
                        flex: 1,
                        cursor: "pointer",
                        fontSize: "var(--font-size-sm)",
                        fontWeight: 600,
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                      onClick={() => {
                        setSelectedTaskId(task.id);
                        setView("detail");
                      }}
                    >
                      {task.title}
                    </span>
                    <StreakBadge current={current} longest={longest} />
                    <button
                      className="btn btn-danger btn-sm habit-kebab"
                      onClick={async (e) => {
                        e.stopPropagation();
                        const ok = await confirm({
                          title: "Delete this habit?",
                          confirmLabel: "Delete",
                          confirmVariant: "danger",
                        });
                        if (ok) {
                          deleteMutation.mutate(task.id);
                        }
                      }}
                      title="Delete habit"
                      aria-label="Delete habit"
                    >
                      ⋯
                    </button>
                  </div>

                  {/* Row 2: Completion buttons — primary interaction */}
                    <div style={{ marginBottom: "0.35rem" }}>
                      <WeekHeatmap
                        recurrenceDays={task.recurrence_days ?? []}
                        completions={taskCompletions}
                        onToggleDay={(date, completed) => {
                          habitToggleMutation.mutate({ taskId: task.id, date, completed });
                        }}
                        disabled={habitToggleMutation.isPending}
                        weekOffset={weekOffset}
                      />
                    </div>

                  {/* Row 3: Compact progress bar + count */}
                  <div className="progress-bar-compact">
                    <div className="progress-bar">
                      <div
                        className="progress-bar-fill habit"
                        style={{ width: `${weeklyPct}%` }}
                      />
                    </div>
                    <span className={`progress-count ${allDone ? "done" : ""}`}>
                      {weeklyDone}/{weeklyTotal}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* ===== BOTTOM: LISTS SECTION ===== */}
      <div className="task-lists-section">
        <h2 className="section-header">
          📁 Lists <span className="badge" style={{ background: "var(--color-list-light)", color: "#047857" }}>{taskLists.length}</span>
        </h2>

        <div className="lists-inner">
          {/* ===== LEFT: LIST PICKER ===== */}
          <div className="list-picker">
            {/* Create new list */}
            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (newListName.trim()) createListMutation.mutate(newListName.trim());
              }}
              style={{ display: "flex", gap: "0.5rem", marginBottom: "var(--space-md)" }}
            >
              <input
                type="text"
                className="input"
                value={newListName}
                onChange={(e) => setNewListName(e.target.value)}
                placeholder="New list name..."
              />
              <button
                type="submit"
                className="btn btn-primary"
                disabled={createListMutation.isPending || !newListName.trim()}
              >
                {createListMutation.isPending ? "..." : "Create"}
              </button>
            </form>

            {taskLists.length === 0 && (
              <div className="empty-state" style={{ padding: "var(--space-md)" }}>
                <div className="empty-state-icon" style={{ fontSize: "1.5rem" }}>📁</div>
                <p className="empty-state-text">No lists yet.<br />Create your first one above!</p>
              </div>
            )}

            <div style={{ display: "flex", flexDirection: "column", gap: "0.35rem" }}>
              {taskLists.map((list) => (
                <div
                  key={list.id}
                  className={`list-picker-item ${selectedListForDetail === list.id ? "selected" : ""}`}
                  onClick={() => setSelectedListForDetail(list.id)}
                >
                  <span style={{ fontSize: "1rem" }}>📁</span>
                  <span
                    style={{
                      flex: 1,
                      fontWeight: selectedListForDetail === list.id ? 600 : 500,
                      fontSize: "var(--font-size-sm)",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {list.name}
                  </span>
                  <button
                    className="btn btn-danger btn-sm"
                    style={{ padding: "0.15rem 0.4rem", fontSize: "0.65rem" }}
                    onClick={async (e) => {
                      e.stopPropagation();
                      const ok = await confirm({
                        title: `Delete list "${list.name}"?`,
                        confirmLabel: "Delete",
                        confirmVariant: "danger",
                      });
                      if (ok) {
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

          {/* ===== RIGHT: LIST DETAIL ===== */}
          <div className="list-detail">
            {selectedListForDetail ? (
              <ListDetailPanel
                listId={selectedListForDetail}
                onListDeleted={() => {
                  queryClient.invalidateQueries({ queryKey: ["taskLists"] });
                  setSelectedListForDetail(null);
                }}
              />
            ) : (
              <div className="empty-state">
                <div className="empty-state-icon">👈</div>
                <p className="empty-state-text">Select a list from the left to view and manage its tasks</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ===== Inline List Detail Panel ===== */
function ListDetailPanel({ listId, onListDeleted }: { listId: string; onListDeleted: () => void }) {
  const queryClient = useQueryClient();
  const [newTitle, setNewTitle] = useState("");
  const [editingName, setEditingName] = useState(false);
  const [editName, setEditName] = useState("");
  const confirm = useConfirm();

  const { data: taskList, isLoading: listLoading } = useQuery({
    queryKey: ["taskList", listId],
    queryFn: () => getTaskList(listId),
  });

  const { data: tasks = [], isLoading: tasksLoading } = useQuery<Task[]>({
    queryKey: ["taskListTasks", listId],
    queryFn: () => listTasksByListID(listId),
  });

  const createMutation = useMutation({
    mutationFn: (title: string) =>
      createTask({ title, list_id: listId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskListTasks", listId] });
      setNewTitle("");
    },
  });

  const deleteTaskMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskListTasks", listId] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ id, status }: { id: string; status: TaskStatus }) => {
      if (status === "todo") {
        playTodoCompletionSound();
      }
      return updateTask(id, { status: status === "done" ? "todo" : "done" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskListTasks", listId] });
    },
  });

  const deleteListMutation = useMutation({
    mutationFn: () => deleteTaskList(listId),
    onSuccess: onListDeleted,
  });

  const updateListMutation = useMutation({
    mutationFn: (name: string) => updateTaskList(listId, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskList", listId] });
      queryClient.invalidateQueries({ queryKey: ["taskLists"] });
      setEditingName(false);
    },
  });

  const handleQuickAdd = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTitle.trim()) return;
    createMutation.mutate(newTitle.trim());
  };

  if (listLoading) {
    return (
      <div className="card" style={{ textAlign: "center", padding: "var(--space-lg)" }}>
        <p style={{ color: "var(--color-text-secondary)" }}>Loading list...</p>
      </div>
    );
  }

  if (!taskList) {
    return (
      <div className="card" style={{ borderLeft: "3px solid var(--color-danger)" }}>
        <p style={{ color: "var(--color-danger)" }}>List not found</p>
      </div>
    );
  }

  const doneCount = tasks.filter((t) => t.status === "done").length;
  const progress = tasks.length > 0 ? Math.round((doneCount / tasks.length) * 100) : 0;

  return (
    <div>
      {/* Title / Edit */}
      {editingName ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (editName.trim()) updateListMutation.mutate(editName.trim());
          }}
          style={{ marginBottom: "var(--space-md)", display: "flex", gap: "0.5rem" }}
        >
          <input
            type="text"
            className="input"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            autoFocus
          />
          <button type="submit" className="btn btn-primary btn-sm">
            Save
          </button>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setEditingName(false)}>
            Cancel
          </button>
        </form>
      ) : (
        <div style={{ marginBottom: "var(--space-md)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
            <span style={{ fontSize: "1.2rem" }}>📁</span>
            <h3 style={{ fontSize: "var(--font-size-lg)", fontWeight: 700, flex: 1 }}>{taskList.name}</h3>
            <button
              className="btn btn-ghost btn-sm"
              onClick={() => {
                setEditName(taskList.name);
                setEditingName(true);
              }}
            >
              ✏️
            </button>
            <button
              className="btn btn-danger btn-sm"
              onClick={async () => {
                const ok = await confirm({
                  title: `Delete list "${taskList.name}"?`,
                  message: "All tasks in this list will also be deleted.",
                  confirmLabel: "Delete",
                  confirmVariant: "danger",
                });
                if (ok) {
                  deleteListMutation.mutate();
                }
              }}
            >
              🗑
            </button>
          </div>
          {tasks.length > 0 && (
            <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)", marginTop: "var(--space-sm)" }}>
              <span className="badge badge-done">{doneCount}/{tasks.length} done</span>
              <div style={{ flex: 1, maxWidth: "150px" }}>
                <div className="progress-bar">
                  <div className="progress-bar-fill" style={{ width: `${progress}%` }} />
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Quick-add form */}
      <form
        onSubmit={handleQuickAdd}
        style={{ display: "flex", gap: "0.5rem", marginBottom: "var(--space-md)" }}
      >
        <input
          type="text"
          className="input"
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          placeholder="Add a task to this list..."
        />
        <button
          type="submit"
          className="btn btn-primary"
          disabled={createMutation.isPending || !newTitle.trim()}
        >
          {createMutation.isPending ? "..." : "Add"}
        </button>
      </form>

      {tasksLoading && (
        <div className="card" style={{ textAlign: "center", padding: "var(--space-lg)" }}>
          <p style={{ color: "var(--color-text-secondary)" }}>Loading tasks...</p>
        </div>
      )}

      {!tasksLoading && tasks.length === 0 && (
        <div className="empty-state" style={{ padding: "var(--space-md)" }}>
          <div className="empty-state-icon" style={{ fontSize: "1.5rem" }}>📝</div>
          <p className="empty-state-text">No tasks in this list yet.<br />Add your first one above!</p>
        </div>
      )}

      {tasks.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
          {tasks.map((task) => (
            <div
              key={task.id}
              className={`card card-list ${task.status === "done" ? "card-done" : ""}`}
              style={{
                display: "flex",
                alignItems: "center",
                gap: "var(--space-sm)",
                padding: "0.6rem 0.85rem",
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
                  textDecoration: task.status === "done" ? "line-through" : "none",
                  color: task.status === "done" ? "var(--color-text-muted)" : "var(--color-text)",
                  fontSize: "var(--font-size-sm)",
                  fontWeight: 500,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
              >
                {task.title}
              </span>
              <button
                className="btn btn-danger btn-sm"
                onClick={async () => {
                  const ok = await confirm({
                    title: "Delete this task?",
                    confirmLabel: "Delete",
                    confirmVariant: "danger",
                  });
                  if (ok) {
                    deleteTaskMutation.mutate(task.id);
                  }
                }}
              >
                ✕
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
