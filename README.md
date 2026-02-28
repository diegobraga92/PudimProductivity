# ToDo App

## Project Idea
1. Simple App with the following types of To Do lists:
    - Daily (To Do tasks that refresh everyday, for routines or habit-making)
    - Regular To Dos (Things that need to be done, once checked, that's it)
    - List of lists (To support categorized lists. E.g: Shopping lists, project ideas, etc)

2. App must be accessible through Web (for computer usage) and Android.
    - It must be able to sync between clients, even with async changes
    - Ideally, for Android, it should also have widgets for easy to use

3. The project is meant to be hosted in a local server, accessible over LAN.
    - Since the server won't always be available (e.g. left home), it must be capable of storing changes locally and then syncing when in LAN
    - Other clients must be able to be updated with the synced changes, even if done at different times

4. UX must be easy to use in both platforms, and configurable:
    - Showing checked or disappearing with them must be configurable per list type
    - For dailys, should have some simple plots, like a graph or a linked list showing how often daily tasks were done, and streaks
    - Daily must be configurable by days of the week where they will refresh (e.g. Won't bike on weekends)

5. Future functionalities
    - Pomodoro functionality for focus sessions
    - White noise options for the same reason
    - Pomodoro can use white noises (e.g. noise runs when focusing, and stops when timer's up)

## Stack
- Stack will be React (front), Rust + Axum (backend), and Kotlin (Android)
    - Considered using React Native, but dedicated Kotlin App will be more robust, and made more sense since Widgets are a requirement

## Concerns
- How to store async changes offline in Android version (no backend)?
- Will need to define a conflict resolution approach
- Show/hide approach will cause all old tasks to be saved, even if not shown (maybe add some compaction)?
- Pomodoro and white noises will not connect to tasks, so implementation should be independent

## Thing to be done (TODO)

### Core Features
- [ ] **Task Management**
  - [ ] Add, edit, delete tasks
  - [ ] Mark tasks as completed/reopened
  - [ ] Reorder tasks via drag-and-drop
  - [ ] Set due dates on tasks
  - [ ] Set recurrence patterns (daily, weekly, etc.)

- [ ] **List Management**
  - [ ] Create Todo lists (standard task lists)
  - [ ] Create Daily lists (auto-generated daily tasks)
  - [ ] Create Collection lists (groups of other lists)
  - [ ] Rename, archive, delete lists
  - [ ] Hierarchical organization (collections contain lists)

- [ ] **Daily View**
  - [ ] Automatic generation of daily tasks from recurring patterns
  - [ ] Dedicated daily task view
  - [ ] Completion tracking for daily tasks

- [ ] **Cross-Platform Support**
  - [ ] Web application (browser-based)
  - [ ] Mobile applications (iOS/Android)
  - [ ] Real-time sync across devices
  - [ ] Offline functionality with sync on reconnect

### Technical Requirements
- [ ] **Backend (Rust)**
  - [ ] Modular monolith architecture
  - [ ] RESTful API endpoints
  - [ ] Database integration (PostgreSQL/SQLite)
  - [ ] Authentication middleware
  - [ ] Domain-driven design structure

- [ ] **Frontend (React Native/TypeScript)**
  - [ ] Shared core logic between web and mobile
  - [ ] Responsive UI components
  - [ ] State management (Redux/Context)
  - [ ] API client with error handling
  - [ ] Offline data persistence

- [ ] **Performance & Reliability**
  - [ ] API response time < 200ms (95th percentile)
  - [ ] App startup time < 2 seconds
  - [ ] 99.9% uptime for core services
  - [ ] Data encryption at rest and in transit
  - [ ] Protection against common vulnerabilities

### Future Features (Roadmap)
- [ ] **Phase 2: Focus Tools**
  - [ ] Pomodoro timer with customizable intervals
  - [ ] Focus session tracking
  - [ ] Integration with task completion

- [ ] **Phase 3: Enhanced Productivity**
  - [ ] White noise/ambient sound player
  - [ ] Focus session recommendations
  - [ ] Team collaboration (shared lists)

- [ ] **Phase 4: Analytics**
  - [ ] Productivity insights and trends
  - [ ] Time tracking integration
  - [ ] Goal setting and progress tracking

 - [ ] **Phase 5: Cloud Deployment** 
  - [ ]**User Authentication**
   - [ ] User registration with email
   - [ ] Secure login/logout
   - [ ] Session management


## LAN Deployment

This project is configured so backend + web can run in Docker on a LAN server.

### Services

- Web UI: `http://192.168.3.12`
- Backend API: `http://192.168.3.12:3000`

### Manual steps to run on the server

1. Install Docker Engine + Docker Compose plugin on the server.
2. Clone this repository on the server.
3. From repo root, run:

```bash
docker compose up -d --build
```

4. Verify:

```bash
docker compose ps
curl http://192.168.3.12:3000/lists
```

5. (If needed) open firewall ports:

```bash
sudo ufw allow 80/tcp
sudo ufw allow 3000/tcp
```

### Android notes

- Android app is configured to use `http://192.168.3.12:3000/lists` for backend availability checks.
- Install APK on devices normally (available in repo releases)