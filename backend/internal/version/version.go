// Package version exposes the application version baked in at build time.
//
// version.txt is generated from the repository-root VERSION file (the single
// source of truth) by scripts/sync-version.mjs — do not edit it by hand. It is
// only the default: the VERSION environment variable still overrides it, so
// deployments can set their own value at runtime.
package version

import (
	_ "embed"
	"strings"
)

//go:embed version.txt
var embedded string

// Value is the application version, e.g. "1.0.0".
var Value = strings.TrimSpace(embedded)
