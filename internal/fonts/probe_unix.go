//go:build unix

package fonts

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// ttyCellAspect derives the cell shape from the pixel size the pty reports for
// the whole window. Zero means the terminal does not report pixel sizes.
func ttyCellAspect() float64 {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return 0
	}
	defer tty.Close()

	size, err := unix.IoctlGetWinsize(int(tty.Fd()), unix.TIOCGWINSZ)
	if err != nil || size.Col == 0 || size.Row == 0 || size.Xpixel == 0 || size.Ypixel == 0 {
		return 0
	}
	return parseCellAspectValue(
		float64(size.Ypixel)/float64(size.Row),
		float64(size.Xpixel)/float64(size.Col),
	)
}

// readResponse waits up to timeout for the terminal reply. Readiness is
// checked with select(2) rather than os.File deadlines: deadlines are not
// supported for a blocking character device such as /dev/tty, and the reply
// must be consumed before the TUI starts reading stdin.
func readResponse(tty *os.File, timeout time.Duration, complete func([]byte) bool) []byte {
	fd := int(tty.Fd())
	deadline := time.Now().Add(timeout)
	buffer := make([]byte, 256)
	var response []byte

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return response
		}

		readSet := unix.FdSet{}
		readSet.Set(fd)
		wait := unix.NsecToTimeval(remaining.Nanoseconds())
		ready, err := unix.Select(fd+1, &readSet, nil, nil, &wait)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil || ready == 0 {
			return response
		}

		n, err := unix.Read(fd, buffer)
		if n > 0 {
			response = append(response, buffer[:n]...)
			if complete(response) {
				return response
			}
		}
		if err != nil && !errors.Is(err, unix.EINTR) && !errors.Is(err, unix.EAGAIN) {
			return response
		}
	}
}
