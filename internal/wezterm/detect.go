package wezterm

import (
	"os"
	"strings"
)

// RunningInWezTerm reports whether tsp runs inside WezTerm, so commands outside
// this package can decide whether a setting can be saved to its config.
func RunningInWezTerm() bool {
	if strings.EqualFold(os.Getenv("TERM_PROGRAM"), "wezterm") {
		return true
	}
	return os.Getenv("WEZTERM_PANE") != "" || os.Getenv("WEZTERM_EXECUTABLE") != ""
}
