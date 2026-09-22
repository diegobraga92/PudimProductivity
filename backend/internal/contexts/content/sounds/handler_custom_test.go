package sounds

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

// id3Audio returns bytes the upload sniffer recognises as an MP3 (an ID3 tag
// plus filler), so the tests do not need real audio.
func id3Audio(extraBytes int) []byte {
	b := make([]byte, 0, 3+extraBytes)
	b = append(b, "ID3"...)
	return append(b, make([]byte, extraBytes)...)
}

// multipartUpload builds a POST /sounds request. A file part is added
// only when fileField is non-empty.
func multipartUpload(t *testing.T, fields map[string]string, fileField string, body []byte) *http.Request {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if fileField != "" {
		fw, err := mw.CreateFormFile(fileField, "upload.mp3")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sounds", buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func newTestRouter(t *testing.T) (*chi.Mux, string) {
	t.Helper()
	dir := t.TempDir()
	r := chi.NewRouter()
	RegisterSoundsRoutes(r, dir, DefaultCatalog)
	return r, dir
}

func TestSoundsCreateThenListThenDelete(t *testing.T) {
	r, dir := newTestRouter(t)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, multipartUpload(t, map[string]string{"label": "My Rain", "icon": "🏴"}, "file", id3Audio(64)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var created Sound
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("created response is not JSON: %v", err)
	}
	if !created.Custom || created.Label != "My Rain" {
		t.Fatalf("unexpected created sound: %+v", created)
	}

	// The catalog grows and the uploaded file is served.
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sounds", nil))
	var catalog struct {
		Sounds []Sound `json:"sounds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Sounds) != len(DefaultCatalog)+1 {
		t.Fatalf("catalog has %d sounds, want %d", len(catalog.Sounds), len(DefaultCatalog)+1)
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sounds/"+created.File, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET uploaded file status = %d, want 200", rec.Code)
	}

	// Delete removes it from the library and from disk.
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/sounds/"+created.ID, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, created.File)); !os.IsNotExist(err) {
		t.Fatalf("audio file still on disk after delete (err = %v)", err)
	}
}

func TestSoundsUpdateAndBuiltinProtection(t *testing.T) {
	r, _ := newTestRouter(t)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, multipartUpload(t, map[string]string{"label": "Original"}, "file", id3Audio(32)))
	var created Sound
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// Rename the user's sound.
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sounds/"+created.ID, bytes.NewBufferString(`{"label":"Renamed","icon":"🌊"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var updated Sound
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Label != "Renamed" || updated.Icon != "🌊" {
		t.Fatalf("update did not apply: %+v", updated)
	}

	for _, tc := range []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodPut, "/api/v1/sounds/rain", `{"label":"x"}`, http.StatusConflict},
		{http.MethodDelete, "/api/v1/sounds/rain", "", http.StatusConflict},
		{http.MethodPut, "/api/v1/sounds/nope", `{"label":"x"}`, http.StatusNotFound},
		{http.MethodDelete, "/api/v1/sounds/nope", "", http.StatusNotFound},
	} {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s %s status = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}

func TestSoundsCreateRejectsInvalidUploads(t *testing.T) {
	r, _ := newTestRouter(t)

	cases := []struct {
		name   string
		fields map[string]string
		field  string
		body   []byte
		want   int
	}{
		{"missing label", map[string]string{"icon": "🎵"}, "file", id3Audio(16), http.StatusBadRequest},
		{"missing file", map[string]string{"label": "x"}, "", nil, http.StatusBadRequest},
		{"not audio", map[string]string{"label": "x"}, "file", []byte("%PDF-1.7 not audio"), http.StatusUnsupportedMediaType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, multipartUpload(t, tc.fields, tc.field, tc.body))
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestSoundsCreateAcceptsUploadLargerThanOneMegabyte(t *testing.T) {
	r, _ := newTestRouter(t)

	// nginx only lets 12 MB through (web/nginx.conf), so a file heavier than
	// its 1 MB default must reach the handler and be stored.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, multipartUpload(t, map[string]string{"label": "Big Rain"}, "file", id3Audio(2<<20)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestSoundsCreateRejectsOversizedUpload(t *testing.T) {
	r, _ := newTestRouter(t)

	// A file past the user-facing cap is rejected by the handler, even though
	// the request body is still small enough to parse.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, multipartUpload(t, map[string]string{"label": "Huge"}, "file", id3Audio(maxSoundFileBytes)))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize file status = %d, want 413 (body: %s)", rec.Code, rec.Body.String())
	}

	// Past the request cap: MaxBytesReader must stop the multipart parse.
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, multipartUpload(t, map[string]string{"label": "Huge"}, "file", id3Audio(maxUploadBytes)))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize request status = %d, want 413 (body: %s)", rec.Code, rec.Body.String())
	}
}
