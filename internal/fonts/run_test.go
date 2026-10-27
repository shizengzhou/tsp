package fonts

import (
	"strings"
	"testing"
)

func TestRunBrowserRequiresTerminal(t *testing.T) {
	families := []Family{{Name: "Test", Faces: []Face{{Family: "Test", Style: "Regular"}}}}

	// stdout is not a terminal while running tests.
	err := runBrowser(families, Options{})
	if err == nil {
		t.Fatal("runBrowser() error = nil, want a terminal error")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("error = %q, want it to mention an interactive terminal", err)
	}
}

func TestFinishAppliesChosenFamily(t *testing.T) {
	ui := NewUI([]Family{{Name: "Hack", Faces: []Face{{Family: "Hack", Style: "Regular"}}}})
	ui.chosen = "Hack"
	ui.chosenSlots = []Slot{{Name: SlotNormal, Label: "Normal", Face: ui.families[0].Faces[0]}}

	var applied Choice
	err := finish(ui, Options{Apply: func(choice Choice) (string, error) {
		applied = choice
		return "/tmp/alacritty.toml", nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if applied.Family != "Hack" {
		t.Errorf("Apply received family %q, want Hack", applied.Family)
	}
	if len(applied.Slots) != 1 || applied.Slots[0].Name != SlotNormal {
		t.Errorf("Apply received slots %+v, want the normal slot", applied.Slots)
	}
}

func TestRunFontsNoFonts(t *testing.T) {
	err := RunFonts(Options{Dirs: []string{t.TempDir()}})
	if err == nil {
		t.Fatal("RunFonts() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "no fonts found") {
		t.Errorf("error = %q, want it to mention no fonts found", err)
	}
}
