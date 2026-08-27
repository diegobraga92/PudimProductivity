/**
 * PudimProductivity Electron main process.
 *
 * Loads the built React SPA from a custom `app://bundle` origin and provides
 * the native integration: notifications, tray, auto-update, window-state
 * persistence, runtime API base override and power-save blocking.
 *
 * The SPA talks to the Go backend over HTTP + WebSocket exactly as in the
 * browser; the backend must allow the `app://bundle` origin
 * (CORS_ALLOWED_ORIGINS=app://bundle).
 */
import {
  app,
  BrowserWindow,
  dialog,
  ipcMain,
  Menu,
  Notification,
  powerSaveBlocker,
  protocol,
  screen,
  shell,
  Tray,
} from "electron";
import type { MenuItemConstructorOptions, Rectangle } from "electron";
import { autoUpdater } from "electron-updater";
import fs from "node:fs";
import path from "node:path";

const APP_SCHEME = "app";
const APP_HOST = "bundle"; // app://bundle/index.html

let mainWindow: BrowserWindow | null = null;
let tray: Tray | null = null;
let isQuitting = false;
let powerSaveBlockId: number | null = null;
let flashStopTimer: ReturnType<typeof setTimeout> | null = null;

// `app://` must be registered as a privileged scheme before app.ready().
protocol.registerSchemesAsPrivileged([
  {
    scheme: APP_SCHEME,
    privileges: { standard: true, secure: true, supportFetchAPI: true, stream: true },
  },
]);

// ─── Runtime settings (API base override, window state) ─────────────────────

function settingsFilePath(): string {
  return path.join(app.getPath("userData"), "settings.json");
}

function readJsonFile<T>(file: string): T | null {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8")) as T;
  } catch {
    return null;
  }
}

function writeJsonFile(file: string, data: unknown): void {
  try {
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, JSON.stringify(data, null, 2), "utf8");
  } catch (err) {
    console.error(`[desktop] failed to write ${path.basename(file)}`, err);
  }
}

function readSettings(): Record<string, unknown> {
  return readJsonFile(settingsFilePath()) ?? {};
}

function writeSettings(patch: Record<string, unknown>): void {
  writeJsonFile(settingsFilePath(), { ...readSettings(), ...patch });
}

/**
 * Backend base URL passed to the renderer. Precedence: persisted user override
 * (setApiBaseUrl), then the PUDIM_API_BASE_URL env var, then "", the renderer
 * falls back to the VITE_API_BASE_URL baked from web/.env.desktop.
 */
function getConfiguredApiBaseUrl(): string {
  const stored = readSettings().apiBaseUrl;
  if (typeof stored === "string" && stored.length > 0) {
    return stored.replace(/\/+$/, "");
  }
  const env = process.env.PUDIM_API_BASE_URL;
  if (env && env.length > 0) {
    return env.replace(/\/+$/, "");
  }
  return "";
}

function windowStateFilePath(): string {
  return path.join(app.getPath("userData"), "window-state.json");
}

interface WindowState {
  width?: number;
  height?: number;
  x?: number;
  y?: number;
  isMaximized?: boolean;
}

function readWindowState(): WindowState {
  return readJsonFile(windowStateFilePath()) ?? {};
}

function saveWindowState(win: BrowserWindow): void {
  const { x, y, width, height } = win.getNormalBounds();
  writeJsonFile(windowStateFilePath(), {
    x,
    y,
    width,
    height,
    isMaximized: win.isMaximized(),
  });
}

/** True when at least part of the bounds intersects a connected display. */
function isVisibleOnSomeDisplay(bounds: Rectangle): boolean {
  return screen.getAllDisplays().some(({ workArea }) => {
    return (
      bounds.x < workArea.x + workArea.width &&
      bounds.x + bounds.width > workArea.x &&
      bounds.y < workArea.y + workArea.height &&
      bounds.y + bounds.height > workArea.y
    );
  });
}

// ─── Web bundle resolution + app:// protocol ────────────────────────────────

function resolveDistDir(): string {
  if (!app.isPackaged) {
    const repo = path.join(__dirname, "..", "..", "web", "dist");
    if (fs.existsSync(repo)) return repo;
    console.error("[desktop] web build not found. Run: npm --prefix ../web run build:desktop");
    app.exit(1);
  }
  // Packaged: dist-web/ is copied next to dist-electron/ inside the asar.
  return path.join(__dirname, "..", "dist-web");
}

