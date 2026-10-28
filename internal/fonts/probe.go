package fonts

import (
	"bytes"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
)

// kittyQuery is the graphics-protocol query action carrying a 1x1 RGB image.
// Supporting terminals answer with an OK status instead of drawing anything.
const kittyQuery = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"

// itermQuery asks for the size of a terminal cell. Terminals that implement
// the iTerm2 inline image protocol (iTerm2, xterm.js image addon) answer it,
// so a reply proves that images can be displayed.
const itermQuery = "\x1b]1337;ReportCellSize\x07"

// backgroundColorQuery asks for the default background color. Most terminals
// answer with an OSC 11 reply such as "rgb:ffff/ffff/ffff".
const backgroundColorQuery = "\x1b]11;?\x07"

// cellSizeQuery asks for the pixel size of a character cell (XTWINOPS CSI 16 t).
// Supported terminals answer with "CSI 6 ; height ; width t".
const cellSizeQuery = "\x1b[16t"

const probeTimeout = 100 * time.Millisecond

// terminalProbe reports the protocol the terminal answers with, and whether the
// terminal could be queried at all. It is a variable so tests can skip talking
// to the real terminal.
var terminalProbe = probeImageProtocol

// probeImageProtocol asks the terminal itself which graphics protocol it
// supports. Terminals that do not implement a protocol ignore the query, so a
// missing reply is treated as "not supported" after the short timeout.
func probeImageProtocol() (imageProtocol, bool) {
	response, queried := queryTerminal(kittyQuery, kittyResponseComplete)
	if !queried {
		return imageProtocolNone, false
	}
	if kittyQuerySucceeded(response) {
		return imageProtocolKitty, true
	}

	response, _ = queryTerminal(itermQuery, itermResponseComplete)
	if itermResponseComplete(response) {
		return imageProtocolIterm, true
	}
	return imageProtocolNone, true
}

// queryTerminal writes one query to the controlling terminal and reads the
// reply. The controlling terminal is opened separately from stdin so the TUI's
// own input stream is left untouched, and the reply is always consumed so it
// cannot leak into the TUI as keystrokes. queried is false when the terminal
// cannot be queried (no controlling terminal or it is not a TTY).
func queryTerminal(query string, complete func([]byte) bool) (response []byte, queried bool) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, false
	}
	defer tty.Close()

	fd := tty.Fd()
	if !term.IsTerminal(fd) {
		return nil, false
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, false
	}
	defer term.Restore(fd, oldState)

	if _, err := io.WriteString(tty, query); err != nil {
		return nil, false
	}
	return readResponse(tty, probeTimeout, complete), true
}

// kittyResponseComplete reports whether an APC reply terminated properly.
func kittyResponseComplete(response []byte) bool {
	start := bytes.Index(response, []byte("\x1b_G"))
	return start >= 0 && bytes.Contains(response[start:], []byte("\x1b\\"))
}

// kittyQuerySucceeded reports whether the terminal accepted the query image.
func kittyQuerySucceeded(response []byte) bool {
	return bytes.Contains(response, []byte("OK"))
}

// itermResponseComplete reports whether a ReportCellSize reply (terminated with
// BEL or ST) was received.
func itermResponseComplete(response []byte) bool {
	start := bytes.Index(response, []byte("ReportCellSize="))
	if start < 0 {
		return false
	}
	rest := response[start:]
	return bytes.Contains(rest, []byte("\x07")) || bytes.Contains(rest, []byte("\x1b\\"))
}

// terminalBackgroundColor asks the terminal for its default background color.
// Terminals without OSC 11 support simply do not reply, in which case the
// caller keeps the COLORFGBG fallback.
func terminalBackgroundColor() (color.RGBA, bool) {
	response, queried := queryTerminal(backgroundColorQuery, backgroundResponseComplete)
	if !queried {
		return color.RGBA{}, false
	}
	return parseBackgroundColor(response)
}

// backgroundResponseComplete reports whether an OSC 11 reply (terminated with
// BEL or ST) was received.
func backgroundResponseComplete(response []byte) bool {
	start := bytes.Index(response, []byte("\x1b]11;"))
	if start < 0 {
		return false
	}
	rest := response[start:]
	return bytes.Contains(rest, []byte("\x07")) || bytes.Contains(rest, []byte("\x1b\\"))
}

