import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useConfirm } from "./useConfirm";
import {
  deleteTask,
  updateTask,
  createTask,
  type Task,
  type TaskStatus,
} from "../api/tasks";
import {
  deleteTaskList,
  getTaskList,
  listTasksByListID,
  updateTaskList,
} from "../api/taskLists";
import QuickAddForm from "./QuickAddForm";
import TaskCard from "./TaskCard";
import SortSelect from "./SortSelect";
import { usePersistedSort } from "../hooks/usePersistedSort";
import { useI18n } from "../i18n";
import { sortTasks } from "../utils/sort";
import {
  playTodoCompletionSound,
  playTodoUncompletionSound,
} from "../utils/sounds";

interface ListDetailPanelProps {
  listId: string;
  onListDeleted: () => void;
}

export default function ListDetailPanel({ listId, onListDeleted }: ListDetailPanelProps) {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [newTitle, setNewTitle] = useState("");
  const [editingName, setEditingName] = useState(false);
  const [editName, setEditName] = useState("");
  const [listSort, setListSort] = usePersistedSort("taskSort.list", "created-desc");
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
      } else {
        playTodoUncompletionSound();
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

  const handleQuickAdd = () => {
    if (!newTitle.trim()) return;
    createMutation.mutate(newTitle.trim());
  };

  if (listLoading) {
    return (
      <div className="card loading-card" style={{ padding: "var(--space-lg)" }}>
        <p className="text-secondary">{t("lists.loading")}</p>
      </div>
    );
  }

  if (!taskList) {
    return (
      <div className="card error-card" style={{ borderWidth: "3px 0 0" }}>
        <p className="error-text">{t("lists.notFound")}</p>
      </div>
    );
  }

  const doneCount = tasks.filter((t) => t.status === "done").length;
  const progress = tasks.length > 0 ? Math.round((doneCount / tasks.length) * 100) : 0;

  const sortedTasks = sortTasks(tasks, listSort);

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
            {t("common.save")}
          </button>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setEditingName(false)}>
            {t("common.cancel")}
          </button>
        </form>
      ) : (
        <div style={{ marginBottom: "var(--space-md)" }}>
          <div className="flex-center" style={{ gap: "var(--space-sm)" }}>
            <span style={{ fontSize: "1.2rem" }}>📁</span>
            <h3 className="flex-1" style={{ fontSize: "var(--font-size-lg)", fontWeight: 700 }}>{taskList.name}</h3>
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
                  title: t("lists.deleteConfirm", { name: taskList.name }),
                  message: t("lists.deleteMessage"),
                  confirmLabel: t("tasks.delete"),
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
            <div className="flex-center" style={{ gap: "var(--space-sm)", marginTop: "var(--space-sm)" }}>
              <span className="badge badge-done">{t("lists.progressDone", { done: doneCount, total: tasks.length })}</span>
              <div style={{ flex: 1, maxWidth: "150px" }}>
                <div className="progress-bar">
                  <div className="progress-bar-fill" style={{ width: `${progress}%` }} />
                </div>
              </div>
              <SortSelect
                value={listSort}
                onChange={setListSort}
                options={["created-desc", "created-asc", "alpha-asc", "alpha-desc"]}
              />
            </div>
          )}
        </div>
      )}

      {/* Quick-add form */}
      <QuickAddForm
        value={newTitle}
        onChange={setNewTitle}
        onSubmit={handleQuickAdd}
        placeholder={t("lists.quickAdd")}
        isPending={createMutation.isPending}
      />

      {tasksLoading && (
        <div className="card loading-card" style={{ padding: "var(--space-lg)" }}>
          <p className="text-secondary">{t("tasks.loading")}</p>
        </div>
      )}

      {!tasksLoading && tasks.length === 0 && (
        <div className="empty-state" style={{ padding: "var(--space-md)" }}>
          <div className="empty-state-icon" style={{ fontSize: "1.5rem" }}>📝</div>
          <p className="empty-state-text">{t("lists.emptyTasks")}</p>
        </div>
      )}

      {tasks.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: "0.4rem" }}>
          {sortedTasks.map((task) => (
            <TaskCard
              key={task.id}
              variant="list"
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
                  deleteTaskMutation.mutate(task.id);
                }
              }}
              compact
            />
          ))}
        </div>
      )}
    </div>
  );
}