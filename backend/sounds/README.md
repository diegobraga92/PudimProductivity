## Adding sounds

1. Add the sound file to this dir, and name it `<sound-id>.mp3`, where
   `<sound-id>` matches a `SoundID` from `web/src/utils/audio.ts`.
2. Add a matching entry to `backend/internal/contexts/content/sounds/domain.go`.
3. Rebuild the backend image. The files are copied to `/app/sounds-default`
   inside the image and seeded into the served `SOUNDS_DIR` (default
   `/app/sounds`, the `soundsdata` volume) on startup. Existing files in the
   served directory are **never overwritten**, so to override a sound on a
   running deployment just drop a file with the same name into the volume.

## Replacing an existing sound

Each catalog entry carries a cache-busting token (`File: "rain.mp3?v=2"`).
When you replace a bundled sound, bump that `?v=` token in `domain.go` so
previously-cached clients fetch the new bytes immediately instead of replaying
the old file.
