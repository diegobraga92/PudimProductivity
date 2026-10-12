import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense, useState } from "react";
import { getHealth, type HealthResponse } from "./api/client";
import { AlarmProvider } from "./components/AlarmProvider";
import { AlarmToast } from "./components/AlarmToast";
import AppHeader from "./components/AppHeader";
import { ConfirmProvider } from "./components/ConfirmProvider";
import HealthPage from "./components/HealthPage";
import { ToastProvider } from "./components/ToastProvider";
import { ToastStack } from "./components/ToastStack";
import { type Page } from "./components/navigation";
import { useAlarmNotifier } from "./hooks/useAlarmNotifier";
import { useErrorReporter } from "./hooks/useErrorReporter";
import { useLiveUpdates } from "./hooks/useLiveUpdates";
import { usePomodoroSoundSync } from "./hooks/usePomodoroSoundSync";
import { usePomodoroMediaSync } from "./hooks/usePomodoroMediaSync";
import { useTaskNotifier } from "./hooks/useTaskNotifier";
import Dashboard from "./pages/Dashboard";
import { useI18n } from "./i18n";
import "./styles.css";

// Secondary pages are code-split (React.lazy) so the initial bundle only
// contains the app shell + Dashboard (the landing page).
// Run `npm run build:analyze` to inspect chunk sizes.
const TaskList = lazy(() => import("./pages/TaskList"));
const Lists = lazy(() => import("./pages/Lists"));
const Planner = lazy(() => import("./pages/Planner"));
const Pomodoro = lazy(() => import("./pages/Pomodoro"));
const Soundscape = lazy(() => import("./pages/Soundscape"));
const RecipeList = lazy(() => import("./pages/RecipeList"));
const RecipeDetail = lazy(() => import("./pages/RecipeDetail"));
const MealPlanner = lazy(() => import("./pages/MealPlanner"));
const Library = lazy(() => import("./pages/Library"));
const ServerSettings = lazy(() => import("./pages/ServerSettings"));

function PageFallback() {
  const { t } = useI18n();
  return (
    <div className="container" style={{ paddingTop: "var(--space-xl)", textAlign: "center", color: "var(--color-text-secondary)" }}>
      {t("common.loading")}
    </div>
  );
}

function AppInner() {
  const [page, setPage] = useState<Page>("dashboard");
  const [selectedRecipeId, setSelectedRecipeId] = useState<string | null>(null);
  const { t } = useI18n();

  // App-root singletons (each hook documents its own purpose).
  useAlarmNotifier();
  useLiveUpdates();
  usePomodoroSoundSync();
  usePomodoroMediaSync();
  useTaskNotifier();
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
    if (view !== "recipes") setSelectedRecipeId(null);
  };

  const isBackendOk = healthData?.status === "ok" && healthData?.db === "connected";

  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column" }}>
      <a className="skip-link" href="#main">
        {t("a11y.skipToContent")}
      </a>

      <AppHeader page={page} isBackendOk={isBackendOk} onNavigate={go} />

      {/* ===== Main Content ===== */}
      <main id="main" style={{ flex: 1, padding: "var(--space-lg) 0" }}>
        <div className="container">
          <Suspense fallback={<PageFallback />}>
            {page === "dashboard" && <Dashboard onNavigate={handleNavigate} />}

            {page === "tasks" && <TaskList />}

            {page === "lists" && <Lists />}

            {page === "planner" && <Planner onNavigate={handleNavigate} />}

            {page === "pomodoro" && <Pomodoro onOpenSounds={() => go("soundscape")} />}

            {page === "soundscape" && <Soundscape />}

            {page === "recipes" &&
              (selectedRecipeId ? (
                <RecipeDetail recipeId={selectedRecipeId} onBack={() => setSelectedRecipeId(null)} />
              ) : (
                <RecipeList onNew={() => setSelectedRecipeId("__new__")} onEdit={(id) => setSelectedRecipeId(id)} />
              ))}

            {page === "mealPlan" && <MealPlanner />}

            {page === "library" && <Library />}

            {page === "settings" && <ServerSettings />}
          </Suspense>

          {page === "health" && <HealthPage />}
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
