import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  const backendPort = env.BACKEND_PORT ?? "8080";
  const frontendPort = Number(env.FRONTEND_PORT) || 3000;

  return {
    plugins: [react()],
    server: {
      port: frontendPort,
      proxy: {
        "/api": {
          target: `http://localhost:${backendPort}`,
          changeOrigin: true,
          // Enable WebSocket proxying for the real-time sync endpoint
          // (/api/v1/ws).
          ws: true,
        },
      },
    },
    build: {
      outDir: "dist",
      sourcemap: true,
    },
  };
});