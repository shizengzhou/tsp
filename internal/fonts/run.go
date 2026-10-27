package fonts

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Choice is the family and the per-style faces picked in the second step.
type Choice struct {
	Family string
	Slots  []Slot
}

// Options controls font discovery and what happens after a family is chosen.
type Options struct {
	Dirs []string

	// Apply, when non-nil, receives the chosen family and its style slots
	// instead of the family name being printed to stdout. It returns the path
	// written, for the confirmation.
	Apply func(Choice) (string, error)

	// ChooseHint replaces the preview hint shown above the sample.
	ChooseHint string
}

// RunFonts opens the interactive font browser. The chosen family is printed to
// stdout, or handed to Options.Apply, after the program exits.
func RunFonts(opts Options) error {
	families, err := Discover(opts.Dirs...)
	if err != nil {
		return err
	}
	return runBrowser(families, opts)
}

func runBrowser(families []Family, opts Options) error {
	if !isTerminal(os.Stdout) {
		return fmt.Errorf("tsp fonts needs an interactive terminal")
	}

	ui := NewUI(families)
	ui.saveMode = opts.Apply != nil
	if opts.ChooseHint != "" {
		ui.chooseHint = opts.ChooseHint
	}
	ui.protocol = detectImageProtocol()
	if ui.protocol != imageProtocolNone {
		// The sample is a transparent image, so its glyphs must contrast with
		// the terminal background rather than assume a dark theme.
		if background, ok := terminalBackgroundColor(); ok {
			ui.foreground = textColorForBackground(background)
		}
		// Size the single preview row with the terminal's real cell shape;
		// terminals whose cells are not the usual 1:2 otherwise stretch the
		// sample and lose the font's proportions.
		if aspect := terminalCellAspect(); aspect > 0 {
			ui.cellAspect = aspect
		}
	}

	final, err := tea.NewProgram(ui, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	return finish(final, opts)
}

// finish prints the chosen family, or writes it out when an Apply callback is
// configured.
func finish(final tea.Model, opts Options) error {
	ui, ok := final.(*UI)
	if !ok || ui.chosen == "" {
		return nil
	}
	if opts.Apply == nil {
		fmt.Println(ui.chosen)
		return nil
	}
	path, err := opts.Apply(Choice{Family: ui.chosen, Slots: ui.chosenSlots})
	if err != nil {
		return err
	}
	fmt.Printf("saved %q to %s\n", ui.chosen, path)
	return nil
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
