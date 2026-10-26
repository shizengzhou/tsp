package fonts

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

// imagePlan is one image placement inside the preview pane.
type imagePlan struct {
	png        []byte
	cols, rows int
}

// graphicsSupported reports whether the current terminal speaks the Kitty
// graphics protocol. TSP_GRAPHICS=1/0 overrides the detection.
func graphicsSupported() bool {
	switch strings.ToLower(os.Getenv("TSP_GRAPHICS")) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	}

	// Inside tmux the escapes reach tmux, not the outer terminal.
	if os.Getenv("TMUX") != "" {
		return false
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	if strings.Contains(strings.ToLower(os.Getenv("TERM")), "kitty") {
		return true
	}
	if os.Getenv("WEZTERM_PANE") != "" {
		return true
	}
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "kitty", "ghostty", "wezterm":
		return true
	}
	return false
}

// imageClearEscapes removes any placement and frees the image data. The
// image is addressed by its id (i=1) and placement id (p=1) so every frame
// replaces the exact same placement instead of stacking a new one.
func imageClearEscapes() string {
	return "\x1b_Ga=d,d=a,q=2\x1b\\" +
		"\x1b_Ga=d,d=I,i=1,p=1,q=2\x1b\\"
}

// imageEscapes returns the escape sequences that remove the old placement,
// free its data and transmit/display the new sample at the current cursor.
// The erase-to-end-of-line before the transmit drops text left behind by
// a previous text-fallback preview, which would otherwise show through the
// transparent image. The cursor is kept in place (C=1) and then advanced
// past the image with CUF: WezTerm binds images to terminal cells, so an
// erase-to-end-of-line issued at the image origin would wipe the placement.
func imageEscapes(plan *imagePlan) string {
	var b strings.Builder
	b.WriteString(imageClearEscapes())
	b.WriteString("\x1b[K")
	writeImage(&b, plan, 1)
	fmt.Fprintf(&b, "\x1b[%dC", plan.cols)
	return b.String()
}

// writeImage transmits and displays a PNG using the Kitty graphics protocol.
func writeImage(w io.Writer, plan *imagePlan, number uint32) {
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
