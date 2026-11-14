//go:build !windows

// Package winproc reads the parent process chain on Windows only; elsewhere
// the terminal identifies itself through the environment.
package winproc

// Process is one ancestor process.
type Process struct {
	// Name is the lower-case executable name without the .exe suffix.
	Name string
	// Path is the full executable path, empty when it cannot be read.
	Path string
}

// Ancestors returns the parent processes of the current process. It is empty
// off Windows, where ConPTY does not hide the terminal environment.
func Ancestors() []Process { return nil }
