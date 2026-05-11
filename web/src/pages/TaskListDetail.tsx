import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  getTaskList,
  listTasksByListID,
  deleteTaskList,
  updateTaskList,
} from "../api/taskLists";
import {
  createTask,
  deleteTask,
  updateTask,
  type Task,
  type TaskStatus,
} from "../api/tasks";
import Checkbox from "../components/Checkbox";

interface TaskListDetailProps {
  listId: string;
  onBack: () => void;
  onDeleted: () => void;
}

export default function TaskListDetail({
  listId,
  onBack,
  onDeleted,
}: TaskListDetailProps) {
  const queryClient = useQueryClient();
  const [newTitle, setNewTitle] = useState("");
  const [editingName, setEditingName] = useState(false);
  const [editName, setEditName] = useState("");

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
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) =>
      updateTask(id, { status: status === "done" ? "todo" : "done" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskListTasks", listId] });
    },
  });

  const deleteListMutation = useMutation({
    mutationFn: () => deleteTaskList(listId),
    onSuccess: onDeleted,
  });

  const updateListMutation = useMutation({
    mutationFn: (name: string) => updateTaskList(listId, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskList", listId] });
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
      <div className="card" style={{ textAlign: "center", padding: "var(--space-xl)" }}>
        <p style={{ color: "var(--color-text-secondary)" }}>Loading list...</p>
      </div>
    );
  }

  if (!taskList) {
    return (
      <div className="card" style={{ borderLeft: "3px solid var(--color-danger)" }}>
        <p style={{ color: "var(--color-danger)" }}>List not found</p>
        <button className="btn btn-ghost mt-sm" onClick={onBack}>
          &larr; Back
        </button>
      </div>
    );
  }

  const doneCount = tasks.filter((t) => t.status === "done").length;
  const progress = tasks.length > 0 ? Math.round((doneCount / tasks.length) * 100) : 0;

  return (
    <div className="animate-fade-in" style={{ maxWidth: "600px" }}>
      {/* Header */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "var(--space-md)",
        }}
      >
        <button className="btn btn-ghost" onClick={onBack}>
          &larr; Back
        </button>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            className="btn btn-ghost"
            onClick={() => {
              setEditName(taskList.name);
              setEditingName(true);
            }}
          >
            ✏️ Rename
          </button>
          <button
            className="btn btn-danger"
            onClick={() => {
              if (confirm(`Delete list "${taskList.name}" and all its tasks?`)) {
                deleteListMutation.mutate();
              }
            }}
          >
            🗑 Delete List
          </button>
        </div>
      </div>

      {/* Title / Edit */}
      {editingName ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (editName.trim()) updateListMutation.mutate(editName.trim());
          }}
          style={{ marginBottom: "var(--space-lg)", display: "flex", gap: "0.5rem" }}
        >
          <input
            type="text"
            className="input"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            autoFocus
          />
          <button type="submit" className="btn btn-primary">
            Save
          </button>
          <button type="button" className="btn btn-ghost" onClick={() => setEditingName(false)}>
            Cancel
          </button>
        </form>
      ) : (
        <div style={{ marginBottom: "var(--space-lg)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)" }}>
            <span style={{ fontSize: "1.5rem" }}>📁</span>
            <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>{taskList.name}</h2>
          </div>
          {tasks.length > 0 && (
            <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)", marginTop: "var(--space-sm)" }}>
              <span className="badge badge-done">{doneCount}/{tasks.length} done</span>
              <div style={{ flex: 1, maxWidth: "200px" }}>
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
        style={{ display: "flex", gap: "0.5rem", marginBottom: "var(--space-lg)" }}
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
          style={{ background: "var(--color-list)" }}
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
        <div className="empty-state">
          <div className="empty-state-icon">📝</div>
          <p className="empty-state-text">No tasks in this list yet.</p>
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
                padding: "0.6rem 0.8rem",
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
                  fontSize: "var(--font-size-base)",
                }}
              >
                {task.title}
              </span>
              <button
                className="btn btn-danger btn-sm"
                onClick={() => {
                  if (confirm("Delete this task?")) {
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
