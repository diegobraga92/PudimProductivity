/**
 * Types for the Electron desktop bridge exposed via contextBridge. 
 * The global is absent in plain-browser builds, all access must be optional-chained.
 */
interface DesktopBridge {
  platform: string;
  versions: {
    app: string;
    electron: string;
  };
  /** Fires a native OS notification (used for alarm reminders). */
  notify: (options: { title: string; body?: string }) => void;
  /** Returns the configured backend base URL (runtime override). */
  getApiBaseUrl: () => string;
  /** Persists a new backend base URL for the next launch. */
  setApiBaseUrl: (url: string) => void;
  /** Opens an http(s) URL in the system browser. */
  openExternal: (url: string) => void;
  /** Toggles "open at login". */
  setLoginItem: (enabled: boolean) => void;
  /** Prevents OS sleep while the focus timer / soundscape is active. */
  setPowerSaveBlocker: (active: boolean) => void;
  /** Flashes the taskbar/dock while an alarm is pending. */
  flashFrame: (active: boolean) => void;
}

interface Window {
  desktop?: DesktopBridge;
}
