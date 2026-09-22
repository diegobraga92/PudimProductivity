## Sounds directory

This directory holds the **built-in** soundscape loops that ship inside the
backend image. On startup `SeedBundledDefaults` copies the known audio files
here into the served `SOUNDS_DIR` (default `/app/sounds`, backed by the
`soundsdata` volume). Files already present are **never overwritten**, so an
operator can override a bundled sound by dropping a file with the same name
into the volume. Non-audio files in this directory (like this README) are
ignored by the seeding step.

## Adding a built-in sound

1. Add the sound file to this dir, named `<sound-id>.mp3`.
2. Add `bundled("<sound-id>")` to `DefaultCatalog` in
   `internal/contexts/content/sounds/domain.go`.
3. Add its label and icon to `web/src/utils/soundCatalog.ts` plus translations
   in `shared/i18n/{en,pt-BR}.json`.
4. Rebuild the backend image.

## Replacing an existing sound

Bundled catalog entries carry a cache-busting token built from
`catalogVersion` (`File: "rain.mp3?v=2"`). When you replace a bundled file,
bump `catalogVersion` in `domain.go` — one constant, one line — so
previously-cached clients fetch the new bytes immediately instead of replaying
the old file.

