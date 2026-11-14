//go:build windows

// Package gui configures Gui (gui-cli's terminal on Windows). Its config uses
// the Ghostty format, so the Ghostty config editors are reused as-is.
package gui

import (
	"path/filepath"

	"github.com/shizengzhou/tsp/internal/winproc"
)

// ancestor returns the Gui process that hosts this one, if any.
func ancestor() (winproc.Process, bool) {
	for _, process := range winproc.Ancestors() {
		if process.Name == "gui" {
			return process, true
		}
	}
	return winproc.Process{}, false
}

// RunningInGui reports whether tsp runs inside Gui. Gui is libghostty based and
// may inherit TERM=xterm-ghostty, so the parent process decides before any
// environment hint is trusted.
func RunningInGui() bool {
	_, ok := ancestor()
	return ok
}

// InstallDir is the Gui installation directory, which ships the themes, or ""
// when the process path cannot be read.
func InstallDir() string {
	process, ok := ancestor()
	if !ok || process.Path == "" {
		return ""
	}
	return filepath.Dir(process.Path)
}
