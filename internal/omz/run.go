package omz

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Options controls the themes browser.
type Options struct {
	ZshDir    string
	CustomDir string
}

// RunThemes opens the interactive theme browser.
func RunThemes(opts Options) error {
	zshDir, customDir := ResolveDirs(opts.ZshDir, opts.CustomDir)
	themes, err := Discover(zshDir, customDir)
	if err != nil {
		return err
	}

	if !isTerminal(os.Stdout) {
		return fmt.Errorf("tsp omz themes needs an interactive terminal")
	}

	renderer, err := NewRenderer(zshDir, customDir)
	if err != nil {
		return err
	}
	defer renderer.Close()

	_, err = tea.NewProgram(NewUI(themes, renderer), tea.WithAltScreen()).Run()
	return err
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
