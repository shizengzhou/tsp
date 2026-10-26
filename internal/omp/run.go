package omp

import (
	"fmt"
	"os"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// Options controls the oh-my-posh themes browser.
type Options struct {
	ThemesDir string
	Shell     string
}

// RunThemes opens the interactive oh-my-posh theme browser.
func RunThemes(opts Options) error {
	shell := opts.Shell
	if shell == "" {
		shell = DefaultShell()
	}

	themes, err := Discover(ResolveThemeDirs(opts.ThemesDir)...)
	if err != nil {
		return err
	}

	if !isTerminal(os.Stdout) {
		return fmt.Errorf("tsp omp themes needs an interactive terminal")
	}

	profile, err := ProfilePath(shell)
	if err != nil {
		return err
	}

	renderer, err := NewRenderer(shell)
	if err != nil {
		return err
	}
	defer renderer.Close()

	apply := func(theme Theme) error { return SetTheme(profile, theme) }
	_, err = tea.NewProgram(NewUI(themes, renderer, apply, shell), tea.WithAltScreen()).Run()
	return err
}

// DefaultShell is the shell whose profile a chosen theme is written to.
func DefaultShell() string {
	if runtime.GOOS == "windows" {
		return "pwsh"
	}
	return "zsh"
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
