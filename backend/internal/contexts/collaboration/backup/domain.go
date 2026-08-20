// Package backup implements full export/restore of the application's
// non-sensitive data.
//
// The feature exists for disaster recovery: take a JSON snapshot of everything
// the user cares about, and later restore it — for example when the database
// is lost and a fresh instance was provisioned. The backup format is plain
// JSON (no encryption) because it deliberately excludes sensitive data
// (users, audit_log). See the backupTables registry in service.go for the
// exact include/exclude list.
package backup

import (
	"encoding/json"
	"time"
)

// BackupFormatVersion is the format written by this module. Import rejects
// backups with a different version so future format changes are handled
// explicitly instead of silently corrupting data.
const BackupFormatVersion = "1"

// BackupFile is the on-disk format of an export. Tables maps a table name to a
// JSON array of row objects (as produced by Postgres row_to_json).
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
