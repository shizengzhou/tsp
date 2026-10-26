package fonts

import (
	"strings"
	"testing"
)

func TestRunBrowserRequiresTerminal(t *testing.T) {
	families := []Family{{Name: "Test", Faces: []Face{{Family: "Test", Style: "Regular"}}}}

	// stdout is not a terminal while running tests.
	err := runBrowser(families)
	if err == nil {
		t.Fatal("runBrowser() error = nil, want a terminal error")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("error = %q, want it to mention an interactive terminal", err)
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
