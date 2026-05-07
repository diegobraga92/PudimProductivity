import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  listTasks,
  deleteTask,
  type Task,
  type TaskStatus,
  type TaskPriority,
} from "../api/tasks";
import TaskCreate from "./TaskCreate";
import TaskDetail from "./TaskDetail";

type View = "list" | "create" | "detail";

export default function TaskList() {
  const queryClient = useQueryClient();
  const [view, setView] = useState<View>("list");
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<TaskStatus | "">("");
  const [priorityFilter, setPriorityFilter] = useState<TaskPriority | "">("");

  const { data: tasks, isLoading, error } = useQuery<Task[]>({
    queryKey: ["tasks", statusFilter, priorityFilter],
    queryFn: () =>
      listTasks(
        statusFilter ? (statusFilter as TaskStatus) : undefined,
        priorityFilter ? (priorityFilter as TaskPriority) : undefined
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
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
    <div style={{ maxWidth: "800px", margin: "0 auto", padding: "1rem" }}>
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

      {/* Filters */}
      <div
        style={{
          display: "flex",
          gap: "0.5rem",
          marginBottom: "1rem",
        }}
      >
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as TaskStatus | "")}
          style={{ padding: "0.3rem" }}
        >
          <option value="">All Statuses</option>
          <option value="todo">Todo</option>
          <option value="in_progress">In Progress</option>
          <option value="done">Done</option>
        </select>

        <select
          value={priorityFilter}
          onChange={(e) =>
            setPriorityFilter(e.target.value as TaskPriority | "")
          }
          style={{ padding: "0.3rem" }}
        >
          <option value="">All Priorities</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
        </select>
      </div>

      {isLoading && <p>Loading tasks...</p>}

      {error && (
        <p style={{ color: "red" }}>
          Error: {(error as Error).message}
        </p>
      )}

      {tasks && tasks.length === 0 && (
        <p style={{ color: "#666" }}>No tasks found. Create one!</p>
      )}

      {tasks && tasks.length > 0 && (
        <ul style={{ listStyle: "none", padding: 0 }}>
          {tasks.map((task) => (
            <li
              key={task.id}
              style={{
                padding: "0.75rem",
                marginBottom: "0.5rem",
                border: "1px solid #ddd",
                borderRadius: "6px",
                cursor: "pointer",
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                background: task.status === "done" ? "#f0f0f0" : "white",
              }}
              onClick={() => {
                setSelectedTaskId(task.id);
                setView("detail");
              }}
            >
              <div>
                <strong
                  style={{
                    textDecoration:
                      task.status === "done" ? "line-through" : "none",
                  }}
                >
                  {task.title}
                </strong>
                <div style={{ fontSize: "0.85rem", color: "#666", marginTop: "0.25rem" }}>
                  <span
                    style={{
                      display: "inline-block",
                      padding: "0.1rem 0.4rem",
                      borderRadius: "3px",
                      fontSize: "0.75rem",
                      fontWeight: "bold",
                      textTransform: "uppercase",
                      background:
                        task.priority === "high"
                          ? "#ffebee"
                          : task.priority === "medium"
                          ? "#fff3e0"
                          : "#e8f5e9",
                      color:
                        task.priority === "high"
                          ? "#c62828"
                          : task.priority === "medium"
                          ? "#ef6c00"
                          : "#2e7d32",
                      marginRight: "0.5rem",
                    }}
                  >
                    {task.priority}
                  </span>
                  <span>{task.status.replace("_", " ")}</span>
                  {task.due_date && (
                    <span style={{ marginLeft: "0.5rem" }}>
                      Due: {new Date(task.due_date).toLocaleDateString()}
                    </span>
                  )}
                </div>
              </div>
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
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
