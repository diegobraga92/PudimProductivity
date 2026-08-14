# Sound files

Optional audio files for the ambient **Soundscape** sounds.

The Soundscape engine normally synthesizes every ambient sound in the browser
using the Web Audio API ("the JS Sound API"). To use a real audio file instead
for a given sound:

1. Drop the file in this directory (`web/public/sounds/`). Any browser-playable
   format works (`.mp3`, `.ogg`, `.wav`, `.m4a`). Loops are handled by the
   engine, so the file should be a clean, loopable clip.
2. Register it in `web/src/utils/soundFiles.ts`:

   ```ts
   AMBIENT_SOUND_FILES = {
     rain: "/sounds/rain.mp3",
     ocean: "/sounds/ocean.mp3",
   };
   ```

   The key must match a `SoundID` (e.g. `white-noise`, `pink-noise`,
   `brown-noise`, `rain`, `ocean`, `wind`, `campfire`, `binaural-beat`,
   `isochronic-tone`, `meditation-bowl`, `ambient-pad`).

3. Rebuild/restart the web app. Sounds without a registered file keep the
   synthesized version — file-based and synthesized sounds can be mixed freely.

File-based sounds are routed through the same master volume / reverb /
visualizer graph as the synthesized ones, so existing controls apply to them
too.
