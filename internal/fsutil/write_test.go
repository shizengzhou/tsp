package fsutil

import (
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
