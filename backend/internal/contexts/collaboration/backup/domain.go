// Package backup implements full export/restore of the application's
// non-sensitive data.
package backup

import (
	"encoding/json"
	"time"
)

// BackupFormatVersion controls the current format version.
// Import rejects backups with a different version to avoid corruption.
const BackupFormatVersion = "1"

// BackupFile is the on-disk format of an export.
// Tables maps a table name to a JSON array of row objects.
type BackupFile struct {
	Version    string                     `json:"version"`
	ExportedAt time.Time                  `json:"exported_at"`
	AppVersion string                     `json:"app_version"`
	RowCounts  map[string]int             `json:"row_counts"`
	Tables     map[string]json.RawMessage `json:"tables"`
}

// ImportResult describes what a restore operation did.
type ImportResult struct {
	RestoredAt time.Time      `json:"restored_at"`
	RowCounts  map[string]int `json:"row_counts"`
}
