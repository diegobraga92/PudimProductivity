# PudimProductivity Desktop (Electron)

A thin Electron wrapper around the React web client: it loads the built SPA
from a custom `app://bundle` origin and adds native integration. It's just a
client, the backend must be running and reachable.

## Run (development)

```bash
# 1. From the desktop/ directory, install dependencies once
npm install

# 2. Start the backend + infra (from the repo root)
../scripts/run.sh

# 3. Build the web app for desktop + the shell, then launch Electron
npm run dev
```

## Backend CORS

Configure the backend to allow it:

```bash
CORS_ALLOWED_ORIGINS=app://bundle
```

## Packages

```bash
npm run package       # full installers (AppImage/deb on Linux, dmg/zip on macOS, nsis on Windows)
npm run package:dir   # unpacked app only 
```

## Deploy with a LAN backend

To use the desktop app with the backend running in another server:

1. On the LAN Server, allow the desktop origin and recreate the backend:

   ```bash
   # .env (server): add
   CORS_ALLOWED_ORIGINS=app://bundle

   docker compose up -d
   ```

2. On the machine running the Electron app, edit `../web/.env.desktop`:

   ```ini
   VITE_API_BASE_URL=http://<lan-ip>:<port>/api/v1             # backend directly
   # VITE_API_BASE_URL=http://<lan-ip>:3000/api/v1          # via the nginx frontend
   # VITE_MEDIA_BASE_URL=http://<lan-ip>:<port>/api/v1/media   # Add for media
   ```

3. Build and install the packaged app:

   ```bash
   npm run package

   sudo apt install ./release/*.deb          # Debian/Ubuntu package, or:

   ./release/PudimProductivity-*.AppImage    # portable AppImage (chmod +x first)
   ```

