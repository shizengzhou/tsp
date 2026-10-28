package ghostty

import (
	"os"
	"strings"
)

// RunningInGhostty reports whether tsp runs inside Ghostty, so commands outside
// this package can decide whether a setting can be saved to its config.
func RunningInGhostty() bool {
	if strings.EqualFold(os.Getenv("TERM_PROGRAM"), "ghostty") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(os.Getenv("TERM")), "xterm-ghostty") {
		return true
	}
	return os.Getenv("GHOSTTY_RESOURCES_DIR") != "" || os.Getenv("GHOSTTY_BIN_DIR") != ""
}
