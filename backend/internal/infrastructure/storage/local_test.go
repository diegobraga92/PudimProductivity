package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
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

func TestFilesystemUploader_StorageRoundTrip(t *testing.T) {
	up, err := NewFilesystemUploader(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewFilesystemUploader: %v", err)
	}

	key := "abc-123/file.txt"
	payload := []byte("hello storage")

	if err := up.Put(context.Background(), key, "text/plain", bytes.NewReader(payload)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	f, err := up.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = f.Content.Close() }()
	got, err := io.ReadAll(f.Content)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("body = %q, want %q", got, payload)
	}
	if f.Size != int64(len(payload)) {
		t.Fatalf("size = %d, want %d", f.Size, len(payload))
	}

	if err := up.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := up.Get(context.Background(), key); err == nil {
		t.Fatal("expected ErrNotFound after delete")
	}
}
