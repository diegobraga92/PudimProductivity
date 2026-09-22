package sounds

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// manifestFileName is the user-added-sound manifest, stored inside the served
// sounds directory next to the audio files it describes.
const manifestFileName = "sounds.json"

// manifestVersion is the schema version of sounds.json.
const manifestVersion = 1

var (
	// ErrDefaultSound is returned when a caller tries to change a built-in.
	ErrDefaultSound = errors.New("sounds: built-in sounds cannot be modified")
	// ErrUnknownSound is returned when a user-added sound id does not exist.
	ErrUnknownSound = errors.New("sounds: unknown sound")
)

// manifest is the persisted record of the sounds a user has added. Built-in
// sounds are never written here, so the two sources cannot drift apart.
type manifest struct {
	Version int             `json:"version"`
	Sounds  []manifestSound `json:"sounds"`
}

// manifestSound is one user-added sound in the manifest.
type manifestSound struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Icon      string    `json:"icon"`
	File      string    `json:"file"`
	MIME      string    `json:"mime"`
	CreatedAt time.Time `json:"created_at"`
}

// sound converts a stored entry into a catalog entry.
func (m manifestSound) sound() Sound {
	return Sound{ID: m.ID, File: m.File, MIME: m.MIME, Label: m.Label, Icon: m.Icon, Custom: true}
}

// Store keeps the sounds a user has added: the audio files on disk in dir plus
// their metadata in the manifest beside them. Built-in sounds are never written
// to the manifest, so the two sources cannot drift apart. Mutations are
// serialized and the manifest is replaced atomically.
type Store struct {
	dir        string
	builtins   []Sound
	builtinIDs map[string]bool

	mu   sync.Mutex
	meta []manifestSound
}

// NewStore loads (or lazily creates) the store rooted at dir. builtins is the
// shipped catalog, used to keep user-added ids from shadowing built-in sounds.
func NewStore(dir string, builtins []Sound) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("sounds: create sounds dir %q: %w", dir, err)
	}
	builtinIDs := make(map[string]bool, len(builtins))
	for _, s := range builtins {
		builtinIDs[s.ID] = true
	}
	s := &Store{dir: dir, builtins: builtins, builtinIDs: builtinIDs}

	m, err := readManifest(filepath.Join(dir, manifestFileName))
	if err != nil {
		// A broken manifest must not take the library down: log it, serve the
		// built-ins and drop the unreadable custom entries.
		log.Warn().Err(err).Msg("sounds: ignoring unreadable sound manifest")
		return s, nil
	}
	s.meta = s.sanitize(m.Sounds)
	return s, nil
}

// List returns the built-in catalog followed by the sounds the user added, in
// the order they were added.
func (s *Store) List() []Sound {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Sound, 0, len(s.builtins)+len(s.meta))
	out = append(out, s.builtins...)
	for _, c := range s.meta {
		out = append(out, c.sound())
	}
	return out
}

// Add stores r as a new audio file and records its metadata, returning the
// created entry. The id and file name are generated here so a caller can
// neither collide with a built-in nor write outside the sounds directory.
func (s *Store) Add(label, icon string, r io.Reader, ext, mime string) (Sound, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := "custom-" + newID()
	entry := manifestSound{
		ID:        id,
		Label:     label,
		Icon:      icon,
		File:      id + ext,
		MIME:      mime,
		CreatedAt: time.Now().UTC(),
	}

	if err := writeFileAtomic(filepath.Join(s.dir, entry.File), r); err != nil {
		return Sound{}, fmt.Errorf("sounds: write %q: %w", entry.File, err)
	}
	next := append(s.meta[:len(s.meta):len(s.meta)], entry)
	if err := s.writeManifest(next); err != nil {
		// Keep disk and manifest consistent: without an entry the file would
		// be an unreachable orphan.
		_ = os.Remove(filepath.Join(s.dir, entry.File))
		return Sound{}, err
	}
	s.meta = next
	return entry.sound(), nil
}

// Update changes the name and icon of a sound the user added.
func (s *Store) Update(id, label, icon string) (Sound, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(id)
	if idx < 0 {
		return Sound{}, s.notFound(id)
	}
	next := make([]manifestSound, len(s.meta))
	copy(next, s.meta)
	next[idx].Label = label
	next[idx].Icon = icon
	if err := s.writeManifest(next); err != nil {
		return Sound{}, err
	}
	s.meta = next
	return next[idx].sound(), nil
}

