import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Fragment, useState, useCallback } from "react";
import { useConfirm } from "../components/useConfirm";
import {
  listTasks,
  deleteTask,
  updateTask,
  completeTask,
  uncompleteTask,
  createTask,
  reorderTasks,
  type Task,
  type TaskStatus,
} from "../api/tasks";
import TaskCreate from "./TaskCreate";
import TaskEditModal from "../components/TaskEditModal";
import QuickAddForm from "../components/QuickAddForm";
import TaskCard from "../components/TaskCard";
import WeekHeatmap from "../components/WeekHeatmap";
import StreakBadge from "../components/StreakBadge";
import SortSelect from "../components/SortSelect";
import { TasksIcon } from "../components/icons";
import { usePersistedSort } from "../hooks/usePersistedSort";
import { useHabitCompletions } from "../hooks/useHabitCompletions";
import { useDragReorder } from "../hooks/useDragReorder";
import { useI18n } from "../i18n";
import { computeStreaks } from "../utils/streaks";
import { getRollingWindowDates, formatWeekRange } from "../utils/dates";
import { sortTasks, type SortOption } from "../utils/sort";
import { positionsFor } from "../utils/reorder";
import { invalidateTasks, queryKeys } from "../utils/queryKeys";
import {
  playHabitCompletionSound,
  playTodoCompletionSound,
  playHabitUncompletionSound,
  playTodoUncompletionSound,
} from "../utils/sounds";

type TaskKind = "one-off" | "habit";

const taskQueryKey = (kind: TaskKind) =>
  kind === "habit" ? queryKeys.tasksHabit : queryKeys.tasksOneOff;

const TODO_SORTS: SortOption[] = ["manual", "created-desc", "created-asc", "alpha-asc", "alpha-desc"];
const HABIT_SORTS: SortOption[] = [...TODO_SORTS, "time-asc", "time-desc"];