const MIME_TYPES: Record<string, string> = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".mjs": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".ttf": "font/ttf",
  ".mp3": "audio/mpeg",
  ".ogg": "audio/ogg",
  ".wav": "audio/wav",
  ".flac": "audio/flac",
  ".m4a": "audio/mp4",
  ".map": "application/json; charset=utf-8",
};

function registerAppProtocol(distDir: string): void {
  protocol.handle(APP_SCHEME, (request) => {
    let requested: string;
    try {
      requested = decodeURIComponent(new URL(request.url).pathname);
    } catch {
      return new Response("bad request", { status: 400 });
    }
    const file = path.normalize(path.join(distDir, requested));
    // Confine to distDir (no path traversal).
    const relative = path.relative(distDir, file);
    if (relative.startsWith("..") || path.isAbsolute(relative)) {
      return new Response("forbidden", { status: 403 });
    }
    if (!fs.existsSync(file)) {
      return new Response("not found", { status: 404 });
    }
    const mime = MIME_TYPES[path.extname(file).toLowerCase()] ?? "application/octet-stream";
    return new Response(fs.readFileSync(file), {
      headers: { "Content-Type": mime, "Cache-Control": "no-cache" },
    });
  });
}

// ─── Window ─────────────────────────────────────────────────────────────────

function createWindow(): void {
  const saved = readWindowState();
  const restoredBounds = {
    width: saved.width ?? 1280,
    height: saved.height ?? 800,
  };
  const bounds =
    saved.x !== undefined &&
    saved.y !== undefined &&
    isVisibleOnSomeDisplay({ x: saved.x, y: saved.y, ...restoredBounds })
      ? { ...restoredBounds, x: saved.x, y: saved.y }
      : restoredBounds;

  const win = new BrowserWindow({
    ...bounds,
    minWidth: 720,
    minHeight: 480,
    show: false,
    autoHideMenuBar: true,
    title: "PudimProductivity",
    icon: path.join(__dirname, "..", "assets", "icon.png"),
    backgroundColor: "#141414",
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      additionalArguments: [
        `--pudim-api-base-url=${getConfiguredApiBaseUrl()}`,
        `--pudim-app-version=${app.getVersion()}`,
      ],
    },
  });

  if (saved.isMaximized) win.maximize();

  win.once("ready-to-show", () => win.show());
  win.on("focus", () => win.flashFrame(false));

  win.on("close", (event) => {
    if (isQuitting) {
      saveWindowState(win);
      return;
    }
    // Minimize to tray instead of quitting.
    event.preventDefault();
    win.hide();
  });

  win.on("closed", () => {
    mainWindow = null;
  });

  // External links (target=_blank / window.open) go to the system browser.
  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:\/\//.test(url)) void shell.openExternal(url);
    return { action: "deny" };
  });

  // Never navigate away from the app origin.
  win.webContents.on("will-navigate", (event, url) => {
    if (!url.startsWith(`${APP_SCHEME}://`)) event.preventDefault();
  });

  void win.loadURL(`${APP_SCHEME}://${APP_HOST}/index.html`);
  mainWindow = win;
}

function showMainWindow(): void {
  if (!mainWindow) {
    createWindow();
    return;
  }
  if (mainWindow.isMinimized()) mainWindow.restore();
  mainWindow.show();
  mainWindow.focus();
}

/**
 * Flashes the taskbar/dock entry when the window is not focused. Auto-stops
 * after 30s (and is cleared on focus) so it never blinks forever.
 */
function flashMainWindow(): void {
  if (!mainWindow || mainWindow.isFocused()) return;
  if (flashStopTimer !== null) clearTimeout(flashStopTimer);
  mainWindow.flashFrame(true);
  flashStopTimer = setTimeout(() => {
    mainWindow?.flashFrame(false);
    flashStopTimer = null;
  }, 30_000);
}

// ─── Tray ───────────────────────────────────────────────────────────────────

function createTray(): void {
  try {
    tray = new Tray(path.join(__dirname, "..", "assets", "trayTemplate.png"));
    tray.setToolTip("PudimProductivity");
    tray.setContextMenu(
      Menu.buildFromTemplate([
        {
          label: "Show PudimProductivity",
          click: () => showMainWindow(),
        },
        { type: "separator" },
        {
          label: "Quit",
          click: () => {
            isQuitting = true;
            app.quit();
          },
        },
      ]),
    );
    tray.on("click", () => {
      if (mainWindow?.isVisible()) mainWindow.hide();
      else showMainWindow();
    });
  } catch (err) {
    // Some Linux desktops have no StatusNotifier host. 
    // Degrade gracefully instead of crashing the app after the window is already up.
    tray = null;
    console.warn("[desktop] system tray unavailable — running without tray", err);
  }
}

