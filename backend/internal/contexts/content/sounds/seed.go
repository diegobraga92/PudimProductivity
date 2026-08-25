package sounds

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TODO: Check if this is necessary

// SeedBundledDefaults copies the default sound library shipped inside the
// image (bundledDir) into the served directory (dir) when files are missing.
func SeedBundledDefaults(bundledDir, dir string) error {
	if bundledDir == "" || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sounds: create sounds dir %q: %w", dir, err)
	}
	entries, err := os.ReadDir(bundledDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("sounds: read bundled dir %q: %w", bundledDir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		dst := filepath.Join(dir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // already present — keep the existing (possibly overridden) file
		}
		if err := copyFile(filepath.Join(bundledDir, e.Name()), dst); err != nil {
			return fmt.Errorf("sounds: seed %q: %w", e.Name(), err)
		}
	}
	return nil
}

// copyFile copies src to dst, creating dst with 0644 permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
