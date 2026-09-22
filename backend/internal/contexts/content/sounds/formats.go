package sounds

import (
	"bytes"
	"io"
	"mime/multipart"
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

// extensionByMIME is the inverse of mimeByExtension: the extension an accepted
// format is stored under.
var extensionByMIME = map[string]string{
	mimeMP3: ".mp3",
	mimeOGG: ".ogg",
	mimeM4A: ".m4a",
	mimeWAV: ".wav",
}

// mimeForFile returns the content type for a file whose name has a known audio
// extension.
func mimeForFile(name string) (string, bool) {
	mime, ok := mimeByExtension[strings.ToLower(filepath.Ext(name))]
	return mime, ok
}

// sniffAudioMIME identifies an audio format from its leading bytes. Bytes are
// used instead of the client-supplied name or Content-Type so a renamed file
// cannot be stored (and later served) as something it is not.
func sniffAudioMIME(head []byte) string {
	switch {
	case bytes.HasPrefix(head, []byte("ID3")):
		return mimeMP3 // MP3 carrying an ID3 tag
	case len(head) >= 2 && head[0] == 0xFF && head[1]&0xE0 == 0xE0:
		return mimeMP3 // bare MPEG frame sync
	case bytes.HasPrefix(head, []byte("OggS")):
		return mimeOGG
	case len(head) >= 12 && bytes.Equal(head[4:8], []byte("ftyp")):
		return mimeM4A // ISO base media (AAC in an MP4/M4A container)
	case len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WAVE")):
		return mimeWAV
	}
	return ""
}

// detectAudioFormat sniffs an uploaded file and returns its MIME type plus the
// extension to store it under, leaving the reader rewound for the copy.
func detectAudioFormat(f multipart.File) (mime, ext string, ok bool) {
	head := make([]byte, 16)
	n, _ := io.ReadFull(f, head)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", "", false
	}
	mime = sniffAudioMIME(head[:n])
	if mime == "" {
		return "", "", false
	}
	return mime, extensionByMIME[mime], true
}
