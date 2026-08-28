/// <reference types="vite/client" />

// Electron desktop builds receive the backend URL from the shell (passed via
// additionalArguments -> contextBridge).
const desktopApiBaseUrl = window.desktop?.getApiBaseUrl?.();

const config = {
  apiBaseUrl: desktopApiBaseUrl || import.meta.env.VITE_API_BASE_URL || "/api/v1",
  // Public base URL for uploaded media objects.
  mediaBaseUrl: import.meta.env.VITE_MEDIA_BASE_URL ?? "",
} as const;

export default config;

