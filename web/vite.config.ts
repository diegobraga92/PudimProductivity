import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { visualizer } from "rollup-plugin-visualizer";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  const backendPort = env.BACKEND_PORT ?? "8080";
  const frontendPort = Number(env.FRONTEND_PORT) || 3000;
  const analyze = env.ANALYZE === "1";

  return {
    // rollup-plugin-visualizer is opt-in: `ANALYZE=1 npm run build` emits
    // dist/stats.html with a treemap of chunk sizes (see package.json
    // build:analyze). Normal builds are unaffected.
    plugins: [react(), ...(analyze ? [visualizer({ filename: "dist/stats.html", gzipSize: true, open: false })] : [])],
    server: {
      port: frontendPort,
      // The shared i18n dictionary lives one level above the web project
      // root; allow the dev server to serve it.
      fs: { allow: [".."] },
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