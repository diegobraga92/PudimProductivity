import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useState } from "react";
import { getHealth, type HealthResponse } from "./api/client";
import { AlarmProvider } from "./components/AlarmProvider";
import { AlarmToast } from "./components/AlarmToast";
import { ConfirmProvider } from "./components/ConfirmProvider";
import { ToastProvider } from "./components/ToastProvider";
import { ToastStack } from "./components/ToastStack";
import { useAlarm } from "./components/useAlarm";
import { useAlarmNotifier } from "./hooks/useAlarmNotifier";
import { useErrorReporter } from "./hooks/useErrorReporter";
import { useLiveUpdates } from "./hooks/useLiveUpdates";
import { useTaskNotifier } from "./hooks/useTaskNotifier";
import Dashboard from "./pages/Dashboard";
import "./styles.css";

// Secondary pages are code-split (React.lazy) so the initial bundle only
// contains the app shell + Dashboard (the landing page). TaskList, Planner,
// Pomodoro and Soundscape chunks load on demand. Run `npm run build:analyze`
// to inspect chunk sizes.
const TaskList = lazy(() => import("./pages/TaskList"));
const Planner = lazy(() => import("./pages/Planner"));
const Pomodoro = lazy(() => import("./pages/Pomodoro"));
const Soundscape = lazy(() => import("./pages/Soundscape"));
const RecipeList = lazy(() => import("./pages/RecipeList"));
const RecipeDetail = lazy(() => import("./pages/RecipeDetail"));
const BookList = lazy(() => import("./pages/BookList"));
const MealPlanList = lazy(() => import("./pages/MealPlanList"));
const MealPlanDetail = lazy(() => import("./pages/MealPlanDetail"));
const DailyPlan = lazy(() => import("./pages/DailyPlan"));
const Insights = lazy(() => import("./pages/Insights"));

type Page = "dashboard" | "tasks" | "planner" | "pomodoro" | "soundscape" | "recipes" | "booktrack" | "mealplan" | "plan" | "insights" | "health";

const pageFallback = (
  <div className="container" style={{ paddingTop: "var(--space-xl)", textAlign: "center", color: "var(--color-text-secondary)" }}>
    Loading…
  </div>
);

function HeaderBadge() {
  const { activeAlarms } = useAlarm();
  if (activeAlarms.length === 0) return null;
  return (
    <span className="alarm-badge" title={`${activeAlarms.length} active alarm${activeAlarms.length > 1 ? "s" : ""}`}>
      🔔 {activeAlarms.length}
    </span>
  );
}

