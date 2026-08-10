import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useEffect, useState } from "react";
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

const NAV_ITEMS: { id: Page; label: string; icon: string }[] = [
  { id: "dashboard", label: "Dashboard", icon: "🏠" },
  { id: "tasks", label: "Tasks", icon: "📋" },
  { id: "planner", label: "Planner", icon: "📅" },
  { id: "pomodoro", label: "Timer", icon: "🍅" },
  { id: "soundscape", label: "Sounds", icon: "🎵" },
  { id: "recipes", label: "Recipes", icon: "🍳" },
  { id: "booktrack", label: "Books", icon: "📚" },
  { id: "mealplan", label: "Meals", icon: "🗓" },
  { id: "plan", label: "Daily Plan", icon: "🤖" },
  { id: "insights", label: "Insights", icon: "🧠" },
  { id: "health", label: "Status", icon: "💚" },
];

const pageFallback = (
  <div className="container" style={{ paddingTop: "var(--space-xl)", textAlign: "center", color: "var(--color-text-secondary)" }}>
    Loading…
  </div>
);

/** Theme hook — resolves the system preference, then honors the manual toggle. */
function useTheme() {
  const [theme, setTheme] = useState<"light" | "dark">(() => {
    const saved = localStorage.getItem("theme");
    if (saved === "light" || saved === "dark") return saved;
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  });

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("theme", theme);
  }, [theme]);

  return { theme, toggleTheme: () => setTheme((t) => (t === "dark" ? "light" : "dark")) };
}

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
  const [menuOpen, setMenuOpen] = useState(false);
  const { theme, toggleTheme } = useTheme();

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

  /** Central navigation: switches page and resets detail selections. */
  const go = (view: Page) => {
    setPage(view);
    setMenuOpen(false);
    if (view !== "recipes") setSelectedRecipeId(null);
    if (view !== "mealplan") setMealPlanId(null);
  };

  const isBackendOk = healthData?.status === "ok" && healthData?.db === "connected";

  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column" }}>
      <a className="skip-link" href="#main">
        Skip to content
      </a>

      {/* ===== Header ===== */}
      <header className="app-header">
        <div className="container header-inner">
          {/* Logo + Title */}
          <button className="logo" onClick={() => go("dashboard")} aria-label="Go to dashboard">
            <span className="logo-emoji">🍮</span>
            <span className="logo-text">Pudim</span>
          </button>

          {/* Desktop Navigation Tabs */}
          <nav className="nav-tabs" aria-label="Primary navigation">
            {NAV_ITEMS.map((tab) => (
              <button
                key={tab.id}
                className={`nav-tab ${page === tab.id ? "active" : ""}`}
                onClick={() => go(tab.id)}
              >
                <span>{tab.icon}</span>
                <span>{tab.label}</span>
              </button>
            ))}
          </nav>

          {/* Status Badge + Theme Toggle + Alarm Badge */}
          <div className="header-right">
            <HeaderBadge />
            <button
              className="theme-toggle"
              onClick={toggleTheme}
              title={theme === "dark" ? "Switch to light mode" : "Switch to dark mode"}
              aria-label={theme === "dark" ? "Switch to light mode" : "Switch to dark mode"}
            >
              {theme === "dark" ? "☀️" : "🌙"}
            </button>
            <span className={`status-dot ${isBackendOk ? "online" : "offline"}`} />
            <span className="conn-label">{isBackendOk ? "Connected" : "Offline"}</span>
          </div>

          {/* Mobile hamburger */}
          <button className="menu-button" onClick={() => setMenuOpen(true)} aria-label="Open menu">
            ☰
          </button>
        </div>
      </header>

      {/* ===== Mobile Slide-Out Drawer ===== */}
      {menuOpen && (
        <div className="nav-backdrop" onClick={() => setMenuOpen(false)}>
          <aside className="nav-drawer" onClick={(e) => e.stopPropagation()} aria-label="Menu">
            <div className="nav-drawer-header">
              <span className="nav-drawer-logo">
                <span>🍮</span> Pudim
              </span>
              <button className="nav-drawer-close" onClick={() => setMenuOpen(false)} aria-label="Close menu">
                ✕
              </button>
            </div>
            {NAV_ITEMS.map((item) => (
              <button
                key={item.id}
                className={`nav-drawer-item ${page === item.id ? "active" : ""}`}
                onClick={() => go(item.id)}
              >
                <span className="nav-icon">{item.icon}</span>
                <span>{item.label}</span>
              </button>
            ))}
          </aside>
        </div>
      )}

      {/* ===== Main Content ===== */}
      <main id="main" style={{ flex: 1, padding: "var(--space-lg) 0" }}>
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
