import { useQuery } from "@tanstack/react-query";
import { getHealth, type HealthResponse } from "./api/client";

function App() {
  const { data, isLoading, error } = useQuery<HealthResponse>({
    queryKey: ["health"],
    queryFn: getHealth,
    refetchInterval: 30_000, // Poll every 30s
  });

  return (
    <main style={{ padding: "2rem", fontFamily: "system-ui, sans-serif" }}>
      <h1>🍮 PudimProductivity</h1>

      <section>
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
    </main>
  );
}

export default App;
