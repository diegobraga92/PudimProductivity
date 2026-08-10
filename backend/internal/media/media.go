// Package media provides direct-to-object-storage uploads via presigned URLs.
// The client requests a short-lived PUT URL, uploads the object itself, and
// stores the returned object key in the domain record (e.g. recipe.image_url).
//
// Uploads are optional at runtime: if no storage backend is configured the
// module runs in degraded mode and upload-URL requests fail with a clear error
// (see ADR 007 for the graceful-degradation matrix).
package media

import (
	"context"
	"errors"
	"time"
)

// UploadURL is a short-lived presigned PUT URL plus the object key that should
// be persisted in place of a real URL.
type UploadURL struct {
	// URL is the presigned PUT URL the client uploads to.
	URL string `json:"url"`
	// Key is the object key (no bucket) to store in the domain record.
	Key string `json:"key"`
}

// Generator issues presigned upload URLs.
type Generator interface {
	GenerateUploadURL(ctx context.Context, key, contentType string, ttl time.Duration) (*UploadURL, error)
}

// ErrNotConfigured indicates no storage backend is configured.
var ErrNotConfigured = errors.New("media: storage backend not configured")
