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

  if (listLoading) return <p>Loading list...</p>;
  if (!taskList) return <p>List not found</p>;

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
        <button
          onClick={onBack}
          style={{
            padding: "0.3rem 0.6rem",
            background: "transparent",
            border: "1px solid #ccc",
            borderRadius: "4px",
            cursor: "pointer",
          }}
        >
          &larr; Back
        </button>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            onClick={() => {
              setEditName(taskList.name);
              setEditingName(true);
            }}
            style={{
              padding: "0.3rem 0.6rem",
              background: "#ffc107",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Rename
          </button>
          <button
            onClick={() => {
              if (confirm(`Delete list "${taskList.name}" and all its tasks?`)) {
                deleteListMutation.mutate();
              }
            }}
            style={{
              padding: "0.3rem 0.6rem",
              background: "#dc3545",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Delete List
          </button>
        </div>
      </div>

      {editingName ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (editName.trim()) updateListMutation.mutate(editName.trim());
          }}
          style={{ marginBottom: "1rem", display: "flex", gap: "0.5rem" }}
        >
          <input
            type="text"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            style={{
              flex: 1,
              padding: "0.5rem",
              border: "1px solid #ccc",
              borderRadius: "4px",
              fontSize: "1rem",
            }}
            autoFocus
          />
          <button
            type="submit"
            style={{
              padding: "0.5rem 1rem",
              background: "#007bff",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Save
          </button>
          <button
            type="button"
            onClick={() => setEditingName(false)}
            style={{
              padding: "0.5rem 1rem",
              background: "#6c757d",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
        </form>
      ) : (
        <h2 style={{ margin: "0 0 1rem 0" }}>{taskList.name}</h2>
      )}

      {/* Quick-add form */}
      <form
        onSubmit={handleQuickAdd}
        style={{
          display: "flex",
          gap: "0.5rem",
          marginBottom: "1rem",
        }}
      >
        <input
          type="text"
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          placeholder="Add a task to this list..."
          style={{
            flex: 1,
            padding: "0.5rem",
            border: "1px solid #ccc",
            borderRadius: "4px",
            fontSize: "1rem",
          }}
        />
        <button
          type="submit"
          disabled={createMutation.isPending || !newTitle.trim()}
          style={{
            padding: "0.5rem 1rem",
            background:
              createMutation.isPending || !newTitle.trim()
                ? "#6c757d"
                : "#007bff",
            color: "white",
            border: "none",
            borderRadius: "4px",
            cursor:
              createMutation.isPending || !newTitle.trim()
                ? "not-allowed"
                : "pointer",
          }}
        >
          Add
        </button>
      </form>

      {tasksLoading && <p>Loading tasks...</p>}

      {!tasksLoading && tasks.length === 0 && (
        <p style={{ color: "#666" }}>No tasks in this list yet.</p>
      )}

      {tasks.length > 0 && (
        <ul style={{ listStyle: "none", padding: 0 }}>
          {tasks.map((task) => (
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
                  textDecoration:
                    task.status === "done" ? "line-through" : "none",
                  color: task.status === "done" ? "#888" : "inherit",
                }}
              >
                {task.title}
              </span>
              <button
                onClick={() => {
                  if (confirm("Delete this task?")) {
                    deleteTaskMutation.mutate(task.id);
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
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
