package sounds

import (
	"path/filepath"
	"strings"
)

// Audio content types supported by the sound library.
const (
	mimeMP3 = "audio/mpeg"
	mimeOGG = "audio/ogg"
	mimeM4A = "audio/mp4"
	mimeWAV = "audio/wav"
)

// mimeByExtension is the single source of truth for which files count as
// sounds. Seeding, the catalog's MIME values and the upload endpoint all read
// it, so the accepted formats only need to change in one place.
var mimeByExtension = map[string]string{
	".mp3": mimeMP3,
	".ogg": mimeOGG,
	".m4a": mimeM4A,
	".wav": mimeWAV,
}

// mimeForFile returns the content type for a file whose name has a known audio
// extension.
func mimeForFile(name string) (string, bool) {
	mime, ok := mimeByExtension[strings.ToLower(filepath.Ext(name))]
	return mime, ok
}
