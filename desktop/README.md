# PudimProductivity Desktop (Electron)

A native desktop wrapper around the existing React web client (`../web`). The
renderer code is reused 100% — the shell only loads the built SPA from a custom
`app://bundle` origin and adds native integration (notifications, tray,
auto-update, window state, runtime API base override).

## Prerequisites

- The Go backend + infrastructure running (see the repo README `Quick Start`).
  The desktop app is a **thin client**: it needs the backend reachable.

## Run (development)

```bash
# 1. From the desktop/ directory, install dependencies once
npm install

# 2. Start the backend + infra (from the repo root) — or run a backend already
#    reachable on localhost:8080
../scripts/run.sh

# 3. Build the web app for desktop + the shell, then launch Electron
npm run dev
```

That runs `vite build --mode desktop` (which reads `../web/.env.desktop` and
bakes in `VITE_API_BASE_URL=http://localhost:8080/api/v1`), compiles the main +
preload TS, generates the icons, and launches Electron.

## Backend CORS

The desktop app serves the SPA from the `app://bundle` origin, so every request
to the Go API is cross-origin. Configure the backend to allow it:

```bash
CORS_ALLOWED_ORIGINS=app://bundle
```

(comma-separated list; see `backend/internal/shared/cors_middleware.go`).

## Package (installers)

```bash
npm run package       # full installers (AppImage/deb on Linux, dmg/zip on macOS, nsis on Windows)
npm run package:dir   # unpacked app only (faster, for smoke tests)
```

Auto-update is wired via `electron-updater` against the GitHub repo releases
(`publish` in `electron-builder.yml`). It only runs in packaged builds and
requires signed builds on macOS/Windows for production distribution.

## Deploy against a LAN backend

The backend + web stack run on a LAN server (see the repo README "Deploy on
LAN"). To use the desktop app from another Linux machine against that server:

1. **On the LAN server** — allow the desktop origin and recreate the backend:

   ```bash
   # .env (server): add
   CORS_ALLOWED_ORIGINS=app://bundle

   docker compose up -d          # recreates the backend with the new env
   ```

2. **On the desktop machine** — point the build at the server. Edit
   `../web/.env.desktop`:

   ```ini
   VITE_API_BASE_URL=http://<lan-ip>:8080/api/v1      # backend directly
   # or VITE_API_BASE_URL=http://<lan-ip>:3000/api/v1 # via the nginx frontend
   # If recipe images (media) are configured on the server, also set:
   # VITE_MEDIA_BASE_URL=http://<lan-ip>:8080/api/v1/media
   ```

3. Build and install the packaged app:

   ```bash
   npm run package
   # Install the produced artifacts in desktop/release/:
   sudo apt install ./release/*.deb          # Debian/Ubuntu package, or:
   ./release/PudimProductivity-*.AppImage    # portable AppImage (chmod +x first)
   ```

4. The packaged app now talks to the LAN backend over HTTP + WebSocket with no
   runtime configuration — real-time sync works across desktop/web/Android.

> Precedence: a runtime override (`window.desktop.setApiBaseUrl(...)` or the
> `PUDIM_API_BASE_URL` env var) always wins over the baked `VITE_API_BASE_URL`.

## Runtime backend URL override

Without rebuilding, an installed app can point at another backend:

- from DevTools: `window.desktop.setApiBaseUrl("http://<other-ip>:8080/api/v1")`
- the URL persists in `~/.config/pudimproductivity-desktop/settings.json`
  (`%APPDATA%` on Windows, `~/Library/Application Support` on macOS) and takes
  effect on the next launch.

## Where things live

| Path                       | Purpose                                        |
|----------------------------|------------------------------------------------|
| `src/main.ts`              | Electron main process                          |
| `src/preload.ts`           | contextBridge API (`window.desktop`)           |
| `../web/src/desktop.d.ts`  | Renderer-side types for the bridge             |
| `electron-builder.yml`     | Packaging / auto-update config                 |
| `scripts/generate-icons.mjs` | Regenerates `assets/*.png` (placeholder art) |
| `scripts/copy-web.mjs`     | Copies `web/dist` into the package             |
