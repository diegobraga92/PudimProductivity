import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { getHealth, type HealthResponse } from "./api/client";
import TaskList from "./pages/TaskList";

type Page = "health" | "tasks";

function App() {
  const [page, setPage] = useState<Page>("tasks");

  const { data, isLoading, error } = useQuery<HealthResponse>({
    queryKey: ["health"],
    queryFn: getHealth,
    refetchInterval: 30_000,
  });

  return (
    <main style={{ fontFamily: "system-ui, sans-serif" }}>
      <header
        style={{
          padding: "1rem 2rem",
          borderBottom: "1px solid #ddd",
          display: "flex",
          alignItems: "center",
          gap: "2rem",
          background: "#f8f9fa",
        }}
      >
        <h1 style={{ margin: 0, fontSize: "1.5rem" }}>🍮 PudimProductivity</h1>
        <nav style={{ display: "flex", gap: "1rem" }}>
          <button
            onClick={() => setPage("tasks")}
            style={{
              padding: "0.4rem 0.8rem",
              background: page === "tasks" ? "#007bff" : "transparent",
              color: page === "tasks" ? "white" : "#333",
              border: page === "tasks" ? "none" : "1px solid #ccc",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Tasks
          </button>
          <button
            onClick={() => setPage("health")}
            style={{
              padding: "0.4rem 0.8rem",
              background: page === "health" ? "#007bff" : "transparent",
              color: page === "health" ? "white" : "#333",
              border: page === "health" ? "none" : "1px solid #ccc",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Health
          </button>
        </nav>
      </header>

      {page === "tasks" && <TaskList />}

      {page === "health" && (
        <section style={{ padding: "2rem" }}>
          <h2>Backend Health</h2>

          {isLoading && <p>Checking backend status...</p>}

          {error && (
            <p style={{ color: "red" }}>
              ❌ Failed to reach backend: {(error as Error).message}
            </p>
          )}

          {data && (
            <div>
              <p>
                Status:{" "}
                <strong
                  style={{
                    color: data.status === "ok" ? "green" : "orange",
                  }}
                >
                  {data.status}
                </strong>
              </p>
              <p>Version: {data.version}</p>
              <p>
                Database:{" "}
                <strong
                  style={{
                    color: data.db === "connected" ? "green" : "red",
                  }}
                >
                  {data.db}
                </strong>
              </p>
            </div>
          )}
        </section>
      )}
    </main>
  );
}

export default App;
