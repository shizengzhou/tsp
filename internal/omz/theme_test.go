package omz

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTheme(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("PROMPT='> '\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverThemes(t *testing.T) {
	root := t.TempDir()
	zshDir := filepath.Join(root, "oh-my-zsh")
	customDir := filepath.Join(zshDir, "custom")

	writeTheme(t, filepath.Join(zshDir, "themes", "beta.zsh-theme"))
	writeTheme(t, filepath.Join(zshDir, "themes", "alpha.zsh-theme"))
	writeTheme(t, filepath.Join(customDir, "beta.zsh-theme"))
	writeTheme(t, filepath.Join(customDir, "themes", "gamma.zsh-theme"))

	themes, err := Discover(zshDir, customDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(themes) != 3 {
		t.Fatalf("got %d themes, want 3", len(themes))
	}

	wantNames := []string{"alpha", "beta", "gamma"}
	for i, want := range wantNames {
		if themes[i].Name != want {
			t.Errorf("theme[%d] = %q, want %q", i, themes[i].Name, want)
		}
	}

	// A custom theme overrides a built-in theme with the same name.
	beta := themes[1]
	if !beta.Custom {
		t.Error("beta should be marked as custom")
	}
	if !strings.HasPrefix(beta.Path, customDir) {
		t.Errorf("beta path = %q, want under %q", beta.Path, customDir)
	}

	gamma := themes[2]
	if !gamma.Custom || filepath.Base(gamma.Path) != "gamma.zsh-theme" {
		t.Errorf("gamma = %+v, want custom gamma.zsh-theme", gamma)
	}
}

func TestDiscoverMissingBuiltinThemes(t *testing.T) {
	zshDir := t.TempDir()
	_, err := Discover(zshDir, filepath.Join(zshDir, "custom"))
	if err == nil {
		t.Fatal("Discover() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "no themes found") {
		t.Errorf("error = %q, want it to mention no themes found", err)
	}
}

func TestDiscoverIgnoresMissingCustomDir(t *testing.T) {
	zshDir := t.TempDir()
	writeTheme(t, filepath.Join(zshDir, "themes", "only.zsh-theme"))

	themes, err := Discover(zshDir, filepath.Join(zshDir, "does-not-exist"))
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(themes) != 1 || themes[0].Name != "only" {
		t.Fatalf("themes = %+v, want only builtin theme", themes)
	}
}

func TestResolveDirs(t *testing.T) {
	t.Setenv("ZSH", "/env/zsh")
	t.Setenv("ZSH_CUSTOM", "/env/custom")

	zshDir, customDir := ResolveDirs("", "")
	if zshDir != "/env/zsh" || customDir != "/env/custom" {
		t.Errorf("ResolveDirs() = %q, %q; want env values", zshDir, customDir)
	}

	zshDir, customDir = ResolveDirs("/flag/zsh", "/flag/custom")
	if zshDir != "/flag/zsh" || customDir != "/flag/custom" {
		t.Errorf("ResolveDirs() = %q, %q; want flag values", zshDir, customDir)
	}

	zshDir, customDir = ResolveDirs("/flag/zsh", "")
	if zshDir != "/flag/zsh" || customDir != "/env/custom" {
		t.Errorf("ResolveDirs() = %q, %q; want custom from env", zshDir, customDir)
	}
}

func TestThemeName(t *testing.T) {
	cases := map[string]string{
		"/themes/agnoster.zsh-theme": "agnoster",
		"af-magic.zsh-theme":         "af-magic",
	}
	for path, want := range cases {
		if got := themeName(path); got != want {
			t.Errorf("themeName(%q) = %q, want %q", path, got, want)
		}
	}
}