// Delete removes a sound the user added, together with its audio file.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(id)
	if idx < 0 {
		return s.notFound(id)
	}

	// Remove the audio first: if that fails, the manifest still describes a
	// file that exists. A manifest entry whose file is gone is dropped the
	// next time the store loads, so a partial failure self-heals.
	if err := os.Remove(filepath.Join(s.dir, s.meta[idx].File)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("sounds: remove %q: %w", s.meta[idx].File, err)
	}
	next := append(s.meta[:idx:idx], s.meta[idx+1:]...)
	if err := s.writeManifest(next); err != nil {
		return err
	}
	s.meta = next
	return nil
}

// indexOf returns the position of a user-added sound, or -1.
func (s *Store) indexOf(id string) int {
	for i, c := range s.meta {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// notFound distinguishes a built-in (which must not be modified) from an id
// that does not exist at all.
func (s *Store) notFound(id string) error {
	if s.builtinIDs[id] {
		return ErrDefaultSound
	}
	return ErrUnknownSound
}

// sanitize drops manifest entries that could not be served safely or would
// shadow a built-in sound. Bad entries are skipped and logged rather than
// failing startup, so a hand-edited manifest degrades instead of breaking the
// library — and an entry whose audio file vanished is cleaned up implicitly.
func (s *Store) sanitize(entries []manifestSound) []manifestSound {
	out := make([]manifestSound, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		switch {
		case e.ID == "" || e.Label == "":
			log.Warn().Interface("sound", e).Msg("sounds: skipping manifest entry without an id or label")
			continue
		case s.builtinIDs[e.ID]:
			log.Warn().Str("id", e.ID).Msg("sounds: skipping manifest entry that shadows a built-in sound")
			continue
		case seen[e.ID]:
			log.Warn().Str("id", e.ID).Msg("sounds: skipping duplicate manifest entry")
			continue
		case !validateFile(e.File):
			log.Warn().Str("id", e.ID).Msg("sounds: skipping manifest entry with an unsafe file name")
			continue
		}
		if _, ok := mimeForFile(e.File); !ok {
			log.Warn().Str("id", e.ID).Str("file", e.File).Msg("sounds: skipping manifest entry with an unsupported audio file")
			continue
		}
		if _, err := os.Stat(filepath.Join(s.dir, e.File)); err != nil {
			log.Warn().Str("id", e.ID).Msg("sounds: skipping manifest entry whose audio file is missing")
			continue
		}
		if e.Icon == "" {
			e.Icon = defaultSoundIcon
		}
		if e.MIME == "" {
			e.MIME, _ = mimeForFile(e.File)
		}
		seen[e.ID] = true
		out = append(out, e)
	}
	return out
}

// readManifest loads the manifest, tolerating a missing file.
func readManifest(path string) (manifest, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return manifest{Version: manifestVersion}, nil
	}
	if err != nil {
		return manifest{}, fmt.Errorf("sounds: read manifest %q: %w", path, err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return manifest{}, fmt.Errorf("sounds: parse manifest %q: %w", path, err)
	}
	return m, nil
}

// writeManifest atomically replaces the manifest with entries.
func (s *Store) writeManifest(entries []manifestSound) error {
	if entries == nil {
		entries = []manifestSound{}
	}
	data, err := json.MarshalIndent(manifest{Version: manifestVersion, Sounds: entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("sounds: encode manifest: %w", err)
	}
	data = append(data, '\n')
	if err := writeFileAtomic(filepath.Join(s.dir, manifestFileName), bytes.NewReader(data)); err != nil {
		return fmt.Errorf("sounds: write manifest: %w", err)
	}
	return nil
}

// writeFileAtomic writes r to path through a temporary file plus a rename, so
// readers never observe a partially written file.
func writeFileAtomic(path string, r io.Reader) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	// CreateTemp uses 0600. The served files use the usual 0644.
	if err := os.Chmod(tmpName, 0o644); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// newID returns a random identifier for a user-added sound.
func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand is unavailable, fall back to the clock.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
