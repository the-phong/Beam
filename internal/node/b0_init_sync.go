package node

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// syncInitArtifacts asks the host filesystem to synchronize the new home,
// including file contents and directory entries. The D-024 Pebble Sync write
// remains the H0 record boundary. These calls do not prove physical power-loss
// or device-cache behavior on any filesystem.
func syncInitArtifacts(home string) error {
	var dirs []string
	err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected initialization artifact type at %s", path)
		}
		return syncInitPath(path)
	})
	if err != nil {
		return fmt.Errorf("synchronize B0 initialization files: %w", err)
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := syncInitPath(dirs[i]); err != nil {
			return fmt.Errorf("synchronize B0 initialization directory: %w", err)
		}
	}
	if err := syncInitPath(filepath.Dir(home)); err != nil {
		return fmt.Errorf("synchronize parent of new B0 home: %w", err)
	}
	return nil
}

func syncInitPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s for sync: %w", path, err)
	}
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(syncErr, closeErr); err != nil {
		return fmt.Errorf("sync %s: %w", path, err)
	}
	return nil
}
