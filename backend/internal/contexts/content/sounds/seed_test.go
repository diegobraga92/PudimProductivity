package sounds

import (
	"os"
	"path/filepath"
	"testing"
)

// Bundled non-audio files (the directory README, a future manifest) must never
// be copied into the served directory.
func TestSeedBundledDefaultsSkipsNonAudio(t *testing.T) {
	bundled := t.TempDir()
	dir := t.TempDir()
	for name, body := range map[string]string{
		"rain.mp3":    "a",
		"README.md":   "docs",
		"sounds.json": "{}",
	} {
		if err := os.WriteFile(filepath.Join(bundled, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := SeedBundledDefaults(bundled, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "rain.mp3")); err != nil {
		t.Fatalf("audio file was not seeded: %v", err)
	}
	for _, name := range []string{"README.md", "sounds.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s must not be seeded (stat err = %v)", name, err)
		}
	}
}
