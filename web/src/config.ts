/// <reference types="vite/client" />

const config = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? "/api/v1",
} as const;

export default config;
