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
import TaskCreate from "./TaskCreate";
import TaskEditModal from "../components/TaskEditModal";
import QuickAddForm from "../components/QuickAddForm";
import TaskCard from "../components/TaskCard";
import WeekHeatmap from "../components/WeekHeatmap";
import StreakBadge from "../components/StreakBadge";
import SortSelect from "../components/SortSelect";
import { usePersistedSort } from "../hooks/usePersistedSort";
import { useHabitCompletions } from "../hooks/useHabitCompletions";
import { useI18n } from "../i18n";
import { computeStreaks } from "../utils/streaks";
import { getRollingWindowDates, formatWeekRange } from "../utils/dates";
import { sortTasks } from "../utils/sort";
import {
  playHabitCompletionSound,
  playTodoCompletionSound,
  playHabitUncompletionSound,
  playTodoUncompletionSound,
} from "../utils/sounds";

export default function TaskList() {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [showCreate, setShowCreate] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [weekOffset, setWeekOffset] = useState(0);
  const [todoSort, setTodoSort] = usePersistedSort("taskSort.todos", "created-desc");
  const [habitSort, setHabitSort] = usePersistedSort("taskSort.habits", "created-desc");
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

  // Tasks and habits now open in edit modals (rendered at the bottom of this
  // component) instead of swapping the whole page.

  // Sort tasks based on selected sort options
  const sortedTodoTasks = sortTasks(todoTasks, todoSort);
  const sortedHabitTasks = sortTasks(habitTasks, habitSort);

  return (
    <div className="animate-fade-in">
      {/* Page Header */}
      <div className="page-header">
        <div>
          <h1 className="page-heading" style={{ marginBottom: "0.25rem" }}>
            📋 {t("tasks.title")}
          </h1>
          <p className="page-subtitle">
            {t("tasks.subtitle")}
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
          {t("tasks.newTask")}
        </button>
      </div>

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
            <h2 className="section-header" style={{ marginBottom: 0 }}>
              📋 {t("tasks.todos")} <span className="badge badge-todo">{todoTasks.length}</span>
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
            placeholder={t("tasks.quickAddTodo")}
            isPending={createTodoMutation.isPending}
          />

          {todoTasks.length === 0 && !isLoading && (
            <div className="empty-state">
              <div className="empty-state-icon">📋</div>
              <p className="empty-state-text">{t("tasks.noTodos")}</p>
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
              />
            ))}
          </div>
        </div>

        {/* ===== HABITS COLUMN ===== */}
        <div className="task-column">
          <div className="task-column-header">
            <h2 className="section-header" style={{ marginBottom: 0 }}>
              🔄 {t("tasks.habits")} <span className="badge badge-habit">{habitTasks.length}</span>
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
              aria-label={t("a11y.previousWeek")}
            >
              &larr; {t("week.prevWeek")}
            </button>
            <span
              className="text-sm text-bold text-secondary"
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

      {/* New task / habit modal */}
      {showCreate && (
        <TaskCreate
          onCreated={() => {
            queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
            queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
            setShowCreate(false);
          }}
          onCancel={() => setShowCreate(false)}
        />
      )}

      {/* Edit task / habit modal — opens directly in edit mode on card click */}
      {selectedTaskId && (
        <TaskEditModal
          key={selectedTaskId}
          taskId={selectedTaskId}
          onClose={() => setSelectedTaskId(null)}
          onUpdated={() => {
            queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
            queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
          }}
          onDeleted={() => {
            queryClient.invalidateQueries({ queryKey: ["tasks", "one-off"] });
            queryClient.invalidateQueries({ queryKey: ["tasks", "habit"] });
            setSelectedTaskId(null);
          }}
        />
      )}
    </div>
  );
}