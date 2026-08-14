package media

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestFilesystemUploader_GenerateUploadURL(t *testing.T) {
	up, err := NewFilesystemUploader(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewFilesystemUploader: %v", err)
	}

	key := "abc-123/pancakes.jpg"
	u, err := up.GenerateUploadURL(context.Background(), key, "image/jpeg", 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateUploadURL: %v", err)
	}
	if want := "/api/v1/media/" + key; u.URL != want {
		t.Fatalf("upload URL = %q, want %q", u.URL, want)
	}
	if u.Key != key {
		t.Fatalf("key = %q, want %q", u.Key, key)
	}

	if _, err := up.GenerateUploadURL(context.Background(), "../escape.jpg", "image/jpeg", 0); err == nil {
		t.Fatal("expected error for traversal key")
	}
}

func TestMediaHandler_PutGetRoundTrip(t *testing.T) {
	r := chi.NewRouter()
	RegisterMediaRoutes(r, t.TempDir())

	key := "abc-123/pancakes.jpg"
	payload := []byte("fake-image-bytes")

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+key, bytes.NewReader(payload))
	putReq.Header.Set("Content-Type", "image/jpeg")
	putRec := httptest.NewRecorder()
	r.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200 (body %q)", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/media/"+key, nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", getRec.Code)
	}
	got, err := io.ReadAll(getRec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("GET body = %q, want %q", got, payload)
	}
}

func TestMediaHandler_RejectsTraversal(t *testing.T) {
	r := chi.NewRouter()
	RegisterMediaRoutes(r, t.TempDir())

	for _, key := range []string{"../../evil.jpg", "a/../evil.jpg", "/abs/evil.jpg", ".."} {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+key, strings.NewReader("x"))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("PUT %q status = %d, want 400", key, rec.Code)
		}
	}
}

func TestMediaHandler_RejectsOversize(t *testing.T) {
	r := chi.NewRouter()
	RegisterMediaRoutes(r, t.TempDir())

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/media/abc-123/file.jpg",
		bytes.NewReader(make([]byte, maxUploadBytes+1)),
	)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize PUT status = %d, want 413", rec.Code)
	}
}
