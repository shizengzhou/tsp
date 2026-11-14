//go:build !windows

package gui

// RunningInGui is always false off Windows: Gui is a Windows terminal.
func RunningInGui() bool { return false }

// InstallDir is empty off Windows; Gui is a Windows terminal.
func InstallDir() string { return "" }
