package sounds

import (
	"fmt"
	"strings"
	"testing"
)

// The catalog is built from bundled(), so guard the pieces it contributes:
// a non-empty id/file, the shared audio MIME type and the cache-busting token.
func TestDefaultCatalogEntries(t *testing.T) {
	if len(DefaultCatalog) == 0 {
		t.Fatal("DefaultCatalog is empty")
	}
	seen := make(map[string]bool, len(DefaultCatalog))
	for _, s := range DefaultCatalog {
		if s.ID == "" || s.File == "" || s.MIME != mimeMP3 {
			t.Fatalf("unexpected catalog entry: %+v", s)
		}
		if !strings.HasSuffix(s.File, fmt.Sprintf(".mp3?v=%d", catalogVersion)) {
			t.Fatalf("entry %q is missing the cache-busting token: %q", s.ID, s.File)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate catalog id %q", s.ID)
		}
		seen[s.ID] = true
	}
}

func TestMimeForFile(t *testing.T) {
	known := map[string]string{
		"rain.mp3":    mimeMP3,
		"loop.OGG":    mimeOGG,
		"ambient.m4a": mimeM4A,
		"waves.wav":   mimeWAV,
	}
	for name, want := range known {
		got, ok := mimeForFile(name)
		if !ok || got != want {
			t.Fatalf("mimeForFile(%q) = %q, %v; want %q, true", name, got, ok, want)
		}
	}
	for _, name := range []string{"README.md", "sounds.json", "noext"} {
		if _, ok := mimeForFile(name); ok {
			t.Fatalf("mimeForFile(%q) accepted a non-audio name", name)
		}
	}
	// mimeForFile only looks at the extension.
	if got, ok := mimeForFile("a/b.mp3"); !ok || got != mimeMP3 {
		t.Fatalf("mimeForFile(%q) = %q, %v; want %q, true", "a/b.mp3", got, ok, mimeMP3)
	}
}
