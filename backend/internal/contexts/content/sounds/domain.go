// Package sounds serves the Soundscape (focus) ambient sound library.
package sounds

// Sound describes one entry in the ambient sound library.
type Sound struct {
	// ID matches the web client's SoundID (e.g. "rain").
	ID string `json:"id"`
	// File is served under /api/v1/sounds/{file}.
	File string `json:"file"`
	// MIME is the audio content type (e.g. "audio/mpeg").
	MIME string `json:"mime"`
}

// DefaultCatalog lists the sounds shipped with the app. Every ID matches a
// SoundID in web/src/utils/audio.ts.
var DefaultCatalog = []Sound{
	{ID: "light-rain", File: "light-rain.mp3?v=2", MIME: "audio/mpeg"},
	{ID: "rain", File: "rain.mp3?v=2", MIME: "audio/mpeg"},
	{ID: "rain-and-thunder", File: "rain-and-thunder.mp3?v=2", MIME: "audio/mpeg"},
	{ID: "strong-rain", File: "strong-rain.mp3?v=2", MIME: "audio/mpeg"},
	{ID: "stronger-rain", File: "stronger-rain.mp3?v=2", MIME: "audio/mpeg"},
	{ID: "fire", File: "fire.mp3?v=2", MIME: "audio/mpeg"},
	{ID: "fire-and-thunder", File: "fire-and-thunder.mp3?v=2", MIME: "audio/mpeg"},
	{ID: "ocean", File: "ocean.mp3?v=2", MIME: "audio/mpeg"},
}
