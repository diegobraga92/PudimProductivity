// Package sounds serves the Soundscape ambient sound library (rain, ocean,
// fire, noise loops, …) as audio files. The web client fetches the catalog and
// plays the files as looping media elements; the Soundscape engine still falls
// back to in-browser synthesis when a file is unavailable, so this module is a
// pure enhancement — a missing file or a down backend degrades gracefully.
//
// The default loops ship inside the container image (backend/sounds →
// /app/sounds-default) and are copied into the served directory on startup, so
// operators can override individual sounds without rebuilding the image.
package sounds

// Sound describes one entry in the ambient sound library.
type Sound struct {
	// ID matches the web client's SoundID (e.g. "rain").
	ID string `json:"id"`
	// File is the audio file name served under /api/v1/sounds/{file}.
	File string `json:"file"`
	// MIME is the audio content type (e.g. "audio/mpeg").
	MIME string `json:"mime"`
}

// DefaultCatalog lists the sounds shipped with the app. Every ID matches a
// SoundID in web/src/utils/audio.ts. Add new sounds here (and drop the file
// into backend/sounds/) to expose them to every client with no frontend
// change — clients only ever see the catalog.
var DefaultCatalog = []Sound{
	{ID: "white-noise", File: "white-noise.mp3", MIME: "audio/mpeg"},
	{ID: "pink-noise", File: "pink-noise.mp3", MIME: "audio/mpeg"},
	{ID: "brown-noise", File: "brown-noise.mp3", MIME: "audio/mpeg"},
	{ID: "rain", File: "rain.mp3", MIME: "audio/mpeg"},
	{ID: "ocean", File: "ocean.mp3", MIME: "audio/mpeg"},
	{ID: "wind", File: "wind.mp3", MIME: "audio/mpeg"},
	{ID: "campfire", File: "campfire.mp3", MIME: "audio/mpeg"},
	{ID: "binaural-beat", File: "binaural-beat.mp3", MIME: "audio/mpeg"},
	{ID: "isochronic-tone", File: "isochronic-tone.mp3", MIME: "audio/mpeg"},
	{ID: "meditation-bowl", File: "meditation-bowl.mp3", MIME: "audio/mpeg"},
	{ID: "ambient-pad", File: "ambient-pad.mp3", MIME: "audio/mpeg"},
}
