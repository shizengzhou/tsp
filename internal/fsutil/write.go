// Package fsutil holds filesystem helpers shared by the config writers.
package fsutil

import (
	"errors"
	"os"
	"path/filepath"
)

// DefaultMode is the mode of a file created from scratch. Files that already
// exist keep their own mode.
const DefaultMode = 0o644

// WriteFile replaces path with data atomically: the data goes into a temporary
// file next to the target, which is then renamed over it. An interrupted or
// failed write therefore never truncates the existing file, and a reader never
// sees a half-written config. An existing file keeps its permission bits, and
// a symlink is written through to its target instead of being replaced.
func WriteFile(path string, data []byte) error {
	target := resolve(path)

	mode := os.FileMode(DefaultMode)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	// Set the mode before the rename so the target never appears with the
	// temporary file's 0600.
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, target)
}

// resolve follows a symlinked file so writing through it keeps the link, which
// dotfile managers rely on. A path that does not exist yet resolves its
// directory, so a symlinked config directory works too.
func resolve(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	if dir, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil {
		return filepath.Join(dir, filepath.Base(path))
	}
	return path
}
