package themebrowser

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Run opens the interactive theme browser and returns when the user quits.
func Run(themes []Theme, opts Options, apply func(Theme) error) error {
	_, err := tea.NewProgram(NewUI(themes, opts, apply), tea.WithAltScreen()).Run()
	return err
}

// IsTerminal reports whether f is an interactive terminal.
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