function AppInner() {
  const [page, setPage] = useState<Page>("dashboard");
  const [selectedRecipeId, setSelectedRecipeId] = useState<string | null>(null);
  const [mealPlanId, setMealPlanId] = useState<string | null>(null);

  // Polls scheduled habit tasks and fires sound + in-app toast alarms
  useAlarmNotifier();

  // Real-time task updates from the backend WebSocket stream (Phase 2). This
  // replaces polling: task changes made on any client appear here immediately.
  useLiveUpdates();

  // In-app toast notifications for task events (Phase 3 — the "push" channel
  // on the web, delivered over the same WebSocket stream).
  useTaskNotifier();

  // Report client-side JS errors to the backend beacon (POST /api/v1/errors).
  useErrorReporter();

  const { data: healthData } = useQuery<HealthResponse>({
    queryKey: ["health"],
    queryFn: getHealth,
    refetchInterval: 30_000,
  });

  const handleNavigate = (view: string) => {
    if (view === "tasks") {
      setPage("tasks");
    } else if (view === "dashboard") {
      setPage("dashboard");
    }
  };

  const isBackendOk = healthData?.status === "ok" && healthData?.db === "connected";

  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column" }}>
      {/* ===== Habitica-Inspired Header ===== */}
      <header
        style={{
          background: "linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-dark) 100%)",
          color: "white",
          padding: "0 var(--space-lg)",
          boxShadow: "0 2px 12px rgba(108, 92, 231, 0.3)",
          position: "sticky",
          top: 0,
          zIndex: 100,
        }}
      >
        <div
          className="container"
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            height: "56px",
          }}
        >
          {/* Logo + Title */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "var(--space-sm)",
              cursor: "pointer",
            }}
            onClick={() => setPage("dashboard")}
          >
            <span style={{ fontSize: "1.5rem", lineHeight: 1 }}>🍮</span>
            <span style={{ fontWeight: 700, fontSize: "var(--font-size-lg)", letterSpacing: "-0.5px" }}>
              Pudim
            </span>
          </div>

          {/* Navigation Tabs */}
          <nav style={{ display: "flex", gap: "0.25rem", height: "100%", alignItems: "stretch" }}>
            {[
              { id: "dashboard" as Page, label: "Dashboard", icon: "🏠" },
              { id: "tasks" as Page, label: "Tasks", icon: "📋" },
              { id: "planner" as Page, label: "Planner", icon: "📅" },
              { id: "pomodoro" as Page, label: "Timer", icon: "🍅" },
              { id: "soundscape" as Page, label: "Sounds", icon: "🎵" },
              { id: "recipes" as Page, label: "Recipes", icon: "🍳" },
              { id: "booktrack" as Page, label: "Books", icon: "📚" },
              { id: "mealplan" as Page, label: "Meals", icon: "🗓" },
              { id: "plan" as Page, label: "Daily Plan", icon: "🤖" },
              { id: "insights" as Page, label: "Insights", icon: "🧠" },
              { id: "health" as Page, label: "Status", icon: "💚" },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setPage(tab.id)}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.35rem",
                  padding: "0 1rem",
                  background: page === tab.id ? "rgba(255,255,255,0.2)" : "transparent",
                  color: "white",
                  border: "none",
                  borderBottom: page === tab.id ? "3px solid white" : "3px solid transparent",
                  borderRadius: "0",
                  cursor: "pointer",
                  fontFamily: "var(--font-family)",
                  fontSize: "var(--font-size-sm)",
                  fontWeight: page === tab.id ? 600 : 400,
                  transition: "all var(--transition-fast)",
                  opacity: page === tab.id ? 1 : 0.8,
                }}
              >
                <span>{tab.icon}</span>
                <span>{tab.label}</span>
              </button>
            ))}
          </nav>

          {/* Status Badge + Alarm Badge */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.4rem",
              fontSize: "var(--font-size-xs)",
              opacity: 0.8,
            }}
          >
            <HeaderBadge />
            <span
              style={{
                width: "8px",
                height: "8px",
                borderRadius: "50%",
                background: isBackendOk ? "#00b894" : "#d63031",
                display: "inline-block",
              }}
            />
            <span>{isBackendOk ? "Connected" : "Offline"}</span>
          </div>
        </div>
      </header>

      {/* ===== Main Content ===== */}
      <main style={{ flex: 1, padding: "var(--space-lg) 0" }}>
        <div className="container">
          <Suspense fallback={pageFallback}>
            {page === "dashboard" && <Dashboard onNavigate={handleNavigate} />}

            {page === "tasks" && <TaskList />}

            {page === "planner" && <Planner onNavigate={handleNavigate} />}

            {page === "pomodoro" && <Pomodoro />}

            {page === "soundscape" && <Soundscape />}

            {page === "recipes" &&
              (selectedRecipeId ? (
                <RecipeDetail recipeId={selectedRecipeId} onBack={() => setSelectedRecipeId(null)} />
              ) : (
                <RecipeList onOpen={(r) => setSelectedRecipeId(r.id)} />
              ))}

            {page === "booktrack" && <BookList />}

            {page === "mealplan" &&
              (mealPlanId !== null ? (
                <MealPlanDetail planId={mealPlanId} onBack={() => setMealPlanId(null)} />
              ) : (
                <MealPlanList onOpen={(id) => setMealPlanId(id)} />
              ))}

            {page === "plan" && <DailyPlan />}

            {page === "insights" && <Insights />}
          </Suspense>

          {page === "health" && (
            <div className="animate-fade-in">
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "var(--space-sm)",
                  marginBottom: "var(--space-lg)",
                }}
              >
                <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>
                  💚 Backend Health
                </h2>
                <span
                  className={`badge ${isBackendOk ? "badge-done" : "badge-habit"}`}
                >
                  {isBackendOk ? "Healthy" : "Issues"}
                </span>
              </div>

              {!healthData && (
                <div className="card" style={{ textAlign: "center", padding: "var(--space-xl)" }}>
                  <p style={{ color: "var(--color-text-secondary)" }}>
                    Checking backend status...
                  </p>
                </div>
              )}

              {healthData && (
                <div
                  style={{
                    display: "grid",
                    gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
                    gap: "var(--space-md)",
                  }}
                >
                  <div className="stat-card">
                    <div
                      className="stat-card-value"
                      style={{
                        color: healthData.status === "ok" ? "var(--color-done)" : "var(--color-warning)",
                      }}
                    >
                      {healthData.status}
                    </div>
                    <div className="stat-card-label">Status</div>
                  </div>
                  <div className="stat-card">
                    <div className="stat-card-value" style={{ fontSize: "var(--font-size-lg)" }}>
                      v{healthData.version}
                    </div>
                    <div className="stat-card-label">Version</div>
                  </div>
                  <div className="stat-card">
                    <div
                      className="stat-card-value"
                      style={{
                        color: healthData.db === "connected" ? "var(--color-done)" : "var(--color-danger)",
                        fontSize: "var(--font-size-sm)",
                      }}
                    >
                      {healthData.db}
                    </div>
                    <div className="stat-card-label">Database</div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </main>

      {/* ===== Alarm + Notification Toast Stacks ===== */}
      <AlarmToast />
      <ToastStack />
    </div>
  );
}

function App() {
  return (
    <AlarmProvider>
      <ToastProvider>
        <ConfirmProvider>
          <AppInner />
        </ConfirmProvider>
      </ToastProvider>
    </AlarmProvider>
  );
}

export default App;
