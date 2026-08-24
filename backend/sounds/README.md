# Default Soundscape sound library

Loopable ambient audio files served by the backend at `/api/v1/sounds/…` and
played by the Soundscape feature (rain, ocean, fire, noise loops, binaural
beats, …).

## Origin & license

The current files are **procedurally generated placeholder loops** — every file
was produced by `scripts/generate-sounds.mjs` (no external samples), so they
are original content with no licensing restrictions. They exist to make the
backend sound-serving pipeline testable end-to-end.

**Replace them with properly mastered, license-clean loops before shipping.**
The web client degrades gracefully to the synthesized in-browser versions when
a file is missing, but the whole point of this pipeline is real recordings.

## Adding / replacing a sound

1. Drop a loop file in this directory. Name it `<sound-id>.mp3`, where
   `<sound-id>` matches a `SoundID` from `web/src/utils/audio.ts`
   (`white-noise`, `pink-noise`, `brown-noise`, `rain`, `ocean`, `wind`,
   `campfire`, `binaural-beat`, `isochronic-tone`, `meditation-bowl`,
   `ambient-pad`). MP3 at 128–192 kbps is the recommended format; WAV works but
   is much larger. Files should be **seamless loops** — the engine loops them.
2. If the sound ID is new, add a matching entry to `DefaultCatalog` in
   `backend/internal/contexts/content/sounds/domain.go` (no frontend change
   needed — clients read the catalog).
3. Rebuild the backend image. The files are copied to `/app/sounds-default`
   inside the image and seeded into the served `SOUNDS_DIR` (default
   `/app/sounds`, the `soundsdata` volume) on startup. Existing files in the
   served directory are **never overwritten**, so to override a sound on a
   running deployment just drop a file with the same name into the volume.

## Regenerating the placeholders

```bash
node scripts/generate-sounds.mjs   # requires ffmpeg on PATH
```
