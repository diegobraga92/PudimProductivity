# Sound files

The ambient **Soundscape** sounds no longer live here.

They are served as audio files (MP3 loops) by the **backend** at
`/api/v1/sounds/…` — see `backend/sounds/` and
`backend/internal/contexts/content/sounds/`. The web client fetches the sound
catalog once (`web/src/utils/soundFiles.ts`) and the Soundscape engine plays
the looping files through the same master-volume / reverb / visualizer graph
as the synthesized sounds. If a file is missing or the backend is unreachable,
the engine falls back to the in-browser synthesized version.

This directory is kept (empty) for historical reference only — do not add sound
files here.

