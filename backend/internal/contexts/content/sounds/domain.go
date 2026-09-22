// Package sounds serves the Soundscape (focus) ambient sound library.
package sounds

import "fmt"

// catalogVersion is the cache-busting token appended to every bundled sound's
// file name. Bump it when a bundled file is replaced so clients that cached the
// old bytes fetch the new ones instead of replaying stale audio.
const catalogVersion = 2

// Sound describes one entry in the ambient sound library.
type Sound struct {
	// ID is the stable identifier used by the web client (e.g. "rain").
	ID string `json:"id"`
	// File is served under /api/v1/sounds/{file}.
	File string `json:"file"`
	// MIME is the audio content type (e.g. "audio/mpeg").
	MIME string `json:"mime"`
	// Label is the name of a sound added by the user. Built-in sounds leave it
	// empty: their names are translated by the web client.
	Label string `json:"label,omitempty"`
	// Icon is the emoji of a sound added by the user.
	Icon string `json:"icon,omitempty"`
	// Custom marks sounds added by the user rather than shipped with the app.
	Custom bool `json:"custom,omitempty"`
}

// UpdateSoundRequest is the JSON body of PUT /api/v1/sounds/{soundId}.
type UpdateSoundRequest struct {
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

const (
	// maxLabelLen bounds a user-supplied sound name.
	maxLabelLen = 60
	// maxIconRunes keeps the icon an emoji rather than arbitrary text (some
	// emoji sequences, like flags with a tag, use several runes).
	maxIconRunes = 8
	// defaultSoundIcon is used when the user does not pick an emoji.
	defaultSoundIcon = "🎵"
)

// bundled builds a catalog entry for a sound shipped inside the image. Bundled
// sounds are named after their id and carry the current cache-busting token.
func bundled(name string) Sound {
	return Sound{
		ID:   name,
		File: fmt.Sprintf("%s.mp3?v=%d", name, catalogVersion),
		MIME: mimeMP3,
	}
}

// DefaultCatalog lists the sounds shipped with the app. These ids are the
// built-in identifiers the web client knows by name. Their translated labels
// and icons live in the web sound catalog / i18n dictionary.
var DefaultCatalog = []Sound{
	bundled("light-rain"),
	bundled("rain"),
	bundled("rain-and-thunder"),
	bundled("strong-rain"),
	bundled("stronger-rain"),
	bundled("fire"),
	bundled("fire-and-thunder"),
	bundled("ocean"),
}
