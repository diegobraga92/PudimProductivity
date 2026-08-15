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
import { useI18n } from "./i18n";
import "./styles.css";

// Secondary pages are code-split (React.lazy) so the initial bundle only
// contains the app shell + Dashboard (the landing page). TaskList, Planner,
// Pomodoro and Soundscape chunks load on demand. Run `npm run build:analyze`
// to inspect chunk sizes.
const TaskList = lazy(() => import("./pages/TaskList"));
const Lists = lazy(() => import("./pages/Lists"));
const Planner = lazy(() => import("./pages/Planner"));
const Pomodoro = lazy(() => import("./pages/Pomodoro"));
const Soundscape = lazy(() => import("./pages/Soundscape"));
const RecipeList = lazy(() => import("./pages/RecipeList"));
const RecipeDetail = lazy(() => import("./pages/RecipeDetail"));
const Library = lazy(() => import("./pages/Library"));
const Insights = lazy(() => import("./pages/Insights"));
const ServerSettings = lazy(() => import("./pages/ServerSettings"));

type Page = "dashboard" | "tasks" | "lists" | "planner" | "pomodoro" | "soundscape" | "recipes" | "library" | "insights" | "health" | "settings";

// Primary tabs always visible on the desktop nav bar.
const NAV_ITEMS: { id: Page; labelKey: string; icon: string }[] = [
  { id: "dashboard", labelKey: "nav.dashboard", icon: "🏠" },
  { id: "tasks", labelKey: "nav.tasks", icon: "📋" },
  { id: "lists", labelKey: "nav.lists", icon: "📁" },
  { id: "planner", labelKey: "nav.planner", icon: "📅" },
  { id: "pomodoro", labelKey: "nav.timer", icon: "🍅" },
  { id: "soundscape", labelKey: "nav.sounds", icon: "🎵" },
  { id: "recipes", labelKey: "nav.recipes", icon: "🍳" },
  { id: "library", labelKey: "nav.library", icon: "🎬" },
  { id: "insights", labelKey: "nav.insights", icon: "🧠" },
];

// Secondary pages tucked into the desktop "More" dropdown so the top bar does
// not overflow (the health/status item was previously pushed off-screen).
const MORE_ITEMS: { id: Page; labelKey: string; icon: string }[] = [
  { id: "health", labelKey: "nav.health", icon: "💚" },
  { id: "settings", labelKey: "nav.serverSettings", icon: "⚙️" },
];

function PageFallback() {
  const { t } = useI18n();
  return (
    <div className="container" style={{ paddingTop: "var(--space-xl)", textAlign: "center", color: "var(--color-text-secondary)" }}>
      {t("common.loading")}
    </div>
  );
}

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
  const { t } = useI18n();
  if (activeAlarms.length === 0) return null;
  return (
    <span className="alarm-badge" title={t("alarm.activeBadge", { count: activeAlarms.length })}>
      🔔 {activeAlarms.length}
    </span>
  );
}

