//go:build windows

package alacritty

import "github.com/shizengzhou/tsp/internal/winproc"

// alacrittyAncestor reports whether alacritty.exe is one of our parent
// processes. Alacritty on Windows hosts the shell through ConPTY, which resets
// TERM to xterm-256color and leaves TERM_PROGRAM and the ALACRITTY_* variables
// unset, so the environment alone cannot identify the terminal.
func alacrittyAncestor() bool {
	for _, process := range winproc.Ancestors() {
		if process.Name == "alacritty" {
			return true
		}
	}
	return false
}