// parseBackgroundColor extracts a color from an OSC 11 reply. Both the rgb: and
// rgba: forms are accepted, with 1 to 4 hex digits per component.
func parseBackgroundColor(response []byte) (color.RGBA, bool) {
	start := bytes.Index(response, []byte("\x1b]11;"))
	if start < 0 {
		return color.RGBA{}, false
	}
	value := response[start+len("\x1b]11;"):]
	if end := bytes.IndexByte(value, 0x07); end >= 0 {
		value = value[:end]
	}
	if end := bytes.Index(value, []byte("\x1b\\")); end >= 0 {
		value = value[:end]
	}

	text := string(value)
	switch {
	case strings.HasPrefix(text, "rgb:"):
		text = strings.TrimPrefix(text, "rgb:")
	case strings.HasPrefix(text, "rgba:"):
		text = strings.TrimPrefix(text, "rgba:")
	default:
		return color.RGBA{}, false
	}

	parts := strings.Split(text, "/")
	if len(parts) < 3 {
		return color.RGBA{}, false
	}
	r, okR := parseColorComponent(parts[0])
	g, okG := parseColorComponent(parts[1])
	b, okB := parseColorComponent(parts[2])
	if !okR || !okG || !okB {
		return color.RGBA{}, false
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}, true
}

// parseColorComponent scales one hex component of an OSC color reply to 8 bits.
func parseColorComponent(component string) (uint8, bool) {
	if len(component) == 0 || len(component) > 4 {
		return 0, false
	}
	value, err := strconv.ParseUint(component, 16, 32)
	if err != nil {
		return 0, false
	}
	max := uint64(1)<<(4*uint(len(component))) - 1
	return uint8(value * 0xff / max), true
}

// cellSize is the pixel size of one terminal cell. width and height are zero
// when the terminal does not report pixels; aspect (height/width) is the
// fallback used to size the preview without stretching it.
type cellSize struct {
	width, height int
	aspect        float64
}

// terminalCellSize reports the terminal's real cell size in pixels. The pty's
// pixel size is used when available, otherwise the terminal is asked directly
// (CSI 16 t). A zero-size result means the size could not be determined.
func terminalCellSize() cellSize {
	if width, height := ttyCellSize(); width > 0 && height > 0 {
		if aspect := parseCellAspectValue(float64(height), float64(width)); aspect > 0 {
			return cellSize{width: width, height: height, aspect: aspect}
		}
	}
	response, queried := queryTerminal(cellSizeQuery, cellSizeResponseComplete)
	if !queried {
		return cellSize{}
	}
	return parseCellSize(response)
}

// cellSizeResponseComplete reports whether a CSI 16 t reply was received.
func cellSizeResponseComplete(response []byte) bool {
	start := bytes.Index(response, []byte("\x1b[6;"))
	return start >= 0 && bytes.Contains(response[start:], []byte("t"))
}

// parseCellSize extracts the pixel size from a "CSI 6 ; height ; width t"
// reply. Implausible values are rejected so the caller keeps its fallback.
func parseCellSize(response []byte) cellSize {
	start := bytes.Index(response, []byte("\x1b[6;"))
	if start < 0 {
		return cellSize{}
	}
	rest := response[start+len("\x1b[6;"):]
	end := bytes.IndexByte(rest, 't')
	if end < 0 {
		return cellSize{}
	}
	parts := strings.Split(string(rest[:end]), ";")
	if len(parts) < 2 {
		return cellSize{}
	}
	height, errHeight := strconv.Atoi(parts[0])
	width, errWidth := strconv.Atoi(parts[1])
	if errHeight != nil || errWidth != nil {
		return cellSize{}
	}
	aspect := parseCellAspectValue(float64(height), float64(width))
	if aspect == 0 {
		return cellSize{}
	}
	return cellSize{width: width, height: height, aspect: aspect}
}

// parseCellAspectValue validates a height/width ratio reported by a terminal.
func parseCellAspectValue(height, width float64) float64 {
	if height <= 0 || width <= 0 {
		return 0
	}
	aspect := height / width
	if aspect < 1 || aspect > 4 {
		return 0
	}
	return aspect
}
