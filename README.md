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