package fonts

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// imageProtocol is the terminal graphics protocol used for the preview.
type imageProtocol int

const (
	imageProtocolNone imageProtocol = iota
	imageProtocolKitty
	imageProtocolIterm
)

// imagePlan is one image placement inside the preview pane.
type imagePlan struct {
	png        []byte
	cols, rows int
}

// graphicsReachable reports whether graphics escapes written by the program
// reach the terminal. Console programs on Windows talk to the terminal through
// ConPTY, which normally swallows them, so only terminals that ship a ConPTY
// forwarding the escapes can show the preview (Otty does; WezTerm builds from
// before its ConPTY workarounds do not). TSP_GRAPHICS=1 forces graphics
// anywhere. It is a variable so tests can simulate platforms.
var graphicsReachable = graphicsPassthrough

// graphicsPassthrough reports whether the front-end terminal receives the
// program's graphics escapes on this platform.
func graphicsPassthrough() bool {
	if runtime.GOOS != "windows" {
		return true
	}
	// libghostty-based terminals (Otty and forks) forward the escapes through
	// their bundled ConPTY.
	program := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	return strings.Contains(program, "otty") || strings.Contains(program, "kitty") || strings.Contains(program, "ghostty")
}

// detectImageProtocol picks the graphics protocol for the current terminal.
// TSP_GRAPHICS=1/0 overrides the detection.
func detectImageProtocol() imageProtocol {
	switch strings.ToLower(os.Getenv("TSP_GRAPHICS")) {
	case "1", "true", "yes":
		if protocol := detectTerminalProtocol(); protocol != imageProtocolNone {
			return protocol
		}
		return imageProtocolKitty
	case "0", "false", "no":
		return imageProtocolNone
	}

	// Inside tmux the escapes reach tmux, not the outer terminal.
	if os.Getenv("TMUX") != "" {
		return imageProtocolNone
	}
	if !graphicsReachable() {
		return imageProtocolNone
	}
	return detectTerminalProtocol()
}

// detectTerminalProtocol asks the terminal which protocol it supports. The
// Kitty answer is authoritative. The iTerm2 inline image protocol has no
// reliable query of its own (the ReportCellSize reply is optional), so a
// recognized iTerm2-style terminal is still used when the probe stays silent.
func detectTerminalProtocol() imageProtocol {
	if protocol, queried := terminalProbe(); queried && protocol != imageProtocolNone {
		return protocol
	}
	return envImageProtocol()
}

// envImageProtocol maps known terminals to their protocol from environment
// variables. It backs up terminals that do not answer the capability query,
// notably iTerm2-style terminals that implement inline images without
// replying to ReportCellSize.
func envImageProtocol() imageProtocol {
	program := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	term := strings.ToLower(os.Getenv("TERM"))

	// iTerm2 and VS Code implement the iTerm2 inline image protocol.
	if program == "iterm.app" || program == "vscode" {
		return imageProtocolIterm
	}

	// Kitty protocol: Kitty, Ghostty, WezTerm, Otty and forks. Applications
	// that embed libghostty keep their own TERM_PROGRAM (Otty uses "otty" and
	// TERM=xterm-256color) or still advertise TERM=xterm-ghostty, so all of
	// these checks lead to the same protocol.
	if strings.Contains(program, "kitty") || strings.Contains(program, "ghostty") ||
		strings.Contains(program, "wezterm") || strings.Contains(program, "otty") {
		return imageProtocolKitty
	}
	if strings.EqualFold(os.Getenv("LC_TERMINAL"), "iTerm2") {
		return imageProtocolIterm
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("KITTY_PID") != "" {
		return imageProtocolKitty
	}
	if strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") {
		return imageProtocolKitty
	}
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" || os.Getenv("GHOSTTY_BIN_DIR") != "" {
		return imageProtocolKitty
	}
	if os.Getenv("WEZTERM_PANE") != "" {
		return imageProtocolKitty
	}
	return imageProtocolNone
}

// imageClearEscapes removes any Kitty placement and frees its data. The image
// is addressed by its id (i=1) and placement id (p=1) so every frame replaces
// the exact same placement instead of stacking a new one.
func imageClearEscapes() string {
	return "\x1b_Ga=d,d=a,q=2\x1b\\" +
		"\x1b_Ga=d,d=I,i=1,p=1,q=2\x1b\\"
}

// imageEscapes returns the escape sequences that remove the old sample and
// transmit/display the new one at the current cursor. The leading
// erase-to-end-of-line drops text left behind by a previous text-fallback
// preview, which would otherwise show through the transparent image.
func imageEscapes(protocol imageProtocol, plan *imagePlan) string {
	var b strings.Builder

	if protocol == imageProtocolIterm {
		// iTerm2 moves the cursor past the image by itself.
		b.WriteString("\x1b[K")
		writeItermImage(&b, plan)
		return b.String()
	}

	// The cursor is kept in place (C=1) and then advanced past the image with
	// CUF: WezTerm binds images to terminal cells, so an erase-to-end-of-line
	// issued at the image origin would wipe the placement.
	b.WriteString(imageClearEscapes())
	b.WriteString("\x1b[K")
	writeKittyImage(&b, plan, 1)
	fmt.Fprintf(&b, "\x1b[%dC", plan.cols)
	return b.String()
}

// writeKittyImage transmits and displays a PNG using the Kitty graphics
// protocol.
func writeKittyImage(w io.Writer, plan *imagePlan, number uint32) {
	data := base64.StdEncoding.EncodeToString(plan.png)
	const chunkSize = 4096

	first := true
	for len(data) > 0 {
		size := chunkSize
		if len(data) < size {
			size = len(data)
		}
		chunk := data[:size]
		data = data[size:]

		more := 0
		if len(data) > 0 {
			more = 1
		}

		var params string
		if first {
			params = fmt.Sprintf("a=T,f=100,i=%d,p=1,q=2,C=1,c=%d,r=%d,z=1,m=%d",
				number, plan.cols, plan.rows, more)
		} else {
			params = fmt.Sprintf("q=2,m=%d", more)
		}
		fmt.Fprintf(w, "\x1b_G%s;%s\x1b\\", params, chunk)
		first = false
	}
}

// writeItermImage transmits and displays a PNG using the iTerm2 inline image
// protocol. The image is sized in character cells and keeps its aspect ratio
// within the reserved single row.
func writeItermImage(w io.Writer, plan *imagePlan) {
	data := base64.StdEncoding.EncodeToString(plan.png)
	fmt.Fprintf(w, "\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=1:%s\x07",
		len(plan.png), plan.cols, plan.rows, data)
}