// ─── IPC (preload bridge) ───────────────────────────────────────────────────

function registerIpcHandlers(): void {
  ipcMain.handle("desktop:notify", (_event, options: { title?: string; body?: string }) => {
    if (!Notification.isSupported()) return;
    new Notification({
      title: options.title ?? "PudimProductivity",
      body: options.body ?? "",
    }).show();
    flashMainWindow();
  });

  ipcMain.handle("desktop:open-external", (_event, url: unknown) => {
    if (typeof url === "string" && /^https?:\/\//.test(url)) void shell.openExternal(url);
  });

  ipcMain.handle("desktop:set-api-base-url", (_event, url: unknown) => {
    if (typeof url === "string" && /^https?:\/\//.test(url)) {
      writeSettings({ apiBaseUrl: url.replace(/\/+$/, "") });
    }
  });

  ipcMain.handle("desktop:set-login-item", (_event, enabled: unknown) => {
    try {
      app.setLoginItemSettings({ openAtLogin: Boolean(enabled) });
    } catch (err) {
      console.error("[desktop] failed to set login item", err);
    }
  });

  ipcMain.handle("desktop:set-power-save", (_event, active: unknown) => {
    if (active && powerSaveBlockId === null) {
      powerSaveBlockId = powerSaveBlocker.start("prevent-app-suspension");
    } else if (!active && powerSaveBlockId !== null) {
      powerSaveBlocker.stop(powerSaveBlockId);
      powerSaveBlockId = null;
    }
  });

  ipcMain.on("desktop:flash-frame", (_event, active: unknown) => {
    if (active) {
      flashMainWindow();
    } else {
      if (flashStopTimer !== null) {
        clearTimeout(flashStopTimer);
        flashStopTimer = null;
      }
      mainWindow?.flashFrame(false);
    }
  });
}

// ─── Auto-update (electron-updater, packaged builds only) ───────────────────

function setupAutoUpdater(): void {
  if (!app.isPackaged) return;
  autoUpdater.autoDownload = true;
  autoUpdater.autoInstallOnAppQuit = true;
  autoUpdater.on("error", (err) => {
    console.warn("[desktop] auto-update error", err);
  });
  autoUpdater.on("update-downloaded", (info) => {
    const choice = dialog.showMessageBoxSync({
      type: "info",
      title: "Update available",
      message: `PudimProductivity ${info.version} is ready to install. Restart now?`,
      buttons: ["Restart", "Later"],
      defaultId: 0,
      cancelId: 1,
    });
    if (choice === 0) autoUpdater.quitAndInstall();
  });
  void autoUpdater.checkForUpdatesAndNotify().catch((err) => {
    console.warn("[desktop] update check failed", err);
  });
}

// ─── Application menu (keeps copy/paste working, esp. on macOS) ─────────────
// The menu stays registered so keyboard accelerators (Ctrl/Cmd+C/V/X/Z, and
// reload/devtools in dev) keep working. The window toolbar is hidden on
// Windows/Linux via `autoHideMenuBar: true` in createWindow(); on macOS the
// menu lives in the system menu bar, so nothing is rendered in the window.

function createApplicationMenu(): void {
  const isMac = process.platform === "darwin";
  const template: MenuItemConstructorOptions[] = [
    ...(isMac ? [{ role: "appMenu" as const }] : []),
    { role: "editMenu" },
    ...(!app.isPackaged ? [{ role: "viewMenu" as const }] : []),
  ];
  Menu.setApplicationMenu(Menu.buildFromTemplate(template));
}

// ─── Lifecycle ──────────────────────────────────────────────────────────────

const gotSingleInstanceLock = app.requestSingleInstanceLock();

if (!gotSingleInstanceLock) {
  app.quit();
} else {
  app.on("second-instance", () => {
    showMainWindow();
  });

  void app.whenReady().then(() => {
    const distDir = resolveDistDir();
    registerAppProtocol(distDir);
    registerIpcHandlers();
    createApplicationMenu();
    createWindow();
    createTray();
    setupAutoUpdater();

    // macOS dock convention: re-create the window when the dock icon is clicked.
    app.on("activate", () => {
      if (BrowserWindow.getAllWindows().length === 0) createWindow();
    });
  });

  app.on("before-quit", () => {
    isQuitting = true;
    if (powerSaveBlockId !== null) {
      powerSaveBlocker.stop(powerSaveBlockId);
      powerSaveBlockId = null;
    }
  });

  app.on("window-all-closed", () => {
    // The app lives in the tray; quit only via the explicit Quit menu item.
    if (isQuitting) app.quit();
  });
}