function AppInner() {
  const [page, setPage] = useState<Page>("dashboard");
  const [selectedRecipeId, setSelectedRecipeId] = useState<string | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [moreOpen, setMoreOpen] = useState(false);
  const { theme, toggleTheme } = useTheme();
  const { t, lang, toggleLang } = useI18n();

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
    setMoreOpen(false);
    if (view !== "recipes") setSelectedRecipeId(null);
  };

  const isBackendOk = healthData?.status === "ok" && healthData?.db === "connected";

  return (
    <div style={{ minHeight: "100vh", display: "flex", flexDirection: "column" }}>
      <a className="skip-link" href="#main">
        {t("a11y.skipToContent")}
      </a>

      {/* ===== Header ===== */}
      <header className="app-header">
        <div className="container header-inner">
          {/* Logo + Title */}
          <button className="logo" onClick={() => go("dashboard")} aria-label={t("a11y.goToDashboard")}>
            <span className="logo-emoji">🍮</span>
            <span className="logo-text">Pudim</span>
          </button>

          {/* Desktop Navigation Tabs */}
          <nav className="nav-tabs" aria-label={t("a11y.primaryNav")}>
            {NAV_ITEMS.map((tab) => (
              <button
                key={tab.id}
                className={`nav-tab ${page === tab.id ? "active" : ""}`}
                onClick={() => go(tab.id)}
              >
                <span>{tab.icon}</span>
                <span>{t(tab.labelKey)}</span>
              </button>
            ))}
          </nav>

          {/* Desktop "More" overflow menu (health/status + server settings) */}
          <div className="more-wrap">
            <button
              className={`nav-tab more-button ${moreOpen ? "active" : ""}`}
              onClick={() => setMoreOpen((o) => !o)}
              aria-haspopup="true"
              aria-expanded={moreOpen}
            >
              <span>⋯</span>
              <span>{t("nav.more")}</span>
            </button>
            {moreOpen && (
              <>
                <div className="more-backdrop" onClick={() => setMoreOpen(false)} />
                <div className="more-menu" role="menu" aria-label={t("nav.more")}>
                  {MORE_ITEMS.map((item) => (
                    <button
                      key={item.id}
                      role="menuitem"
                      className={`more-menu-item ${page === item.id ? "active" : ""}`}
                      onClick={() => go(item.id)}
                    >
                      <span className="more-menu-icon">{item.icon}</span>
                      <span>{t(item.labelKey)}</span>
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>

          {/* Status Badge + Language/Theme Toggles + Alarm Badge */}
          <div className="header-right">
            <HeaderBadge />
            <button
              className="theme-toggle"
              onClick={toggleLang}
              title={lang === "en" ? t("lang.portuguese") : t("lang.english")}
              aria-label={lang === "en" ? t("lang.portuguese") : t("lang.english")}
              style={{ fontSize: "0.95rem" }}
            >
              {lang === "en" ? "🇧🇷" : "🇺🇸"}
            </button>
            <button
              className="theme-toggle"
              onClick={toggleTheme}
              title={theme === "dark" ? t("theme.light") : t("theme.dark")}
              aria-label={theme === "dark" ? t("theme.light") : t("theme.dark")}
            >
              {theme === "dark" ? "☀️" : "🌙"}
            </button>
            <span className={`status-dot ${isBackendOk ? "online" : "offline"}`} />
            <span className="conn-label">{isBackendOk ? t("status.connected") : t("status.offline")}</span>
          </div>

          {/* Mobile hamburger */}
          <button className="menu-button" onClick={() => setMenuOpen(true)} aria-label={t("a11y.openMenu")}>
            ☰
          </button>
        </div>
      </header>

      {/* ===== Mobile Slide-Out Drawer ===== */}
      {menuOpen && (
        <div className="nav-backdrop" onClick={() => setMenuOpen(false)}>
          <aside className="nav-drawer" onClick={(e) => e.stopPropagation()} aria-label={t("nav.more")}>
            <div className="nav-drawer-header">
              <span className="nav-drawer-logo">
                <span>🍮</span> Pudim
              </span>
              <button className="nav-drawer-close" onClick={() => setMenuOpen(false)} aria-label={t("a11y.closeMenu")}>
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
                <span>{t(item.labelKey)}</span>
              </button>
            ))}
            {MORE_ITEMS.map((item) => (
              <button
                key={item.id}
                className={`nav-drawer-item ${page === item.id ? "active" : ""}`}
                onClick={() => go(item.id)}
              >
                <span className="nav-icon">{item.icon}</span>
                <span>{t(item.labelKey)}</span>
              </button>
            ))}
          </aside>
        </div>
      )}

      {/* ===== Main Content ===== */}
      <main id="main" style={{ flex: 1, padding: "var(--space-lg) 0" }}>
        <div className="container">
          <Suspense fallback={<PageFallback />}>
            {page === "dashboard" && <Dashboard onNavigate={handleNavigate} />}

            {page === "tasks" && <TaskList />}

            {page === "lists" && <Lists />}

            {page === "planner" && <Planner onNavigate={handleNavigate} />}

            {page === "pomodoro" && <Pomodoro />}

            {page === "soundscape" && <Soundscape />}

            {page === "recipes" &&
              (selectedRecipeId ? (
                <RecipeDetail recipeId={selectedRecipeId} onBack={() => setSelectedRecipeId(null)} />
              ) : (
                <RecipeList onOpen={(r) => setSelectedRecipeId(r.id)} />
              ))}

            {page === "library" && <Library />}

            {page === "insights" && <Insights />}

            {page === "settings" && <ServerSettings />}
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
                  💚 {t("nav.health")}
                </h2>
                <span
                  className={`badge ${isBackendOk ? "badge-done" : "badge-habit"}`}
                >
                  {isBackendOk ? t("status.healthy") : t("status.issues")}
                </span>
              </div>

              {!healthData && (
                <div className="card" style={{ textAlign: "center", padding: "var(--space-xl)" }}>
                  <p style={{ color: "var(--color-text-secondary)" }}>
                    {t("status.checking")}
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
                    <div className="stat-card-label">{t("nav.status")}</div>
                  </div>
                  <div className="stat-card">
                    <div className="stat-card-value" style={{ fontSize: "var(--font-size-lg)" }}>
                      v{healthData.version}
                    </div>
                    <div className="stat-card-label">{t("status.version")}</div>
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
                    <div className="stat-card-label">{t("status.database")}</div>
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
