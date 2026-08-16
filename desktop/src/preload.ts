/**
 * PudimProductivity — Electron preload script.
 *
 * Runs sandboxed with contextIsolation enabled. Exposes a minimal, whitelisted
 * API on window.desktop for the React app (see web/src/desktop.d.ts for the
 * renderer-side types).
 */
import { contextBridge, ipcRenderer } from "electron";

/** Reads a `--flag=value` from additionalArguments passed by the main process. */
function argValue(prefix: string): string | null {
  const arg = process.argv.find((a) => a.startsWith(prefix));
  return arg ? arg.slice(prefix.length) : null;
}

// Empty string = no override configured; web/src/config.ts then falls back to
// the build-time VITE_API_BASE_URL baked from web/.env.desktop.
const apiBaseUrl = argValue("--pudim-api-base-url=") ?? "";
const appVersion = argValue("--pudim-app-version=") ?? "0.0.1";

contextBridge.exposeInMainWorld("desktop", {
  platform: process.platform,
  versions: {
    app: appVersion,
    electron: process.versions.electron,
  },
  // Synchronous on purpose: web/src/config.ts reads it at module load time.
  getApiBaseUrl: () => apiBaseUrl,
  setApiBaseUrl: (url: string) => ipcRenderer.invoke("desktop:set-api-base-url", url),
  notify: (options: { title: string; body?: string }) =>
    ipcRenderer.invoke("desktop:notify", options),
  openExternal: (url: string) => ipcRenderer.invoke("desktop:open-external", url),
  setLoginItem: (enabled: boolean) => ipcRenderer.invoke("desktop:set-login-item", enabled),
  setPowerSaveBlocker: (active: boolean) =>
    ipcRenderer.invoke("desktop:set-power-save", active),
  flashFrame: (active: boolean) => ipcRenderer.send("desktop:flash-frame", active),
});
