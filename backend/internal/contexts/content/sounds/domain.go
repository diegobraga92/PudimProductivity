// Package sounds serves the Soundscape (focus) ambient sound library.
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
// SoundID in web/src/utils/audio.ts.
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
