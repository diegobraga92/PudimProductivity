import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import { useConfirm } from "../components/useConfirm";
import {
  listTasks,
  deleteTask,
  updateTask,
  completeTask,
  uncompleteTask,
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
import QuickAddForm from "../components/QuickAddForm";
import TaskCard from "../components/TaskCard";
import ListDetailPanel from "../components/ListDetailPanel";
import TaskListShare from "../components/TaskListShare";
import WeekHeatmap from "../components/WeekHeatmap";
import StreakBadge from "../components/StreakBadge";
import ProgressBar from "../components/ProgressBar";
import SortSelect from "../components/SortSelect";
import { usePersistedSort } from "../hooks/usePersistedSort";
import { useHabitCompletions } from "../hooks/useHabitCompletions";
import { usePresence } from "../hooks/usePresence";
import { DEV_USER_ID } from "../api/client";
import { computeStreaks } from "../utils/streaks";
import { getRollingWindowDates, formatWeekRange } from "../utils/dates";
import { sortTasks } from "../utils/sort";
import {
  playHabitCompletionSound,
  playTodoCompletionSound,
  playHabitUncompletionSound,
  playTodoUncompletionSound,
} from "../utils/sounds";

type View = "list" | "create" | "detail";

export default function TaskList() {
  const queryClient = useQueryClient();
  const [view, setView] = useState<View>("list");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [selectedListForDetail, setSelectedListForDetail] = useState<string | null>(null);
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [newHabitTitle, setNewHabitTitle] = useState("");
  const [newListName, setNewListName] = useState("");
  const [weekOffset, setWeekOffset] = useState(0);
  const [todoSort, setTodoSort] = usePersistedSort("taskSort.todos", "created-desc");
  const [habitSort, setHabitSort] = usePersistedSort("taskSort.habits", "created-desc");
  // Phase 8: which list's share dialog is open.
  const [shareListId, setShareListId] = useState<string | null>(null);
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

  // Phase 8: live presence — who is online per list.
  const { online: onlineByList } = usePresence(taskLists.map((l) => l.id));

  const isLoading = todoLoading || habitLoading;
  const error = todoError || habitError;

  // Fetch completion history for all habit tasks using a single batch request.
  const weekDates = getRollingWindowDates(weekOffset);
  const allCompletions = useHabitCompletions(habitTasks);

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
      } else {
        playTodoUncompletionSound();
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
        playHabitUncompletionSound();
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

  // Sort tasks based on selected sort options
  const sortedTodoTasks = sortTasks(todoTasks, todoSort);
  const sortedHabitTasks = sortTasks(habitTasks, habitSort);

  return (
    <div className="animate-fade-in">
      {/* Page Header */}
      <div className="page-header">
        <div>
          <h1 className="page-heading" style={{ marginBottom: "0.25rem" }}>
            📋 Tasks
          </h1>
          <p className="page-subtitle">
            Manage your todos, habits, and lists
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setView("create")}>
          + New Task
        </button>
      </div>

      {/* Progress Row */}
      <div
        className="flex flex-wrap"
        style={{
          gap: "var(--space-lg)",
          marginBottom: "var(--space-xl)",
          background: "var(--color-surface)",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-md)",
          padding: "var(--space-md) var(--space-lg)",
        }}
      >
        <span className="text-sm text-bold text-secondary">
          Today's Progress
        </span>
        <div className="flex" style={{ gap: "var(--space-lg)", alignItems: "center" }}>
          <div className="flex-center" style={{ gap: "var(--space-sm)" }}>
            <span className="badge badge-todo">{doneTodos}/{todoTasks.length}</span>
            <div style={{ width: "100px" }}>
              <ProgressBar value={todoProgress} variant="todo" />
            </div>
          </div>
        </div>
      </div>

      {isLoading && (
        <div className="card loading-card">
          <p className="text-secondary">Loading tasks...</p>
        </div>
      )}
      {error && (
        <div className="card error-card">
          <p className="error-text">
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
            <SortSelect
              value={todoSort}
              onChange={setTodoSort}
              options={["created-desc", "created-asc", "alpha-asc", "alpha-desc"]}
            />
          </div>

          {/* Quick-add for todos */}
          <QuickAddForm
            value={newTodoTitle}
            onChange={setNewTodoTitle}
            onSubmit={() => {
              if (newTodoTitle.trim()) createTodoMutation.mutate(newTodoTitle.trim());
            }}
            placeholder="Quick add todo..."
            isPending={createTodoMutation.isPending}
          />

          {todoTasks.length === 0 && !isLoading && (
            <div className="empty-state">
              <div className="empty-state-icon">📋</div>
              <p className="empty-state-text">No todos yet.<br />Add your first one above!</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
            {sortedTodoTasks.map((task, index) => (
              <TaskCard
                key={task.id}
                variant="todo"
                title={task.title}
                done={task.status === "done"}
                onToggle={() => toggleMutation.mutate({ id: task.id, status: task.status })}
                onDelete={async () => {
                  const ok = await confirm({
                    title: "Delete this task?",
                    confirmLabel: "Delete",
                    confirmVariant: "danger",
                  });
                  if (ok) {
                    deleteMutation.mutate(task.id);
                  }
                }}
                onTitleClick={() => {
                  setSelectedTaskId(task.id);
                  setView("detail");
                }}
                animationDelay={`${index * 30}ms`}
              />
            ))}
          </div>
        </div>

        {/* ===== HABITS COLUMN ===== */}
        <div className="task-column">
          <div className="task-column-header">
            <h2 className="section-header" style={{ marginBottom: 0 }}>
              🔄 Habits <span className="badge badge-habit">{habitTasks.length}</span>
            </h2>
            <SortSelect
              value={habitSort}
              onChange={setHabitSort}
              options={["created-desc", "created-asc", "alpha-asc", "alpha-desc", "time-asc", "time-desc"]}
            />
          </div>

          {/* Shared week navigation for all habits */}
          <div
            className="flex-between"
            style={{
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
              className="text-sm text-bold text-secondary"
            >
              {weekOffset === 0
                ? "Last 7 Days"
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
          <QuickAddForm
            value={newHabitTitle}
            onChange={setNewHabitTitle}
            onSubmit={() => {
              if (newHabitTitle.trim()) createHabitMutation.mutate(newHabitTitle.trim());
            }}
            placeholder="Quick add habit (weekdays)..."
            isPending={createHabitMutation.isPending}
          />

          {habitTasks.length === 0 && !isLoading && (
            <div className="empty-state">
              <div className="empty-state-icon">🔄</div>
              <p className="empty-state-text">No habits yet.<br />Start building one above!</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
            {sortedHabitTasks.map((task, index) => {
              const taskCompletions = allCompletions?.[task.id] ?? [];
              const { current, longest } = computeStreaks(taskCompletions, task.recurrence_days);

              return (
                <div
                  key={task.id}
                  className="card card-habit"
                  style={{ animationDelay: `${index * 30}ms` }}
                >
                  {/* Row 1: Title + Streak + Kebab (hover reveal) */}
                  <div className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "0.35rem" }}>
                    <span
                      className="flex-1"
                      style={{
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
            <QuickAddForm
              value={newListName}
              onChange={setNewListName}
              onSubmit={() => {
                if (newListName.trim()) createListMutation.mutate(newListName.trim());
              }}
              placeholder="New list name..."
              submitLabel="Create"
              isPending={createListMutation.isPending}
            />

            {taskLists.length === 0 && (
              <div className="empty-state" style={{ padding: "var(--space-md)" }}>
                <div className="empty-state-icon" style={{ fontSize: "1.5rem" }}>📁</div>
                <p className="empty-state-text">No lists yet.<br />Create your first one above!</p>
              </div>
            )}

            <div style={{ display: "flex", flexDirection: "column", gap: "0.35rem" }}>
              {taskLists.map((list) => {
                const ownerId = list.owner_id ?? "";
                const isOwner = ownerId === DEV_USER_ID;
                const ownerOnline = onlineByList.get(list.id)?.has(ownerId) ?? false;
                return (
                  <div
                    key={list.id}
                    className={`list-picker-item ${selectedListForDetail === list.id ? "selected" : ""}`}
                    onClick={() => setSelectedListForDetail(list.id)}
                  >
                    <span style={{ fontSize: "1rem" }}>📁</span>
                    {/* Phase 8: owner presence dot */}
                    <span
                      style={{
                        width: 8,
                        height: 8,
                        borderRadius: "50%",
                        background: ownerOnline ? "#00b894" : "#b2bec3",
                        flexShrink: 0,
                      }}
                      title={`Owner ${list.owner_id} ${ownerOnline ? "online" : "offline"}`}
                    />
                    <span
                      className="flex-1"
                      style={{
                        fontWeight: selectedListForDetail === list.id ? 600 : 500,
                        fontSize: "var(--font-size-sm)",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {list.name}
                    </span>
                    {/* Phase 8: share dialog */}
                    <button
                      className="btn btn-ghost btn-sm"
                      style={{ padding: "0.15rem 0.4rem", fontSize: "0.7rem" }}
                      title="Share list"
                      onClick={(e) => {
                        e.stopPropagation();
                        setShareListId(list.id);
                      }}
                    >
                      👥
                    </button>
                    {/* Only the owner can delete a list (Phase 8) */}
                    {isOwner && (
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
                    )}
                  </div>
                );
              })}
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

      {/* Phase 8: share dialog */}
      {shareListId &&
        (() => {
          const list = taskLists.find((l) => l.id === shareListId);
          if (!list) return null;
          return (
            <TaskListShare
              listId={list.id}
              listName={list.name}
              isOwner={list.owner_id === DEV_USER_ID}
              onlineUsers={onlineByList.get(list.id)}
              onClose={() => setShareListId(null)}
            />
          );
        })()}
    </div>
  );
}