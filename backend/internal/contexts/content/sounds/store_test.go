package sounds

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestStore builds a store over a temp dir with the default catalog.
func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir, DefaultCatalog)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s, dir
}

func TestStoreAddListUpdateDelete(t *testing.T) {
	s, dir := newTestStore(t)

	created, err := s.Add("My Rain", "🏴", strings.NewReader("fake-audio"), ".mp3", mimeMP3)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !created.Custom || created.Label != "My Rain" || created.Icon != "🏴" {
		t.Fatalf("unexpected created entry: %+v", created)
	}
	if !strings.HasPrefix(created.ID, "custom-") || created.File != created.ID+".mp3" {
		t.Fatalf("unexpected generated id/file: %+v", created)
	}
	if _, err := os.Stat(filepath.Join(dir, created.File)); err != nil {
		t.Fatalf("audio file not written: %v", err)
	}

	// The merged list is built-ins first, then the user's sounds.
	list := s.List()
	if len(list) != len(DefaultCatalog)+1 {
		t.Fatalf("List() returned %d entries, want %d", len(list), len(DefaultCatalog)+1)
	}
	if list[len(list)-1].ID != created.ID {
		t.Fatalf("custom sound not appended: %+v", list[len(list)-1])
	}

	// A fresh store over the same dir must see the persisted sound.
	reopened, err := NewStore(dir, DefaultCatalog)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := reopened.List(); len(got) != len(DefaultCatalog)+1 {
		t.Fatalf("reopened store returned %d entries, want %d", len(got), len(DefaultCatalog)+1)
	}

	updated, err := s.Update(created.ID, "Renamed", "🌊")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Label != "Renamed" || updated.Icon != "🌊" {
		t.Fatalf("update did not apply: %+v", updated)
	}
	if got := s.List()[len(DefaultCatalog)].Label; got != "Renamed" {
		t.Fatalf("updated label not listed, got %q", got)
	}

	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, created.File)); !os.IsNotExist(err) {
		t.Fatalf("audio file still present after delete (err = %v)", err)
	}
	if got := s.List(); len(got) != len(DefaultCatalog) {
		t.Fatalf("List() returned %d entries after delete, want %d", len(got), len(DefaultCatalog))
	}
}

func TestStoreProtectsBuiltinsAndReportsUnknown(t *testing.T) {
	s, _ := newTestStore(t)

	if _, err := s.Update("rain", "x", "y"); !errors.Is(err, ErrDefaultSound) {
		t.Fatalf("Update(builtin) error = %v, want ErrDefaultSound", err)
	}
	if err := s.Delete("rain"); !errors.Is(err, ErrDefaultSound) {
		t.Fatalf("Delete(builtin) error = %v, want ErrDefaultSound", err)
	}
	if _, err := s.Update("nope", "x", "y"); !errors.Is(err, ErrUnknownSound) {
		t.Fatalf("Update(unknown) error = %v, want ErrUnknownSound", err)
	}
	if err := s.Delete("nope"); !errors.Is(err, ErrUnknownSound) {
		t.Fatalf("Delete(unknown) error = %v, want ErrUnknownSound", err)
	}
}

func TestStoreSanitizesManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shared.mp3"), []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":1,"sounds":[
      {"id":"rain","label":"Shadows a built-in","file":"shared.mp3","mime":"audio/mpeg"},
      {"id":"dup","label":"First","file":"shared.mp3","mime":"audio/mpeg"},
      {"id":"dup","label":"Second","file":"shared.mp3","mime":"audio/mpeg"},
      {"id":"escape","label":"Escape","file":"../escape.mp3","mime":"audio/mpeg"},
      {"id":"wrongtype","label":"Wrong type","file":"shared.pdf","mime":"application/pdf"},
      {"id":"ghost","label":"Missing file","file":"missing.mp3","mime":"audio/mpeg"},
      {"id":"","label":"No id","file":"shared.mp3","mime":"audio/mpeg"}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, manifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(dir, DefaultCatalog)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	var customs []Sound
	for _, sound := range s.List() {
		if sound.Custom {
			customs = append(customs, sound)
		}
	}
	if len(customs) != 1 || customs[0].Label != "First" {
		t.Fatalf("expected only the first valid custom entry, got %+v", customs)
	}
}

func TestStoreCorruptManifestKeepsBuiltins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, manifestFileName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(dir, DefaultCatalog)
	if err != nil {
		t.Fatalf("a corrupt manifest must not fail startup: %v", err)
	}
	if got := s.List(); len(got) != len(DefaultCatalog) {
		t.Fatalf("List() returned %d entries, want %d", len(got), len(DefaultCatalog))
	}
}
