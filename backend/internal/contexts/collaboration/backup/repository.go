package backup

import (
	"context"
	"errors"
)

// Sentinel errors returned by Repository.Import.
var (
	ErrInvalidBackup      = errors.New("invalid backup document")
	ErrUnsupportedVersion = errors.New("unsupported backup version")
)

// Repository is the port for full export/restore of non-sensitive data.
type Repository interface {
	// Export snapshots every backup table into a single BackupFile and returns
	// it serialized as indented JSON.
	Export(ctx context.Context, appVersion string) ([]byte, error)

	// Import validates a backup and restores it into the database, replacing
	// the current contents. On any error every change is rolled back and the
	// pre-restore data is left untouched.
	Import(ctx context.Context, data []byte) (ImportResult, error)
}
