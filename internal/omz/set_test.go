package omz

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSetTheme applies a real theme through the oh-my-zsh CLI in a throwaway
// HOME. It is skipped when oh-my-zsh (or its CLI) is unavailable.
func TestSetTheme(t *testing.T) {
	zshBin := findZsh(t)
	zshDir, customDir := ResolveDirs("", "")
	if _, err := os.Stat(filepath.Join(zshDir, "lib", "cli.zsh")); err != nil {
		t.Skipf("omz CLI not installed: %v", err)
	}
	themes, err := Discover(zshDir, customDir)
	if err != nil {
		t.Skipf("oh-my-zsh not available: %v", err)
	}
	target, ok := pickSettableTheme(themes, customDir)
	if !ok {
		t.Skip("no theme that omz theme set can resolve")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZDOTDIR", home)

	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("ZSH_THEME=\"robbyrussell\"\nplugins=(git)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	renderer := &Renderer{zshBin: zshBin, zshDir: zshDir, customDir: customDir}
	if err := renderer.SetTheme(target); err != nil {
		t.Fatalf("SetTheme(%s) error = %v", target.Name, err)
	}

	data, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("ZSH_THEME=%q", target.Name)
	if !strings.Contains(string(data), want) {
		t.Errorf(".zshrc does not contain %q:\n%s", want, data)
	}
	if !strings.Contains(string(data), "# set by `omz`") {
		t.Errorf(".zshrc missing omz marker:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc.bck")); err != nil {
		t.Errorf("backup file missing: %v", err)
	}
}

func TestSetThemeUnknownName(t *testing.T) {
	zshBin := findZsh(t)
	zshDir, customDir := ResolveDirs("", "")
	if _, err := os.Stat(filepath.Join(zshDir, "lib", "cli.zsh")); err != nil {
		t.Skipf("omz CLI not installed: %v", err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZDOTDIR", home)

	zshrc := filepath.Join(home, ".zshrc")
	original := "ZSH_THEME=\"robbyrussell\"\nplugins=(git)\n"
	if err := os.WriteFile(zshrc, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	renderer := &Renderer{zshBin: zshBin, zshDir: zshDir, customDir: customDir}
	err := renderer.SetTheme(Theme{Name: "tsp-no-such-theme"})
	if err == nil {
		t.Fatal("SetTheme() error = nil, want theme not found")
	}
	if !strings.Contains(err.Error(), "theme not found") {
		t.Errorf("error = %q, want it to mention the missing theme", err)
	}

	data, readErr := os.ReadFile(zshrc)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != original {
		t.Errorf(".zshrc changed after a failed set:\n%s", data)
	}
}

func TestSetThemeWithoutCLI(t *testing.T) {
	renderer := &Renderer{zshBin: findZsh(t), zshDir: t.TempDir()}
	err := renderer.SetTheme(Theme{Name: "anything"})
	if err == nil {
		t.Fatal("SetTheme() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "omz CLI not found") {
		t.Errorf("error = %q, want it to mention the missing omz CLI", err)
	}
}

func TestEnvWith(t *testing.T) {
	env := envWith(
		[]string{"HOME=/real", "ZSH=/old", "PATH=/bin"},
		map[string]string{"ZSH": "/new"},
	)
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "ZSH=/old") {
		t.Errorf("env still contains overridden value:\n%s", joined)
	}
	for _, want := range []string{"ZSH=/new", "HOME=/real", "PATH=/bin"} {
		if !strings.Contains(joined, want) {
			t.Errorf("env missing %q:\n%s", want, joined)
		}
	}
}

// pickSettableTheme returns a theme that "omz theme set" can resolve, i.e. one
// in $ZSH/themes, $ZSH_CUSTOM, or $ZSH_CUSTOM/themes.
func pickSettableTheme(themes []Theme, customDir string) (Theme, bool) {
	for _, theme := range themes {
		if !theme.Custom {
			return theme, true
		}
	}
	for _, theme := range themes {
		dir := filepath.Dir(theme.Path)
		if dir == customDir || dir == filepath.Join(customDir, "themes") {
			return theme, true
		}
	}
	return Theme{}, false
}
