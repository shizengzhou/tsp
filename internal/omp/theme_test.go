package omp

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTheme(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "beta.omp.json")
	writeTheme(t, dir, "alpha.omp.json")
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	themes, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(themes) != 2 {
		t.Fatalf("got %d themes, want 2", len(themes))
	}
	if themes[0].Name != "alpha" || themes[1].Name != "beta" {
		t.Errorf("names = %q, %q; want alpha, beta", themes[0].Name, themes[1].Name)
	}
	if themes[0].Path != filepath.Join(dir, "alpha.omp.json") {
		t.Errorf("path = %q", themes[0].Path)
	}
}

func TestDiscoverPrecedence(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	want := writeTheme(t, first, "shared.omp.json")
	writeTheme(t, second, "shared.omp.json")

	themes, err := Discover(first, second)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(themes) != 1 {
		t.Fatalf("got %d themes, want 1", len(themes))
	}
	if themes[0].Path != want {
		t.Errorf("path = %q, want the first directory %q", themes[0].Path, want)
	}
}

func TestDiscoverEmpty(t *testing.T) {
	if _, err := Discover(t.TempDir()); err == nil {
		t.Fatal("Discover() error = nil, want error")
	}
}

func TestResolveThemeDirsExplicit(t *testing.T) {
	dir := t.TempDir()
	dirs := ResolveThemeDirs(dir)
	if len(dirs) != 1 || dirs[0] != dir {
		t.Errorf("dirs = %v, want just %q", dirs, dir)
	}
}