export default function TaskList() {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [showCreate, setShowCreate] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [weekOffset, setWeekOffset] = useState(0);
  const [todoSort, setTodoSort] = usePersistedSort("taskSort.todos", "created-desc");
  const [habitSort, setHabitSort] = usePersistedSort("taskSort.habits", "created-desc");
  // Which sub-list of to-dos is shown: unchecked ("open") or completed.
  const [todoTab, setTodoTab] = useState<"open" | "completed">("open");
  // Announced to screen readers after a keyboard or pointer reorder.
  const [reorderMessage, setReorderMessage] = useState("");
  const handleWeekOffsetChange = useCallback((newOffset: number) => {
    setWeekOffset(newOffset);
  }, []);

  const { data: todoTasks = [], isLoading: todoLoading, error: todoError } = useQuery<Task[]>({
    queryKey: queryKeys.tasksOneOff,
    queryFn: () => listTasks(undefined, "one-off"),
  });

  const { data: habitTasks = [], isLoading: habitLoading, error: habitError } = useQuery<Task[]>({
    queryKey: queryKeys.tasksHabit,
    queryFn: () => listTasks(undefined, "habit"),
  });

  const isLoading = todoLoading || habitLoading;
  const error = todoError || habitError;

  // Fetch completion history for all habit tasks using a single batch request.
  const weekDates = getRollingWindowDates(weekOffset);
  const allCompletions = useHabitCompletions(habitTasks);

  const confirm = useConfirm();

  // Writes an order into the cache straight away, so rows do not jump back
  // while the request is in flight.
  const applyLocalPositions = useCallback(
    (kind: TaskKind, orderedIds: string[]) => {
      const positions = positionsFor(orderedIds);
      queryClient.setQueryData<Task[]>(taskQueryKey(kind), (old) =>
        old?.map((task) => {
          const position = positions.get(task.id);
          return position === undefined ? task : { ...task, position };
        })
      );
    },
    [queryClient]
  );

  // Mutations
  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      invalidateTasks(queryClient);
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
      invalidateTasks(queryClient);
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
      queryClient.invalidateQueries({ queryKey: queryKeys.habitCompletions });
    },
  });

  const createTodoMutation = useMutation({
    mutationFn: (title: string) => createTask({ title }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasksOneOff });
      setNewTodoTitle("");
    },
  });

  const reorderMutation = useMutation({
    mutationFn: ({ kind, status, orderedIds }: { kind: TaskKind; status: TaskStatus; orderedIds: string[] }) =>
      reorderTasks({ type: kind, status, ordered_ids: orderedIds }),
    onMutate: ({ kind, orderedIds }) => {
      applyLocalPositions(kind, orderedIds);
    },
    // A rejection means the list moved under us (deleted or completed item),
    // so drop the optimistic order and take the server's again.
    onError: () => {
      invalidateTasks(queryClient);
    },
    onSettled: () => {
      invalidateTasks(queryClient);
    },
  });

  const persistOrder = useCallback(
    (kind: TaskKind, status: TaskStatus, orderedIds: string[]) => {
      if (orderedIds.length === 0) return;
      reorderMutation.mutate({ kind, status, orderedIds });
    },
    [reorderMutation]
  );

    // Tasks and habits open in edit modals rather than swapping the page.

  // Two sub-tabs so completed to-dos don't clutter the unchecked view; sorting is shared.
  const openTodos = todoTasks.filter((t) => t.status === "todo");
  const completedTodos = todoTasks.filter((t) => t.status === "done");
  const activeTodos = todoTab === "open" ? openTodos : completedTodos;
  const sortedTodoTasks = sortTasks(activeTodos, todoSort);
  const sortedHabitTasks = sortTasks(habitTasks, habitSort);

  const todoStatus: TaskStatus = todoTab === "open" ? "todo" : "done";
  const displayedTodoIds = sortedTodoTasks.map((task) => task.id);
  const displayedHabitIds = sortedHabitTasks.map((task) => task.id);

  const announceMove = useCallback(
    (movedId: string, orderedIds: string[]) => {
      const moved = [...todoTasks, ...habitTasks].find((task) => task.id === movedId);
      if (!moved) return;
      setReorderMessage(
        t("tasks.reorderAnnounce", {
          title: moved.title,
          position: orderedIds.indexOf(movedId) + 1,
        })
      );
    },
    [habitTasks, t, todoTasks]
  );

  // Dragging only makes sense in manual order, so the first drag adopts the
  // order already on screen instead of the stored ranks.
  const todoDrag = useDragReorder({
    ids: displayedTodoIds,
    onReorder: (orderedIds, movedId) => {
      persistOrder("one-off", todoStatus, orderedIds);
      announceMove(movedId, orderedIds);
    },
    onDragStart: () => {
      if (todoSort === "manual") return;
      applyLocalPositions("one-off", displayedTodoIds);
      setTodoSort("manual");
    },
  });

  const habitDrag = useDragReorder({
    ids: displayedHabitIds,
    onReorder: (orderedIds, movedId) => {
      persistOrder("habit", "todo", orderedIds);
      announceMove(movedId, orderedIds);
    },
    onDragStart: () => {
      if (habitSort === "manual") return;
      applyLocalPositions("habit", displayedHabitIds);
      setHabitSort("manual");
    },
  });

  return (
    <div className="animate-fade-in">
      {/* Page Header */}
      <div className="page-header">
        <div>
          <h1 className="page-heading" style={{ marginBottom: "0.25rem" }}>
            <TasksIcon size={24} />
            {t("tasks.title")}
          </h1>
          <p className="page-subtitle">
            {t("tasks.subtitle")}
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
          {t("tasks.newTask")}
        </button>
      </div>

      {/* Reorder feedback for keyboard and screen reader users */}
      <p className="sr-only" role="status" aria-live="polite">
        {reorderMessage}
      </p>

      {isLoading && (
        <div className="card loading-card">
          <p className="text-secondary">{t("tasks.loading")}</p>
        </div>
      )}
      {error && (
        <div className="card error-card">
          <p className="error-text">
            {t("common.error")}: {(error as Error).message}
          </p>
        </div>
      )}

      {/* ===== TOP: TO-DOS + HABITS (2-COLUMN GRID) ===== */}
      <div className="task-columns">
        {/* ===== TODOS COLUMN ===== */}
        <div className="task-column">
          <div className="task-column-header">
            <div className="flex-between" style={{ gap: "var(--space-sm)", flexWrap: "wrap" }}>
              <h2 className="section-header" style={{ marginBottom: 0 }}>
                📋 {t("tasks.todos")} <span className="badge badge-todo">{activeTodos.length}</span>
              </h2>
              <SortSelect
                value={todoSort}
                onChange={setTodoSort}
                options={TODO_SORTS}
              />
            </div>
          </div>

          {/* Open / Completed sub-tabs */}
          <div className="task-tabs" role="tablist" aria-label={t("tasks.todos")}>
            <button
              type="button"
              role="tab"
              aria-selected={todoTab === "open"}
              className={`task-tab${todoTab === "open" ? " active" : ""}`}
              onClick={() => setTodoTab("open")}
            >
              {t("tasks.openTodos")}
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={todoTab === "completed"}
              className={`task-tab${todoTab === "completed" ? " active" : ""}`}
              onClick={() => setTodoTab("completed")}
            >
              {t("tasks.completedTodos")}
            </button>
          </div>

          <div className="task-column-body">
            {/* Quick-add for todos, a new item always starts unchecked, so it
                only belongs on the "Open" sub-tab */}
            {todoTab === "open" && (
              <QuickAddForm
                value={newTodoTitle}
                onChange={setNewTodoTitle}
                onSubmit={() => {
                  if (newTodoTitle.trim()) createTodoMutation.mutate(newTodoTitle.trim());
                }}
                placeholder={t("tasks.quickAddTodo")}
                isPending={createTodoMutation.isPending}
              />
            )}

            {activeTodos.length === 0 && !isLoading && (
              <div className="empty-state">
                <div className="empty-state-icon">{todoTab === "open" ? "📋" : "✅"}</div>
                <p className="empty-state-text">
                  {todoTab === "open" ? t("tasks.noTodos") : t("tasks.noCompletedTodos")}
                </p>
              </div>
            )}

            <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
              {sortedTodoTasks.map((task, index) => (
                <Fragment key={task.id}>
                  {todoDrag.draggingId !== null && todoDrag.landingIndex === index && (
                    <div className="reorder-slot" aria-hidden="true" />
                  )}
                  <TaskCard
                    variant="todo"
                    title={task.title}
                    done={task.status === "done"}
                    onToggle={() => toggleMutation.mutate({ id: task.id, status: task.status })}
                    onDelete={async () => {
                      const ok = await confirm({
                        title: t("tasks.deleteTaskTitle"),
                        confirmLabel: t("tasks.delete"),
                        confirmVariant: "danger",
                      });
                      if (ok) {
                        deleteMutation.mutate(task.id);
                      }
                    }}
                    onTitleClick={() => {
                      setSelectedTaskId(task.id);
                    }}
                    animationDelay={`${index * 30}ms`}
                    rowRef={todoDrag.registerItem(task.id)}
                    dragHandle={todoDrag.handleProps(task.id)}
                    dragging={todoDrag.draggingId === task.id}
                  />
                </Fragment>
              ))}
              {todoDrag.draggingId !== null && todoDrag.landingIndex === sortedTodoTasks.length && (
                <div className="reorder-slot" aria-hidden="true" />
              )}
            </div>
          </div>
        </div>

        {/* ===== HABITS COLUMN ===== */}
        <div className="task-column">
          <div className="task-column-header">
            <div className="flex-between" style={{ gap: "var(--space-sm)", flexWrap: "wrap" }}>
              <h2 className="section-header" style={{ marginBottom: 0 }}>
                🔄 {t("tasks.habits")} <span className="badge badge-habit">{habitTasks.length}</span>
              </h2>
              <SortSelect
                value={habitSort}
                onChange={setHabitSort}
                options={HABIT_SORTS}
              />
            </div>
          </div>

          <div className="task-column-body">
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
                aria-label={t("a11y.previousWeek")}
              >
                &larr; {t("week.prevWeek")}
              </button>
              <span
                className="text-sm text-bold text-secondary"
                style={{ flex: 1, textAlign: "center", whiteSpace: "nowrap", minWidth: 0, overflow: "hidden", textOverflow: "ellipsis" }}
              >
                {weekOffset === 0
                  ? t("week.last7Days")
                  : formatWeekRange(weekDates)}
              </span>
              <button
                className="btn btn-ghost"
                style={{ padding: "0.2rem 0.5rem", fontSize: "var(--font-size-sm)" }}
                onClick={() => handleWeekOffsetChange(weekOffset + 1)}
                disabled={weekOffset >= 0}
                aria-label={t("a11y.nextWeek")}
              >
                {t("week.nextWeek")} &rarr;
              </button>
            </div>

            {habitTasks.length === 0 && !isLoading && (
              <div className="empty-state">
                <div className="empty-state-icon">🔄</div>
                <p className="empty-state-text">{t("tasks.noHabits")}</p>
              </div>
            )}

            <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
              {sortedHabitTasks.map((task, index) => {
                const taskCompletions = allCompletions?.[task.id] ?? [];
                const { current, longest } = computeStreaks(taskCompletions, task.recurrence_days);

                return (
                  <Fragment key={task.id}>
                    {habitDrag.draggingId !== null && habitDrag.landingIndex === index && (
                      <div className="reorder-slot" aria-hidden="true" />
                    )}
                    <div
                      ref={habitDrag.registerItem(task.id)}
                      className={`card card-habit ${habitDrag.draggingId === task.id ? "card-dragging" : ""}`}
                      style={{ animationDelay: `${index * 30}ms` }}
                    >
                      {/* Row 1: Handle + Title + Streak + Delete (hover reveal) */}
                      <div className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "0.35rem" }}>
                        <button
                          type="button"
                          className="reorder-handle"
                          {...habitDrag.handleProps(task.id)}
                          aria-label={t("tasks.reorderAria", { title: task.title })}
                          title={t("tasks.reorderAria", { title: task.title })}
                        >
                          ⠿
                        </button>
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
                          }}
                        >
                          {task.title}
                        </span>
                        <StreakBadge current={current} longest={longest} />
                        <button
                          className="btn btn-danger btn-sm habit-delete"
                          onClick={async (e) => {
                            e.stopPropagation();
                            const ok = await confirm({
                              title: t("tasks.deleteHabitTitle"),
                              confirmLabel: t("tasks.delete"),
                              confirmVariant: "danger",
                            });
                            if (ok) {
                              deleteMutation.mutate(task.id);
                            }
                          }}
                          title={t("tasks.deleteHabitAria")}
                          aria-label={t("tasks.deleteHabitAria")}
                        >
                          ✕
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
                  </Fragment>
                );
              })}
              {habitDrag.draggingId !== null && habitDrag.landingIndex === sortedHabitTasks.length && (
                <div className="reorder-slot" aria-hidden="true" />
              )}
            </div>
          </div>
        </div>
      </div>

      {/* New task / habit modal */}
      {showCreate && (
        <TaskCreate
          onCreated={() => {
            invalidateTasks(queryClient);
            setShowCreate(false);
          }}
          onCancel={() => setShowCreate(false)}
        />
      )}

      {/* Edit task / habit modal, opens directly in edit mode on card click */}
      {selectedTaskId && (
        <TaskEditModal
          key={selectedTaskId}
          taskId={selectedTaskId}
          onClose={() => setSelectedTaskId(null)}
          onUpdated={() => {
            invalidateTasks(queryClient);
          }}
          onDeleted={() => {
            invalidateTasks(queryClient);
            setSelectedTaskId(null);
          }}
        />
      )}
    </div>
  );
}