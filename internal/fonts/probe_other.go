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

// ttyCellAspect is unavailable without /dev/tty and TIOCGWINSZ.
func ttyCellAspect() float64 {
	return 0
}
