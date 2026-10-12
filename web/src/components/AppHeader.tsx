import { useEffect, useState } from "react";
import { useI18n } from "../i18n";
import {
  BellIcon,
  CloseIcon,
  GlobeIcon,
  MenuIcon,
  MoonIcon,
  MoreIcon,
  SunIcon,
} from "./icons";
import { MORE_ITEMS, NAV_ITEMS, type Page } from "./navigation";
import { useAlarm } from "./useAlarm";

/** Theme hook: resolves the system preference, then honors the manual toggle. */
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

/** Active-alarm counter shown in the header while an alarm is ringing. */
function HeaderBadge() {
  const { activeAlarms } = useAlarm();
  const { t } = useI18n();
  if (activeAlarms.length === 0) return null;
  return (
    <span className="alarm-badge" title={t("alarm.activeBadge", { count: activeAlarms.length })}>
      <BellIcon size={13} /> {activeAlarms.length}
    </span>
  );
}

interface AppHeaderProps {
  page: Page;
  isBackendOk: boolean;
  onNavigate: (view: Page) => void;
}

/**
 * App chrome: top bar (nav tabs, "More" menu, status + toggles) and the mobile
 * slide-out drawer. Owns its own open/closed state so the page shell does not
 * have to.
 */
export default function AppHeader({ page, isBackendOk, onNavigate }: AppHeaderProps) {
  const { t, lang, toggleLang } = useI18n();
  const { theme, toggleTheme } = useTheme();
  const [menuOpen, setMenuOpen] = useState(false);
  const [moreOpen, setMoreOpen] = useState(false);

  /** Navigates and closes whichever menu triggered it. */
  const go = (view: Page) => {
    setMenuOpen(false);
    setMoreOpen(false);
    onNavigate(view);
  };

  return (
    <>
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
                <span className="nav-icon"><tab.icon size={16} /></span>
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
              <span className="nav-icon"><MoreIcon size={16} /></span>
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
                      <span className="more-menu-icon"><item.icon size={16} /></span>
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
            >
              <GlobeIcon size={15} />
            </button>
            <button
              className="theme-toggle"
              onClick={toggleTheme}
              title={theme === "dark" ? t("theme.light") : t("theme.dark")}
              aria-label={theme === "dark" ? t("theme.light") : t("theme.dark")}
            >
              {theme === "dark" ? <SunIcon size={15} /> : <MoonIcon size={15} />}
            </button>
            <span className={`status-dot ${isBackendOk ? "online" : "offline"}`} />
            <span className="conn-label">{isBackendOk ? t("status.connected") : t("status.offline")}</span>
          </div>

          {/* Mobile hamburger */}
          <button className="menu-button" onClick={() => setMenuOpen(true)} aria-label={t("a11y.openMenu")}>
            <MenuIcon size={20} />
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
                <CloseIcon size={16} />
              </button>
            </div>
            {[...NAV_ITEMS, ...MORE_ITEMS].map((item) => (
              <button
                key={item.id}
                className={`nav-drawer-item ${page === item.id ? "active" : ""}`}
                onClick={() => go(item.id)}
              >
                <span className="nav-icon"><item.icon size={18} /></span>
                <span>{t(item.labelKey)}</span>
              </button>
            ))}
          </aside>
        </div>
      )}
    </>
  );
}
