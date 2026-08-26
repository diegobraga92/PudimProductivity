# PudimProductivity

Personal productivity app with tasks, habits, pomodoro timer, focus soundscapes, recipes, and a media library. Everything stays in sync between web (React), desktop (Electron), and Android (Kotlin), with a Go backend.


## Quick start (development)

Go 1.26+, Node 20+ (see `.nvmrc`), and Docker.

```bash
cp .env.example .env
./scripts/run.sh 
```

Open http://localhost:3000. The backend health check is at http://localhost:8080/api/v1/health.

- **Android app:** `./scripts/run-mobile.sh` (boots an emulator, builds and installs the app)
- **Desktop app:** see [`desktop/README.md`](desktop/README.md)

## Deploy on LAN

Run the whole thing as a self-contained Docker stack on any Linux machine in a local network:

```bash
git clone https://github.com/diegobraga92/PudimProductivity.git ~/pudimproductivity
cd ~/pudimproductivity
cp .env.example .env
docker compose up -d
```

Then open `http://<server-ip>:3000` from any device on the LAN. Ports are configurable in `.env`. The nginx frontend serves the built app and proxies `/api/` (and WebSockets) to the backend.


## What it does (for now)

- **Tasks & habits** — task lists, recurring habits with streaks and weekly heatmaps
- **Weekly planner** — with alarms: in-app toasts on web, native notifications on desktop, and local alarms on Android
- **Pomodoro focus timer** and **soundscape** (ambient loops like rain and ocean) — web + desktop
- **Recipes**
- **Media library** — movies, series, books and games, with CSV import and optional ratings pulled from OMDb / RAWG / IGDB (configured at runtime in Server Settings)
- **Real-time sync** — changes push to every client over WebSocket. The Android app is local-first (SQLite) and catches up when it reconnects. The database always wins.

## To be added

- Users and observability, likely other stuff too.

## License

MIT
