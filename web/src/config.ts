/// <reference types="vite/client" />

// Electron desktop builds receive the backend URL from the shell (passed via
// additionalArguments -> contextBridge, see desktop/src/preload.ts): either a
// user override (settings.json / PUDIM_API_BASE_URL) or an empty string when
// none is set. Empty falls through to the build-time VITE_API_BASE_URL, so a
// desktop build can point at a LAN backend just by setting web/.env.desktop.
// In the browser there is no bridge, so the env var (or same-origin default)
// is used, which works because nginx/Vite proxy /api to the backend.
const desktopApiBaseUrl = window.desktop?.getApiBaseUrl?.();

const config = {
  apiBaseUrl: desktopApiBaseUrl || import.meta.env.VITE_API_BASE_URL || "/api/v1",
  // Public base URL for uploaded media objects (e.g. the S3 bucket or CDN
  // domain). Recipe image keys returned by the upload flow are resolved
  // against this. Leave empty when no media backend is configured.
  mediaBaseUrl: import.meta.env.VITE_MEDIA_BASE_URL ?? "",
} as const;

export default config;

