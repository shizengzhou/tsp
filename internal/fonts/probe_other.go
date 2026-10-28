//go:build !unix

package fonts

import (
	"os"
	"time"
)

// readResponse is unavailable on platforms without a pollable controlling
// terminal; there is no /dev/tty to open, so detection uses environment hints.
func readResponse(_ *os.File, _ time.Duration, _ func([]byte) bool) []byte {
	return nil
}

// ttyCellSize is unavailable without /dev/tty and TIOCGWINSZ.
func ttyCellSize() (width, height int) {
	return 0, 0
}
