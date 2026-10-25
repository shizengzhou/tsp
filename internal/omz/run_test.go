package omz

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunThemesRequiresTerminal(t *testing.T) {
	zshDir := t.TempDir()
	writeTheme(t, filepath.Join(zshDir, "themes", "only.zsh-theme"))

	// stdout is not a terminal while running tests.
	err := RunThemes(Options{ZshDir: zshDir, CustomDir: filepath.Join(zshDir, "custom")})
	if err == nil {
		t.Fatal("RunThemes() error = nil, want a terminal error")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("error = %q, want it to mention an interactive terminal", err)
	}
}

func TestRunThemesNoThemes(t *testing.T) {
	zshDir := t.TempDir()

	err := RunThemes(Options{ZshDir: zshDir})
	if err == nil {
		t.Fatal("RunThemes() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "no themes found") {
		t.Errorf("error = %q, want it to mention no themes found", err)
	}
}
