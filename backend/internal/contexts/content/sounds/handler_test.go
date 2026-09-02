package sounds

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSoundsCatalog(t *testing.T) {
	r := chi.NewRouter()
	RegisterSoundsRoutes(r, t.TempDir(), DefaultCatalog)

	// Both the exact and the trailing-slash path must serve the catalog.
	for _, path := range []string{"/api/v1/sounds", "/api/v1/sounds/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("GET %s Cache-Control = %q, want %q", path, got, "no-store")
		}
		var body struct {
			Sounds []Sound `json:"sounds"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("catalog response is not JSON: %v", err)
		}
		if len(body.Sounds) != len(DefaultCatalog) {
			t.Fatalf("catalog has %d sounds, want %d", len(body.Sounds), len(DefaultCatalog))
		}
		if body.Sounds[0].ID == "" || body.Sounds[0].File == "" {
			t.Fatalf("catalog entry missing id/file: %+v", body.Sounds[0])
		}
	}
}

func TestSoundsGetFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rain.mp3"), []byte("fake-mp3-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	RegisterSoundsRoutes(r, dir, DefaultCatalog)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sounds/rain.mp3", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET file status = %d, want 200", rec.Code)
	}
	got, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fake-mp3-bytes" {
		t.Fatalf("GET file body = %q, want %q", got, "fake-mp3-bytes")
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("GET file Cache-Control = %q, want %q", cc, "no-cache")
	}

	// The catalog advertises cache-busting query tokens (e.g. "rain.mp3?v=2");
	// the handler must ignore the query and still serve the on-disk file.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/sounds/rain.mp3?v=2", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET versioned file status = %d, want 200", rec2.Code)
	}
	if got, _ := io.ReadAll(rec2.Body); string(got) != "fake-mp3-bytes" {
		t.Fatalf("GET versioned file body = %q, want %q", got, "fake-mp3-bytes")
	}
}

func TestSoundsRejectsTraversal(t *testing.T) {
	r := chi.NewRouter()
	RegisterSoundsRoutes(r, t.TempDir(), DefaultCatalog)

	for _, key := range []string{
		"../../etc/passwd",
		"a/../b.mp3",
		"sub/rain.mp3",
		"/abs.mp3",
		`back\slash.mp3`,
		"..",
		".",
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/sounds/"+key, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %q status = %d, want 400", key, rec.Code)
		}
	}
}

func TestSeedBundledDefaults(t *testing.T) {
	bundled := t.TempDir()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(bundled, "rain.mp3"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundled, "ocean.mp3"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SeedBundledDefaults(bundled, dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "rain.mp3"))
	if err != nil || string(data) != "a" {
		t.Fatalf("seeded file missing/corrupt: %v %q", err, data)
	}

	// Existing files must never be overwritten (operator overrides win).
	if err := os.WriteFile(filepath.Join(dir, "ocean.mp3"), []byte("override"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SeedBundledDefaults(bundled, dir); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(dir, "ocean.mp3"))
	if err != nil || string(data) != "override" {
		t.Fatalf("operator override lost: %v %q", err, data)
	}

	// Missing bundled dir is tolerated (no-op).
	if err := SeedBundledDefaults(filepath.Join(t.TempDir(), "nope"), dir); err != nil {
		t.Fatalf("missing bundled dir should be a no-op, got %v", err)
	}
}
