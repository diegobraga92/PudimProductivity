package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// File represents a stored object and its content.
type File struct {
	// Key is the object key (no bucket).
	Key string
	// Content is the object body. Callers must close it.
	Content io.ReadCloser
	// Size is the object size in bytes, when known.
	Size int64
}

// Storage is the domain port for reading, writing and deleting objects in the media store.
type Storage interface {
	// Get returns the object for key, or ErrNotFound when absent.
	Get(ctx context.Context, key string) (*File, error)
	// Put stores r under key with the given content type.
	Put(ctx context.Context, key, contentType string, r io.Reader) error
	// Delete removes the object for key. Deleting an absent key returns nil.
	Delete(ctx context.Context, key string) error
}

// ErrNotFound is returned by Storage.Get when the object does not exist.
var ErrNotFound = errors.New("media: object not found")

// ValidateKey rejects object keys that could escape the media root (path
// traversal) or reference non-file segments.
func ValidateKey(key string) error {
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return fmt.Errorf("unsafe key %q", key)
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("unsafe key %q", key)
		}
		for _, r := range part {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '.' && r != '-' && r != '_' {
				return fmt.Errorf("unsafe key %q", key)
			}
		}
	}
	return nil
}
