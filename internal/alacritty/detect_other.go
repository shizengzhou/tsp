//go:build !windows

package alacritty

// alacrittyAncestor is only needed on Windows, where ConPTY hides the terminal
// environment; elsewhere alacritty sets TERM or the ALACRITTY_* variables.
func alacrittyAncestor() bool { return false }
