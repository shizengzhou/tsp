package fonts

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Options controls font discovery.
type Options struct {
	Dirs []string
}

// RunFonts opens the interactive font browser. The chosen family name is
// printed to stdout after the program exits.
func RunFonts(opts Options) error {
	families, err := Discover(opts.Dirs...)
	if err != nil {
		return err
	}
	return runBrowser(families)
}

func runBrowser(families []Family) error {
	if !isTerminal(os.Stdout) {
		return fmt.Errorf("tsp fonts needs an interactive terminal")
	}

	ui := NewUI(families)
	ui.graphics = graphicsSupported()

	final, err := tea.NewProgram(ui, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	printChosen(final)
	return nil
}

func printChosen(final tea.Model) {
	if ui, ok := final.(*UI); ok && ui.chosen != "" {
		fmt.Println(ui.chosen)
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
