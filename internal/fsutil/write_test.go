package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileCreatesWithDefaultMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	if err := WriteFile(path, []byte("theme = Demo\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "theme = Demo\n" {
		t.Errorf("content = %q", data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if mode := info.Mode().Perm(); mode != DefaultMode {
		t.Errorf("mode = %v, want %v", mode, os.FileMode(DefaultMode))
	}

	// The temporary file must not outlive the write.
	if leftovers, _ := filepath.Glob(filepath.Join(dir, ".config.tmp-*")); len(leftovers) != 0 {
		t.Errorf("temporary files left behind: %v", leftovers)
	}
}

func TestWriteFileKeepsExistingMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatalf("seed %s: %v", path, err)
	}

	if err := WriteFile(path, []byte("new\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "new\n" {
		t.Errorf("content = %q", data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("mode = %v, want 0600", mode)
	}
}

func TestWriteFileFollowsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles", "config")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(target, []byte("old\n"), 0o640); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	link := filepath.Join(dir, "config")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if err := WriteFile(link, []byte("new\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link was replaced by a regular file")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "new\n" {
		t.Errorf("target content = %q", data)
	}

	if info, err := os.Stat(target); err == nil && info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", info.Mode().Perm())
	}
}

func TestWriteFileFailsWithoutDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing", "config")

	if err := WriteFile(path, []byte("x")); err == nil {
		t.Fatal("WriteFile succeeded without the config directory")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("target exists after a failed write: %v", err)
	}
}

func TestUpdateFileCreatesFileAndDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config")

	if err := UpdateFile(path, func(raw []byte) ([]byte, error) {
		if raw != nil {
			t.Errorf("update got %q for a missing file, want nil", raw)
		}
		return []byte("new\n"), nil
	}); err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "new\n" {
		t.Errorf("content = %q", data)
	}
	if _, err := os.Stat(path + ".bck"); !os.IsNotExist(err) {
		t.Errorf("a backup was created for a file that did not exist: %v", err)
	}
}

func TestUpdateFileBacksUpPreviousContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("seed %s: %v", path, err)
	}

	if err := UpdateFile(path, func(raw []byte) ([]byte, error) {
		if string(raw) != "old\n" {
			t.Errorf("update got %q, want the previous contents", raw)
		}
		return []byte("new\n"), nil
	}); err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}

	backup, err := os.ReadFile(path + ".bck")
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backup) != "old\n" {
		t.Errorf("backup = %q, want the previous contents", backup)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "new\n" {
		t.Errorf("content = %q", data)
	}
}

func TestUpdateFileLeavesFileWhenUpdateFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("seed %s: %v", path, err)
	}

	if err := UpdateFile(path, func([]byte) ([]byte, error) {
		return nil, errors.New("boom")
	}); err == nil {
		t.Fatal("UpdateFile succeeded although update failed")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != "old\n" {
		t.Errorf("content = %q, want the file untouched", data)
	}
	if _, err := os.Stat(path + ".bck"); !os.IsNotExist(err) {
		t.Errorf("a backup was written for a failed update: %v", err)
	}
}
