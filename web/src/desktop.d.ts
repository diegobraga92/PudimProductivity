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
  /** Reports whether media control is supported and playerctl is installed. */
  mediaStatus: () => Promise<{ supported: boolean; available: boolean }>;
  /** Lists the MPRIS players playerctl can reach. */
  mediaListPlayers: () => Promise<{ players: string[] }>;
  /** Sends play or pause to the system media player. */
  mediaControl: (options: {
    action: "play" | "pause";
    player?: string;
  }) => Promise<{ ok: boolean; reason?: "unavailable" | "no-players" | "error"; message?: string }>;
}

interface Window {
  desktop?: DesktopBridge;
}
